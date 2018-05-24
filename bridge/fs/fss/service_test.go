package fss

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/windmilleng/mish/bridge/fs"
	"github.com/windmilleng/mish/bridge/fs/proto"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/storage/storages"
	"github.com/windmilleng/mish/os/ospath"
	"github.com/windmilleng/mish/os/temp"
	"github.com/windmilleng/mish/os/watch"
)

type CheckoutExpectation struct {
	id    data.SnapshotID
	files files
}

type files map[string]string

func TestCheckout(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	id1 := f.writeFileWM("foo.txt", "bar")
	id2 := f.writeFileWM("baz.txt", "quux")
	id3 := f.writeFileWM("foo.txt", "bat")

	// now test
	expected := []CheckoutExpectation{
		{id1, files{"foo.txt": "bar"}},
		{id2, files{"foo.txt": "bar", "baz.txt": "quux"}},
		{id3, files{"foo.txt": "bat", "baz.txt": "quux"}},
	}

	for i, e := range expected {
		t.Run(fmt.Sprintf("%d: %v", i, e.id), func(t *testing.T) {
			if err := f.expectCheckout(e); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCheckoutSymlink(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	f.writeFileWM("foo.txt", "foo")
	id := f.writeSymlinkWM("foo.txt", "bar.txt")
	checkout, err := f.fs.Checkout(f.ctx, id, "")
	if err != nil {
		t.Fatal(err)
	}

	link, err := os.Readlink(filepath.Join(checkout.Path, "bar.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if link != "foo.txt" {
		t.Errorf("Expected link to foo.txt, actual %s", link)
	}
}

func TestResetCheckout(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	f.writeFileWM("foo.txt", "foo")
	f.writeFileWM("bar.txt", "bar")
	id := f.writeFileWM("baz.txt", "baz")

	root := f.root.Path()
	_, err = f.fs.Checkout(f.ctx, id, f.root.Path())
	if err != nil {
		t.Fatal(err)
	}

	originalFiles := files{"foo.txt": "foo", "bar.txt": "bar", "baz.txt": "baz"}
	err = f.expectFiles(root, originalFiles)
	if err != nil {
		t.Error(err)
	}

	f.writeFileFS("foo.txt", "foo2")
	f.writeFileFS("log.txt", "log")
	err = os.Remove(filepath.Join(root, "bar.txt"))
	if err != nil {
		t.Fatal(err)
	}

	err = f.expectFiles(root, files{"foo.txt": "foo2", "baz.txt": "baz", "log.txt": "log"})
	if err != nil {
		t.Error(err)
	}

	err = f.fs.ResetCheckout(f.ctx, fs.CheckoutStatus{SnapID: id, Path: f.root.Path()})
	if err != nil {
		t.Error(err)
	}

	err = f.expectFiles(root, originalFiles)
	if err != nil {
		t.Error(err)
	}
}

func TestMirrorFS2WM(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Test times out on macOS. Deadlock somewhere in fsnotify.")
	}

	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.writeFileFS("foo.txt", "1")
	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, ospath.NewAllMatcher())
	f.writeFileFS("bar.txt", "2")

	f.fs.ToWMFSync(f.ctx, ptr)
	head, _ := dbint.HeadSnap(f.ctx, f.db, ptr)
	rev1Snap, err := f.db.Get(f.ctx, data.PointerAtRev{ID: ptr, Rev: 1})
	if err != nil {
		t.Fatal(err)
	}

	err = f.expectSnapshotFiles(head.SnapID, files{
		"foo.txt": "1",
		"bar.txt": "2",
	})
	if err != nil {
		t.Fatal(err)
	}

	recipe1a, err := f.db.LookupPathToSnapshot(f.ctx, data.RecipeRTagEdit, rev1Snap.SnapID)
	if err != nil {
		t.Fatal(err)
	}

	recipe1b, err := f.db.LookupPathToSnapshot(f.ctx, data.RecipeRTagOptimal, rev1Snap.SnapID)
	if err != nil {
		t.Fatal(err)
	}

	if recipe1a.Tag.Type != data.RecipeTagTypeTemp || recipe1b.Tag.Type != data.RecipeTagTypeOptimal {
		t.Errorf("Unexpected recipe tags: %v, %v", recipe1a.Tag, recipe1b.Tag)
	}

	recipe2, err := f.db.LookupPathToSnapshot(f.ctx, data.RecipeRTagEdit, head.SnapID)
	if err != nil {
		t.Fatal(err)
	}

	if recipe2.Tag.Type != data.RecipeTagTypeEdit {
		t.Errorf("Unexpected recipe tags: %v", recipe2.Tag)
	}

	status, err := f.fs.ToWMStatus(f.ctx, f.root.Path())
	if err != nil {
		t.Fatal(err)
	}
	if status == nil || status.Pointer != ptr.String() {
		t.Errorf("Unexpected status: %v", status)
	}

	status, err = f.fs.ToWMPointerStatus(f.ctx, ptr)
	if err != nil {
		t.Fatal(err)
	}
	if status == nil || string(status.Path) != f.root.Path() {
		t.Errorf("Unexpected status: %v", status)
	}
}

func TestMirrorFS2WMIgnore(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Test times out on macOS. Deadlock somewhere in fsnotify.")
	}

	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.writeFileFS("foo.txt", "1")
	f.writeFileFS("bar.txt", "2")

	matcher, _ := ospath.NewFileMatcher("foo.txt")
	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, matcher)

	f.fs.ToWMFSync(f.ctx, ptr)
	head, _ := dbint.HeadSnap(f.ctx, f.db, ptr)
	err = f.expectSnapshotFiles(head.SnapID, files{
		"foo.txt": "1",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMirrorFS2WMEmptyIgnoreList(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.writeFileFS("foo.txt", "1")
	f.writeFileFS("bar.txt", "2")

	matcher := ospath.NewEmptyMatcher()
	err = f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, matcher)

	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestStopFS2WM(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, ospath.NewAllMatcher())
	f.writeFileFS("foo.txt", "1")
	f.fs.ToWMFSync(f.ctx, ptr)
	f.fs.ToWMStop(f.ctx, ptr)

	f.writeFileFS("bar.txt", "2")
	time.Sleep(10 * time.Millisecond)

	head, err := dbint.HeadSnap(f.ctx, f.db, ptr)
	if err != nil {
		t.Fatal(err)
	}

	err = f.expectSnapshotFiles(head.SnapID, files{
		"foo.txt": "1",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStopAndRestartFS2WM(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, ospath.NewAllMatcher())
	f.writeFileFS("foo.txt", "1")
	f.fs.ToWMFSync(f.ctx, ptr)
	f.fs.ToWMStop(f.ctx, ptr)

	// Rev 1 = foo.txt write
	head, _ := dbint.HeadSnap(f.ctx, f.db, ptr)
	if head.Rev != 1 {
		recipes, _ := f.db.RecipesNeeded(f.ctx, data.EmptySnapshotID, head.SnapID, ptr)
		t.Errorf("Expected rev 1. Actual: %+v\n%+v", head, recipes)
	}

	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, ospath.NewAllMatcher())

	f.writeFileFS("bar.txt", "2")
	f.fs.ToWMFSync(f.ctx, ptr)

	// Rev 2 = bar.txt write
	head, _ = dbint.HeadSnap(f.ctx, f.db, ptr)
	if head.Rev != 2 {
		recipes, _ := f.db.RecipesNeeded(f.ctx, data.EmptySnapshotID, head.SnapID, ptr)
		t.Errorf("Expected rev 2. Actual: %+v\nRecipes: %+v", head, recipes)
	}

	err = f.expectSnapshotFiles(head.SnapID, files{
		"foo.txt": "1",
		"bar.txt": "2",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStopAndRestartFS2WMIgnore(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, ospath.NewAllMatcher())
	f.writeFileFS("foo.txt", "1")
	f.writeFileFS("bar.txt", "2")
	f.fs.ToWMFSync(f.ctx, ptr)
	f.fs.ToWMStop(f.ctx, ptr)

	head, _ := dbint.HeadSnap(f.ctx, f.db, ptr)
	err = f.expectSnapshotFiles(head.SnapID, files{
		"foo.txt": "1",
		"bar.txt": "2",
	})
	if err != nil {
		t.Fatal(err)
	}

	matcher, _ := ospath.NewFileMatcher("foo.txt")
	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, matcher)
	f.fs.ToWMFSync(f.ctx, ptr)

	head, _ = dbint.HeadSnap(f.ctx, f.db, ptr)
	err = f.expectSnapshotFiles(head.SnapID, files{
		"foo.txt": "1",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestShutdownFS2WM(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, ospath.NewAllMatcher())
	f.writeFileFS("foo.txt", "1")
	f.fs.ToWMFSync(f.ctx, ptr)
	f.fs.Shutdown(f.ctx)

	f.writeFileFS("bar.txt", "2")
	time.Sleep(10 * time.Millisecond)

	head, err := dbint.HeadSnap(f.ctx, f.db, ptr)
	if err != nil {
		t.Fatal(err)
	}

	err = f.expectSnapshotFiles(head.SnapID, files{
		"foo.txt": "1",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMirrorFS2WMOnlyOnePointerEdit(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	os.Mkdir(path.Join(f.root.Path(), "bar"), os.FileMode(0777))
	os.Mkdir(path.Join(f.root.Path(), "baz"), os.FileMode(0777))

	ptr := data.MustParsePointerID("ptr")
	f.writeFileFS("foo1.txt", "1")
	f.writeFileFS("bar/foo2.txt", "2")
	f.writeFileFS("bar/foo3.txt", "3")
	f.writeFileFS("baz/foo4.txt", "4")
	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, ospath.NewAllMatcher())

	f.fs.ToWMFSync(f.ctx, ptr)
	head, _ := dbint.HeadSnap(f.ctx, f.db, ptr)
	if head.Rev != 1 {
		t.Errorf("Expected exactly one pointer rev, got %d", head.Rev)
	}

	err = f.expectSnapshotFiles(head.SnapID, files{
		"foo1.txt":     "1",
		"bar/foo2.txt": "2",
		"bar/foo3.txt": "3",
		"baz/foo4.txt": "4",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFS2WMFSync(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Test times out on macOS. Deadlock somewhere in fsnotify.")
	}

	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.writeFileFS("foo.txt", "1")
	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, ospath.NewAllMatcher())
	for i := 2; i < 5; i++ {
		f.writeFileFS("bar.txt", strconv.Itoa(i))
		head, err := f.fs.ToWMFSync(f.ctx, ptr)
		if err != nil {
			t.Fatal(err)
		}

		if int(head.Rev) != i {
			t.Errorf("FSync did not sync to head. Expected %d, Actual %d", i, head.Rev)
		}
	}
}

func TestMirrorFS2WMAutoSave(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.writeFileFS("foo.txt", "1")

	ctx, cancel := context.WithTimeout(f.ctx, time.Second)
	defer cancel()
	ch := make(chan *proto.FsBridgeState)
	f.fs.AddChangeListener(ctx, ch)
	f.fs.ToWMStart(ctx, f.root.Path(), ptr, ospath.NewAllMatcher())

	state := <-ch

	if state == nil || len(state.FsToWmMirrors) != 1 {
		t.Errorf("No FsToWmMirrors found: %v", state)
	}
}

func TestSnapshotDir(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	f.writeFileWM("foo.txt", "bar")
	f.writeFileWM("baz.txt", "quux")
	base := f.writeFileWM("foo.txt", "bat")

	checkout, err := f.fs.Checkout(f.ctx, base, "")
	if err != nil {
		t.Fatal(err)
	}

	id, err := f.fs.SnapshotDir(f.ctx, checkout.Path, ospath.NewAllMatcher(), data.UserTestID, data.RecipeWTagOptimal, fs.Hint{Base: base})
	if err != nil {
		t.Fatal(err)
	}

	if id != base {
		t.Fatalf("didn't use base: %v; expected %v", id, base)
	}

	defer os.RemoveAll(checkout.Path)

}

func TestMirrorErrorHandler(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	f.writeFileFS("foo.txt", "1")

	var mirrorErr error
	mu := sync.Mutex{}
	f.fs.SetMirrorErrorHandler(func(e error) {
		mu.Lock()
		mirrorErr = e
		mu.Unlock()
	})

	getMirrorErr := func() error {
		mu.Lock()
		defer mu.Unlock()
		return mirrorErr
	}

	err = f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, ospath.NewAllMatcher())
	if err != nil {
		t.Fatal(err)
	} else if getMirrorErr() != nil {
		t.Fatal(getMirrorErr())
	}

	checkout, ok := f.fs.fs2wm.active[ptr]
	if !ok {
		t.Fatal("Checkout not created properly")
	}

	fakeErr := fmt.Errorf("fake error")
	checkout.w.Events <- watch.WatchErrEvent{Err: fakeErr}
	start := time.Now()
	for time.Now().Sub(start) < time.Second {
		if getMirrorErr() != nil {
			break
		}
	}

	if mirrorErr != fakeErr {
		t.Errorf("Expected fake error. Actual: %v", mirrorErr)
	}

	_, err = f.fs.ToWMFSync(f.ctx, ptr)
	if err == nil || err != fakeErr {
		t.Errorf("Expected error. Actual: %v", err)
	}
}

func TestMirrorErrorsOnTooManyFiles(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	ptr := data.MustParsePointerID("ptr")
	var mirrorErr error
	mu := sync.Mutex{}
	f.fs.SetMirrorErrorHandler(func(e error) {
		mu.Lock()
		mirrorErr = e
		mu.Unlock()
	})

	getMirrorErr := func() error {
		mu.Lock()
		defer mu.Unlock()
		return mirrorErr
	}

	f.fs.ToWMStart(f.ctx, f.root.Path(), ptr, ospath.NewAllMatcher())
	if mirrorErr != nil {
		t.Fatal(mirrorErr)
	}

	f.writeFileFS("foo.txt", "1")
	f.fs.ToWMFSync(f.ctx, ptr)
	f.writeFileFS("foo.txt", "2")
	f.fs.ToWMFSync(f.ctx, ptr)
	f.writeFileFS("foo.txt", "3")
	f.fs.ToWMFSync(f.ctx, ptr)
	f.writeFileFS("foo.txt", "4")
	f.fs.ToWMFSync(f.ctx, ptr)
	f.writeFileFS("foo.txt", "5")
	f.fs.ToWMFSync(f.ctx, ptr)
	f.writeFileFS("foo.txt", "6")
	f.fs.ToWMFSync(f.ctx, ptr)

	start := time.Now()
	for time.Now().Sub(start) < time.Second {
		if getMirrorErr() != nil {
			break
		}
	}

	if mirrorErr == nil {
		t.Errorf("Expected too many file writes error. Actual: %v", mirrorErr)
	}
}

type fixture struct {
	ctx    context.Context
	cancel func()
	db     dbint.DB2
	err    error
	lastID data.SnapshotID
	t      *testing.T
	fs     *LocalFSBridge
	root   *temp.TempDir
}

func setup(t *testing.T) (*fixture, error) {
	watch.SetLimitChecksEnabled(false)

	var tmp *temp.TempDir
	var err error
	if runtime.GOOS == "darwin" {
		tmp, err = temp.NewDirAtSlashTmp(t.Name())
	} else {
		tmp, err = temp.NewDir(t.Name())
	}
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := storages.NewTestMemoryRecipeStore()
	ptrs := storages.NewMemoryPointers()

	db := db2.NewDB2(s, ptrs)
	opt := db2.NewOptimizer(db, s)

	fs := NewLocalFSBridge(ctx, db, opt, tmp)
	fs.SetMirrorErrorHandler(func(err error) {
		t.Fatalf("checkin failed: %v", err)
	})

	return &fixture{ctx: ctx, cancel: cancel, db: db, fs: fs, t: t, root: tmp}, nil
}

func (f *fixture) tearDown() {
	f.fs.Shutdown(f.ctx)
	f.root.TearDown()
	f.cancel()
	watch.SetLimitChecksEnabled(true)
}

func (f *fixture) writeLinearOp(op data.Op) data.SnapshotID {
	if f.err != nil {
		return data.SnapshotID{}
	}

	inputs := []data.SnapshotID{}
	if !f.lastID.Nil() {
		inputs = []data.SnapshotID{f.lastID}
	}

	f.lastID, _, f.err = f.db.Create(f.ctx, data.Recipe{Op: op, Inputs: inputs}, data.UserTestID, data.RecipeWTagEdit)
	if f.err != nil {
		f.t.Fatal(f.err)
	}
	return f.lastID
}

func (f *fixture) writeFileWM(filename string, text string) data.SnapshotID {
	op := &data.WriteFileOp{
		Path: filename,
		Data: data.NewBytes([]byte(text)),
	}
	return f.writeLinearOp(op)
}

func (f *fixture) writeSymlinkWM(oldName, newName string) data.SnapshotID {
	op := &data.WriteFileOp{
		Path: newName,
		Data: data.BytesFromString(oldName),
		Type: data.FileSymlink,
	}
	return f.writeLinearOp(op)
}

func (f *fixture) writeFileFS(filename string, text string) {
	f.err = ioutil.WriteFile(path.Join(f.root.Path(), filename), []byte(text), os.FileMode(0755))
	if f.err != nil {
		f.t.Fatal(f.err)
	}
}

// Compare the snapshot in the DB by doing a checkout, then comparing dir structures.
func (f *fixture) expectCheckout(ex CheckoutExpectation) error {
	checkout, err := f.fs.Checkout(f.ctx, ex.id, "")
	if err != nil {
		return err
	}

	defer os.RemoveAll(checkout.Path)
	err = f.expectFiles(checkout.Path, ex.files)
	if err != nil {
		return fmt.Errorf("snapshot %s: %v", ex.id, err)
	}
	return nil
}

// Compare dir structures.
func (f *fixture) expectFiles(checkout string, exFiles files) error {
	unseen := make(map[string]bool)
	for k, _ := range exFiles {
		unseen[k] = true
	}

	fn := func(p string, info os.FileInfo, err error) error {
		rel, err := filepath.Rel(checkout, p)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		expectedData, ok := exFiles[rel]
		if !ok {
			return fmt.Errorf("checkout %v contains unexpected file %v", checkout, rel)
		}

		actualBytes, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}

		actualData := string(actualBytes)
		if actualData != expectedData {
			return fmt.Errorf("checkout %v contains bad data for file %v:\n"+
				"actual: %q\n"+
				"expected: %q\n", checkout, rel, actualData, expectedData)
		}

		delete(unseen, rel)

		return nil
	}

	if err := filepath.Walk(checkout, fn); err != nil {
		return err
	}

	if len(unseen) != 0 {
		return fmt.Errorf("checkout %v should contain files %v", checkout, unseen)
	}

	return nil
}

func (f *fixture) expectSnapshotFiles(id data.SnapshotID, exFiles files) error {
	unseen := make(map[string]bool)
	for k, _ := range exFiles {
		unseen[k] = true
	}

	snapshot, err := f.db.Snapshot(f.ctx, id)
	if err != nil {
		return err
	}

	pathSet := snapshot.PathSet()
	for path := range pathSet {
		delete(unseen, path)

		exFile, ok := exFiles[path]
		if !ok {
			return fmt.Errorf("Did not expect file in snapshot: %s", path)
		}

		actual, err := f.db.SnapshotFile(f.ctx, id, path)
		if err != nil {
			return err
		}

		if actual.Contents.String() != exFile {
			return fmt.Errorf("Unexpected file data for path %q:\nexpected: %s\nactual: %s\n",
				path, exFile, actual.Contents.String())
		}
	}

	if len(unseen) != 0 {
		return fmt.Errorf("Expected files in snapshot: %v", unseen)
	}

	return nil
}
