package dbint_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/storage/storages"
)

var ctx = context.Background()

func TestDirAssemblerOrdering(t *testing.T) {
	f := setup(t)
	a1 := dbint.NewDirAssembler(ctx, f.db, data.UserTestID, data.RecipeWTagOptimal)
	a2 := dbint.NewDirAssembler(ctx, f.db, data.UserTestID, data.RecipeWTagOptimal)

	a1.WriteString("foo.txt", "foo")
	a1.WriteString("bar.txt", "bar")

	a2.WriteString("bar.txt", "bar")
	a2.WriteString("foo.txt", "foo")

	s1, err := a1.Commit()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := a2.Commit()
	if err != nil {
		t.Fatal(err)
	}

	if s1 != s2 {
		t.Errorf("Expected snapshots to be equal: %s, %s", s1, s2)
	}

	r1, _ := a1.AsRecipe()
	r2, _ := a2.AsRecipe()
	if !reflect.DeepEqual(r1.Inputs, r2.Inputs) {
		t.Errorf("Expected recipe inputs to be equal: %v, %v", r1.Inputs, r2.Inputs)
	}

	op1 := r1.Op.(*data.DirOp)
	op2 := r2.Op.(*data.DirOp)

	if !reflect.DeepEqual(op1.Names, op2.Names) {
		t.Errorf("Expected op names to be equal: %v, %v", op1.Names, op2.Names)
	}
}

type fixture struct {
	t  *testing.T
	db *db2.DB2
}

func setup(t *testing.T) *fixture {
	db := db2.NewDB2(storages.NewTestMemoryRecipeStore(), storages.NewMemoryPointers())
	return &fixture{t: t, db: db}
}
