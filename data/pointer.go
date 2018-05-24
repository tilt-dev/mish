package data

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/windmilleng/mish/data/db/dbpath"
)

// PointerID uniquely identifies a Pointer. A Pointer points to one Snapshot currently,
// and has a history of each Snapshot it has pointed at.
type PointerID struct {
	owner UserID

	base string

	ext PointerType
}

type PointerType string

const (
	UserPtr     PointerType = ""
	RunPtr      PointerType = "run"
	OrcPtr      PointerType = "orc"
	ArtifactPtr PointerType = "art"
	CmdDBPtr    PointerType = "cmddb" // Used to store the command db in data/cmd/cmds
)

// Parse a pointer ID or panic
func MustParsePointerID(str string) PointerID {
	id, err := ParsePointerID(str)
	if err != nil {
		panic(err)
	}
	return id
}

func ParsePointerID(str string) (PointerID, error) {
	uIndex := strings.Index(str, "/")
	owner := AnonymousID
	if uIndex != -1 {
		ownerStr := str[:uIndex]
		ownerInt, err := strconv.ParseUint(ownerStr, 16, 64)
		if err == nil {
			owner = UserID(ownerInt)
			str = str[uIndex+1:]
		} else {
			return PointerID{}, fmt.Errorf("Error parsing pointer ID: %s", str)
		}
	}

	index := strings.LastIndex(str, ".")
	if index == -1 {
		return PointerID{owner: owner, base: str, ext: UserPtr}, nil
	}
	name := str[:index]
	t := PointerType(str[index+1:])
	return PointerID{owner: owner, base: name, ext: t}, nil
}

func NewPointerID(owner UserID, name string, t PointerType) (PointerID, error) {
	err := validatePointerName(name)
	if err != nil {
		return PointerID{}, err
	}
	if t == "" {
		return PointerID{owner: owner, base: name, ext: UserPtr}, nil
	}
	return PointerID{owner: owner, base: name, ext: t}, nil
}

func MustNewPointerID(owner UserID, name string, t PointerType) PointerID {
	id, err := NewPointerID(owner, name, t)
	if err != nil {
		panic(err)
	}
	return id
}

func validatePointerName(name string) error {
	invalid := "/."
	if strings.ContainsAny(name, invalid) {
		return fmt.Errorf("Pointer names can't contain any of the following characters: %s", invalid)
	}

	return nil
}

func (id PointerID) Nil() bool {
	return id.base == ""
}

func (id PointerID) Owner() UserID {
	return id.owner
}

// Base.Ext
func (id PointerID) Local() string {
	if id.ext == "" {
		return id.base
	}
	return fmt.Sprintf("%s.%s", id.base, id.ext)
}

func (id PointerID) Base() string {
	return id.base
}

func (id PointerID) Ext() PointerType {
	return id.ext
}

// Temporary pointers are not committed to the persistent database.
func (id PointerID) IsTemporary() bool {
	return false
}

func (id PointerID) String() string {
	if id.ext == "" {
		return fmt.Sprintf("%x/%s", id.owner, id.base)
	}
	return fmt.Sprintf("%x/%s.%s", id.owner, id.base, id.ext)
}

