package db2

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/storage"
	"github.com/windmilleng/mish/data/db/storage/monitors"
	"github.com/windmilleng/mish/errors"
)

func TestWaitForFrozen(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr-1")
	var expected data.PointerAtSnapshot
	go func() {
		f.writeToPtr(ptr, wf("foo.txt", "foo"))
		head := f.writeToPtr(ptr, wf("bar.txt", "bar"))

		expected = data.PtrEdit(head, head.SnapID)
		expected.Frozen = true

		dbint.SetFrozen(f.ctx, f.db, head)
	}()

	actual, err := dbint.WaitForFrozen(f.ctx, f.db, monitors.NewEmptyLockMonitor(), ptr)
	if err != nil {
		t.Fatal(err)
	} else if actual.Rev != expected.Rev {
		t.Errorf("Expected: %+v\nActual: %+v", expected, actual)
	}
}

func TestWaitForFrozenLockLost(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr-1")
	monitor := &fakeLockMonitor{}

	go func() {
		time.Sleep(10 * time.Millisecond)
		monitor.setLost(true)
	}()

	_, err = dbint.WaitForFrozen(f.ctx, f.db, monitor, ptr)
	if err == nil || !errors.IsCanceled(err) {
		t.Errorf("Expected canceled error. Actual: %v", err)
	}
}

type fakeLockMonitor struct {
	mu   sync.Mutex
	lost bool
}

func (m *fakeLockMonitor) setLost(val bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lost = val
}

func (m *fakeLockMonitor) IsLockLost(ctx context.Context, id data.PointerID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lost, nil
}

func (m *fakeLockMonitor) IsActivelyHeld(ctx context.Context, id data.PointerID) (bool, error) {
	return false, nil
}

var _ storage.PointerLockMonitor = &fakeLockMonitor{}
