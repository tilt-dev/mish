package storages

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/windmilleng/mish/data"
)

var InvalidTargetError = errors.New("invalid target snapshot ID")
var FrozenWriteError = errors.New("pointer is frozen")

// The data for one pointer.
//
// PointerAtSnapshots are stored in sorted order but are not necessarily
// complete (because they may be backfilled locally from the storage server).
type pointerData struct {
	history  []data.PointerAtSnapshot
	metadata data.PointerMetadata
	// watchers are waiting for this pointer to be updated, when it is, close each watcher
	watchers []chan struct{}
}

func (d *pointerData) head() data.PointerAtSnapshot {
	r := d.history[len(d.history)-1]
	return r
}

func (d *pointerData) append(n data.PointerAtSnapshot) error {
	l := len(d.history)
	head := d.history[l-1]
	if head.Frozen {
		return FrozenWriteError
	}
	d.history = append(d.history, n)
	for _, w := range d.watchers {
		close(w)
	}

	d.watchers = nil
	return nil
}

func (d *pointerData) insert(n data.PointerAtSnapshot) error {
	l := len(d.history)
	head := d.history[l-1]
	insertAtEnd := l == 0 || head.Rev < n.Rev
	if insertAtEnd {
		return d.append(n)
	}

	// Search() requires that d.history be in sorted order,
	// and f(i) == true implies f(i+1) == true, and finds the smallest
	// index satisfying the search criteria.
	index := sort.Search(l, func(i int) bool { return d.history[i].Rev >= n.Rev })
	if index == l {
		// If index == l,
		// then that implies d.history[i].Rev < n.Rev for all i,
		// which implies that `insertAtEnd` must be true and we can't have hit this codepath.
		// The only way this could happen is if pointerData is in a non-sorted state.
		return fmt.Errorf("pointerData consistency error")
	}

	rev := d.history[index].Rev
	if rev == n.Rev {
		return nil
	}

	d.history = append(d.history[0:index], append([]data.PointerAtSnapshot{n}, d.history[index:]...)...)
	return nil
}

type MemoryPointers struct {
	mu   sync.RWMutex
	data map[data.PointerID]*pointerData
}

func NewMemoryPointers() *MemoryPointers {
	return &MemoryPointers{
		data: make(map[data.PointerID]*pointerData),
	}
}

func (p *MemoryPointers) getOrMake(id data.PointerID) *pointerData {
	d := p.data[id]
	if d == nil {
		p.data[id] = newPtrData(id)
	}
	return p.data[id]
}

func newPtrData(id data.PointerID) *pointerData {
	return &pointerData{
		history: []data.PointerAtSnapshot{data.PointerAtSnapshotZero(id)},
	}
}

func (p *MemoryPointers) MakeTemp(ctx context.Context, userID data.UserID, prefix string, t data.PointerType) (data.PointerAtSnapshot, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	i := 1 + rand.Intn(10)
	for {
		k, err := data.NewPointerID(userID, fmt.Sprintf("%s-%d", prefix, i), t)
		if err != nil {
			return data.PointerAtSnapshot{}, err
		}
		if _, ok := p.data[k]; !ok {
			pData := newPtrData(k)
			head := data.PointerAtSnapshotTempInit(k)
			pData.history = append(pData.history, head)
			p.data[k] = pData
			return head, nil
		}

		// Generate the next digit
		i = 10*i + 1 + rand.Intn(10)
	}
}

func (p *MemoryPointers) AcquirePointer(ctx context.Context, id data.PointerID) (data.PointerAtRev, error) {
	return p.AcquirePointerWithHost(ctx, id, "")
}

func (p *MemoryPointers) AcquirePointerWithHost(ctx context.Context, id data.PointerID, host data.Host) (data.PointerAtRev, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	d := p.getOrMake(id)
	d.metadata = data.PointerMetadata{WriteHost: host}
	return d.head().AsPointerAtRev(), nil
}

func (p *MemoryPointers) PointerMetadata(ctx context.Context, id data.PointerID) (data.PointerMetadata, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	d := p.data[id]
	if d == nil {
		return data.PointerMetadata{}, grpc.Errorf(codes.NotFound, "Pointer not found: %q", id)
	}
	return d.metadata, nil
}

