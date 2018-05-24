package fss

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/golang/protobuf/proto"
	"github.com/windmilleng/mish/bridge/fs"
	"github.com/windmilleng/mish/bridge/fs/proto"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/errors"
	"github.com/windmilleng/mish/logging"
	"github.com/windmilleng/mish/os/ospath"
	"github.com/windmilleng/mish/os/temp"
	"github.com/windmilleng/mish/os/watch"
)

type TooManyWrites struct {
	path string
}

func (t TooManyWrites) Error() string {
	return fmt.Sprintf("Too many writes seen for file %s", t.path)
}

const mirrorWriteLimit = 5

// ctx: A context that should exist for the lifetime of the fs2wm.
func newFS2WM(ctx context.Context, db dbint.DB2, optimizer dbint.Optimizer, tmp *temp.TempDir) *fs2wm {
	return &fs2wm{
		ctx:                ctx,
		db:                 db,
		optimizer:          optimizer,
		tmp:                tmp,
		active:             make(map[data.PointerID]*checkin),
		change:             sync.NewCond(&sync.Mutex{}),
		mirrorErrorHandler: defaultMirrorErrorHandler,
	}
}

type fs2wm struct {
	ctx                context.Context
	db                 dbint.DB2
	optimizer          dbint.Optimizer
	tmp                *temp.TempDir
	change             *sync.Cond
	active             map[data.PointerID]*checkin
	mirrorErrorHandler func(err error)
}

func defaultMirrorErrorHandler(err error) {
	logging.Global().Errorln("checkin quitting:", err)
}

// Add a change listener that stops listening when the context is done.
func (m *fs2wm) AddChangeListener(ctx context.Context, l chan *proto.FsBridgeState) {
	oldState := m.serialize()

	go func() {
		<-ctx.Done()
		m.change.Broadcast()
	}()

	go func() {
		done := false
		oldState := oldState
		for !done {
			m.change.L.Lock()

			newState := m.serialize()
			if pb.Equal(oldState, newState) {
				m.change.Wait()
			}

			m.change.L.Unlock()

			select {
			case <-ctx.Done():
				done = true
			case l <- newState:
				oldState = newState
			}
		}
		close(l)
	}()
}

func (m *fs2wm) SetMirrorErrorHandler(f func(err error)) {
	m.mirrorErrorHandler = f
}

func (m *fs2wm) emitChange() {
	go m.change.Broadcast()
}

func (m *fs2wm) serialize() *proto.FsBridgeState {
	result := make([]*proto.FsToWmState, 0, len(m.active))
	for ptr, checkin := range m.active {
		result = append(result, &proto.FsToWmState{
			Pointer: ptr.String(),
			Path:    []byte(checkin.w.Root()),
			Matcher: ospath.MatcherD2P(checkin.w.Matcher()),
		})
	}
	return &proto.FsBridgeState{FsToWmMirrors: result}
}

func (m *fs2wm) Start(ctx context.Context, path string, ptr data.PointerID, matcher *ospath.Matcher) error {
	m.change.L.Lock()
	defer m.emitChange()
	defer m.change.L.Unlock()

	if matcher.Empty() {
		return fmt.Errorf("Mirror patterns should not be empty")
	}

	head, err := dbint.AcquireSnap(ctx, m.db, ptr)

	if err != nil {
		return fmt.Errorf("Start: can't head pointer %s: %v", ptr, err)
	}

	c := m.active[ptr]
	if c != nil {
		if ospath.MatchersEqual(matcher, c.matcher) {
			return nil // yay! we're already mirroring!
		} else {
			// oh no! we're mirroring but with a different matcher. Stop and restart.
			err := m.stopLocked(ctx, ptr)
			if err != nil {
				return fmt.Errorf("Start: can't stop old mirror of %s: %v", ptr, err)
			}
		}
	}

	return m.mirror(ctx, path, head, matcher)
}

func (m *fs2wm) Status(path string) (*proto.FsToWmState, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("Path is not absolute: %s", path)
	}

	m.change.L.Lock()
	defer m.change.L.Unlock()

	// os/watch stores real absolute paths, so we need to resolve symlinks on this path.
	path, err := ospath.RealAbs(path)
	if err != nil {
		return nil, err
	}

	state := m.serialize()
	for _, s := range state.FsToWmMirrors {
		_, ok := ospath.Child(string(s.Path), path)
		if ok {
			return s, nil
		}
	}

	return nil, nil
}

func (m *fs2wm) PointerStatus(id data.PointerID) (*proto.FsToWmState, error) {
	m.change.L.Lock()
	defer m.change.L.Unlock()

	state := m.serialize()
	for _, s := range state.FsToWmMirrors {
		if s.Pointer == id.String() {
			return s, nil
		}
	}

	return nil, nil
}

