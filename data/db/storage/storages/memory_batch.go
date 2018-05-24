package storages

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/storage"
	"github.com/windmilleng/mish/errors"
)

type MemoryBatchCodeStore struct {
	recipes *MemoryRecipeStore
	ptrs    *MemoryPointers

	backfiller storage.StorageReader
}

var _ storage.BatchCodeStore = &MemoryBatchCodeStore{}

type memoryBackfiller struct {
	*MemoryRecipeStore
	*MemoryPointers
}

// MemoryBatchStore can be used both as a "source-of-truth" storage that can
// query recipes and pointers.  Or it can serve as the storage system for a
// cache on top of a backend storage service.
//
// Use this method when it's the source of truth.
func NewMemoryBatchCodeStore(recipes *MemoryRecipeStore, ptrs *MemoryPointers) *MemoryBatchCodeStore {
	return NewMemoryBatchCodeStoreWithBackfill(recipes, ptrs, memoryBackfiller{recipes, ptrs})
}

// MemoryBatchStore can be used both as a "source-of-truth" storage that can
// query recipes and pointers.  Or it can serve as the storage system for a
// cache on top of a backend storage service.
//
// backfiller: If this is not the source of truth, then the backfiller allows us
// query the source of truth and backfill the store.
func NewMemoryBatchCodeStoreWithBackfill(recipes *MemoryRecipeStore, ptrs *MemoryPointers, backfiller storage.StorageReader) *MemoryBatchCodeStore {
	return &MemoryBatchCodeStore{
		recipes:    recipes,
		ptrs:       ptrs,
		backfiller: backfiller,
	}
}

func (s *MemoryBatchCodeStore) WriteMany(ctx context.Context, writes []data.Write) error {
	for _, w := range writes {
		if err := s.write(ctx, w); err != nil {
			return err
		}
	}

	return nil
}

func (s *MemoryBatchCodeStore) WriteExisting(ctx context.Context, writes []data.Write) error {
	for _, w := range writes {
		if err := s.writeExisting(ctx, w); err != nil {
			return err
		}
	}

	return nil
}

func (s *MemoryBatchCodeStore) write(ctx context.Context, w data.Write) error {
	switch w := w.(type) {
	case data.CreateSnapshotWrite:
		return s.recipes.Replicate(ctx, w.Recipe)
	case data.SetPointerWrite:
		return s.ptrs.Set(ctx, w.Next)
	case data.AcquirePointerWrite:
		_, err := s.ptrs.AcquirePointerWithHost(ctx, w.ID, w.Host)
		return err
	default:
		return fmt.Errorf("memory_batch.write: unexpected Write type %T %v", w, w)
	}
}

func (s *MemoryBatchCodeStore) writeExisting(ctx context.Context, w data.Write) error {
	switch w := w.(type) {
	case data.CreateSnapshotWrite:
		return s.recipes.Replicate(ctx, w.Recipe)
	case data.SetPointerWrite:
		return s.ptrs.SetExisting(ctx, w.Next)
	case data.AcquirePointerWrite:
		_, err := s.ptrs.AcquirePointerWithHost(ctx, w.ID, w.Host)
		return err
	default:
		return fmt.Errorf("memory_batch.writeExisting: unexpected Write type %T %v", w, w)
	}
}

// Overfetches writes. For example, if the client requested a pointer head, then
// we send back the CreateSnapshotWrite for that pointer, assuming they're going
// to ask for that next.
func (s *MemoryBatchCodeStore) HeadMany(ctx context.Context, ptrHeads []data.PointerAtRev) (storage.WriteStream, error) {
	stream := storage.NewSimpleWriteStream()
	go func() {
		defer stream.Close()

		err := s.fetchManyHelper(ctx, stream, ptrHeads)
		if err != nil {
			stream.SendError(err)
		}
	}()
	return stream, nil
}

func (s *MemoryBatchCodeStore) fetchManyHelper(ctx context.Context, stream *storage.SimpleWriteStream, ptrHeads []data.PointerAtRev) error {
	for _, head := range ptrHeads {
		newPtrAtRev, err := s.ptrs.Head(ctx, head.ID)
		if err != nil {
			if grpc.Code(err) == codes.NotFound {
				stream.SendError(storage.PointerNotFoundError{ID: head.ID})
				continue
			}

			return errors.Propagatef(err, "FetchMany#head(%v)", head)
		}

		if newPtrAtRev.Rev <= head.Rev {
			continue
		}

		newHead, err := s.ptrs.Get(ctx, newPtrAtRev)
		if err != nil {
			return err
		}

		startSnapID := data.EmptySnapshotID
		if head.Rev > 0 {
			start, err := s.backfiller.Get(ctx, head)
			if err != nil {
				return err
			}

			startSnapID = start.SnapID
		}

		recipes, err := storage.NonOptimizedRecipes(ctx, s.backfiller, startSnapID, newHead.SnapID)
		if err != nil {
			return errors.Propagatef(err, "FetchMany#alongptr(%v)", newHead)
		}

		if len(recipes) > 0 {
			// Send back the optimized IdentityOp first, if there is one.
			inputs := recipes[len(recipes)-1].Inputs
			if len(inputs) > 0 {
				optRecipe, err := s.backfiller.LookupPathToSnapshot(ctx, data.RecipeRTagOptimal, inputs[0])
				if err != nil {
					return errors.Propagatef(err, "FetchMany#optimal(%v)", newHead)
				}

				_, ok := optRecipe.Op.(*data.IdentityOp)
				if ok {
					stream.Send(data.CreateSnapshotWrite{Recipe: optRecipe})
				}
			}
		}

		// We want to send back the results in reverse topological order.
		for i := len(recipes) - 1; i >= 0; i-- {
			stream.Send(data.CreateSnapshotWrite{Recipe: recipes[i]})
		}

		// Send the SetPointer after all constituting recipes have been sent.
		stream.Send(data.SetPointerWrite{Next: newHead})
	}

	return nil
}

func (s *MemoryBatchCodeStore) HasSnapshots(ctx context.Context, snapIDs []data.SnapshotID, c data.Consistency) ([]data.SnapshotID, error) {
	return s.recipes.HasSnapshots(ctx, snapIDs)
}
