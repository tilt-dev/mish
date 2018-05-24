package transform_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/dbpath"
	"github.com/windmilleng/mish/data/db/storage/storages"
	"github.com/windmilleng/mish/data/transform"
)

var ctx = context.Background()

type TransformExpectation struct {
	base data.SnapshotID
	a    data.Op
	b    data.Op

	// If defined as nil, that means we expect the transform to fail.
	files files
}

type files map[string]string

func TestTransform(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	bazFile := f.create(wf("baz.txt", "baz"))
	bazBarFile := f.create(wf("bar.txt", "bar"), bazFile)

	expected := []TransformExpectation{
		{data.EmptySnapshotID, wf("foo.txt", "foo"), dir("bar"), files{"bar/foo.txt": "foo"}},
		{bazFile, ins("baz.txt", 3, "x"), dir("bar"), files{"bar/baz.txt": "bazx"}},
		{bazFile, del("baz.txt", 1, 1), dir("bar"), files{"bar/baz.txt": "bz"}},
		{bazFile, del("baz.txt", 1, 1), f.preserve("baz.txt"), files{"baz.txt": "bz"}},
		{bazFile, del("baz.txt", 1, 1), f.preserveStrip("baz.txt"), files{"baz.txt": ""}},
		{bazFile, rm("baz.txt"), f.preserve("baz.txt"), files{}},
		{bazFile, wf("baz.txt", "baz"), f.preserveStrip("baz.txt"), files{"baz.txt": ""}},
		{bazFile, del("baz.txt", 1, 1), f.preserve("foo.txt"), files{}},
		{bazBarFile, f.preserveFile("bar.txt"), f.preserveFile("baz.txt"), files{}},
		{bazBarFile, f.preserveFile("bar.txt"), f.preserve("*.txt"), files{"bar.txt": "bar"}},
		{bazBarFile, f.preserveFile("bar.txt"), dir("dir"), files{"dir/bar.txt": "bar"}},
		{bazBarFile, f.preserveFile("bar.txt"), rm("bar.txt"), files{}},
		{bazBarFile, f.preserveFile("bar.txt"), rm("baz.txt"), files{"bar.txt": "bar"}},
		{data.EmptySnapshotID, wf("foo.txt", "foo"), dir("bar", "baz"), nil},
		{bazFile, editIns("baz.txt", 3, "x"), dir("bar"), files{"bar/baz.txt": "bazx"}},
		{bazFile, editDel("baz.txt", 1, 1), dir("bar"), files{"bar/baz.txt": "bz"}},
		{bazFile, rm("baz.txt"), dir("bar"), files{}},
		{data.EmptySnapshotID, wf("foo.txt", "foo"), identity(), files{"foo.txt": "foo"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("Transform%d(%s, %s)", i, e.a, e.b), func(t *testing.T) {
			if err := f.expectTransform(e); err != nil {
				t.Error(err)
			}
		})
	}
}

type fixture struct {
	t  *testing.T
	db dbint.DB2
}

func setup(t *testing.T) (*fixture, error) {
	store := storages.NewTestMemoryRecipeStore()
	db := db2.NewDB2(store, storages.NewMemoryPointers())

	return &fixture{t: t, db: db}, nil
}

func (f *fixture) tearDown() {}

func (f *fixture) expectTransform(ex TransformExpectation) error {
	err := f.expectTransformHelper(ex)
	if err != nil {
		return err
	}

	err = f.expectTransformHelper(TransformExpectation{base: ex.base, a: ex.b, b: ex.a, files: ex.files})
	if err != nil {
		return fmt.Errorf("InvertedTransform: %v", err)
	}
	return nil
}

func (f *fixture) expectTransformHelper(ex TransformExpectation) error {
	expectNoTransform := ex.files == nil

	aPrime, bPrime, err := transform.Transform(ex.a, ex.b)
	if err != nil {
		if transform.IsNoTransformErr(err) && expectNoTransform {
			return nil
		}
		return err
	}

	if expectNoTransform {
		return fmt.Errorf("Expected Transform() to fail")
	}

	id1 := f.create(bPrime, f.create(ex.a, ex.base))
	err = f.expectSnapshot(id1, ex.files)
	if err != nil {
		return fmt.Errorf("Applying A, B': (%s, %s): %v", ex.a, bPrime, err)
	}

	id2 := f.create(aPrime, f.create(ex.b, ex.base))
	err = f.expectSnapshot(id2, ex.files)
	if err != nil {
		return fmt.Errorf("Applying B, A': (%s, %s): %v", ex.b, aPrime, err)
	}
	return nil
}

func (f *fixture) expectSnapshot(id data.SnapshotID, files files) error {
	snapshot, err := f.db.Snapshot(ctx, id)
	if err != nil {
		return err
	}

	if len(snapshot.PathSet()) != len(files) {
		return fmt.Errorf("Expected path count: %d. Actual: %d. %v", len(files), len(snapshot.PathSet()), snapshot.PathSet())
	}

	for path, contents := range files {
		file, err := f.db.SnapshotFile(ctx, id, path)
		if err != nil {
			return err
		}

		if file == nil {
			return fmt.Errorf("File missing from snapshot: %s, %v", path, files)
		}

		if file.Contents.String() != contents {
			return fmt.Errorf("Expected file %s contents %q, actual %q", path, contents, file.Contents.String())
		}
	}

	return nil
}

func identity() *data.IdentityOp {
	return &data.IdentityOp{}
}

func wf(path string, bytes string) *data.WriteFileOp {
	return &data.WriteFileOp{Path: path, Data: data.NewBytes([]byte(bytes))}
}

func rm(path string) *data.RemoveFileOp {
	return &data.RemoveFileOp{Path: path}
}

func dir(names ...string) *data.DirOp {
	return &data.DirOp{Names: names}
}

func ins(path string, index int64, bytes string) *data.InsertBytesFileOp {
	return &data.InsertBytesFileOp{Path: path, Index: index, Data: data.NewBytes([]byte(bytes))}
}

func del(path string, index int64, count int64) *data.DeleteBytesFileOp {
	return &data.DeleteBytesFileOp{Path: path, Index: index, DeleteCount: count}
}

func editIns(path string, index int64, bytes string) *data.EditFileOp {
	return &data.EditFileOp{Path: path, Splices: []data.EditFileSplice{
		&data.InsertBytesSplice{Index: index, Data: data.NewBytes([]byte(bytes))},
	}}
}

func editDel(path string, index int64, count int64) *data.EditFileOp {
	return &data.EditFileOp{Path: path, Splices: []data.EditFileSplice{
		&data.DeleteBytesSplice{Index: index, DeleteCount: count},
	}}
}

func (f *fixture) preserve(pattern string) *data.PreserveOp {
	m, err := dbpath.NewMatcherFromPattern(pattern)
	if err != nil {
		f.t.Fatal(err)
	}

	return &data.PreserveOp{Matcher: m}
}

func (f *fixture) preserveFile(path string) *data.PreserveOp {
	m, err := dbpath.NewFileMatcher(path)
	if err != nil {
		f.t.Fatal(err)
	}

	return &data.PreserveOp{Matcher: m}
}

func (f *fixture) preserveStrip(pattern string) *data.PreserveOp {
	m, err := dbpath.NewMatcherFromPattern(pattern)
	if err != nil {
		f.t.Fatal(err)
	}

	return &data.PreserveOp{Matcher: m, StripContents: true}
}

func (f *fixture) create(op data.Op, inputs ...data.SnapshotID) data.SnapshotID {
	r, _, err := f.db.Create(ctx, data.Recipe{Op: op, Inputs: inputs}, data.UserTestID, data.RecipeWTagEdit)
	if err != nil {
		f.t.Fatal(err)
	}
	return r
}
