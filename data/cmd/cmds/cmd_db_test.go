package cmds

import (
	"context"
	"fmt"
	"testing"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/cmd"
	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/storage/storages"
)

func TestSimple(t *testing.T) {
	f := setup(t)
	defer f.tearDown()

	// first make snapshots
	s1 := f.edit(data.EmptySnapshotID, "foo.txt")
	s2 := f.edit(s1, "foo.txt")

	a1 := f.newArgs()

	f.assertGet(a1, s1, "", "foo.txt")
	f.assertGet(a1, s2, "", "foo.txt")
}

type fixture struct {
	t     *testing.T
	db    dbint.DB2
	ctx   context.Context
	cmdDB cmd.CmdDB

	nextArgID  int
	nextEditID int
}

func setup(t *testing.T) *fixture {
	recipes := storages.NewTestMemoryRecipeStore()
	ptrs := storages.NewMemoryPointers()
	db := db2.NewDB2(recipes, ptrs)
	runStore := NewEmptyRunStore()
	cmdDB := NewCmdDB(recipes, runStore)

	return &fixture{
		t:     t,
		db:    db,
		ctx:   context.Background(),
		cmdDB: cmdDB,
	}
}

func (f *fixture) tearDown() {
}

func (f *fixture) edit(in data.SnapshotID, filename string) data.SnapshotID {
	nonce := f.nextEditID
	f.nextEditID++

	r, _, err := f.db.Create(f.ctx,
		data.Recipe{
			Op:     &data.WriteFileOp{Path: filename, Data: data.BytesFromString(fmt.Sprintf("data-%d", nonce))},
			Inputs: []data.SnapshotID{in},
		},
		data.UserTestID,
		data.RecipeWTagEdit)
	if err != nil {
		f.t.Fatal(err)
	}
	return r
}

func (f *fixture) newArgs() cmd.Cmd {
	id := f.nextArgID
	f.nextArgID++
	return cmd.Cmd{
		Argv: []string{"echo", fmt.Sprintf("%d", id)},
	}
}

func (f *fixture) atSnap(c cmd.Cmd, snap data.SnapshotID) cmd.Cmd {
	if !c.Dir.Nil() {
		f.t.Fatalf("non-empty args cmd: %v", c)
	}

	c.Dir = snap
	return c
}

func (f *fixture) assertGet(c cmd.Cmd, s data.SnapshotID, ex cmd.AssociatedID, exChanged ...string) {
	c = f.atSnap(c, s)
	actual, changedOps, err := f.cmdDB.GetAssociatedID(f.ctx, c)
	if err != nil {
		f.t.Fatal(err)
	}

	if ex != actual {
		f.t.Fatalf("get got %q; expected %q", actual, ex)
	}

	actualChanged := make([]string, len(changedOps))
	for i, op := range changedOps {
		if op, ok := op.(data.FileOp); ok {
			actualChanged[i] = op.FilePath()
		} else {
			f.t.Fatalf("unexpected op type: %T %v", op, op)
		}
	}

	if len(exChanged) != len(actualChanged) {
		f.t.Fatalf("get got changed %q; expected %q", actualChanged, exChanged)
	}

	for i, actual := range actualChanged {
		ex := exChanged[i]
		if ex != actual {
			f.t.Fatalf("get got changed %q; expected %q", actualChanged, exChanged)
		}
	}
}
