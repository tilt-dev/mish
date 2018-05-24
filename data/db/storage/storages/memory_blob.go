package storages

import (
	"context"
	"sync"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type MemoryBlobStore struct {
	blobs map[storage.BlobID]data.Bytes
	mu    sync.Mutex
}

func NewMemoryBlobStore() *MemoryBlobStore {
	return &MemoryBlobStore{
		blobs: make(map[storage.BlobID]data.Bytes),
	}
}

func (s *MemoryBlobStore) ReadBlob(ctx context.Context, id storage.BlobID) (data.Bytes, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.blobs[id]
	if !ok {
		return data.NewEmptyBytes(), grpc.Errorf(codes.NotFound, "Blob not found: %s", id)
	}
	return b, nil
}

func (s *MemoryBlobStore) StoreBlob(ctx context.Context, bytes data.Bytes) (storage.BlobID, error) {
	id := storage.ToBlobID(bytes)

	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.blobs[id]
	if !ok {
		s.blobs[id] = bytes
	}

	return id, nil
}
