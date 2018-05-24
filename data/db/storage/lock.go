package storage

import (
	"context"

	"github.com/windmilleng/mish/data"
)

type PointerLocker interface {
	// Returns whether the current process holds the lock
	IsHoldingPointer(ctx context.Context, id data.PointerID) (bool, error)
}

type PointerLockMonitor interface {
	// Returns whether a living machine holds the lock.
	//
	// For this to return true, process A must think it has the lock AND
	// the main datastore must agree that process A holds the lock, for some A.
	IsActivelyHeld(ctx context.Context, id data.PointerID) (bool, error)

	// Returns true if the database says a living machine holds the lock,
	// but we can't confirm it.
	//
	// For this to return true, the main datastore must think process A holds the lock,
	// but process A is either dead or disagrees.
	IsLockLost(ctx context.Context, id data.PointerID) (bool, error)
}
