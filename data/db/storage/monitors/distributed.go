package monitors

import (
	"context"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/proto"
	"github.com/windmilleng/mish/data/db/storage"
	"github.com/windmilleng/mish/data/db/storage/locks"
	"github.com/windmilleng/mish/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const cacheTTL = time.Hour

type DistributedLockMonitor struct {
	store storage.PointerMetadataStore

	// map[data.Host]storage.PointerLocker
	lockers *cache.Cache

	// map[data.PointerID]data.Host
	hosts *cache.Cache
}

func NewDistributedLockMonitor(store storage.PointerMetadataStore) DistributedLockMonitor {
	return DistributedLockMonitor{
		store:   store,
		lockers: cache.New(cacheTTL, cacheTTL),
		hosts:   cache.New(cacheTTL, cacheTTL),
	}
}

func (m DistributedLockMonitor) IsActivelyHeld(ctx context.Context, id data.PointerID) (bool, error) {
	host, err := m.host(ctx, id)
	if err != nil {
		return false, err
	} else if host == "" {
		return false, nil
	}

	locker, err := m.locker(ctx, host)
	if err != nil {
		if errors.IsDeadlineExceeded(err) || grpc.Code(err) == codes.Unavailable {
			return false, nil
		}
		return false, err
	}

	active, err := locker.IsHoldingPointer(ctx, id)
	if err != nil {
		if grpc.Code(err) == codes.Unavailable {
			return false, nil
		}
		return false, err
	}
	return active, nil
}

func (m DistributedLockMonitor) IsLockLost(ctx context.Context, id data.PointerID) (bool, error) {
	host, err := m.host(ctx, id)
	if err != nil {
		return false, err
	} else if host == "" {
		return false, nil
	}

	locker, err := m.locker(ctx, host)
	if err != nil {
		if errors.IsDeadlineExceeded(err) || grpc.Code(err) == codes.Unavailable {
			return true, nil
		}
		return false, err
	}

	active, err := locker.IsHoldingPointer(ctx, id)
	if err != nil {
		if grpc.Code(err) == codes.Unavailable {
			return true, nil
		}
		return false, err
	}
	return !active, nil
}

// Retrieve the host that holds the given pointer. Caches non-empty hosts.
func (m DistributedLockMonitor) host(ctx context.Context, id data.PointerID) (data.Host, error) {
	hostStruct, ok := m.hosts.Get(id.String())
	if ok {
		host, ok := hostStruct.(data.Host)
		if ok {
			return host, nil
		}
	}

	metadata, err := m.store.PointerMetadata(ctx, id)
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return "", nil
		}
		return "", err
	}

	host := metadata.WriteHost
	if host != "" {
		m.hosts.SetDefault(id.String(), host)
	}

	return host, nil
}

// Returns an error if the machine at the given host doesn't respond.
func (m DistributedLockMonitor) locker(ctx context.Context, host data.Host) (storage.PointerLocker, error) {
	var locker storage.PointerLocker
	lockerStruct, ok := m.lockers.Get(string(host))
	if ok {
		locker, ok = lockerStruct.(storage.PointerLocker)
	}

	if locker == nil {
		// Dial the host written in the database.
		conn, err := m.dial(ctx, string(host))
		if err != nil {
			return nil, err
		}

		client := proto.NewPointerLockerClient(conn)
		locker = locks.NewGRPCPointerLocker(client)
		m.lockers.SetDefault(string(host), locker)
	}
	return locker, nil
}

func (m DistributedLockMonitor) dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, dialOptions()...)
	if err != nil {
		return nil, errors.Propagatef(err, "LockMonitor#dial(%q)", addr)
	}
	return conn, nil
}

func dialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}
}

var _ storage.PointerLockMonitor = DistributedLockMonitor{}