func (id PointerID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

// The hostname and port of a machine that is writing to a pointer.
// Should only be used for a DNS address that's reachable within the cluster.
type Host string

type PointerMetadata struct {
	// The host that currently holds the writelock
	WriteHost Host
}

// PointerRev is the revision of a Pointer. Nonexistent Pointers are at rev 0,
// and each successive Set operation increments the Pointer's revision
type PointerRev int64

// A Pointer at a revision (should identify a Snapshot)
type PointerAtRev struct {
	ID  PointerID
	Rev PointerRev
}

func (p PointerAtRev) String() string {
	return fmt.Sprintf("%s:%d", p.ID, p.Rev)
}

func (p PointerAtRev) Equals(o PointerAtRev) bool {
	return p.ID == o.ID && p.Rev == o.Rev
}

func (p PointerAtRev) Since(o PointerAtRev) []PointerAtRev {
	if p.ID != o.ID {
		panic(fmt.Errorf("PointerAtRev.Since: different ids %q %q", p.ID, o.ID))
	}

	result := []PointerAtRev{}

	for r := o.Rev; r < p.Rev; r++ {
		result = append(result, PointerAtRev{ID: p.ID, Rev: r + 1})
	}

	return result
}

func PointerAtRevFromString(s string) (PointerAtRev, error) {
	splits := strings.Split(s, ":")
	if len(splits) != 2 {
		return PointerAtRev{}, fmt.Errorf("could not parse PointerAtRev from %q; expected 1 colon, got: %d", s, len(splits)-1)
	}
	i, err := strconv.Atoi(splits[1])
	if err != nil {
		return PointerAtRev{}, fmt.Errorf("could not parse PointerAtRev from %q; %s is not an int: %v", s, splits[1], err)
	}
	id, err := ParsePointerID(splits[0])
	if err != nil {
		return PointerAtRev{}, err
	}
	return PointerAtRev{ID: id, Rev: PointerRev(i)}, nil
}

func PointerAtRevsToPointerRevs(ptrAtRevs []PointerAtRev) []PointerRev {
	result := make([]PointerRev, 0, len(ptrAtRevs))
	for _, ptrAtRev := range ptrAtRevs {
		result = append(result, ptrAtRev.Rev)
	}
	return result
}

// A Pointer at a revision, resolved to the snapshot ID it points at.
type PointerAtSnapshot struct {
	ID        PointerID
	Rev       PointerRev
	SnapID    SnapshotID
	Frozen    bool
	UpdatedAt time.Time
}

func (p PointerAtSnapshot) AsPointerAtRev() PointerAtRev {
	return PointerAtRev{ID: p.ID, Rev: p.Rev}
}

func (p PointerAtSnapshot) Owner() UserID {
	return p.ID.Owner()
}

func (p PointerAtSnapshot) Nil() bool {
	return p.ID.Nil()
}

// Rev 0 of every pointer starts at the same place.
func PointerAtSnapshotZero(id PointerID) PointerAtSnapshot {
	return PointerAtSnapshot{ID: id, Rev: 0, SnapID: EmptySnapshotID}
}

// Every temp pointer starts at the same place, rev 1.
//
// This is hacking around the behavior of db.Head. The problem
// is that if you Head() a pointer that doesn't exist, it creates a new
// pointer. You can't distinguish between a pointer that doesn't exist
// and one at rev 0. We don't want this behavior for temp pointers,
// because we want to reserve the pointer. So we add a spurious rev.
func PointerAtSnapshotTempInit(id PointerID) PointerAtSnapshot {
	return PointerAtSnapshot{ID: id, Rev: 1, SnapID: EmptySnapshotID}
}

func PointerAtSnapshotsToSnapIDs(ptrAtSnaps []PointerAtSnapshot) []SnapshotID {
	result := make([]SnapshotID, 0, len(ptrAtSnaps))
	for _, ptrAtSnap := range ptrAtSnaps {
		result = append(result, ptrAtSnap.SnapID)
	}
	return result
}

func PtrEdit(cur PointerAtSnapshot, next SnapshotID) PointerAtSnapshot {
	return PointerAtSnapshot{
		ID:     cur.ID,
		Rev:    cur.Rev + 1,
		SnapID: next,
	}
}

type PointerHistoryQuery struct {
	// The pointer to query.
	PtrID PointerID

	// Only return revisions where the last recipe changed the file matched.
	Matcher *dbpath.Matcher

	// The maximum revision inclusive. If not specified, we'll use HEAD as the max revision.
	MaxRev PointerRev

	// The minimum revision inclusive. If not speficied, we'll use 0 as the min revision.
	MinRev PointerRev

	// The maximum number of results to return.
	MaxResults int32

	// By default, we return revisions in descending order.
	// If ascending is specified, we return them in ascending order.
	Ascending bool
}

// Iterator over revisions in the history of a pointer,
// to make it easier to query pointer history.
type PointerHistoryIter func() (*PointerAtRev, error)

// Return a new iterator with elements filtered out that do not match the filter function.
func (f PointerHistoryIter) Filter(filterFn func(PointerAtRev) (bool, error)) PointerHistoryIter {
	return func() (*PointerAtRev, error) {
		// In the future, we might buffer the results from f, and
		// run multiple filterFn() calls in parallel. This would speed
		// up history queries where we're filtering by path.
		for true {
			el, err := f()
			if err != nil {
				return nil, err
			}

			if el == nil {
				break
			}

			pass, err := filterFn(*el)
			if err != nil {
				return nil, err
			}

			if pass {
				return el, nil
			}
		}
		return nil, nil
	}
}

func (f PointerHistoryIter) Take(n int) ([]PointerAtRev, error) {
	result := make([]PointerAtRev, 0)

	for len(result) < n {
		el, err := f()
		if err != nil {
			return nil, err
		}

		if el == nil {
			break
		}

		result = append(result, *el)
	}

	return result, nil
}

func (f PointerHistoryIter) TakeAll() ([]PointerAtRev, error) {
	return f.Take(math.MaxInt32)
}

type PointerHistoryIterParams struct {
	MaxRev    PointerRev
	MinRev    PointerRev
	Ascending bool
}

func NewPointerHistoryIter(id PointerID, headRev PointerRev, params PointerHistoryIterParams) PointerHistoryIter {
	// Set the max rev from the params
	maxRev := headRev
	if params.MaxRev != 0 && maxRev > params.MaxRev {
		maxRev = params.MaxRev
	}

	// Set the min rev from the params
	minRev := PointerRev(1)
	if params.MinRev != 0 && minRev < params.MinRev {
		minRev = params.MinRev
	}

	if params.Ascending {
		current := minRev
		return func() (*PointerAtRev, error) {
			if current > maxRev {
				return nil, nil
			}

			result := &PointerAtRev{ID: id, Rev: current}
			current++
			return result, nil
		}
	} else {
		current := maxRev
		return func() (*PointerAtRev, error) {
			if current < minRev {
				return nil, nil
			}

			result := &PointerAtRev{ID: id, Rev: current}
			current--
			return result, nil
		}
	}
}
