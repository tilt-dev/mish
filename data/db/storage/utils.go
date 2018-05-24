package storage

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/windmilleng/mish/data"
)

type recipeTagFilter func(tag data.RecipeWTag) bool

var allRecipeTagFilter recipeTagFilter = func(tag data.RecipeWTag) bool { return true }

// Get a tree of the recipes needed, with `want` at the root and `have` in a leaf node.
func RecipeTreeNeeded(ctx context.Context, reader RecipeReader, have data.SnapshotID, want data.SnapshotID, ptr data.PointerID) (StoredRecipeTree, error) {
	finder := newRecipeFinder(reader, ptr, have, allRecipeTagFilter)
	recipes, err := finder.find(ctx, want)
	if err != nil {
		return storedRecipeLeaf, err
	}
	return recipes, nil
}

// Get all the recipes needed, traversing backwards from `want` to `have`
func RecipesNeeded(ctx context.Context, reader RecipeReader, have data.SnapshotID, want data.SnapshotID, ptr data.PointerID) ([]data.StoredRecipe, error) {
	tree, err := RecipeTreeNeeded(ctx, reader, have, want, ptr)
	if err != nil {
		return nil, err
	}
	return tree.AsSlice(), nil
}

// Find recipes, traversing backwards from `want` to `have`,
// but only looking at recipes tagged with the given pointer.
func RecipesAlongPointer(ctx context.Context, reader RecipeReader, have data.SnapshotID, want data.SnapshotID, ptr data.PointerID) ([]data.StoredRecipe, error) {

	ptrStr := ptr.String()
	filter := func(tag data.RecipeWTag) bool {
		return tag.Type == data.RecipeTagTypeEdit && tag.ID == ptrStr
	}
	finder := newRecipeFinder(reader, ptr, have, filter)
	recipes, err := finder.find(ctx, want)
	if err != nil {
		return nil, err
	}
	return recipes.AsSlice(), nil
}

// Prefetch as many recipes as we need to get back to an optimal snapshot.
func NonOptimizedRecipes(ctx context.Context, reader RecipeReader, have data.SnapshotID, want data.SnapshotID) ([]data.StoredRecipe, error) {
	filter := func(tag data.RecipeWTag) bool {
		return tag.Type != data.RecipeTagTypeOptimal
	}
	finder := newRecipeFinder(reader, data.PointerID{}, have, filter)
	recipes, err := finder.find(ctx, want)
	if err != nil {
		return nil, err
	}
	return recipes.AsSlice(), nil
}

type recipeFinder struct {
	reader RecipeReader
	tag    data.RecipeRTag
	have   data.SnapshotID
	filter recipeTagFilter
}

func newRecipeFinder(reader RecipeReader, id data.PointerID, have data.SnapshotID, filter recipeTagFilter) *recipeFinder {
	var tag data.RecipeRTag
	tag = data.RecipeRTagOptimal
	if !id.Nil() {
		tag = data.RecipeRTagForPointer(id)
	}

	return &recipeFinder{
		reader: reader,
		tag:    tag,
		have:   have,
		filter: filter,
	}
}

func (f *recipeFinder) find(ctx context.Context, want data.SnapshotID) (StoredRecipeTree, error) {
	if want == data.EmptySnapshotID {
		return storedRecipeLeaf, nil
	}

	if want == f.have {
		return storedRecipeLeaf, nil
	}

	recipe, err := f.reader.LookupPathToSnapshot(ctx, f.tag, want)
	if err != nil {
		return storedRecipeLeaf, err
	}

	if !f.filter(recipe.Tag) {
		return storedRecipeLeaf, nil
	}

	inputs := recipe.Inputs
	result := StoredRecipeTree{
		value:    recipe,
		children: make([]StoredRecipeTree, len(inputs)),
	}
	if len(inputs) == 0 {
		return result, nil
	} else if len(inputs) == 1 {
		tree, err := f.find(ctx, inputs[0])
		if err != nil {
			return storedRecipeLeaf, err
		}

		result.children[0] = tree
		return result, nil
	} else {
		g, ctx := errgroup.WithContext(ctx)
		for i, input := range inputs {
			i, input := i, input
			g.Go(func() error {
				tree, err := f.find(ctx, input)
				if err != nil {
					return err
				}
				result.children[i] = tree
				return nil
			})
		}

		err = g.Wait()
		if err != nil {
			return storedRecipeLeaf, err
		}

		return result, nil
	}
}

// A tree structure for accumulating recipes.
type StoredRecipeTree struct {
	// If this is nil, then this is a leaf node.
	value    data.StoredRecipe
	children []StoredRecipeTree
}

func (t StoredRecipeTree) Nil() bool {
	return t.value.Snap.Nil()
}

// A topologically sorted list of recipes, from most recent to least recent.
func (t StoredRecipeTree) AsSlice() []data.StoredRecipe {
	recipes := make([]data.StoredRecipe, 0, 1)
	return t.appendRecipes(recipes)
}

var pathNotFound = fmt.Errorf("StoredRecipeTree: path not found")

func IsPathNotFoundError(err error) bool {
	return err == pathNotFound
}

// Given a snapshot ID, we want to find a path from that snapshot ID to the root of the tree.
// Then, for each recipe on the path, we want to pass back both
// 1) The recipe for this edge
// 2) The index of the input that we came from.
//
// Returns nil if we successfully found a path, pathNotFound error otherwise.
//
// The callback should return an error to stop the iteration.
func (t StoredRecipeTree) ClimbTreeFrom(start data.SnapshotID, f func(r data.StoredRecipe, index int) error) error {
	if t.Nil() {
		if start == data.EmptySnapshotID {
			return nil
		}
		return pathNotFound
	}

	if t.value.Snap == start {
		return nil
	}

	for i, child := range t.children {
		input := t.value.Recipe.Inputs[i]
		if input == start {
			// We found the start of the path!
			return f(t.value, i)
		}

		if child.Nil() {
			continue
		}

		err := child.ClimbTreeFrom(start, f)

		// If we did not find `start` on this branch, try the next one.
		if err == pathNotFound {
			continue
		} else if err != nil {
			return err
		}

		// We found another leg of the path!
		return f(t.value, i)
	}

	if len(t.children) == 0 && start == data.EmptySnapshotID {
		return nil
	}

	return pathNotFound
}

func (t StoredRecipeTree) appendRecipes(recipes []data.StoredRecipe) []data.StoredRecipe {
	if t.Nil() || t.value.Snap == data.EmptySnapshotID {
		return recipes
	}

	recipes = append(recipes, t.value)
	for _, c := range t.children {
		recipes = c.appendRecipes(recipes)
	}
	return recipes
}

var storedRecipeLeaf = StoredRecipeTree{}
