package db2

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/dbpath"
	"github.com/windmilleng/mish/data/db/storage"
	"github.com/windmilleng/mish/data/db/storage/storages"
)

type SnapshotExpectation struct {
	id    data.SnapshotID
	files files
}

type SnapshotReaderExpectation struct {
	reader dbint.SnapshotFileReader
	files  files
}

type SnapshotIDExpectation struct {
	id   data.SnapshotID
	path string
	ids  []data.SnapshotID
}

type PointerQueryExpectation struct {
	id   data.PointerID
	path string
	revs []data.PointerRev
}

type PathsChangedExpectation struct {
	start data.SnapshotID
	end   data.SnapshotID
	paths []string
}

type CostExpectation struct {
	id      data.SnapshotID
	pattern string
	ops     int32
}

type files map[string]string

func TestWriteFile(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("foo.txt", "foo"))
	s2 := f.create(wf("bar.txt", "bar"), s1)
	s3 := f.create(wf("foo.txt", "foo2"), s2)

	// now test
	expected := []SnapshotExpectation{
		{s1, files{"foo.txt": "foo"}},
		{s2, files{"foo.txt": "foo", "bar.txt": "bar"}},
		{s3, files{"foo.txt": "foo2", "bar.txt": "bar"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}

	expectedIDs := []SnapshotIDExpectation{
		{s3, "foo.txt", []data.SnapshotID{s3, s1}},
		{s3, "bar.txt", []data.SnapshotID{s2}},
	}

	for i, e := range expectedIDs {
		t.Run(fmt.Sprintf("Accumulator %d: %v", i, e.path), func(t *testing.T) {
			if err := f.expectSnapshotIDs(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestWriteFileToPtr(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr-1")
	f.writeToPtr(ptr, wf("foo.txt", "foo"))
	f.writeToPtr(ptr, wf("bar.txt", "bar"))
	f.writeToPtr(ptr, wf("foo.txt", "foo2"))

	expected := []PointerQueryExpectation{
		{ptr, "foo.txt", []data.PointerRev{3, 1}},
		{ptr, "bar.txt", []data.PointerRev{2}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("PointerQuery %d: %v", i, e.path), func(t *testing.T) {
			if err := f.expectPointerQuery(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPathsChanged(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("", "foo"))
	s2 := f.create(wf("", "bar"))
	s3 := f.create(dir("foo.txt", "bar.txt"), s1, s2)
	s4 := f.create(dir("another"), s3)
	s5 := f.create(subdir("another"), s4)
	s6 := f.create(wf("baz.txt", "baz"), s5)
	s7 := f.create(wf("foo.txt", "foo2"), s3)
	s8 := f.create(wf("bar.txt", "bar2"), s7)
	s9 := f.create(dir("dir"), s4)
	s10 := f.create(ins("dir/another/bar.txt", 1, "x"), s9)
	s11 := f.create(del("dir/another/foo.txt", 1, 1), s10)

	expected := []PathsChangedExpectation{
		{data.EmptySnapshotID, s5, []string{"bar.txt", "foo.txt"}},
		{s2, s5, []string{"", "bar.txt", "foo.txt"}},
		{s2, s4, []string{"", "another/bar.txt", "another/foo.txt"}},
		{s2, s6, []string{"", "bar.txt", "foo.txt", "baz.txt"}},
		{data.EmptySnapshotID, s6, []string{"bar.txt", "foo.txt", "baz.txt"}},
		{s3, s6, []string{"baz.txt"}},
		{s7, s3, []string{"foo.txt"}},
		{s8, s7, []string{"bar.txt"}},
		{s8, s6, []string{"foo.txt", "bar.txt", "baz.txt"}},
		{s8, s3, []string{"foo.txt", "bar.txt"}},
		{s10, s9, []string{"dir/another/bar.txt"}},
		{s11, s9, []string{"dir/another/bar.txt", "dir/another/foo.txt"}},
		{s8, s8, []string{}},
		{s11, s11, []string{}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("PathsChanged %d: %v, %v", i, e.start, e.end), func(t *testing.T) {
			if err := f.expectPathsChanged(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOverlay(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("", "foo"))
	s2 := f.create(wf("", "bar"))
	s3 := f.create(dir("foo.txt", "bar.txt"), s1, s2)
	s4 := f.create(dir("dir"), s3)

	s5 := f.create(wf("", "bar2"))
	s6 := f.create(wf("", "baz2"))
	s7 := f.create(dir("bar.txt", "baz.txt"), s5, s6)
	s8 := f.create(dir("dir"), s7)
	s8b := f.create(dir("dir2"), s7)

	s9 := f.create(overlay(), s3, s3)
	s10 := f.create(overlay(), s3, s7)
	s11 := f.create(overlay(), s4, s8)
	s12 := f.create(overlay(), s4, s8b)

	expected := []SnapshotExpectation{
		{s9, files{"foo.txt": "foo", "bar.txt": "bar"}},
		{s10, files{"foo.txt": "foo", "bar.txt": "bar2", "baz.txt": "baz2"}},
		{s11, files{"dir/foo.txt": "foo", "dir/bar.txt": "bar2", "dir/baz.txt": "baz2"}},
		{s12, files{"dir/foo.txt": "foo", "dir/bar.txt": "bar", "dir2/bar.txt": "bar2", "dir2/baz.txt": "baz2"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDir(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("", "foo"))
	s2 := f.create(wf("", "bar"))
	s3 := f.create(dir("foo.txt", "bar.txt"), s1, s2)
	s4 := f.create(dir("another"), s3)

	// now test
	expected := []SnapshotExpectation{
		{s3, files{"foo.txt": "foo", "bar.txt": "bar"}},
		{s4, files{"another/foo.txt": "foo", "another/bar.txt": "bar"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}

	expectedIDs := []SnapshotIDExpectation{
		{s4, "another/foo.txt", []data.SnapshotID{s4, s3, s1}},
		{s4, "another/bar.txt", []data.SnapshotID{s4, s3, s2}},
	}

	for i, e := range expectedIDs {
		t.Run(fmt.Sprintf("Accumulator %d: %v", i, e.path), func(t *testing.T) {
			if err := f.expectSnapshotIDs(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSubDir(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("", "foo"))
	s2 := f.create(wf("", "bar"))
	s3 := f.create(dir("foo.txt", "bar.txt"), s1, s2)
	s4 := f.create(dir("another"), s3)
	s5 := f.create(subdir("another"), s4)

	expected := []SnapshotExpectation{
		{s3, files{"foo.txt": "foo", "bar.txt": "bar"}},
		{s5, files{"foo.txt": "foo", "bar.txt": "bar"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}

	expectedIDs := []SnapshotIDExpectation{
		{s5, "foo.txt", []data.SnapshotID{s5, s4, s3, s1}},
		{s5, "bar.txt", []data.SnapshotID{s5, s4, s3, s2}},
	}

	for i, e := range expectedIDs {
		t.Run(fmt.Sprintf("Accumulator %d: %v", i, e.path), func(t *testing.T) {
			if err := f.expectSnapshotIDs(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDirNested(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("", "foo"))
	s2 := f.create(dir("foo/bar/baz/foo.txt"), s1)
	s3 := f.create(subdir("foo"), s2)
	s4 := f.create(subdir("bar"), s3)
	s5 := f.create(subdir("baz"), s4)

	e := SnapshotExpectation{s5, files{"foo.txt": "foo"}}
	if err := f.expectSnapshot(e); err != nil {
		t.Fatal(err)
	}

	eIDs := SnapshotIDExpectation{s5, "foo.txt", []data.SnapshotID{s5, s4, s3, s2, s1}}
	if err := f.expectSnapshotIDs(eIDs); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveFile(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("", "foo"))
	s2 := f.create(rm(""), s1)
	s3 := f.create(wf("foo/bar.txt", "bar"))
	s4 := f.create(wf("foo/baz.txt", "baz"), s3)
	s5 := f.create(rm("foo/bar.txt"), s4)

	expected := []SnapshotExpectation{
		{s2, files{}},
		{s5, files{"foo/baz.txt": "baz"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPreserve(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("foo.txt", "foo"))
	s2 := f.create(wf("bar.txt", "bar"), s1)

	foo2 := f.create(f.preserve("foo.txt"), s2)
	bar2 := f.create(f.preserve("bar.txt"), s2)

	s3 := f.create(wf("foo.txt", "foo3"), s2)
	foo3 := f.create(f.preserve("foo.txt"), s3)
	bar3 := f.create(f.preserve("bar.txt"), s3)
	fooStrip3 := f.create(f.preserveStrip("foo.txt"), s3)
	barStrip3 := f.create(f.preserveStrip("bar.txt"), s3)

	expected := []SnapshotExpectation{
		{foo2, files{"foo.txt": "foo"}},
		{bar2, files{"bar.txt": "bar"}},
		{foo3, files{"foo.txt": "foo3"}},
		{bar3, files{"bar.txt": "bar"}},
		{fooStrip3, files{"foo.txt": ""}},
		{barStrip3, files{"bar.txt": ""}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInsertBytesFile(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("test.txt", "hello"))
	s2 := f.create(ins("test.txt", 4, " n"), s1)

	expected := []SnapshotExpectation{
		{s2, files{"test.txt": "hell no"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDeleteBytesFile(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("test.txt", "hello"))
	s2 := f.create(del("test.txt", 4, 1), s1)

	expected := []SnapshotExpectation{
		{s2, files{"test.txt": "hell"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEditFile(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("test.txt", "hello"))
	s2 := f.create(editIns("test.txt", 4, " n"), s1)
	s3 := f.create(editDel("test.txt", 4, 1), s1)

	expected := []SnapshotExpectation{
		{s2, files{"test.txt": "hell no"}},
		{s3, files{"test.txt": "hell"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestChmodFile(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("test.txt", "hello"))
	s2 := f.create(chmod("test.txt", true), s1)

	// TODO(dbentley): how to test

	expected := []SnapshotExpectation{
		{s2, files{"test.txt": "hello"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRmdir(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("foo/bar/baz.txt", "baz"))
	s2 := f.create(rmdir(""), s1)
	s3 := f.create(rmdir("foo"), s1)
	s4 := f.create(rmdir("foo/bar"), s1)
	s5 := f.create(rmdir("quux"), s1)

	expected := []SnapshotExpectation{
		{s2, files{}},
		{s3, files{}},
		{s4, files{}},
		{s5, files{"foo/bar/baz.txt": "baz"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}

	expectedIDs := []SnapshotIDExpectation{
		{s2, "foo/bar/baz.txt", []data.SnapshotID{s2, s1}},
		{s3, "foo/bar/baz.txt", []data.SnapshotID{s3, s1}},
		{s4, "foo/bar/baz.txt", []data.SnapshotID{s4, s1}},
		{s5, "foo/bar/baz.txt", []data.SnapshotID{s1}},
	}

	for i, e := range expectedIDs {
		t.Run(fmt.Sprintf("Accumulator %d: %v", i, e.path), func(t *testing.T) {
			if err := f.expectSnapshotIDs(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSnapshotEvalShortCircuits(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("foo.txt", "foo"))
	s2 := f.create(wf("bar.txt", "bar"), s1)

	matcher, _ := dbpath.NewFileMatcher("foo.txt")
	result, err := f.db.eval(f.ctx, data.RecipeRTagEdit, s2, &snapshotEvaluator{matcher: matcher})
	if err != nil {
		t.Fatal(err)
	}

	foo, err := lookupFile(result.(snapshotVal), "foo.txt", onNotFoundError)
	if err != nil {
		t.Fatal(err)
	}

	if foo.data.String() != "foo" {
		t.Errorf("Unexpected contents of foo.txt: %s", foo.data.String())
	}

	_, err = lookupFile(result.(snapshotVal), "bar.txt", onNotFoundError)
	if err == nil {
		t.Errorf("Expected error looking up bar")
	} else if err.Error() != "Lookup(\"bar.txt\"): does not exist" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestContentIDBookkeeping(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("", "foo"))
	s2 := f.create(dir("foo.txt"), s1)
	s3 := f.create(ins("foo.txt", 0, "hello "), s2)

	matcher, _ := dbpath.NewFileMatcher("foo.txt")
	result, err := f.db.eval(f.ctx, data.RecipeRTagEdit, s2, &snapshotEvaluator{matcher: matcher})
	if err != nil {
		t.Fatal(err)
	}

	foo, err := lookupFile(result.(snapshotVal), "foo.txt", onNotFoundError)
	if err != nil {
		t.Fatal(err)
	}

	cID := foo.contentID()
	if cID != s1 {
		t.Errorf("Expected content ID %q, actual %q", s1, cID)
	}

	result, err = f.db.eval(f.ctx, data.RecipeRTagEdit, s3, &snapshotEvaluator{matcher: matcher})
	if err != nil {
		t.Fatal(err)
	}

	cID = result.(snapshotVal).contentID()
	if !cID.Nil() {
		t.Errorf("Expected no content ID, actual %q", cID)
	}
}

func TestOptimizeSnapshot(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("foo.txt", "foo"))
	s2 := f.create(wf("bar.txt", "bar"), s1)
	s3 := f.create(wf("foo.txt", "foo2"), s2)
	s4 := f.create(dir("baz"), s1)
	s5 := f.create(wf("baz/foo.txt", "foo3"), s4)

	o := NewOptimizer(f.db, f.store)
	f.countingStore.createCount = 0

	_, err = o.OptimizeSnapshot(f.ctx, s1)
	if err != nil {
		t.Fatal(err)
	}

	if f.countingStore.createCount != 2 {
		t.Fatalf("Expected 2 create. Actual: %d", f.countingStore.createCount)
	}
	f.countingStore.createCount = 0

	_, err = o.OptimizeSnapshot(f.ctx, s2)
	if err != nil {
		t.Fatal(err)
	}

	if f.countingStore.createCount != 2 {
		t.Errorf("Expected 2 create. Actual: %d", f.countingStore.createCount)
	}
	f.countingStore.createCount = 0

	_, err = o.OptimizeSnapshot(f.ctx, s3)
	if err != nil {
		t.Fatal(err)
	}

	if f.countingStore.createCount != 2 {
		t.Fatalf("Expected 2 create. Actual: %d", f.countingStore.createCount)
	}
	f.countingStore.createCount = 0

	_, err = o.OptimizeSnapshot(f.ctx, s3)
	if err != nil {
		t.Fatal(err)
	}

	if f.countingStore.createCount != 0 {
		t.Errorf("Expected 0 create. Actual: %d", f.countingStore.createCount)
	}
	f.countingStore.createCount = 0

	_, err = o.OptimizeSnapshot(f.ctx, s5)
	if err != nil {
		t.Fatal(err)
	}

	if f.countingStore.createCount != 3 {
		t.Errorf("Expected 3 creates. Actual: %d", f.countingStore.createCount)
	}
	f.countingStore.createCount = 0

	// now test
	expected := []SnapshotExpectation{
		{s1, files{"foo.txt": "foo"}},
		{s2, files{"foo.txt": "foo", "bar.txt": "bar"}},
		{s3, files{"foo.txt": "foo2", "bar.txt": "bar"}},
		{s5, files{"baz/foo.txt": "foo3"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}

			if err := f.expectTwoPaths(e.id); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOptimizeSnapshotToEmpty(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("a/b/c.txt", "c"))
	s2 := f.create(rm("a/b/c.txt"), s1)
	o := NewOptimizer(f.db, f.store)
	s3, err := o.OptimizeSnapshot(f.ctx, s2)
	if err != nil {
		t.Fatal(err)
	}

	if s3 != data.EmptySnapshotID {
		t.Errorf("Expected empty snapshot ID. Actual: %v", s2)
	}
}

func TestOptimizeSnapshotFastPath(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("a/b/c.txt", "c"))
	s2 := f.create(wf("d/e/f.txt", "f"), s1)
	s3 := f.create(wf("d/g/h.txt", "h"), s2)
	s4 := f.create(wf("i/j.txt", "j"), s3)
	s5 := f.create(rm("a/b/c.txt"), s4)
	s6 := f.create(wf("i/j.txt", "k"), s5)

	// now test
	expected := []SnapshotExpectation{
		{s1, files{"a/b/c.txt": "c"}},
		{s2, files{"a/b/c.txt": "c", "d/e/f.txt": "f"}},
		{s3, files{"a/b/c.txt": "c", "d/e/f.txt": "f", "d/g/h.txt": "h"}},
		{s4, files{"a/b/c.txt": "c", "d/e/f.txt": "f", "d/g/h.txt": "h", "i/j.txt": "j"}},
		{s4, files{"a/b/c.txt": "c", "d/e/f.txt": "f", "d/g/h.txt": "h", "i/j.txt": "j"}},
		{s5, files{"d/e/f.txt": "f", "d/g/h.txt": "h", "i/j.txt": "j"}},
		{s6, files{"d/e/f.txt": "f", "d/g/h.txt": "h", "i/j.txt": "k"}},
	}

	o := NewOptimizer(f.db, f.store)
	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			_, err := o.optimizeSnapshotFastPath(f.ctx, e.id)
			if err != nil {
				t.Fatal(err)
			}

			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}

			if err := f.expectTwoPaths(e.id); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOptimizeSnapshotFilesMatching(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("rev/1/foo.txt", "foo"))
	s2 := f.create(wf("rev/1/bar.txt", "bar"), s1)
	s3 := f.create(wf("rev/2/foo.txt", "foo"), s2)
	s4 := f.create(ins("rev/1/foo.txt", 3, "1"), s3)

	o := NewOptimizer(f.db, f.store)
	_, err = o.OptimizeSnapshot(f.ctx, s3)
	if err != nil {
		t.Fatal(err)
	}

	// now test
	expected := []SnapshotReaderExpectation{
		{
			f.filesMatching(s4, ".wmcmds/*"),
			files{},
		},
		{
			f.filesMatching(s4, "rev/*/foo.txt"),
			files{"rev/1/foo.txt": "foo1", "rev/2/foo.txt": "foo"},
		},
		{
			f.filesMatching(s4, "*/*/foo.txt"),
			files{"rev/1/foo.txt": "foo1", "rev/2/foo.txt": "foo"},
		},
		{
			f.filesMatching(s4, "*/1/*.txt"),
			files{"rev/1/foo.txt": "foo1", "rev/1/bar.txt": "bar"},
		},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("OptimizeSnapshotFilesMatching-%d", i), func(t *testing.T) {
			if err := f.expectSnapshotReader(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLookupFile(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("foo.txt", "foo"))

	// Test different ways of looking up a file.
	f1, err := f.db.SnapshotFile(f.ctx, s1, "foo.txt")
	if err != nil {
		t.Error(err)
	} else if f1.Contents.String() != "foo" {
		t.Errorf("Expected foo, actual: %s", f1.Contents.String())
	}

	matcher, _ := dbpath.NewFileMatcher("foo.txt")
	files, err := f.db.SnapshotFilesMatching(f.ctx, s1, matcher)
	if err != nil {
		t.Fatal(err)
	}

	f1, err = files.File(f.ctx, "foo.txt")
	if err != nil {
		t.Error(err)
	} else if f1.Contents.String() != "foo" {
		t.Errorf("Expected foo, actual: %s", f1.Contents.String())
	}

	_, err = f.db.SnapshotFile(f.ctx, s1, "bar.txt")
	if err == nil {
		t.Errorf("Expected error")
	} else if grpc.Code(err) != codes.NotFound {
		t.Errorf("Unexpected error: %v", err)
	}

	matcher, _ = dbpath.NewFileMatcher("bar.txt")
	files, err = f.db.SnapshotFilesMatching(f.ctx, s1, matcher)
	if err != nil {
		t.Fatal(err)
	}

	f1, err = files.File(f.ctx, "bar.txt")
	if err == nil {
		t.Errorf("Expected error")
	} else if grpc.Code(err) != codes.NotFound {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCost(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	snap := data.EmptySnapshotID
	for i := 0; i < 25; i++ {
		snap = f.create(wf(fmt.Sprintf("log%d.txt", i), "foo"), snap)
	}

	o := NewOptimizer(f.db, f.store)
	_, err = o.OptimizeSnapshot(f.ctx, snap)
	if err != nil {
		t.Fatal(err)
	}

	preserve1 := f.create(f.preserve("log1.txt"), snap)
	preserve1s := f.create(f.preserve("log1*.txt"), snap)

	// now test
	expected := []CostExpectation{
		{snap, "**", 27},
		{data.EmptySnapshotID, "**", 0},
		{snap, "log1.txt", 3},
		{snap, "log1.txt,log2.txt", 4},
		{snap, "log1*.txt", 13},
		{preserve1, "**", 4},
		{preserve1s, "**", 14},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectCost(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPreserveOnRemove(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("foo.txt", "foo"))
	s2 := f.create(rm("foo.txt"), s1)
	s3 := f.create(f.preserveFile("foo.txt"), s2)

	expected := []SnapshotExpectation{
		{s1, files{"foo.txt": "foo"}},
		{s2, files{}},
		{s3, files{}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectSnapshot(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRecipesNeeded(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.writeToPtr(ptr, wf("foo.txt", "foo"))
	f.writeToPtr(ptr, wf("bar.txt", "bar"))
	head := f.writeToPtr(ptr, wf("foo.txt", "foo2"))

	o := NewOptimizer(f.db, f.store)

	_, err = o.OptimizeSnapshot(f.ctx, head.SnapID)
	if err != nil {
		t.Fatal(err)
	}

	ptrRecipes, err := f.db.RecipesNeeded(f.ctx, data.EmptySnapshotID, head.SnapID, ptr)
	if err != nil {
		t.Fatal(err)
	}

	if len(ptrRecipes) != 3 ||
		ptrRecipes[0].Recipe.Op.(*data.WriteFileOp).FilePath() != "foo.txt" ||
		ptrRecipes[1].Recipe.Op.(*data.WriteFileOp).FilePath() != "bar.txt" ||
		ptrRecipes[2].Recipe.Op.(*data.WriteFileOp).FilePath() != "foo.txt" {
		t.Errorf("Unexpected recipes needed: %+v", ptrRecipes)
	}

	optRecipes, err := f.db.RecipesNeeded(f.ctx, data.EmptySnapshotID, head.SnapID, data.PointerID{})
	if err != nil {
		t.Fatal(err)
	}

	if len(optRecipes) != 4 {
		t.Errorf("Unexpected recipes needed: %+v", optRecipes)
	}

	if _, ok := optRecipes[0].Recipe.Op.(*data.IdentityOp); !ok {
		t.Errorf("Unexpected recipes needed: %+v", optRecipes)
	}
}

func TestOptimizeSymlink(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	s1 := f.create(wf("a.txt", "a"))
	s2 := f.create(wsymlink("b.txt", "a.txt"), s1)
	o := NewOptimizer(f.db, f.store)
	s3, err := o.OptimizeSnapshot(f.ctx, s2)
	if err != nil {
		t.Fatal(err)
	}

	err = f.expectSymlink(s2, "b.txt", "a.txt")
	if err != nil {
		t.Fatal(err)
	}

	err = f.expectSymlink(s3, "b.txt", "a.txt")
	if err != nil {
		t.Fatal(err)
	}
}

type fixture struct {
	t     *testing.T
	store *storages.MemoryRecipeStore

	ptrs          data.Pointers
	countingStore *CountingRecipeStore
	db            *DB2
	ctx           context.Context
}

func setup(t *testing.T) (*fixture, error) {
	memStore := storages.NewTestMemoryRecipeStore()
	countingStore := &CountingRecipeStore{RecipeStore: memStore}

	ptrs := storages.NewMemoryPointers()
	db := NewDB2(countingStore, ptrs)

	return &fixture{
		t:             t,
		store:         memStore,
		ptrs:          ptrs,
		countingStore: countingStore,
		db:            db,
		ctx:           context.Background(),
	}, nil
}

func (f *fixture) tearDown() {}

func (f *fixture) reset(t *testing.T) *fixture {
	f.tearDown()
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fixture) expectSnapshotIDs(ex SnapshotIDExpectation) error {
	matcher, _ := dbpath.NewFileMatcher(ex.path)
	actual, err := f.db.eval(f.ctx, data.RecipeRTagEdit, ex.id, &snapshotAccumulator{matcher: matcher})
	if err != nil {
		return err
	}

	actualIDs := actual.(*snapshotIDList).toSlice()

	if len(actualIDs) != len(ex.ids) {
		return fmt.Errorf("Expected %d ids, actual: %v", len(ex.ids), actualIDs)
	}

	for i, id := range actualIDs {
		if ex.ids[i] != id {
			return fmt.Errorf("Expected id %q at position %d, actual %q", ex.ids[i], i, id)
		}
	}

	return nil
}

func (f *fixture) expectPointerQuery(ex PointerQueryExpectation) error {
	matcher, _ := dbpath.NewFileMatcher(ex.path)
	actual, err := f.db.QueryPointerHistory(f.ctx, data.PointerHistoryQuery{
		Matcher: matcher,
		PtrID:   ex.id,
	})
	if err != nil {
		return err
	}

	if len(actual) != len(ex.revs) {
		return fmt.Errorf("Expected %d revs, actual: %v", len(ex.revs), actual)
	}

	for i, ptrAtSnapshot := range actual {
		if ex.revs[i] != ptrAtSnapshot.Rev {
			return fmt.Errorf("Expected rev %q at position %d, actual %q", ex.revs[i], i, ptrAtSnapshot.Rev)
		}
	}

	return nil
}

func (f *fixture) expectPathsChanged(ex PathsChangedExpectation) error {
	// Make sure pathsChanged is symmetric.
	err := f.expectPathsChangedHelper(ex)
	if err != nil {
		return err
	}

	err = f.expectPathsChangedHelper(PathsChangedExpectation{
		start: ex.end,
		end:   ex.start,
		paths: ex.paths,
	})
	if err != nil {
		return fmt.Errorf("Reversed PathsChangedExpectation: %v", err)
	}

	return nil
}

func (f *fixture) expectPathsChangedHelper(ex PathsChangedExpectation) error {
	actual, err := f.db.PathsChanged(f.ctx, ex.start, ex.end, data.RecipeRTagEdit, dbpath.NewAllMatcher())
	if err != nil {
		return err
	}

	sort.Strings(actual)
	sort.Strings(ex.paths)
	if len(actual) != len(ex.paths) {
		return fmt.Errorf("Expected %d paths changed, actual: %v", len(ex.paths), actual)
	}

	for i, p := range actual {
		if ex.paths[i] != p {
			return fmt.Errorf("Expected path %q at position %d, actual %q", ex.paths[i], i, p)
		}
	}

	return nil
}

func (f *fixture) filesMatching(id data.SnapshotID, pattern string) dbint.SnapshotFileReader {
	matcher, err := dbpath.NewMatcherFromPattern(pattern)
	if err != nil {
		f.t.Fatal(err)
	}

	reader, err := f.db.SnapshotFilesMatching(f.ctx, id, matcher)
	if err != nil {
		f.t.Fatal(err)
	}

	return reader
}

func (f *fixture) expectSnapshot(ex SnapshotExpectation) error {
	return f.expectSnapshotReader(SnapshotReaderExpectation{
		reader: dbint.FileReaderForSnapshot(f.db, ex.id),
		files:  ex.files,
	})
}

func (f *fixture) expectSnapshotReader(ex SnapshotReaderExpectation) error {
	fileList, err := ex.reader.FilesMatching(f.ctx, dbpath.NewAllMatcher())
	if err != nil {
		return err
	}

	files := fileList.AsMap()
	if len(files) != len(ex.files) {
		return fmt.Errorf("Expected path count: %d. Actual: %d", len(files), len(ex.files))
	}

	for path, contents := range ex.files {
		// Evaluate this twice: once with SnapshotFilesMatching and once with SnapshotFile
		file := files[path]
		if file == nil {
			return fmt.Errorf("File missing from snapshot: %s, %v", path, files)
		}

		if file.Contents.String() != contents {
			return fmt.Errorf("SnapQuery: Expected file %s contents %q, actual %q", path, contents, file.Contents.String())
		}

		file2, err := ex.reader.File(f.ctx, path)
		if err != nil {
			return err
		}

		if file2.Contents.String() != contents {
			return fmt.Errorf("SnapshotFile: Expected file %s contents %q, actual %q", path, contents, file2.Contents.String())
		}
	}

	return nil
}

func (f *fixture) expectSymlink(id data.SnapshotID, path string, target string) error {
	file, err := f.db.SnapshotFile(f.ctx, id, path)
	if err != nil {
		return err
	}

	if file.Type != data.FileSymlink {
		return fmt.Errorf("Expected symlink but got regular file at %q", path)
	}

	if file.Contents.String() != target {
		return fmt.Errorf("Expected symlink %s contents %q, actual %q", path, target, file.Contents.String())
	}

	return nil
}

func (f *fixture) expectTwoPaths(id data.SnapshotID) error {
	recipeOpt, err := f.db.LookupPathToSnapshot(f.ctx, data.RecipeRTagOptimal, id)
	if err != nil {
		return err
	}

	recipeEdit, err := f.db.LookupPathToSnapshot(f.ctx, data.RecipeRTagEdit, id)
	if err != nil {
		return err
	}

	if fmt.Sprintf("%v", recipeEdit) == fmt.Sprintf("%v", recipeOpt) {
		return fmt.Errorf("Expected two paths but only one recipe to %s: %v", id, recipeOpt)
	}

	// recipeOpt should always be an IdentityOp to a content id
	_, ok := recipeOpt.Op.(*data.IdentityOp)
	if !ok {
		return fmt.Errorf("Expected recipeOpt to be an IdentityOp. Actual: %T", recipeOpt)
	}
	if !recipeOpt.Inputs[0].IsContentID() {
		return fmt.Errorf("Expected recipeOpt to point to a content ID. Actual: %s", recipeOpt.Inputs[0])
	}
	return nil
}

func (f *fixture) expectCost(ex CostExpectation) error {
	patterns := strings.Split(ex.pattern, ",")
	matcher, err := dbpath.NewMatcherFromPatterns(patterns)
	if err != nil {
		return err
	}

	cost, err := f.db.Cost(f.ctx, ex.id, data.RecipeRTagOptimal, matcher)
	if err != nil {
		return err
	}

	if cost.Ops != ex.ops {
		return fmt.Errorf("Expected total cost %d. Actual cost %+v", ex.ops, cost)
	}
	return nil
}

func wf(path string, bytes string) *data.WriteFileOp {
	return &data.WriteFileOp{Path: path, Data: data.NewBytes([]byte(bytes))}
}

func wsymlink(path string, target string) *data.WriteFileOp {
	return &data.WriteFileOp{Path: path, Data: data.NewBytes([]byte(target)), Type: data.FileSymlink}
}

func rm(path string) *data.RemoveFileOp {
	return &data.RemoveFileOp{Path: path}
}

func dir(names ...string) *data.DirOp {
	return &data.DirOp{Names: names}
}

func overlay() *data.OverlayOp {
	return &data.OverlayOp{}
}

func subdir(path string) *data.SubdirOp {
	return &data.SubdirOp{Path: path}
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

func chmod(path string, executable bool) *data.ChmodFileOp {
	return &data.ChmodFileOp{Path: path, Executable: executable}
}

func rmdir(path string) *data.RmdirOp {
	return &data.RmdirOp{Path: path}
}

func (f *fixture) preserve(pattern string) *data.PreserveOp {
	m, err := dbpath.NewMatcherFromPattern(pattern)
	if err != nil {
		f.t.Fatal(err)
	}

	return &data.PreserveOp{Matcher: m}
}

func (f *fixture) preserveFile(pattern string) *data.PreserveOp {
	m, err := dbpath.NewFileMatcher(pattern)
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
	r, _, err := f.db.Create(f.ctx, data.Recipe{Op: op, Inputs: inputs}, data.UserTestID, data.RecipeWTagEdit)
	if err != nil {
		f.t.Fatal(err)
	}
	return r
}

func (f *fixture) writeToPtr(ptrID data.PointerID, op data.Op) data.PointerAtSnapshot {
	head, err := dbint.EditPointer(f.ctx, f.db, ptrID, op)
	if err != nil {
		f.t.Fatal(err)
	}
	return head
}

func ops(ops ...data.Op) []data.Op {
	return ops
}

type CountingRecipeStore struct {
	storage.RecipeStore
	createCount int
}

func (s *CountingRecipeStore) Create(ctx context.Context, r data.Recipe, owner data.UserID, t data.RecipeWTag) (data.SnapshotID, bool, error) {
	s.createCount++
	return s.RecipeStore.Create(ctx, r, owner, t)
}

func (s *CountingRecipeStore) CreatePath(ctx context.Context, r data.StoredRecipe) error {
	return s.RecipeStore.CreatePath(ctx, r)
}
