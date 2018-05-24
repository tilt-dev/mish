package storage

import (
	"context"

	"github.com/windmilleng/mish/data"
)

type RecipeStore interface {
	RecipeReader
	RecipeWriter
}

type RecipeReader interface {
	LookupPathToSnapshot(context context.Context, tag data.RecipeRTag, snapID data.SnapshotID) (data.StoredRecipe, error)
}

type AllRecipeReader interface {
	AllPathsToSnapshot(ctx context.Context, snapID data.SnapshotID) ([]data.StoredRecipe, error)
}

type PointerReader interface {
	Get(ctx context.Context, rev data.PointerAtRev) (data.PointerAtSnapshot, error)
}

type StorageReader interface {
	RecipeReader
	PointerReader
}

type RecipeWriter interface {
	// Create creates a new Snapshot created by r
	Create(context context.Context, r data.Recipe, owner data.UserID, t data.RecipeWTag) (data.SnapshotID, bool, error)

	// CreatePath creates a new path to an existing Snapshot.
	// If the snapshot doesn't exist, it should throw a NotFound error.
	CreatePath(ctx context.Context, sr data.StoredRecipe) error

	// Some implementations of recipe writer write to the backend in the background.
	WriteQueueSize(context context.Context) (data.WriteQueueSize, error)
}

type RecipeSink interface {
	Replicate(ctx context.Context, r data.StoredRecipe) error
}

type BatchCodeStore interface {
	WriteMany(ctx context.Context, writes []data.Write) error

	HeadMany(ctx context.Context, ptrHeads []data.PointerAtRev) (WriteStream, error)

	HasSnapshots(ctx context.Context, snapID []data.SnapshotID, c data.Consistency) ([]data.SnapshotID, error)
}

type RecipeIndex interface {
	LookupRecipe(ctx context.Context, r data.Recipe) (data.StoredRecipe, error)
}
