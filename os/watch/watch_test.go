package watch

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/windmilleng/fsnotify"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/storage/storages"
	"github.com/windmilleng/mish/os/ospath"
	"github.com/windmilleng/mish/os/temp"
)

const PERM = os.FileMode(0600)

const DIR_PERM = os.FileMode(0700)

func TestCreateFile(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.startWatch()
	f.write("test.tmp", "hello")

	f.fsync()
	f.assertWriteOp("test.tmp", "hello", false)
}

func TestCreateSymlink(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.startWatch()

	f.write("a.txt", "a")
	f.fsync()
	f.assertWriteOp("a.txt", "a", false)

	f.writeSymlink("a.txt", "b.txt")
	f.fsync()
	f.assertWriteSymlinkOp("b.txt", "a.txt")
}

// NOTE(dmiller): this fails with fsevents
func TestCreateBrokenSymlink(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.startWatch()

	f.writeSymlink("a.txt", "b.txt")
	f.fsync()
	f.assertNoOp()
}

func TestCreateFileInnerDir(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.startWatch()
	err := os.MkdirAll(filepath.Join(f.WatchDir(), "a", "b"), DIR_PERM)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(
		filepath.Join(f.WatchDir(), "a", "b", "test.tmp"),
		[]byte("hello"),
		PERM)
	if err != nil {
		t.Fatal(err)
	}

	f.fsync()
	f.assertWriteOp(filepath.Join("a", "b", "test.tmp"), "hello", false)
}

func TestAppendFile(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.write("test.tmp", "hello")
	f.startWatch()

	f.append("test.tmp", " world")

	f.fsync()
	f.assertEditOp("test.tmp", ins(5, " world"))
}

func TestCreateThenAppendFile(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.write("test.tmp", "hello")
	f.startWatch()

	f.append("test.tmp", " world")

	f.fsync()
	f.assertEditOp("test.tmp", ins(5, " world"))
	f.assertNoOp()
}

func TestDeleteBytes(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.startWatch()
	f.write("test.tmp", "hello")

	f.fsync()
	f.assertWriteOp("test.tmp", "hello", false)

	f.write("test.tmp", "hell")

	f.fsync()
	f.assertEditOp("test.tmp", del(4, 1))
	f.assertNoOp()
}

func TestInsertAfterDeleteBytes(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.startWatch()

	oldContents := "blah blah blah blah blah blah blah blah blah blah"
	newContents := "blah blah bla blah blah blah blaho blah blah blah"
	f.write("test.tmp", oldContents)

	f.fsync()
	f.assertWriteOp("test.tmp", oldContents, false)

	f.write("test.tmp", newContents)

	f.fsync()
	f.assertEditOp("test.tmp", del(13, 1), ins(33, "o"))
	f.assertNoOp()
}

func TestDeleteAfterInsertBytes(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.startWatch()

	oldContents := "blah blah blah blah blah blah blah blah blah blah"
	newContents := "blah blah blaho blah blah blah bla blah blah blah"
	f.write("test.tmp", oldContents)

	f.fsync()
	f.assertWriteOp("test.tmp", oldContents, false)

	f.write("test.tmp", newContents)

	f.fsync()
	f.assertEditOp("test.tmp", ins(14, "o"), del(34, 1))
	f.assertNoOp()
}

func TestRemoveFile(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	path := filepath.Join(f.WatchDir(), "test.tmp")
	f.write("test.tmp", "hello")
	f.startWatch()

	err := os.Remove(path)
	if err != nil {
		t.Fatal(err)
	}

	f.fsync()
	f.assertRemoveOp("test.tmp")
	f.assertNoOp()
}

// When git modifies a file on Ubuntu, fsnotify emits two REMOVE events.
// We haven't investigated what file system operations cause this,
// or whether this is an fsnotify or inotify bug. But we should be defensive
// against when this happens.
func TestRemoveFileTwice(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	path := filepath.Join(f.WatchDir(), "test.tmp")
	f.write("test.tmp", "hello")
	f.startWatch()

	err := os.Remove(path)
	if err != nil {
		t.Fatal(err)
	}
	f.emitEvent(fsnotify.Event{
		Op:   fsnotify.Remove,
		Name: path,
	})
	f.emitEvent(fsnotify.Event{
		Op:   fsnotify.Remove,
		Name: path,
	})

	f.fsync()
	f.assertRemoveOp("test.tmp")
	f.assertNoOp()
}

