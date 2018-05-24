package storages

import (
	"context"
	"crypto/sha256"
	"sort"
	"sync"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/logging"
)

type IDer interface {
	NewID(onwerID data.UserID, r data.Recipe) data.SnapshotID
}

// An IDer that simply gives incrementing IDs.
type incrementingIDer struct {
	prefix data.SnapshotPrefix

	mu     sync.Mutex
	nextID int64
}

func NewIncrementingIDer(prefix data.SnapshotPrefix) IDer {
	return &incrementingIDer{prefix: prefix}
}

func (i *incrementingIDer) NewID(ownerID data.UserID, r data.Recipe) data.SnapshotID {
	i.mu.Lock()
	defer i.mu.Unlock()
	n := i.nextID
	i.nextID++

	return i.prefix.NewID(ownerID, n)
}

// An IDer that uses content-based shas for recipes.
type contentIDer struct {
	fallback IDer
}

func NewContentIDer(fallback IDer) IDer {
	return &contentIDer{fallback: fallback}
}

func (ci *contentIDer) NewID(ownerID data.UserID, r data.Recipe) data.SnapshotID {
	switch op := r.Op.(type) {
	case *data.WriteFileOp:
		// Only WriteFileOps with empty paths and no inputs have a content-addressable ID.
		if allInputsEmpty(r.Inputs) && op.Path == "" {
			hash := sha256.New()
			hash.Write(op.Data.InternalByteSlice())
			if op.Executable {
				hash.Write([]byte{1})
			}
			if op.Type != data.FileRegular {
				// We add 2 so that this never conflicts with Executable.
				hash.Write([]byte{2 + byte(op.Type)})
			}
			return data.ContentID(ownerID, hash.Sum([]byte{}))
		}
	case *data.DirOp:
		if len(op.Names) == 0 {
			return data.EmptySnapshotID
		}

		// A DirOp is content addressable if all its inputs are content-addressible,
		// owned by the same user, and sorted by name.
		if allInputsContent(r.Inputs) &&
			allInputsOwnedBy(r.Inputs, ownerID) &&
			len(r.Inputs) == len(op.Names) &&
			sort.StringsAreSorted(op.Names) {
			hash := sha256.New()
			for i, id := range r.Inputs {
				// Only use the local part of the snapshot ID, so that the owner
				// doesn't get mixed into the content hash.
				b, err := id.Hash()
				if err != nil {
					logging.With(context.TODO()).Errorf("ContentIDer: %v", err)
					return ci.fallback.NewID(ownerID, r)
				}
				hash.Write(b)
				hash.Write([]byte(op.Names[i]))
			}
			return data.ContentID(ownerID, hash.Sum([]byte{}))
		}
	}

	return ci.fallback.NewID(ownerID, r)
}

func allInputsEmpty(inputs []data.SnapshotID) bool {
	for _, id := range inputs {
		if !id.IsEmptyID() {
			return false
		}
	}
	return true
}

func allInputsOwnedBy(inputs []data.SnapshotID, ownerID data.UserID) bool {
	for _, id := range inputs {
		// The empty ID is "owned" by everyone and allowed to be in a snapshot.
		if id.Owner() != ownerID && !id.IsEmptyID() {
			return false
		}
	}
	return true
}

func allInputsContent(inputs []data.SnapshotID) bool {
	for _, id := range inputs {
		if !id.IsContentID() {
			return false
		}
	}
	return true
}
