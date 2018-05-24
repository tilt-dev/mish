package db2

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/storage"
)

// this ought to be db/dbs.LocalDB
type DB2 struct {
	store storage.RecipeStore
	data.Pointers

	ctor Creator
}

func NewDB2(store storage.RecipeStore, ptrs data.Pointers) *DB2 {
	return &DB2{store: store, Pointers: ptrs, ctor: store}
}

func NewDB2Ctor(store storage.RecipeStore, ptrs data.Pointers, ctor Creator) *DB2 {
	return &DB2{store: store, Pointers: ptrs, ctor: ctor}
}

// We can visualize a snapshot as a tree of recipes.
//
// opEvaluator behaves like a tree processor in a traditional tree traversal
// algorithm for synthesizing properties about the tree.  The evaluator
// specifies:
//
// - a starting value to represent the empty tree (`empty`),
// - a pre-visit step (`visitBackwards`) that propagates inherited values down
//   the tree and allows us to skip subtrees, and
// - a post-visit step (`applyOp`) that synthesizes values up the tree.
//
// Implementations of opEvaluator should pick a concrete opEvalResult.
type opEvaluator interface {
	// Creates an empty result type.
	empty() opEvalResult

	// Applies the op, and returns a new accumulator with the state applied.
	// The inputs of applyOp will always be the output of some combination of
	// empty() and applyOp(). It is generally OK to modify opEvalResult in-place.
	applyOp(outputID data.SnapshotID, op data.Op, inputs []opEvalResult) (opEvalResult, error)

	// Traverses back before an op. Returns instructions on how to traverse the inputs.
	//
	// Params:
	// op: The op
	// input: The start of the input edge.
	// inputNum: The index of the input edge. The semantics of this depends
	//   on the particular op.
	//
	// Returns:
	// opEvalResult: if non-nil, short-circuit traversal and use this value for the input tree.
	// opEvaluator: if non-nil, recursively traverse backwards with this evaluator.
	// If both are nil, short-circuits traversal with the empty() opEvalResult.
	visitBackwards(op data.Op, input data.SnapshotID, inputNum int) (opEvaluator, opEvalResult, error)
}

type opEvalResult interface {
	opEvalResult()
}

// A greedy traversal of all the ops in a snapshot's history.
func (d *DB2) eval(ctx context.Context, tag data.RecipeRTag, snapID data.SnapshotID, e opEvaluator) (opEvalResult, error) {
	if snapID == data.EmptySnapshotID {
		return e.empty(), nil
	}

	recipe, err := d.store.LookupPathToSnapshot(ctx, tag, snapID)
	if err != nil {
		return nil, err
	}

	return d.evalOp(ctx, tag, snapID, e, recipe.Op, recipe.Inputs)
}

func (d *DB2) evalOp(ctx context.Context, tag data.RecipeRTag, snapID data.SnapshotID, e opEvaluator, op data.Op, inputs []data.SnapshotID) (opEvalResult, error) {
	count := len(inputs)
	inputVals := make([]opEvalResult, count)
	if count == 1 {
		val, err := d.evalOpInput(ctx, tag, e, op, inputs[0], 0)
		if err != nil {
			return nil, err
		}
		inputVals[0] = val
	} else if count >= 2 {
		// Traverse the tree in parallel goroutines
		g, ctx := errgroup.WithContext(ctx)

		for i, input := range inputs {
			i, input := i, input
			g.Go(func() error {
				val, err := d.evalOpInput(ctx, tag, e, op, input, i)
				if err != nil {
					return err
				}
				inputVals[i] = val
				return nil
			})
		}

		err := g.Wait()
		if err != nil {
			return nil, err
		}
	}

	return e.applyOp(snapID, op, inputVals)
}

func (d *DB2) evalOpInput(ctx context.Context, tag data.RecipeRTag, e opEvaluator, op data.Op, input data.SnapshotID, i int) (opEvalResult, error) {
	inputEvaluator, inputResult, err := e.visitBackwards(op, input, i)
	if err != nil {
		return nil, err
	}

	if inputResult != nil {
		return inputResult, err
	}

	if inputEvaluator == nil {
		return e.empty(), nil
	}

	return d.eval(ctx, tag, input, inputEvaluator)
}

func (d *DB2) RecipesNeeded(ctx context.Context, have data.SnapshotID, want data.SnapshotID, ptr data.PointerID) ([]data.StoredRecipe, error) {
	return storage.RecipesNeeded(ctx, d.store, have, want, ptr)
}

func (d *DB2) Create(ctx context.Context, r data.Recipe, owner data.UserID, t data.RecipeWTag) (data.SnapshotID, bool, error) {
	return d.ctor.Create(ctx, r, owner, t)
}

func (d *DB2) QueryPointerHistory(ctx context.Context, query data.PointerHistoryQuery) ([]data.PointerAtSnapshot, error) {
	ptrID := query.PtrID
	ptrAtSnap, err := dbint.HeadSnap(ctx, d, ptrID)
	if err != nil {
		return nil, err
	}

	// If maxResults is not specified, defaults to 10
	maxResults := int(query.MaxResults)
	if maxResults == 0 {
		maxResults = 10
	}

	// TODO(nick): If accumulating across the whole history is too slow, we can
	// prune the tree early once we get MaxResults ids.
	snapID := ptrAtSnap.SnapID
	snapIDList, err := d.eval(ctx, data.RecipeRTagForPointer(ptrID), snapID, &snapshotAccumulator{matcher: query.Matcher})
	if err != nil {
		return nil, err
	}

	snapIDMap := snapIDList.(*snapshotIDList).toSet()
	ptrRevToSnapID := map[data.PointerRev]data.SnapshotID{}

	iter := data.NewPointerHistoryIter(ptrID, ptrAtSnap.Rev, data.PointerHistoryIterParams{
		MaxRev:    query.MaxRev,
		MinRev:    query.MinRev,
		Ascending: query.Ascending,
	})

	iter = iter.Filter(func(ptrAtRev data.PointerAtRev) (bool, error) {
		ptrAtSnap, err := d.Get(ctx, ptrAtRev)
		if err != nil {
			return false, err
		}
		ptrRevToSnapID[ptrAtSnap.Rev] = ptrAtSnap.SnapID

		recipes, err := dbint.RecipesFromPreviousRev(ctx, d, ptrAtSnap)
		for _, r := range recipes {
			if snapIDMap[r.Snap] {
				return true, nil
			}
		}

		return false, nil
	})

	ptrAtRevs, err := iter.Take(maxResults)
	if err != nil {
		return nil, err
	}

	result := make([]data.PointerAtSnapshot, 0, len(ptrAtRevs))
	for _, ptrAtRev := range ptrAtRevs {
		result = append(result, data.PointerAtSnapshot{
			ID:     ptrAtRev.ID,
			Rev:    ptrAtRev.Rev,
			SnapID: ptrRevToSnapID[ptrAtRev.Rev],
		})
	}

	return result, nil
}

func (d *DB2) LookupPathToSnapshot(context context.Context, tag data.RecipeRTag, snapID data.SnapshotID) (data.StoredRecipe, error) {
	return d.store.LookupPathToSnapshot(context, tag, snapID)
}
