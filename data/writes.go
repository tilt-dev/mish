package data

import (
	"fmt"
)

type Write interface {
	IsWrite()
}

type CreateSnapshotWrite struct {
	Recipe StoredRecipe
}

type SetPointerWrite struct {
	Next PointerAtSnapshot
}

type AcquirePointerWrite struct {
	ID   PointerID
	Host Host
}

func (w CreateSnapshotWrite) IsWrite() {}
func (w SetPointerWrite) IsWrite()     {}
func (w AcquirePointerWrite) IsWrite() {}

type ClientType string

func (t ClientType) SnapshotType() SnapshotType {
	return SnapshotType(t)
}

type ClientNonce string

type WriteQueueSize struct {
	// Recipes waiting to be written.
	WaitingWrites int

	// The maximum number of recipes recently in the queue.
	TotalWrites int
}

// Split writes into two groups.
// CreateSnapshot has much weaker consistency requirements than SetPointerWrites,
// so is often useful to write separately.
func SplitWritesTyped(writes []Write) ([]CreateSnapshotWrite, []SetPointerWrite, []AcquirePointerWrite, error) {
	// Group writes by type.
	recipes := make([]CreateSnapshotWrite, 0, len(writes))
	pointers := make([]SetPointerWrite, 0, len(writes))
	acquires := make([]AcquirePointerWrite, 0, len(writes))
	for _, w := range writes {
		switch w := w.(type) {
		case CreateSnapshotWrite:
			recipes = append(recipes, w)

		case SetPointerWrite:
			pointers = append(pointers, w)

		case AcquirePointerWrite:
			acquires = append(acquires, w)

		default:
			return nil, nil, nil, fmt.Errorf("SplitWrites: unexpected Write type %T %v", w, w)
		}
	}
	return recipes, pointers, acquires, nil
}

// ARGH GENERICS. This is the same as above, but with generic write arrays.
func SplitWrites(writes []Write) ([]Write, []Write, []Write, error) {
	// Group writes by type.
	recipes := make([]Write, 0, len(writes))
	pointers := make([]Write, 0, len(writes))
	acquires := make([]Write, 0, len(writes))
	for _, w := range writes {
		switch w := w.(type) {
		case CreateSnapshotWrite:
			recipes = append(recipes, w)

		case SetPointerWrite:
			pointers = append(pointers, w)

		case AcquirePointerWrite:
			acquires = append(acquires, w)

		default:
			return nil, nil, nil, fmt.Errorf("SplitWrites: unexpected Write type %T %v", w, w)
		}
	}
	return recipes, pointers, acquires, nil
}
