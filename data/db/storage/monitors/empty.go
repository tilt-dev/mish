package monitors

import (
	"context"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/storage"
)

type EmptyLockMonitor struct{}

func NewEmptyLockMonitor() EmptyLockMonitor {
	return EmptyLockMonitor{}
}

func NewLockMonitorForTesting() storage.PointerLockMonitor {
	return NewEmptyLockMonitor()
}

func (m EmptyLockMonitor) IsActivelyHeld(ctx context.Context, id data.PointerID) (bool, error) {
	return false, nil
}

func (m EmptyLockMonitor) IsLockLost(ctx context.Context, id data.PointerID) (bool, error) {
	return false, nil
}

var _ storage.PointerLockMonitor = EmptyLockMonitor{}
