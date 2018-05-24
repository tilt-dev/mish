package monitors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"net"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/proto"
	"github.com/windmilleng/mish/data/db/storage/locks"
	"github.com/windmilleng/mish/data/db/storage/storages"
	"github.com/windmilleng/mish/storage/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestPointerNotFound(t *testing.T) {
	f := newDistFixture(t)
	defer f.tearDown()

	ptr := f.ptr
	err := f.assertActivelyHeld(ptr, false)
	if err != nil {
		t.Fatal(err)
	}

	err = f.assertLockLost(ptr, false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.ptrs.Head(f.ctx, ptr)
	if err == nil || grpc.Code(err) != codes.NotFound {
		t.Errorf("Expected NotFound. Actual: %v", err)
	}
}

func TestPointerAcquiredButNeverHeld(t *testing.T) {
	f := newDistFixture(t)
	defer f.tearDown()

	ptr := f.ptr
	_, err := f.ptrs.AcquirePointerWithHost(f.ctx, ptr, f.lockerA.addr)
	if err != nil {
		t.Fatal(err)
	}

	err = f.assertActivelyHeld(ptr, false)
	if err != nil {
		t.Fatal(err)
	}

	err = f.assertLockLost(ptr, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPointerAcquiredAndHeld(t *testing.T) {
	f := newDistFixture(t)
	defer f.tearDown()

	ptr := f.ptr
	_, err := f.ptrs.AcquirePointerWithHost(f.ctx, ptr, f.lockerA.addr)
	if err != nil {
		t.Fatal(err)
	}

	f.lockerA.locker.HoldPointer(ptr)

	err = f.assertActivelyHeld(ptr, true)
	if err != nil {
		t.Fatal(err)
	}

	err = f.assertLockLost(ptr, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPointerAcquiredAndHeldByWrongBackend(t *testing.T) {
	f := newDistFixture(t)
	defer f.tearDown()

	ptr := f.ptr
	_, err := f.ptrs.AcquirePointerWithHost(f.ctx, ptr, f.lockerA.addr)
	if err != nil {
		t.Fatal(err)
	}

	f.lockerB.locker.HoldPointer(ptr)

	err = f.assertActivelyHeld(ptr, false)
	if err != nil {
		t.Fatal(err)
	}

	err = f.assertLockLost(ptr, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPointerAcquiredAndHeldByKilledBackend(t *testing.T) {
	f := newDistFixture(t)
	defer f.tearDown()

	ptr := f.ptr
	_, err := f.ptrs.AcquirePointerWithHost(f.ctx, ptr, f.lockerA.addr)
	if err != nil {
		t.Fatal(err)
	}

	f.lockerA.locker.HoldPointer(ptr)

	err = f.assertActivelyHeld(ptr, true)
	if err != nil {
		t.Fatal(err)
	}

	err = f.assertLockLost(ptr, false)
	if err != nil {
		t.Fatal(err)
	}

	f.lockerA.server.GracefulStop()

	err = f.assertActivelyHeld(ptr, false)
	if err != nil {
		t.Fatal(err)
	}

	err = f.assertLockLost(ptr, true)
	if err != nil {
		t.Fatal(err)
	}
}

type lockerFixture struct {
	locker *locks.MemoryPointerLocker
	addr   data.Host
	server *grpc.Server
}

func newLockerFixture(t *testing.T, addr string) lockerFixture {
	locker := locks.NewMemoryPointerLocker()

	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	grpcServer := grpc.NewServer()
	lockServer := server.NewLockServer(locker)
	proto.RegisterPointerLockerServer(grpcServer, lockServer)
	go grpcServer.Serve(l)
	return lockerFixture{
		locker: locker,
		addr:   data.Host(l.Addr().String()),
		server: grpcServer,
	}
}

func (f lockerFixture) tearDown() {
	f.server.GracefulStop()
}

type distFixture struct {
	ctx     context.Context
	cancel  func()
	lockerA lockerFixture
	lockerB lockerFixture
	ptrs    *storages.MemoryPointers
	dist    DistributedLockMonitor
	ptr     data.PointerID
}

func newDistFixture(t *testing.T) distFixture {
	ptrs := storages.NewMemoryPointers()
	dist := NewDistributedLockMonitor(ptrs)
	ptr := data.MustNewPointerID(data.AnonymousID, "ptr", data.UserPtr)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	return distFixture{
		ctx:     ctx,
		cancel:  cancel,
		ptr:     ptr,
		ptrs:    ptrs,
		dist:    dist,
		lockerA: newLockerFixture(t, "localhost:0"),
		lockerB: newLockerFixture(t, "localhost:0"),
	}
}

func (f distFixture) assertActivelyHeld(id data.PointerID, expected bool) error {
	actual, err := f.dist.IsActivelyHeld(f.ctx, id)
	if err != nil {
		return err
	}

	if actual != expected {
		return fmt.Errorf("assertActivelyHeld: expected %v, actual %v", expected, actual)
	}
	return nil
}

func (f distFixture) assertLockLost(id data.PointerID, expected bool) error {
	actual, err := f.dist.IsLockLost(f.ctx, id)
	if err != nil {
		return err
	}

	if actual != expected {
		return fmt.Errorf("assertLockLost: expected %v, actual %v", expected, actual)
	}
	return nil
}

func (f distFixture) tearDown() {
	f.lockerA.tearDown()
	f.lockerB.tearDown()
	f.cancel()
}
