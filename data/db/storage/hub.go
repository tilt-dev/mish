package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/windmilleng/mish/data"
)

const (
	DaemonClientType   data.ClientType = "wmdaemon"
	RunnerClientType                   = "wmrunner"
	APIClientType                      = "wmapi"
	FrontendClientType                 = "wmfrontend"
	CLIClientType                      = "wmcli"
	StorageClientType                  = "wmstorage"
)

type WriteStream interface {
	Next(ctx context.Context) (data.Write, error)
}

type PointerMetadataStore interface {
	PointerMetadata(c context.Context, id data.PointerID) (data.PointerMetadata, error)
}

type Hub interface {
	RecipeReader
	BatchCodeStore
	PointerMetadataStore

	RegisterClient(ctx context.Context, t data.ClientType) (data.ClientNonce, error)

	AcquirePointer(c context.Context, id data.PointerID, host data.Host) (data.PointerAtRev, error)

	Get(c context.Context, v data.PointerAtRev) (data.PointerAtSnapshot, error)

	AllPathsToSnapshot(context context.Context, snapID data.SnapshotID) ([]data.StoredRecipe, error)

	ActivePointerIDs(ctx context.Context, userID data.UserID, types []data.PointerType) ([]data.PointerID, error)

	Flush(ctx context.Context) error
}

type PointerNotFoundError struct {
	ID data.PointerID
}

func (e PointerNotFoundError) Error() string {
	return fmt.Sprintf("Not found: %s", e.ID.String())
}

type SimpleWriteStream struct {
	writeCh chan data.Write
	errCh   chan error
}

// Return a stream to write to. All Send() methods can be called concurrently.
func NewSimpleWriteStream() *SimpleWriteStream {
	return &SimpleWriteStream{writeCh: make(chan data.Write), errCh: make(chan error)}
}

func (s *SimpleWriteStream) Send(w data.Write) {
	s.writeCh <- w
}

func (s *SimpleWriteStream) SendError(err error) {
	s.errCh <- err
}

func (s *SimpleWriteStream) Next(ctx context.Context) (data.Write, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case w, ok := <-s.writeCh:
		if !ok {
			return nil, io.EOF
		}
		return w, nil
	case err, ok := <-s.errCh:
		if !ok {
			return nil, io.EOF
		}
		return nil, err
	}
}

func (s *SimpleWriteStream) Close() {
	close(s.writeCh)
	close(s.errCh)
}

func WriteStreamToSlice(ctx context.Context, s WriteStream) ([]data.Write, error) {
	result := make([]data.Write, 0)
	for {
		w, err := s.Next(ctx)
		if err == io.EOF {
			return result, nil
		}

		if err != nil {
			_, ok := err.(PointerNotFoundError)
			if ok {
				continue
			}
			return nil, err
		}

		result = append(result, w)
	}
}
