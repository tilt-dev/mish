package storages

import (
	"context"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/storage"
)

type CompositeRecipeSink struct {
	sinks []storage.RecipeSink
}

func NewCompositeRecipeSink(sinks ...storage.RecipeSink) *CompositeRecipeSink {
	return &CompositeRecipeSink{sinks: sinks}
}

func (s *CompositeRecipeSink) Replicate(ctx context.Context, r data.StoredRecipe) error {
	for _, sink := range s.sinks {
		if err := sink.Replicate(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// TapRecipeWriter allows writing Recipes and tapping the stream of writes.
// E.g. writing and then indexing.
// TODO(dbentley): we should merge this with code in spoke.go and sync.go
// so that there's one way to:
// *) intercept writes
// *) flush
// *) write queue size, etc.
type TapRecipeWriter struct {
	first  storage.RecipeWriter
	second storage.RecipeSink
}

func NewTapRecipeWriter(first storage.RecipeWriter, second storage.RecipeSink) *TapRecipeWriter {
	return &TapRecipeWriter{
		first:  first,
		second: second,
	}
}

func (w *TapRecipeWriter) Create(ctx context.Context, r data.Recipe, owner data.UserID, t data.RecipeWTag) (data.SnapshotID, bool, error) {
	id, new, err := w.first.Create(ctx, r, owner, t)

	if err != nil {
		return data.SnapshotID{}, false, err
	}

	err = w.second.Replicate(ctx, data.StoredRecipe{Snap: id, Tag: t, Recipe: r})
	if err != nil {
		return data.SnapshotID{}, false, err
	}

	return id, new, nil
}

func (w *TapRecipeWriter) CreatePath(ctx context.Context, r data.StoredRecipe) error {
	if err := w.first.CreatePath(ctx, r); err != nil {
		return err
	}

	return w.second.Replicate(ctx, r)
}

func (w *TapRecipeWriter) WriteQueueSize(ctx context.Context) (data.WriteQueueSize, error) {
	return w.first.WriteQueueSize(ctx)
}

// Create a Store from a Reader and a Writer
type Store struct {
	storage.RecipeReader
	storage.RecipeWriter
}
