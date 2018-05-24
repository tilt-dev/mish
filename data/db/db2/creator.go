package db2

import (
	"context"
	"fmt"
	"path"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
	"github.com/windmilleng/mish/data/db/storage"
	"github.com/windmilleng/mish/data/transform"
)

type Creator interface {
	Create(context context.Context, r data.Recipe, owner data.UserID, t data.RecipeWTag) (data.SnapshotID, bool, error)
}

type stack struct {
	height int
}

func (s stack) push() stack {
	return stack{s.height + 1}
}

type rewritingCreator struct {
	recipes storage.AllRecipeReader
	store   storage.RecipeStore
	index   storage.RecipeIndex
}

func NewRewritingCreator(recipes storage.AllRecipeReader, store storage.RecipeStore, index storage.RecipeIndex) *rewritingCreator {
	r := &rewritingCreator{
		recipes: recipes,
		store:   store,
		index:   index,
	}

	return r
}

func (c *rewritingCreator) Create(ctx context.Context, r data.Recipe, owner data.UserID, t data.RecipeWTag) (data.SnapshotID, bool, error) {
	id, b, err := c.create(ctx, stack{}, r, owner, t)
	return id, b, err
}

func (c *rewritingCreator) create(ctx context.Context, s stack, r data.Recipe, owner data.UserID, t data.RecipeWTag) (data.SnapshotID, bool, error) {
	sr, err := c.index.LookupRecipe(ctx, r)
	if err != nil {
		return data.SnapshotID{}, false, err
	}

	id := sr.Snap
	if !id.Nil() {
		return id, false, nil
	}

	// We don't have exactly this recipe in storage already, so let's try rewriting it
	rewrite, err := c.findRewrite(ctx, s, r)
	if err != nil {
		return data.SnapshotID{}, false, err
	}

	if len(rewrite) > 0 {
		rewriteT := data.RecipeWTag{Type: data.RecipeTagTypeRewritten, ID: data.RewriteIDEditDirection}
		id, new, err := c.tryRewrite(ctx, s, rewrite, owner, rewriteT)
		if err != nil {
			return data.SnapshotID{}, false, err
		}

		if !id.Nil() {
			// rewrite was successful!
			// also write this new op, but to the rewritten name
			sr := data.StoredRecipe{Snap: id, Recipe: r, Tag: t}
			if err := c.store.CreatePath(ctx, sr); err != nil {
				return data.SnapshotID{}, false, err
			}
			return id, new, nil
		}
	}

	if t.Type == data.RecipeTagTypeRewritten {
		// we are in a recursive call to create, and we didn't find a rewrite we could apply,
		// so don't write this
		return data.SnapshotID{}, false, nil
	}
	// rewrite didn't happen; just write this
	return c.store.Create(ctx, r, owner, t)
}

func (c *rewritingCreator) getInput(r data.Recipe) data.SnapshotID {
	if len(r.Inputs) != 1 {
		return data.EmptySnapshotID
	}

	return r.Inputs[0]
}

// it's not worth recurring infinitely, so set a limit
const maxHeight = 50

func (c *rewritingCreator) findRewrite(ctx context.Context, s stack, r data.Recipe) ([]data.StoredRecipe, error) {
	if s.height > maxHeight {
		return nil, nil
	}
	rewrite, err := c.findLinearRewrite(ctx, r)
	if err != nil || len(rewrite) > 0 {
		return rewrite, err
	}

	rewrite, err = c.findTriangleRewrite(ctx, r)
	if err != nil || len(rewrite) > 0 {
		return rewrite, err
	}

	return nil, nil
}

// Suppose we have B(A(x)), and we know that B(A(s)) == B(s) for all s.
// Then if we already have B(x), we can reuse it.
func (c *rewritingCreator) findLinearRewrite(ctx context.Context, r data.Recipe) ([]data.StoredRecipe, error) {
	if len(r.Inputs) != 1 {
		return nil, nil
	}

	paths, err := c.recipes.AllPathsToSnapshot(ctx, r.Inputs[0])
	if err != nil {
		return nil, err
	}

	for _, path := range paths {
		if !isLinearOp(path.Op) || len(path.Inputs) != 1 {
			continue
		}

		aPrime, err := transform.LeftEclipsingTransform(path.Op, r.Op)
		if err != nil {
			return nil, err
		}

		if aPrime == nil {
			continue
		}

		// B(A(X) ==  A'(B(X)) (because B = B')
		return []data.StoredRecipe{
			out(data.TempID("im"), r.Op, path.Inputs[0]),
			out(data.SnapshotID{}, aPrime, data.TempID("im")),
		}, nil

	}

	return nil, nil
}

