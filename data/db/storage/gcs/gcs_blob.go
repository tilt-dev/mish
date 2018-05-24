package gcs

import (
	"context"
	"io/ioutil"

	gstorage "cloud.google.com/go/storage"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/storage"
	"github.com/windmilleng/mish/errors"
)

const CacheControl = "Cache-Control:public,max-age=3600"

type GCSBlobStore struct {
	bucket *gstorage.BucketHandle
}

func NewGCSBlobStore(bucket *gstorage.BucketHandle) *GCSBlobStore {
	return &GCSBlobStore{
		bucket: bucket,
	}
}

func (s *GCSBlobStore) ReadBlob(ctx context.Context, blobID storage.BlobID) (data.Bytes, error) {
	reader, err := s.bucket.Object(string(blobID)).NewReader(ctx)
	if err != nil {
		return data.NewEmptyBytes(), errors.Propagatef(err, "ReadBlob")
	}
	defer reader.Close()

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return data.NewEmptyBytes(), errors.Propagatef(err, "ReadBlob")
	}
	return data.NewBytesWithBacking(bytes), nil
}

func (s *GCSBlobStore) StoreBlob(ctx context.Context, b data.Bytes) (storage.BlobID, error) {
	blobID := storage.ToBlobID(b)
	obj := s.bucket.Object(string(blobID))
	attrs, err := obj.Attrs(ctx)
	if err == nil {
		// The blob is already in storage! yay.

		// Check if the cache control is set correctly.
		if attrs.CacheControl == "" {
			_, err = obj.Update(ctx, gstorage.ObjectAttrsToUpdate{CacheControl: CacheControl})
			if err != nil {
				return "", errors.Propagatef(err, "StoreBlob#Update")
			}
		}

		return blobID, nil
	} else if err != gstorage.ErrObjectNotExist {
		return "", errors.Propagatef(err, "StoreBlob#Attrs")
	}

	// The blob isn't in storage, so we have to write it.
	writer := obj.NewWriter(ctx)
	writer.ObjectAttrs.CacheControl = CacheControl

	// GCS writers work differently than normal writers, because
	// they buffer data and send it asynchronously. Close() is the real
	// way to commit data.
	_, err = writer.Write(b.InternalByteSlice())
	if err != nil {
		writer.Close()
		return "", errors.Propagatef(err, "StoreBlob#Write(%d)", b.Len())
	}

	err = writer.Close()
	if err != nil {
		return "", errors.Propagatef(err, "StoreBlob#Close(%d)", b.Len())
	}
	return blobID, nil
}