func (p *MemoryPointers) Head(ctx context.Context, id data.PointerID) (data.PointerAtRev, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	d := p.data[id]
	if d == nil {
		return data.PointerAtRev{}, grpc.Errorf(codes.NotFound, "Pointer not found: %q", id)
	}

	return d.head().AsPointerAtRev(), nil
}

func (p *MemoryPointers) Get(ctx context.Context, at data.PointerAtRev) (data.PointerAtSnapshot, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	d := p.data[at.ID]
	if d == nil {
		return data.PointerAtSnapshot{}, grpc.Errorf(codes.NotFound, "Unknown pointer: %v", at.ID)
	}

	l := len(d.history)
	index := sort.Search(l, func(i int) bool { return d.history[i].Rev >= at.Rev })
	if index == l {
		return data.PointerAtSnapshot{}, grpc.Errorf(codes.NotFound, "Missing pointer rev: %v", at)
	}

	ptrAtSnap := d.history[index]
	if ptrAtSnap.Rev != at.Rev {
		return data.PointerAtSnapshot{}, grpc.Errorf(codes.NotFound, "Missing pointer rev: %v", at)
	}
	return ptrAtSnap, nil
}

// TODO(nick): Add safeguards to make sure that a pointer is temporary iff all snapshot paths
// are temporary.
func (p *MemoryPointers) Set(ctx context.Context, next data.PointerAtSnapshot) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	d := p.data[next.ID]
	if d == nil {
		return grpc.Errorf(codes.NotFound, "Unknown pointer: %v %+v", next.ID, p.data)
	}

	head := d.head()
	if next.Rev != (head.Rev + 1) {
		return fmt.Errorf("Stale write to pointer (%q, %d). Currently at: %d.", next.ID, next.Rev, head.Rev)
	}

	if next.UpdatedAt.IsZero() {
		next.UpdatedAt = time.Now()
	}

	if head.UpdatedAt.After(next.UpdatedAt) {
		return fmt.Errorf("Stale write to pointer. UpdatedAt %v not after %v", next.UpdatedAt, head.UpdatedAt)
	}

	if next.SnapID.Nil() {
		// TODO(dbentley): also would be good to check we have the new snapshot in storage
		return InvalidTargetError
	}

	if head.Frozen {
		return FrozenWriteError
	}

	return d.append(next)
}

func (p *MemoryPointers) SetExisting(ctx context.Context, ptrAtSnap data.PointerAtSnapshot) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	d := p.getOrMake(ptrAtSnap.ID)

	if ptrAtSnap.SnapID.Nil() {
		// TODO(dbentley): also would be good to check we have the new snapshot in storage
		return InvalidTargetError
	}

	return d.insert(ptrAtSnap)
}

func (p *MemoryPointers) Wait(ctx context.Context, last data.PointerAtRev) error {
	p.mu.Lock()
	locked := true
	defer func() {
		if locked {
			p.mu.Unlock()
		}
	}()
	d := p.getOrMake(last.ID)

	lastRev := data.PointerRev(last.Rev)
	currentRev := d.head().Rev
	if lastRev > currentRev {
		// they couldn't have seen last; it's in the future!
		return fmt.Errorf("Invalid wait on pointer: %v", last)
	}

	if lastRev < currentRev {
		// a write has already happened, return immediately
		return nil
	}

	ch := make(chan struct{})
	d.watchers = append(d.watchers, ch)
	locked = false
	p.mu.Unlock()
	// wait for either a write or our context to finish
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *MemoryPointers) ActivePointerIDs(ctx context.Context, userID data.UserID, types []data.PointerType) ([]data.PointerID, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	typeSet := make(map[data.PointerType]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}

	keys := make([]data.PointerID, 0, len(p.data))
	for k := range p.data {
		isVisible := k.Owner() == userID || userID == data.RootID
		if isVisible && typeSet[k.Ext()] {
			keys = append(keys, k)
		}
	}
	return keys, nil
}
