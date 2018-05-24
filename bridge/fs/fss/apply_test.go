// Tests to ensure that applytoFS correctly interprets Ops emitted by
// WorkspaceWatcher
package fss

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/proto"
	"github.com/windmilleng/mish/data/db/storage/storages"
	"github.com/windmilleng/mish/os/temp"
	"github.com/windmilleng/mish/os/watch"
)

const PERM = os.FileMode(0600)
const DIR_PERM = os.FileMode(0700)

func TestWriteFile(t *testing.T) {
	f := newMirrorFixture(t)
	defer f.tearDown()

	ioutil.WriteFile(f.SrcPath("test.tmp"), []byte("hello"), PERM)

	f.mirror()
	f.assertDestFileContents("test.tmp", "hello")
}

func TestWriteRelSymlink(t *testing.T) {
	f := newMirrorFixture(t)
	defer f.tearDown()

	ioutil.WriteFile(f.SrcPath("a.txt"), []byte("a"), PERM)
	err := os.Symlink("a.txt", f.SrcPath("b.txt"))
	if err != nil {
		t.Fatal(err)
	}

	f.mirror()
	f.assertDestLink("a.txt", "b.txt")
}

func TestWriteAbsSymlink(t *testing.T) {
	f := newMirrorFixture(t)
	defer f.tearDown()

	ioutil.WriteFile(f.SrcPath("a.txt"), []byte("a"), PERM)
	err := os.Symlink(f.SrcPath("a.txt"), f.SrcPath("b.txt"))
	if err != nil {
		t.Fatal(err)
	}

	f.mirror()
	f.assertDestLink("a.txt", "b.txt")
}

func TestWriteMalformedSymlink(t *testing.T) {
	f := newMirrorFixture(t)
	defer f.tearDown()

	err := applyToFS(&data.WriteFileOp{
		Path: "a.txt",
		Data: data.BytesFromString("../password.txt"),
		Type: data.FileSymlink,
	}, f.DestDir())
	if err == nil || !strings.Contains(err.Error(), "symlinks can't point outside of root directory") {
		t.Errorf("Expected error. Actual: %v", err)
	}
}

func TestWriteMalformedAbsSymlink(t *testing.T) {
	f := newMirrorFixture(t)
	defer f.tearDown()

	err := applyToFS(&data.WriteFileOp{
		Path: "a.txt",
		Data: data.BytesFromString("/var/password.txt"),
		Type: data.FileSymlink,
	}, f.DestDir())
	if err == nil || !strings.Contains(err.Error(), "cannot handle absolute symlink") {
		t.Errorf("Expected error. Actual: %v", err)
	}
}

func TestWriteInnerDirFile(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Test times out on macOS. Deadlock somewhere in fsnotify.")
	}

	f := newMirrorFixture(t)
	defer f.tearDown()

	os.MkdirAll(f.SrcPath("a", "b"), DIR_PERM)
	ioutil.WriteFile(f.SrcPath("a", "b", "test.tmp"), []byte("hello"), PERM)

	f.mirror()
	f.assertDestFileContents(filepath.Join("a", "b", "test.tmp"), "hello")
}

func TestWriteAndRemoveInnerDirFile(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Test times out on macOS. Deadlock somewhere in fsnotify.")
	}

	f := newMirrorFixture(t)
	defer f.tearDown()

	os.MkdirAll(f.SrcPath("a", "b"), DIR_PERM)
	ioutil.WriteFile(f.SrcPath("a", "b", "test.tmp"), []byte("hello"), PERM)

	f.mirror()
	f.assertDestFileContents(filepath.Join("a", "b", "test.tmp"), "hello")

	os.RemoveAll(f.SrcPath("a"))
	f.mirror()

	f.assertDestFileNotExists(filepath.Join("a", "b", "test.tmp"))
	f.assertDestFileNotExists(filepath.Join("a", "b"))
	f.assertDestFileNotExists(filepath.Join("a"))
}

func TestWriteChmodFile(t *testing.T) {
	f := newMirrorFixture(t)
	defer f.tearDown()

	srcPath := f.SrcPath("test.tmp")
	ioutil.WriteFile(srcPath, []byte("hello"), os.FileMode(0600))
	f.mirror()
	f.assertDestFileExecutable("test.tmp", false)

	os.Chmod(srcPath, os.FileMode(0700))
	f.mirror()
	f.assertDestFileExecutable("test.tmp", true)
}

func TestAppendToFile(t *testing.T) {
	f := newMirrorFixture(t)
	defer f.tearDown()

	ioutil.WriteFile(f.SrcPath("test.tmp"), []byte("hello"), PERM)

	f.mirror()
	f.assertDestFileContents("test.tmp", "hello")

	ioutil.WriteFile(f.SrcPath("test.tmp"), []byte("hello world"), PERM)

	f.mirror()
	f.assertDestFileContents("test.tmp", "hello world")
}

func TestInsertMidFile(t *testing.T) {
	f := newMirrorFixture(t)
	defer f.tearDown()

	ioutil.WriteFile(f.SrcPath("test.tmp"), []byte("hello world"), PERM)

	f.mirror()
	f.assertDestFileContents("test.tmp", "hello world")

	ioutil.WriteFile(f.SrcPath("test.tmp"), []byte("hello, world"), PERM)

	f.mirror()
	f.assertDestFileContents("test.tmp", "hello, world")
}