func TestMoveFileSameDirectory(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	path := filepath.Join(f.WatchDir(), "test.tmp")
	path2 := filepath.Join(f.WatchDir(), "test2.tmp")
	f.write("test.tmp", "hello")
	f.startWatch()

	err := os.Rename(path, path2)
	if err != nil {
		t.Fatal(err)
	}

	f.fsync()
	f.assertRemoveOpInAnyOrder("test.tmp")
	f.assertWriteOpInAnyOrder("test2.tmp", "hello", false)
}

func TestMoveFileOutOfWorkspace(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	path := filepath.Join(f.WatchDir(), "test.tmp")
	path2 := filepath.Join(f.OtherDir(), "test2.tmp")
	f.write("test.tmp", "hello")

	f.startWatch()

	err := os.Rename(path, path2)
	if err != nil {
		t.Fatal(err)
	}

	f.fsync()
	f.assertRemoveOp("test.tmp")
	f.assertNoOp()
}

func TestMoveFileIntoWorkspace(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	path := filepath.Join(f.WatchDir(), "test.tmp")
	path2 := filepath.Join(f.OtherDir(), "test2.tmp")
	ioutil.WriteFile(path2, []byte("hello"), PERM)

	f.startWatch()

	err := os.Rename(path2, path)
	if err != nil {
		t.Fatal(err)
	}

	f.fsync()
	f.assertWriteOp("test.tmp", "hello", false)
	f.assertNoOp()
}

func TestChmod(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	path := filepath.Join(f.WatchDir(), "test.tmp")
	ioutil.WriteFile(path, []byte("hello"), PERM)

	f.startWatch()

	newMode := os.FileMode(0700)
	err := os.Chmod(path, newMode)
	if err != nil {
		t.Fatal(err)
	}

	f.fsync()
	f.assertChmodOp("test.tmp", true)
	f.assertNoOp()
}

func TestCreateAbsoluteSymlink(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	path := filepath.Join(f.WatchDir(), "test.tmp")
	ioutil.WriteFile(path, []byte("hello"), PERM)
	os.Symlink(path, filepath.Join(f.WatchDir(), "link"))

	dirPath := filepath.Join(f.WatchDir(), "testdir")
	os.MkdirAll(dirPath, 0700)
	path = filepath.Join(dirPath, "test2.tmp")
	ioutil.WriteFile(path, []byte("hello2"), PERM)
	os.Symlink(path, filepath.Join(f.WatchDir(), "link2"))

	f.snapshot()
	f.assertWriteSymlinkOp("link", "test.tmp")
	f.assertWriteSymlinkOp("link2", "testdir/test2.tmp")
	f.assertWriteOp("test.tmp", "hello", false)
	f.assertWriteOp("testdir/test2.tmp", "hello2", false)
	f.assertNoOp()
}

func TestCreateSymlinkOutsideWorkspace(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	path := filepath.Join(f.WatchDir(), "test.tmp")
	err := ioutil.WriteFile(path, []byte("hello"), PERM)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Symlink("../../password.txt", filepath.Join(f.WatchDir(), "link"))
	if err != nil {
		t.Fatal(err)
	}

	f.snapshot()
	f.assertWriteOp("test.tmp", "hello", false)
	f.assertNoOp()
}

func TestDirNotTracked(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	subdir := filepath.Join(f.WatchDir(), "subdir")
	os.MkdirAll(subdir, DIR_PERM)

	f.startWatch()
	f.fsync()
	f.assertNoOp()
}

