package storage

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/windmilleng/mish/data"
)

type BlobID string

func ToBlobID(b data.Bytes) BlobID {
	hash := sha256.New()
	hash.Write(b.InternalByteSlice())
	return BlobID(fmt.Sprintf("sha256-%x", hash.Sum(nil)))
}

// A simple interface for storing large files.
type BlobStorage interface {
	ReadBlob(ctx context.Context, id BlobID) (data.Bytes, error)
	StoreBlob(ctx context.Context, bytes data.Bytes) (BlobID, error)
}
