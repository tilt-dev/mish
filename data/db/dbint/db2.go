// dbint holds the DB INTerface.
// It's a temporary home as we migrate DB's, and will eventually move to data/db (or maybe just db?)
package dbint

import (
	"context"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
)

type Reader interface {
	Snapshot(ctx context.Context, snap data.SnapshotID) (SnapshotMetadata, error)

	// Returns the state of a file at a snapshot.
	// Returns a grpc NotFound if the file cannot be found.
	SnapshotFile(ctx context.Context, snap data.SnapshotID, path string) (*SnapshotFile, error)

	SnapshotFilesMatching(ctx context.Context, snap data.SnapshotID, matcher *dbpath.Matcher) (SnapshotFileList, error)

	// Determine the paths that changed between two snapshot IDs.
	//
	// The definition of what counts as a "change" to a file is a bit fuzzy, but
	// basically we count any direct write or deletes to the file. We
	// do NOT count a rmdir that deletes the entire directory tree a file lives in,
	// or operations that move the whole directory (Dir/Subdir)
	PathsChanged(ctx context.Context, start, end data.SnapshotID, tag data.RecipeRTag, matcher *dbpath.Matcher) ([]string, error)

	// Head returns the current revision of a Pointer
	// non-existent Pointers throw an error
	Head(c context.Context, id data.PointerID) (data.PointerAtRev, error)

	// AcquirePointer returns the current revision of a Pointer, and locks it so that only this
	// machine can write to it.
	// Non-existent Pointers are initialized to rev 0
	AcquirePointer(c context.Context, id data.PointerID) (data.PointerAtRev, error)

	// Get returns the value of a Pointer at a revision
	Get(c context.Context, v data.PointerAtRev) (data.PointerAtSnapshot, error)

	// Wait waits for a Pointer to change from a known value or ctx to be done (returning nil)
	Wait(c context.Context, last data.PointerAtRev) error

	// Returns all the Pointer revs that match a particular query.
	QueryPointerHistory(c context.Context, query data.PointerHistoryQuery) ([]data.PointerAtSnapshot, error)

	LookupPathToSnapshot(context context.Context, tag data.RecipeRTag, snapID data.SnapshotID) (data.StoredRecipe, error)

	// RecipesNeeded finds the recipes needed if you have have and want want
	// The result is topologically sorted with the newest first, so you should
	// apply in reverse order
	RecipesNeeded(ctx context.Context, have data.SnapshotID, want data.SnapshotID, ptr data.PointerID) ([]data.StoredRecipe, error)

	// Estimates the cost of playing back a snapshot along a given path and matcher.
	Cost(ctx context.Context, snapID data.SnapshotID, tag data.RecipeRTag, matcher *dbpath.Matcher) (SnapshotCost, error)
}

// The optimizer is an odd duck because it needs to do both high-level playback (like a DB)
// and also needs low-level writing capabilities that require a higher degree of trust
// (like a MemoryRecipeStore).
//
// In the future, the DB may do this automatically. For the prototyping phase
// we have an explicit call for it. Breaking it out into a separate interface
// should make it easier to waffle between the auto and manual modes.
type Optimizer interface {
	// Rewrite the snapshot in a more efficient way.
	//
	// The hope is that this will rewrite a linear stream of Insert/DeleteBytesOps into
	// a more compact tree of WriteOps and DirOps that are easier to query.
	//
	// Returns the content ID for the optimized snapshot. If the returned ID is different than
	// the parameter, then that means we've created an identityOp between them.
	OptimizeSnapshot(ctx context.Context, snapID data.SnapshotID) (data.SnapshotID, error)

	// Retrieves the optimized ID if this snapshot has already been optimized.
	// Otherwise, returns the empty ID.
	IsOptimized(ctx context.Context, snapID data.SnapshotID) (data.SnapshotID, error)
}

type Writer interface {
	// Returns the new snapshot ID, and whether this is a newly created ID.
	// (Content IDs may already exist in the db)
	Create(ctx context.Context, r data.Recipe, owner data.UserID, tag data.RecipeWTag) (data.SnapshotID, bool, error)

	MakeTemp(c context.Context, userID data.UserID, prefix string, t data.PointerType) (data.PointerAtSnapshot, error)

	Set(c context.Context, next data.PointerAtSnapshot) error
}

type Debug interface {
	// Returns all pointer IDs active for this user.
	ActivePointerIDs(c context.Context, userID data.UserID, types []data.PointerType) ([]data.PointerID, error)
}

// This ought to be db.DB
type DB2 interface {
	Reader
	Writer

	Debug
}