func TestDeleteFromEndFile(t *testing.T) {
	f := newMirrorFixture(t)
	defer f.tearDown()

	ioutil.WriteFile(f.SrcPath("test.tmp"), []byte("hello world"), PERM)

	f.mirror()
	f.assertDestFileContents("test.tmp", "hello world")

	ioutil.WriteFile(f.SrcPath("test.tmp"), []byte("hello"), PERM)

	f.mirror()
	f.assertDestFileContents("test.tmp", "hello")
}

func TestDeleteFromMiddleFile(t *testing.T) {
	f := newMirrorFixture(t)
	defer f.tearDown()

	ioutil.WriteFile(f.SrcPath("test.tmp"), []byte("hello world"), PERM)

	f.mirror()
	f.assertDestFileContents("test.tmp", "hello world")

	ioutil.WriteFile(f.SrcPath("test.tmp"), []byte("hell world"), PERM)

	f.mirror()
	f.assertDestFileContents("test.tmp", "hell world")
}

// A fixture that mirrors changes from one directory
// to another by calling mirror()
type mirrorFixture struct {
	t       *testing.T
	srcDir  *temp.TempDir
	destDir *temp.TempDir
	ww      *watch.WorkspaceWatcher
}

func (f *mirrorFixture) SrcDir() string {
	return f.srcDir.Path()
}

func (f *mirrorFixture) SrcPath(parts ...string) string {
	return filepath.Join(append([]string{f.SrcDir()}, parts...)...)
}

func (f *mirrorFixture) DestDir() string {
	return f.destDir.Path()
}

func (f *mirrorFixture) DestPath(parts ...string) string {
	return filepath.Join(append([]string{f.DestDir()}, parts...)...)
}

func (f *mirrorFixture) assertDestFileContents(relPath, contents string) {
	actual, err := ioutil.ReadFile(f.DestPath(relPath))
	if err != nil {
		f.t.Fatal(err)
	}

	if contents != string(actual) {
		f.t.Errorf("Expected file contents '%s', got '%s'", contents, string(actual))
	}
}

func (f *mirrorFixture) assertDestLink(oldName, newName string) {
	actual, err := os.Readlink(f.DestPath(newName))
	if err != nil {
		f.t.Fatal(err)
	}

	if oldName != actual {
		f.t.Errorf("Expected file link '%s', got '%s'", oldName, string(actual))
	}
}

func (f *mirrorFixture) assertDestFileNotExists(relPath string) {
	_, err := os.Stat(f.DestPath(relPath))
	if err == nil {
		f.t.Errorf("Expected file not to exist: %s", relPath)
	}
}

func (f *mirrorFixture) assertDestFileExecutable(relPath string, expected bool) {
	info, err := os.Stat(f.DestPath(relPath))
	if err != nil {
		f.t.Fatalf("File does not exist %s: %v", relPath, err)
	}

	executable := data.IsExecutableMode(info.Mode())
	if executable != expected {
		if expected {
			f.t.Errorf("Expected executable file at %s", relPath)
		} else {
			f.t.Errorf("Expected non-executable file at %s", relPath)
		}
	}
}

func (f *mirrorFixture) mirror() {
	syncDoneCh := make(chan struct{})
	opsDoneCh := make(chan struct{})

	go func() {
		defer func() {
			close(opsDoneCh)
		}()
		for {
			select {
			case event := <-f.ww.Events:
				switch event := event.(type) {
				case watch.WatchOpsEvent:
					for _, op := range event.Ops {
						op = f.assertOpRoundTrip(op)
						err := applyToFS(op, f.DestDir())
						if err != nil {
							f.t.Fatalf("Error processing op %v: %v", op, err)
						}
					}
				case watch.WatchErrEvent:
					f.t.Fatal(event.Err)
				case watch.WatchSyncEvent:
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

// Try to convert op to a protobuf and back, failing the test if there are any
// errors. If any fields are missing in the conversion, we'll catch that when we
// verify the applyToFS() results.
func (f *mirrorFixture) assertOpRoundTrip(op data.Op) data.Op {
	recipe := data.Recipe{Op: op}
	recipeProto, err := proto.RecipeD2P(recipe)
	if err != nil {
		f.t.Fatalf("Could not convert recipe to protobuf: %v", recipe)
	}

	result, err := proto.RecipeP2D(recipeProto)
	if err != nil {
		f.t.Fatalf("Could not convert protobuf to recipe: %v", recipeProto)
	}
	return result.Op
}

func (f *mirrorFixture) tearDown() {
	f.srcDir.TearDown()
	f.destDir.TearDown()
	// TODO(dmiller) this deadlocks on macOS. There are several PRs open on fsnotify that claim to resolve this
	// but none of them do
	err := f.ww.Close()
	if err != nil {
		f.t.Fatal(err)
	}
	watch.SetLimitChecksEnabled(true)
}

func newMirrorFixture(t *testing.T) *mirrorFixture {
	watch.SetLimitChecksEnabled(false)

	tmp, _ := temp.NewDir(t.Name())
	srcDir, _ := tmp.NewDir("src")
	destDir, _ := tmp.NewDir("dest")

	recipes := storages.NewTestMemoryRecipeStore()
	ptrs := storages.NewMemoryPointers()
	db := db2.NewDB2(recipes, ptrs)

	ww, err := watch.NewWorkspaceWatcher(srcDir.Path(), db, watch.DefaultMatcher(), data.UserTestID, data.RecipeWTagEdit, tmp)
	if err != nil {
		t.Fatal(err)
	}
	return &mirrorFixture{
		t:       t,
		srcDir:  srcDir,
		destDir: destDir,
		ww:      ww,
	}
}