func TestOnlyMatchedDirTracked(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	matcher, _ := ospath.NewMatcherFromPattern("a/b/**")
	f.matcher = matcher

	os.MkdirAll(filepath.Join(f.WatchDir(), "a/b"), DIR_PERM)
	os.MkdirAll(filepath.Join(f.WatchDir(), "a/c"), DIR_PERM)
	ioutil.WriteFile(filepath.Join(f.WatchDir(), "a/c/d.txt"), []byte("hello"), PERM)
	ioutil.WriteFile(filepath.Join(f.WatchDir(), "a/b/c.txt"), []byte("hello"), PERM)

	f.snapshot()
	f.assertWriteOp("a/b/c.txt", "hello", false)
	f.assertNoOp()
}

func TestSocketNotTracked(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	socketPath := filepath.Join(f.WatchDir(), "s")
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}

	f.startWatch()
	f.fsync()
	f.assertNoOp()

	l.Close()
}

func TestCreateFileAndSync(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.write("test.tmp", "hello")
	f.startWatchAndSync(data.EmptySnapshotID)

	f.fsync()
	f.assertWriteOp("test.tmp", "hello", false)
}

func TestSyncRemoveFileInSnapshot(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	previous := f.create(wf("test.tmp", "hello"))

	f.startWatchAndSync(previous)

	f.fsync()
	f.assertRemoveOp("test.tmp")
}

func TestSyncInsertBytesInSnapshot(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	previous := f.create(wf("test.tmp", "hell"))

	f.write("test.tmp", "hello")

	f.startWatchAndSync(previous)

	f.fsync()
	f.assertEditOp("test.tmp", ins(4, "o"))
}

func TestSyncDeleteBytesInSnapshot(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	previous := f.create(wf("test.tmp", "hello"))

	f.write("test.tmp", "hell")

	f.startWatchAndSync(previous)
	f.fsync()
	f.assertEditOp("test.tmp", del(4, 1))
}

func TestSyncChmodInSnapshot(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	previous := f.create(wf("test.tmp", "hello"))

	path := filepath.Join(f.WatchDir(), "test.tmp")
	ioutil.WriteFile(path, []byte("hello"), os.FileMode(0700))

	f.startWatchAndSync(previous)
	f.fsync()
	f.assertChmodOp("test.tmp", true)
}

func TestSyncDeleteBytesAndChmodInSnapshot(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	previous := f.create(wf("test.tmp", "hello hello hello hello"))

	path := filepath.Join(f.WatchDir(), "test.tmp")
	ioutil.WriteFile(path, []byte("hello hello hello hell"), os.FileMode(0700))

	f.startWatchAndSync(previous)
	f.fsync()
	f.assertEditOp("test.tmp", del(22, 1))
	f.assertChmodOp("test.tmp", true)
}

func TestSymlinkAtRoot(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	os.RemoveAll(f.WatchDir())
	os.Symlink(f.OtherDir(), f.WatchDir())

	f.startWatch()
	f.write("test.tmp", "hello")

	f.fsync()
	f.assertWriteOp("test.tmp", "hello", false)
}

func TestChangesSinceModtime(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("skipping test that depends on mtime on MacOS since some Macs use HFS+ which only supports second level precision")
	}

	f := newFixture(t)
	defer f.tearDown()

	f.write("a.txt", "a")
	f.write("b.txt", "b")
	f.write("c.txt", "c")
	f.write("d.txt", "d")

	opsEvent, err := DirectoryToOpsEvent(f.WatchDir(), f.db, ospath.NewAllMatcher(), data.EmptySnapshotID, data.UserTestID, data.RecipeWTagTemp)
	if err != nil {
		t.Fatal(err)
	}
	checkpointSnapID := opsEvent.SnapID

	time.Sleep(5 * time.Millisecond)
	f.append("d.txt", "x")

	// NOTE(nick): The filesystem clock can be skewed from golang's clock.
	// e.g., if you call time.Now(), then modify a file, then Stat it, the Stat will return
	// a modtime BEFORE the first time.Now(). So we depend entirely on Stat for times.
	stat, _ := os.Stat(filepath.Join(f.WatchDir(), "d.txt"))
	checkpointTime := stat.ModTime().Add(time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	f.append("b.txt", "x")
	f.rm("c.txt")

	opsEvent, err = ChangesSinceModTimeToOpsEvent(f.WatchDir(), f.db, checkpointSnapID, data.UserTestID, checkpointTime)
	if err != nil {
		t.Fatal(err)
	}

	pathsChanged := []string{}
	for _, op := range opsEvent.Ops {
		pathsChanged = append(pathsChanged, op.(data.FileOp).FilePath())
	}
	sort.Strings(pathsChanged)

	// We do not detect the changes to d, because it's before the checkpoint time.
	if len(pathsChanged) != 2 || pathsChanged[0] != "b.txt" || pathsChanged[1] != "c.txt" {
		t.Errorf("Unexpected paths changed: %v", pathsChanged)
	}
}