// Suppose we're trying to compute:
// Overlay(A(x), Preserve[paths](y))
//
// One way to think of this is as a mask on "paths" of x.
//
// It's easy to show that if
// Preserve[!paths](A(s)) == A(Preserve[!paths](s)) for all s,
// then
// Overlay(A(s), Preserve[paths](t)) == A(Overlay(s, Preserve[paths](t)))
// i.e., we can flip the order of ops.
//
// we can do a similar transform for
// Overlay(A(x), Dir(path)(y))
// to
// A(Overlay(x, Dir(path)(y)))
// if we know that A doesn't affect anything under `path`
func (c *rewritingCreator) findTriangleRewrite(ctx context.Context, r data.Recipe) ([]data.StoredRecipe, error) {
	_, ok := r.Op.(*data.OverlayOp)
	if !ok || len(r.Inputs) != 2 {
		return nil, nil
	}

	leftPaths, err := c.recipes.AllPathsToSnapshot(ctx, r.Inputs[0])
	if err != nil {
		return nil, err
	}

	matcher, err := c.isolateTree(ctx, r.Inputs[1])
	if err != nil || matcher == nil {
		return nil, err
	}

	invertedMatcher, err := matcher.Invert()
	if err != nil {
		// this is ok, not all matchers are invertible
		return nil, nil
	}

	invertedPreserve := &data.PreserveOp{Matcher: invertedMatcher}

	for _, left := range leftPaths {
		if !transform.IsCommutative(invertedPreserve, left.Op) {
			continue
		}

		leftInput := data.EmptySnapshotID
		if len(left.Inputs) > 0 {
			leftInput = left.Inputs[0]
		}

		return []data.StoredRecipe{
			out(data.TempID("im"), &data.OverlayOp{}, leftInput, r.Inputs[1]),
			out(data.SnapshotID{}, left.Op, data.TempID("im")),
		}, nil
	}

	return nil, nil
}

// A helper for findTriangleRewrite. Find a matcher that captures the contents
// of a snapshot.
//
// For example, if this function returns Matcher(foo/**), that means
// we can guarantee that this snapshot only contains files under foo/.
//
// This needs to be fast, so only checks the top-level recipes.
func (c *rewritingCreator) isolateTree(ctx context.Context, tree data.SnapshotID) (*dbpath.Matcher, error) {
	if tree == data.EmptySnapshotID {
		return dbpath.NewEmptyMatcher(), nil
	}

	recipes, err := c.recipes.AllPathsToSnapshot(ctx, tree)
	if err != nil {
		return nil, err
	}

	for _, recipe := range recipes {
		if len(recipe.Inputs) != 1 {
			continue
		}

		switch op := recipe.Op.(type) {
		case *data.PreserveOp:
			return op.Matcher, nil
		case *data.DirOp:
			return dbpath.NewMatcherFromPattern(path.Join(op.Names[0], "**"))
		}
	}

	return nil, nil
}

func (c *rewritingCreator) tryRewrite(ctx context.Context, s stack, srs []data.StoredRecipe, owner data.UserID, t data.RecipeWTag) (data.SnapshotID, bool, error) {
	binds := make(map[data.SnapshotID]data.SnapshotID)
	var last data.SnapshotID
	var new bool

	for i, sr := range srs {
		inputs := make([]data.SnapshotID, len(sr.Inputs))
		for j, input := range sr.Inputs {
			if input.IsTempID() {
				perm, ok := binds[input]
				if !ok {
					return data.SnapshotID{}, false, fmt.Errorf("no binding for %v", input)
				}
				inputs[j] = perm
			} else {
				inputs[j] = input
			}
		}

		recipe := data.Recipe{Op: sr.Op, Inputs: inputs}

		var err error
		lastSR, err := c.index.LookupRecipe(ctx, recipe)
		if err != nil {
			return data.SnapshotID{}, false, err
		}
		last = lastSR.Snap

		if last.Nil() && i == 0 {
			// srs form a path that only works if the first sr already exists
			// it doesn't, so let's go deeper.
			horTag := data.RecipeWTag{
				Type: data.RecipeTagTypeRewritten,
				ID:   data.RewriteIDBuildDirection,
			}
			last, new, err = c.create(ctx, s.push(), recipe, owner, horTag)
			if err != nil || last.Nil() {
				// welp, recursion didn't solve this problem
				return data.SnapshotID{}, false, err
			}
		}

		if last.Nil() {
			last, new, err = c.store.Create(ctx, recipe, owner, t)
			if err != nil {
				return data.SnapshotID{}, false, err
			}
		}

		binds[sr.Snap] = last
	}

	return last, new, nil
}

func out(snap data.SnapshotID, op data.Op, inputs ...data.SnapshotID) data.StoredRecipe {
	return data.StoredRecipe{Snap: snap, Recipe: data.Recipe{Op: op, Inputs: inputs}}
}
