package storage

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/windmilleng/mish/data"
)

var tooManyRecipesError = fmt.Errorf("Too many recipes to traverse")
var couldNotUpdatePathsError = fmt.Errorf("Could not update paths")

func IsTooManyRecipesError(err error) bool {
	return err == tooManyRecipesError
}

func IsCouldNotUpdatePathsError(err error) bool {
	return err == couldNotUpdatePathsError
}

// Given a set of paths in one snapshot, rebase them into a snapshot further
// down the edit stream.
//
// Returns TooManyRecipesError if traversal excceeded the maximum recipes to traverse.
// Returns CouldNotUpdatePathsError if we couldn't figure out how to rebase the paths.
func RebasePaths(ctx context.Context, reader RecipeReader, have, want data.SnapshotID, ptr data.PointerID, paths []string, maxRecipeCount int) ([]string, error) {
	if have == want {
		return paths, nil
	}

	result := make([]string, len(paths))
	for i, p := range paths {
		result[i] = p
	}

	tree, err := RecipeTreeNeeded(ctx, reader, have, want, ptr)
	if err != nil {
		return nil, err
	}

	// Climb up the recipe tree to the want snapshot.
	// Along the way, transform the paths to match what's in the run dir.
	//
	// and we want to "translate" the diff between
	err = tree.ClimbTreeFrom(have, func(recipe data.StoredRecipe, input int) error {
		maxRecipeCount--
		if maxRecipeCount <= 0 {
			return tooManyRecipesError
		}

		op := recipe.Recipe.Op

		// File ops never change the current path.
		_, isFileOp := op.(data.FileOp)
		if isFileOp {
			return nil
		}

		switch op := op.(type) {
		case *data.PreserveOp, *data.OverlayOp:
			return nil
		case *data.DirOp:
			// Transform the paths for the directory
			for i, path := range result {
				result[i] = filepath.Join(op.Names[input], path)
			}
			return nil
		}

		// Otherwise, we don't know how to update the paths, so just bail.
		return couldNotUpdatePathsError
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