// NOTE(dmiller): this is flaky with fsevents
func TestOverwriteFileWithDir(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.startWatch()
	f.write("test", "hello1")
	f.fsync()
	f.assertWriteOp("test", "hello1", false)
	f.assertNoOp()

	f.rm("test")
	f.mkdirAll("test")
	f.write("test/text.txt", "hello2")

	f.fsync()
	f.assertRemoveOp("test")
	f.assertWriteOp("test/text.txt", "hello2", false)
	f.assertNoOp()
}

func TestCanBeNotifiedThatAFileHasChanged(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.write("test.tmp", "hello")
	f.startWatch()

	c := make(chan struct{})

	f.ww.NotifyOnChange("test.tmp", c)

	f.append("test.tmp", " world")

	x := <-c

	if x != struct{}{} {
		t.Error("Expected to be notified that a file changed, but we weren't")
	}
}

func TestFSyncAfterError(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.startWatch()
	fakeErr := fmt.Errorf("fake error")
	f.ww.watcher.Errors() <- fakeErr

	start := time.Now()
	var err error
	for time.Since(start) < time.Second {
		token := fmt.Sprintf("%s", time.Now())
		err = f.ww.FSync(f.ctx, token)
		if err != nil {
			break
		}
	}

	if err != fakeErr {
		t.Errorf("Expected propagated error. Actual: %v", err)
	}

	// drain
	for _ = range f.ww.Events {
	}
}

func TestWorkspaceWithLimit(t *testing.T) {
	f := newFixture(t)
	defer f.tearDown()

	f.write("test.tmp", "hello")
	f.write("test2.tmp", "world")

	err := f.startWatchWithLimit(1)
	if err == nil {
		t.Error("Expected error, got nil")
	} else if !strings.Contains(err.Error(), "too many files") {
		t.Errorf("Expected error to contain the string 'too many files', but for %s", err.Error())
	}
}

// A fixture that watches a directory.
// Starts watching when startWatch() is called.
type fixture struct {
	t        *testing.T
	watchDir *temp.TempDir
	otherDir *temp.TempDir
	tmp      *temp.TempDir
	ww       *WorkspaceWatcher
	ops      []data.Op
	matcher  *ospath.Matcher

	db  dbint.DB2
	ctx context.Context
}

func (f *fixture) emitEvent(e fsnotify.Event) {
	f.ww.watcher.Events() <- e
}

func (f *fixture) WatchDir() string {
	return f.watchDir.Path()
}

func (f *fixture) OtherDir() string {
	return f.otherDir.Path()
}

func (f *fixture) assertRemoveOp(path string) {
	op := f.assertOp()
	removeOp, ok := op.(*data.RemoveFileOp)
	if !ok || removeOp.Path != path {
		f.t.Errorf("Expected remove op, got %v", op)
	}
}

func (f *fixture) assertRemoveOpInAnyOrder(path string) {
	for _, op := range f.ops {
		removeOp, ok := op.(*data.RemoveFileOp)
		if ok && removeOp.Path == path {
			return
		}
	}

	f.t.Errorf("Expected remove op with path %s in %v", path, f.ops)
}

