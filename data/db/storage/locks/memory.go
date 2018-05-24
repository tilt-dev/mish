package locks

import (
	"context"
	"sync"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/storage"
)

type MemoryPointerLocker struct {
	locks map[data.PointerID]bool
	mu    sync.RWMutex
}

func NewMemoryPointerLocker() *MemoryPointerLocker {
	return &MemoryPointerLocker{
		locks: make(map[data.PointerID]bool),
	}
}

func (l *MemoryPointerLocker) HoldPointer(ptr data.PointerID) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.locks[ptr] = true
}

func (l *MemoryPointerLocker) IsHoldingPointer(ctx context.Context, ptr data.PointerID) (bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.locks[ptr], nil
}

var _ storage.PointerLocker = &MemoryPointerLocker{}
