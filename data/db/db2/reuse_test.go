package db2

import (
	"context"
	"fmt"
	"testing"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/storage/storages"
)

type reuseCase struct {
	init    []data.Op
	edit    []data.Op
	do      []data.Op
	overlay []data.Op
}

func TestReuse(t *testing.T) {
	f := setupReuse(t)

	reuses := []reuseCase{
		{
			nil,
			nil,
			ops(wf("test.txt", "hello")),
			nil,
		},
		{
			ops(wf("in.txt", "hello")),
			ops(wf("out.txt", "goodbye")),
			ops(f.preserveFile("in.txt")),
			nil,
		},
		{
			ops(wf("in.txt", "hello")),
			ops(wf("out.txt", "goodbye")),
			ops(f.preserveFile("in.txt"), f.preserveFile("*.txt")),
			nil,
		},
		{
			ops(wf("in.txt", "hello")),
			ops(wf("out.txt", "goodbye")),
			ops(f.preserveFile("in.txt"), f.preserveFile("*.txt"), f.preserveFile("*.*")),
			nil,
		},
		{
			ops(wf("in.txt", "hello")),
			ops(
				wf("out.txt", "goodbye"),
				wf("alsoout.txt", "also goodbye"),
			),
			ops(f.preserveFile("in.txt")),
			nil,
		},
		{
			ops(wf("in.txt", "hello")),
			ops(wf("out.txt", "goodbye")),
			ops(f.preserve("*in.txt")),
			ops(wf("alsoin.txt", "hi there"), f.preserve("also*")),
		},
		{
			ops(wf("in.txt", "hello")),
			ops(wf("out.txt", "goodbye")),
			ops(dir("newroot"), f.preserve("*in.txt")),
			nil,
		},
		{
			ops(wf("foo.txt", "foo"), wf("bar.txt", "bar")),
			ops(
				wf("foo.txt", "foo2"),
				wf("foo.txt", "foo3"),
			),
			ops(dir("foo"), f.preserve("foo/bar.txt")),
			nil,
		},
		{
			ops(wf("one/in.txt", "hello")),
			ops(wf("two/out.txt", "goodbye")),
			ops(f.preserve("one/**")),
			ops(wf("one/out.txt", "hi there"), dir("one")),
		},
		{
			ops(wf("one/in.txt", "hello")),
			ops(wf("two/out.txt", "goodbye")),
			ops(f.preserve("one/**")),
			ops(wf("three/out.txt", "hi there"), dir("three")),
		},
	}

	for i, c := range reuses {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			f = setupReuse(t)
			f.assertReuse(c)
			f.tearDown()
		})
	}
}

func setupReuse(t *testing.T) *fixture {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	return f
}

type recordingSink struct {
	recipes []data.StoredRecipe
}

func (s *recordingSink) Replicate(ctx context.Context, r data.StoredRecipe) error {
	s.recipes = append(s.recipes, r)
	return nil
}

func (f *fixture) assertReuse(c reuseCase) {
	// Testing Reuse involves many objects, so this comment describes what's going on.

	// 3 DBs:
	// 1) f.db is the editor
	// 2) do1DB is the first evaluation
	// 3) do2DB is the second evaluation, which wants to reuse the info gained during the first

	// So, to do this, we:
	// 1) perform init ops on f.db
	// 2a) create do1DB
	//   2b) in the overlay case, do the other side and the overlay on do1DB
	// 2c) perform do ops on do1DB
	// 3) perform edit ops on f.db
	// 4a) create do2DB
	//   4b) in the overlay case, do the other side and the overlay on do2DB
	// 4c) perform do ops on do2DB

	init, err := dbint.CreateLinear(f.ctx, f.db, c.init, data.EmptySnapshotID, data.UserTestID, data.RecipeWTagEdit)
	if err != nil {
		f.t.Fatal(err)
	}

	do1Input := init

	do1DB, do1RecordingSink := f.createTemp("tmp1", nil)

	if len(c.overlay) > 0 {
		otherSide, err := dbint.CreateLinear(f.ctx, do1DB, c.overlay, data.EmptySnapshotID, data.UserTestID, data.RecipeWTagEdit)
		if err != nil {
			f.t.Fatal(err)
		}
		do1Input, _, err = do1DB.Create(f.ctx,
			data.Recipe{Op: &data.OverlayOp{}, Inputs: []data.SnapshotID{init, otherSide}},
			data.UserTestID, data.RecipeWTagEdit)
		if err != nil {
			f.t.Fatal(err)
		}
	}

	do1, err := dbint.CreateLinear(f.ctx, do1DB, c.do, do1Input, data.UserTestID, data.RecipeWTagEdit)
	if err != nil {
		f.t.Fatal(err)
	}

	edit, err := dbint.CreateLinear(f.ctx, f.db, c.edit, init, data.UserTestID, data.RecipeWTagEdit)
	if err != nil {
		f.t.Fatal(err)
	}

	do2Input := edit

	do2DB, _ := f.createTemp("tmp2", do1RecordingSink)

	if len(c.overlay) > 0 {
		otherSide, err := dbint.CreateLinear(f.ctx, do2DB, c.overlay, data.EmptySnapshotID, data.UserTestID, data.RecipeWTagEdit)
		if err != nil {
			f.t.Fatal(err)
		}
		do2Input, _, err = do2DB.Create(f.ctx,
			data.Recipe{Op: &data.OverlayOp{}, Inputs: []data.SnapshotID{edit, otherSide}},
			data.UserTestID,
			data.RecipeWTagEdit)
		if err != nil {
			f.t.Fatal(err)
		}
	}

	do2, err := dbint.CreateLinear(f.ctx, do2DB, c.do, do2Input, data.UserTestID, data.RecipeWTagEdit)
	if err != nil {
		f.t.Fatal(err)
	}

	f.assertSnapReuse(do2DB, do1, do2)
}

// createTemp creates a temp DB, that uses the info from prev, and saves it output to
// recordingSink.
func (f *fixture) createTemp(id string, prev *recordingSink) (dbint.DB2, *recordingSink) {

	index := storages.NewMemoryRecipeIndex()
	if prev != nil {
		for _, r := range prev.recipes {
			index.Replicate(f.ctx, r)
		}
	}

	s := &recordingSink{}
	rep := storages.NewCompositeRecipeSink(index, s)

	tap := storages.NewTapRecipeWriter(f.store, rep)

	store := storages.Store{
		RecipeReader: f.store,
		RecipeWriter: tap,
	}

	db := NewDB2Ctor(store, f.ptrs, NewRewritingCreator(f.store, store, index))
	return db, s
}

func (f *fixture) assertSnapReuse(db dbint.DB2, s1 data.SnapshotID, s2 data.SnapshotID) {
	// this is similar to RecipesNeeded, but we want to follow the rewrite tag
	if s1 == s2 {
		return
	}

	r, err := db.LookupPathToSnapshot(f.ctx, data.RecipeRTag{Type: data.RecipeTagTypeRewritten, ID: data.RewriteIDEditDirection}, s2)
	if err != nil {
		f.t.Fatal(err)
	}

	if _, ok := r.Op.(*data.IdentityOp); !ok {
		f.t.Fatalf("snapshot %v isn't the result of an IdentityOp: %v", s2, r)
	}

	f.assertSnapReuse(db, s1, r.Inputs[0])
}