func (f *fixture) assertEditOp(path string, splices ...data.EditFileSplice) {
	op := f.assertOp()
	editOp, ok := op.(*data.EditFileOp)
	if !ok || editOp.Path != path {
		f.t.Errorf("Expected edit op, got %v", op)
		return
	}

	if len(editOp.Splices) != len(splices) {
		f.t.Errorf("Expected %d splices. Actual: %d", len(splices), len(editOp.Splices))
	}

	for i, s := range editOp.Splices {
		if !reflect.DeepEqual(s, splices[i]) {
			f.t.Errorf("Splice mismatch at %d. Expected %v. Actual %v", i, splices[i], s)
		}
	}
}

func ins(index int, d string) data.EditFileSplice {
	return &data.InsertBytesSplice{Index: int64(index), Data: data.BytesFromString(d)}
}

func del(index, count int) data.EditFileSplice {
	return &data.DeleteBytesSplice{Index: int64(index), DeleteCount: int64(count)}
}

func (f *fixture) assertWriteOp(path string, contents string, executable bool) {
	op := f.assertOp()
	writeOp, ok := op.(*data.WriteFileOp)

	if !ok || writeOp.Path != path ||
		writeOp.Data.String() != contents ||
		writeOp.Executable != executable ||
		writeOp.Type != data.FileRegular {
		f.t.Errorf("Expected write op (%s, %s, %v), got %+v", path, contents, executable, op)
	}
}

func (f *fixture) assertWriteSymlinkOp(path string, contents string) {
	op := f.assertOp()
	writeOp, ok := op.(*data.WriteFileOp)

	if !ok || writeOp.Path != path ||
		writeOp.Data.String() != contents ||
		writeOp.Type != data.FileSymlink {
		f.t.Errorf("Expected write symlink op (%s, %s), got %v", path, contents, op)
	}
}

func (f *fixture) assertWriteOpInAnyOrder(path string, contents string, executable bool) {
	for _, op := range f.ops {
		writeOp, ok := op.(*data.WriteFileOp)

		if !ok {
			continue
		}

		if writeOp.Path == path &&
			writeOp.Data.String() == contents &&
			writeOp.Executable == executable &&
			writeOp.Type == data.FileRegular {
			return
		}
	}

	f.t.Errorf("Expected write op (%s, %s, %v) in %+v", path, contents, executable, f.ops)
}

func (f *fixture) assertChmodOp(path string, executable bool) {
	op := f.assertOp()
	chmodOp, ok := op.(*data.ChmodFileOp)
	if !ok || chmodOp.Path != path || chmodOp.Executable != executable {
		f.t.Errorf("Expected chmod op, got %v", op)
	}
}

func (f *fixture) assertOp() data.Op {
	if len(f.ops) == 0 {
		f.t.Errorf("Expected op; got none")
		return nil
	}

	op := f.ops[0]
	f.ops = f.ops[1:]

	return op
}

func (f *fixture) assertNoOp() {
	if len(f.ops) > 0 {
		f.t.Errorf("Expected no more ops, got %v", f.ops)
	}
}

func (f *fixture) startWatch() {
	ww, err := NewWorkspaceWatcher(f.WatchDir(), f.db, f.matcher, data.UserTestID, data.RecipeWTagEdit, f.tmp)
	if err != nil {
		f.t.Fatal(err)
	}
	f.ww = ww
}

func (f *fixture) startWatchWithLimit(limit int) error {
	ww, err := NewWorkspaceWatcherWithLimit(f.WatchDir(), f.db, f.matcher, data.UserTestID, data.RecipeWTagEdit, f.tmp, 1)
	if err != nil {
		return err
	}
	f.ww = ww
	return nil
}

func (f *fixture) startWatchAndSync(previous data.SnapshotID) {
	ww, err := NewWorkspaceWatcherAndSync(f.WatchDir(), f.db, previous, f.matcher, data.UserTestID, NewRecipeTagProvider(data.RecipeWTagEdit), f.tmp)
	if err != nil {
		f.t.Fatal(err)
	}
	f.ww = ww
}

