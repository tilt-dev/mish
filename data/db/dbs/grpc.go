package dbs

import (
	"context"
	"net"
	"path/filepath"
	"time"

	"google.golang.org/grpc"

	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/dbint"
	dp "github.com/windmilleng/mish/data/db/proto"
	"github.com/windmilleng/mish/data/db/storage"
	"github.com/windmilleng/mish/data/db/storage/hubs"
	"github.com/windmilleng/mish/data/db/storage/storages"
	wmNet "github.com/windmilleng/mish/net"
	"github.com/windmilleng/mish/os/temp"
	"github.com/windmilleng/mish/storage/server"
)

// Create a fake GRPC Hub that serves on a local file socket
// and listens on the same file socket. This is helpful for tests
// that want to exercise GRPC and timeouts.
func NewFakeGRPCDB2(store *storages.MemoryRecipeStore, ptrs *storages.MemoryPointers, grpcServer *grpc.Server, tmp *temp.TempDir) (dbint.DB2, *hubs.SpokeStorage, error) {
	socketDir, err := tmp.NewDir("socket")
	if err != nil {
		return nil, nil, err
	}

	socket := filepath.Join(socketDir.Path(), "socket")
	l, err := net.Listen("unix", socket)
	if err != nil {
		return nil, nil, err
	}

	memHub := hubs.NewMemoryHub(store, ptrs)
	hubServer := server.NewHubServer(memHub)
	dp.RegisterStorageHubServer(grpcServer, hubServer)
	go grpcServer.Serve(l)

	dial, err := grpc.Dial(socket, grpc.WithInsecure(), grpc.WithDialer(unixDial))
	if err != nil {
		return nil, nil, err
	}

	hubClient := dp.NewStorageHubClient(dial)
	grpcHub := hubs.NewGRPCHub(hubClient)
	storage, err := hubs.NewSpokeStorage(context.Background(), grpcHub, wmNet.NewConnectionReporter(dial), storage.CLIClientType, "")
	if err != nil {
		return nil, nil, err
	}
	return db2.NewDB2(storage, storage), storage, nil
}

func unixDial(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", addr, timeout)
}