// Ensure that we've synchronized this Windmill Pointer with the current file system state.
func (m *fs2wm) FSync(ctx context.Context, ptr data.PointerID) (data.PointerAtSnapshot, error) {
	m.change.L.Lock()
	active, ok := m.active[ptr]
	m.change.L.Unlock()

	if ok {
		return active.fsync(ctx)
	} else {
		return data.PointerAtSnapshot{}, grpc.Errorf(codes.NotFound, "Pointer not found %q", ptr)
	}
}

func (m *fs2wm) mirror(ctx context.Context, path string, head data.PointerAtSnapshot, matcher *ospath.Matcher) error {
	// The first time we write to a pointer, we optimize the snapshot before writing.
	optimizeBeforeWriting := head.Rev == 0
	seenInitialSync := false
	tag := data.RecipeWTagForPointer(head.ID)
	tagProvider := watch.RecipeTagProvider(func() data.RecipeWTag {
		if optimizeBeforeWriting && !seenInitialSync {
			seenInitialSync = true
			return data.RecipeWTagTemp
		}
		return tag
	})

	if !optimizeBeforeWriting {
		// This SnapshotDir call is ~~~optimization magic~~~
		//
		// When we mirror a directory against `head`, we scan the whole directory
		// contents and diff it against the contents in `head`.
		//
		// In the common sync case, we assume the diff between `head` and the
		// current directory is small. So the workspace watcher has to download the
		// entire contents of `head` from the server to diff it against the current
		// directory.  But networks are slow!
		//
		// To trick it into not going to the network, we run SnapshotDir first. This
		// writes an optimized snapshot of the current directory into the local
		// DB. These snapshots are indexed by SHA-256 checksum. When we look up
		// `head` later, most of the contents will already be in the DB without
		// going to the network.
		//
		// The cost of writing these "preemptive" snapshots is low, because the
		// syncer is smart enough not to re-upload them if our servers already have
		// them.
		_, err := m.SnapshotDir(ctx, path, matcher, head.Owner(), data.RecipeWTagOptimal, fs.Hint{Base: data.EmptySnapshotID})
		if err != nil {
			return err
		}
	}

	w, err := watch.NewWorkspaceWatcherAndSync(path, m.db, head.SnapID, matcher, head.Owner(), tagProvider, m.tmp)
	if err != nil {
		return err
	}

	// Create a new long-lived context.
	ctx, stopFunc := context.WithCancel(m.ctx)

	c := &checkin{
		fs2wm:     m,
		db:        m.db,
		optimizer: m.optimizer,
		matcher:   matcher,

		stopFunc: stopFunc,
		ctx:      ctx,

		w: w,

		initialSnap:   head.SnapID,
		currentSnap:   head.SnapID,
		currentHead:   head,
		seenFirstSync: false,

		err:  nil,
		done: make(chan struct{}),

		syncChans: make(map[string]chan data.PointerAtSnapshot),
	}

	m.active[head.ID] = c
	go func() {
		err := c.loopUntilError()
		if err != nil {
			c.fs2wm.mirrorErrorHandler(err)
		}
	}()

	go c.fsync(context.Background())

	return nil
}

func (m *fs2wm) SnapshotDir(ctx context.Context, path string, matcher *ospath.Matcher, owner data.UserID, tag data.RecipeWTag, hint fs.Hint) (data.SnapshotID, error) {
	base := hint.Base
	if base.Nil() {
		base = data.EmptySnapshotID
	}

	opsTag := tag
	shouldOptimize := tag.Type == data.RecipeTagTypeOptimal
	if shouldOptimize {
		// We can't capture DirectoryToOps as optimal ops. That doesn't make sense.
		// So we capture them as temp ops first, then optimize later.
		opsTag = data.RecipeWTagTemp
	}
	opsEvent, err := watch.DirectoryToOpsEvent(path, m.db, matcher, base, owner, opsTag)
	if err != nil {
		return data.SnapshotID{}, err
	}

	// Create optimal recipes to sync to the hub.
	snapID := opsEvent.SnapID

	if shouldOptimize {
		_, err = m.optimizer.OptimizeSnapshot(ctx, snapID)
		if err != nil {
			return data.SnapshotID{}, err
		}
	}

	return snapID, err
}

func (m *fs2wm) Stop(ctx context.Context, ptr data.PointerID) error {
	m.change.L.Lock()
	defer m.emitChange()
	defer m.change.L.Unlock()
	return m.stopLocked(ctx, ptr)
}