func (f *fixture) fsync() {
	f.ops = nil
	syncDoneCh := make(chan struct{})
	opsDoneCh := make(chan struct{})

	go func() {
		defer func() {
			close(opsDoneCh)
		}()
		for {
			select {
			case event, ok := <-f.ww.Events:
				if !ok {
					return
				}

				switch event := event.(type) {
				case WatchOpsEvent:
					f.ops = append(f.ops, filterEmptyWriteOps(event.Ops)...)
				case WatchErrEvent:
					f.t.Fatal(event.Err)
				case WatchSyncEvent:
				default:
					f.t.Fatalf("Unknown event type: %T", event)
				}
			case <-syncDoneCh:
				return
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := f.ww.FSync(ctx, ""); err != nil {
		f.t.Fatal(err)
	}
	close(syncDoneCh)

	<-opsDoneCh
}

func (f *fixture) snapshot() {
	opsEvent, err := DirectoryToOpsEvent(f.watchDir.Path(), f.db, f.matcher, data.EmptySnapshotID, data.UserTestID, data.RecipeWTagTemp)
	if err != nil {
		f.t.Fatal(err)
	}

	f.ops = opsEvent.Ops
}

func (f *fixture) mkdirAll(path string) {
	absPath := filepath.Join(f.WatchDir(), path)
	err := os.MkdirAll(absPath, DIR_PERM)
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) write(path, contents string) {
	absPath := filepath.Join(f.WatchDir(), path)
	err := ioutil.WriteFile(absPath, []byte(contents), PERM)
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) writeSymlink(oldName, newName string) {
	absNewName := filepath.Join(f.WatchDir(), newName)
	err := os.Symlink(oldName, absNewName)
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) append(path, contents string) {
	absPath := filepath.Join(f.WatchDir(), path)
	file, err := os.OpenFile(absPath, os.O_APPEND|os.O_WRONLY, PERM)
	if err != nil {
		f.t.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString(contents)
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) rm(path string) {
	absPath := filepath.Join(f.WatchDir(), path)
	err := os.Remove(absPath)
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) tearDown() {
	if f.ww != nil {
		err := f.ww.Close()
		if err != nil {
			f.t.Fatal(err)
		}

		// drain
		for event := range f.ww.Events {
			errEvent, ok := event.(WatchErrEvent)
			if ok {
				f.t.Fatal(errEvent.Err)
			}
		}
	}
	f.watchDir.TearDown()
	f.otherDir.TearDown()
	SetLimitChecksEnabled(true)
}

func newFixture(t *testing.T) *fixture {
	SetLimitChecksEnabled(false)

	tmp, _ := temp.NewDir(t.Name())
	watchDir, _ := tmp.NewDir("watch")
	otherDir, _ := tmp.NewDir("other")

	db := db2.NewDB2(storages.NewTestMemoryRecipeStore(), storages.NewMemoryPointers())
	ctx := context.Background()
	return &fixture{
		t:        t,
		watchDir: watchDir,
		otherDir: otherDir,
		tmp:      tmp,
		matcher:  DefaultMatcher(),

		db:  db,
		ctx: ctx,
	}
}

func wf(path string, bytes string) *data.WriteFileOp {
	return &data.WriteFileOp{Path: path, Data: data.BytesFromString(bytes)}
}

func (f *fixture) create(op data.Op) data.SnapshotID {
	snap, _, err := f.db.Create(f.ctx, data.Recipe{Op: op}, data.UserTestID, data.RecipeWTagEdit)
	if err != nil {
		f.t.Fatal(err)
	}
	return snap
}

func isEmptyWriteOp(op data.Op) bool {
	wfOp, ok := op.(*data.WriteFileOp)
	if !ok {
		return false
	}
	return wfOp.Data.Len() == 0
}

// On MacOS, filesystem writes sometimes materialize as two events:
// one event that writes a 0-length file, and another event
// that writes the file contents. If this happens, quietly drop the first op.
// This doesn't really matter for testing purposes.
func filterEmptyWriteOps(ops []data.Op) []data.Op {
	result := make([]data.Op, 0, len(ops))
	for _, op := range ops {
		if isEmptyWriteOp(op) {
			log.Println("Filtering empty WriteFileOp:", op)
			continue
		}

		result = append(result, op)
	}
	return result
}