// stopLocked stops a watch, assuming locks are already held. (helper function)
func (m *fs2wm) stopLocked(ctx context.Context, ptr data.PointerID) error {
	c := m.active[ptr]
	if c == nil {
		return fmt.Errorf("Stop: not mirroring into pointer %s", ptr)
	}

	c.stopFunc()

	select {
	case <-c.done:
		delete(m.active, ptr)
		return c.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *fs2wm) Shutdown(ctx context.Context) error {
	m.change.L.Lock()
	defer m.change.L.Unlock()

	for _, c := range m.active {
		c.stopFunc()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.done:
			// Shutdown shouldn't affect serialization state,
			// so don't delete the checkin from the map.
			err := c.err
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type checkin struct {
	fs2wm *fs2wm

	// fields for communication between db/server request goroutines and loop goroutine
	stopFunc context.CancelFunc //stopFunc allows stopping this checkin
	err      error              // the error this encountered (if any); written by loop; read by req after done is closed
	done     chan struct{}      // a channel to wait for this checkin to be finished

	// fields for loop goroutine
	db        dbint.DB2
	optimizer dbint.Optimizer
	matcher   *ospath.Matcher

	ctx context.Context // how we know we should stop

	// for closing the watcher
	w *watch.WorkspaceWatcher

	// current state; only accessed by goroutine
	currentHead data.PointerAtSnapshot

	// If false, we haven't seen the first SyncOp yet. This
	// means we're still uploading the directory contents.
	// We don't want to edit the pointer until we've seen the contents.
	seenFirstSync bool
	initialSnap   data.SnapshotID

	// current snap when we're not editing the pointer yet.
	currentSnap data.SnapshotID

	syncChans map[string]chan data.PointerAtSnapshot
	mu        sync.Mutex
}

func (c *checkin) loopUntilError() (err error) {
	fileHeatMap := map[string]int{}
	defer func() {
		c.w.Close()
		c.mu.Lock()
		c.err = err
		c.mu.Unlock()
		close(c.done)

		go func() {
			// drain
			for _ = range c.w.Events {
			}
		}()
	}()

	t := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-t.C:
			fileHeatMap = map[string]int{}
		case <-c.ctx.Done():
			return

		case event, ok := <-c.w.Events:
			if !ok {
				return nil
			}

			switch event := event.(type) {
			case watch.WatchErrEvent:
				return event.Err

			case watch.WatchOpsEvent:
				filesSeenForOp := map[string]bool{}
				for _, op := range event.Ops {
					fileOp, ok := op.(data.FileOp)
					if !ok {
						continue
					}
					fp := fileOp.FilePath()
					if !filesSeenForOp[fp] {
						fileHeatMap[fp]++
						if fileHeatMap[fp] > mirrorWriteLimit {
							return TooManyWrites{path: fp}
						}
					}
					filesSeenForOp[fp] = true
				}
				err := c.handleOpsEvent(event)
				if err != nil {
					return err
				}

			case watch.WatchSyncEvent:
				err := c.handleSyncEvent(event)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("Unknown event type: %T", event)
			}
		}
	}
}

func (c *checkin) fsync(ctx context.Context) (data.PointerAtSnapshot, error) {
	token := fmt.Sprintf("fsync-file-%d", rand.Uint64())
	ch := make(chan data.PointerAtSnapshot)

	c.mu.Lock()
	err := c.err
	if err != nil {
		c.mu.Unlock()
		return data.PointerAtSnapshot{}, err
	}

	c.syncChans[token] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.syncChans, token)
		c.mu.Unlock()
	}()

	err = c.w.FSync(ctx, token)
	if err != nil {
		return data.PointerAtSnapshot{}, errors.Propagatef(err, "fs2wm#fsync(%s)", token)
	}

	// Wait until the SyncChan has been flushed through the op pipeline.
	select {
	case head := <-ch:
		return head, nil
	case <-ctx.Done():
		return data.PointerAtSnapshot{}, ctx.Err()
	case <-c.done:
		return data.PointerAtSnapshot{}, c.err
	}
}

func (c *checkin) handleSyncEvent(event watch.WatchSyncEvent) error {
	c.mu.Lock()
	ch := c.syncChans[event.Token]
	c.mu.Unlock()

	head := c.currentHead

	if !c.seenFirstSync {
		c.seenFirstSync = true

		// Optimize the snapshot first before setting the pointer,
		// so lookups will be fast...
		_, err := c.optimizer.OptimizeSnapshot(c.ctx, c.currentSnap)
		if err != nil {
			return err
		}

		if c.currentSnap != c.initialSnap {
			// ...but set the pointer to the incremental snapshot ID.
			// db.PathsChanged will use incremental snapshot IDs to trigger
			// op-based PathsChanged calculation.
			newHead := data.PtrEdit(head, c.currentSnap)
			err = c.db.Set(c.ctx, newHead)
			if err != nil {
				return err
			}
			c.currentHead = newHead
			c.currentSnap = newHead.SnapID
			head = newHead
		}
	}

	go func() {
		if ch != nil {
			ch <- head
			close(ch)
		}
	}()

	return nil
}

func (c *checkin) handleOpsEvent(event watch.WatchOpsEvent) error {
	if c.seenFirstSync {
		newHead := data.PtrEdit(c.currentHead, event.SnapID)
		err := c.db.Set(c.ctx, newHead)
		if err != nil {
			return err
		}
		c.currentHead = newHead
	}
	c.currentSnap = event.SnapID
	return nil
}
