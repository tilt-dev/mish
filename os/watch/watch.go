// Watches a directory recursively, translating file
// system events into our internal Op structure.
//
// File events are complicated! Here are some helpful links:
// https://fsnotify.org/
// http://man7.org/linux/man-pages/man7/inotify.7.html
package watch

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/windmilleng/fsnotify"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/errors"
	"github.com/windmilleng/mish/os/ospath"
	"github.com/windmilleng/mish/os/temp"
)

// The default matcher of a watcher. Just ignores .git directories.
func DefaultMatcher() *ospath.Matcher {
	m, err := ospath.NewMatcherFromPattern("!.git/**")
	if err != nil {
		panic(err)
	}
	return m
}

// Watches a directory. Does not generate write ops for existing contents.
func NewWorkspaceWatcher(path string, db dbint.DB2, matcher *ospath.Matcher, owner data.UserID, tag data.RecipeWTag, tmp *temp.TempDir) (*WorkspaceWatcher, error) {
	o := newOptimizer(db, data.EmptySnapshotID, owner, NewRecipeTagProvider(tag))
	return newWorkspaceWatcherHelperWithLimit(path, o, matcher, false, tmp, defaultMaxFilesWatched)
}

func NewWorkspaceWatcherWithLimit(path string, db dbint.DB2, matcher *ospath.Matcher, owner data.UserID, tag data.RecipeWTag, tmp *temp.TempDir, maxFiles int) (*WorkspaceWatcher, error) {
	o := newOptimizer(db, data.EmptySnapshotID, owner, NewRecipeTagProvider(tag))
	return newWorkspaceWatcherHelperWithLimit(path, o, matcher, false, tmp, maxFiles)
}

// Watches a directory.
//
// Generates write ops for any existing contents of the directory that aren't in
// the snapshot, and remove ops for any contents of the snapshot that aren't in
// the directory.
//
// The use of RecipeTagProvider is a major hack so that we can assign a Temp
// tag recipe to the initial ops. I wish we had a better way to do this but
// this is good enough for now.
func NewWorkspaceWatcherAndSync(path string, db dbint.DB2, previous data.SnapshotID, matcher *ospath.Matcher, owner data.UserID,
	tagFn RecipeTagProvider, tmp *temp.TempDir) (*WorkspaceWatcher, error) {
	o := newOptimizer(db, previous, owner, tagFn)
	return newWorkspaceWatcherHelper(path, o, matcher, true, tmp)
}

func newWorkspaceWatcherHelper(path string, o *optimizer, matcher *ospath.Matcher, snapshotInitially bool, tmp *temp.TempDir) (*WorkspaceWatcher, error) {
	return newWorkspaceWatcherHelperWithLimit(path, o, matcher, snapshotInitially, tmp, defaultMaxFilesWatched)
}

func newWorkspaceWatcherHelperWithLimit(path string, o *optimizer, matcher *ospath.Matcher, snapshotInitially bool, tmp *temp.TempDir, maxFiles int) (*WorkspaceWatcher, error) {
	watcher, err := NewWatcher()
	if err != nil {
		return nil, err
	}
	initialOpsEvent, s, err := newWorkspaceHelperWithLimit(path, o, matcher, time.Time{}, watcher, maxFiles)
	if err != nil {
		return nil, err
	}

	eventsCh := make(chan WatchEvent)

	syncDir, err := tmp.NewDir("watch-sync-")
	if err != nil {
		return nil, err
	}

	err = watcher.Add(syncDir.Path())
	if err != nil {
		return nil, errors.Propagatef(err, "Watcher")
	}

	ww := &WorkspaceWatcher{
		s:       s,
		o:       o,
		watcher: watcher,
		Events:  eventsCh,

		syncDir:     syncDir,
		syncChans:   make(map[string]chan error),
		notifyChans: make(map[string]chan struct{}),
		closeChan:   make(chan struct{}),
	}

	if !snapshotInitially {
		initialOpsEvent = WatchOpsEvent{}
	}

	coalescedCh := make(chan fsnotify.Event)
	go coalesceEvents(watcher.Events(), coalescedCh)
	go ww.loop(initialOpsEvent, coalescedCh)
	return ww, nil
}

// Crawls a directory, and generates write ops for everything in it.
func DirectoryToOpsEvent(path string, db dbint.DB2, matcher *ospath.Matcher,
	previous data.SnapshotID, owner data.UserID, tag data.RecipeWTag) (WatchOpsEvent, error) {
	optimizer := newOptimizer(db, previous, owner, NewRecipeTagProvider(tag))
	opsEvent, _, err := newWorkspaceHelper(path, optimizer, matcher, time.Time{}, nil)
	if err != nil {
		return WatchOpsEvent{}, err
	}

	return opsEvent, nil
}

// Crawls a directory for modifications since the given time, and generates temp write ops for everything in it.
func ChangesSinceModTimeToOpsEvent(path string, db dbint.DB2, previous data.SnapshotID, owner data.UserID, minMtime time.Time) (WatchOpsEvent, error) {
	optimizer := newOptimizer(db, previous, owner, NewRecipeTagProvider(data.RecipeWTagTemp))
	opsEvent, _, err := newWorkspaceHelper(path, optimizer, ospath.NewAllMatcher(), minMtime, nil)
	if err != nil {
		return WatchOpsEvent{}, err
	}

	return opsEvent, nil
}

// Returns a new watcher on a directory, and all the paths we know are in that
// directory. There are multiple possible initialization processes, and each
// will handle the existing files differently.
//
// Snapshot is the last known snapshot of the directory. We will use this to
// determine what ops to generate.
func newWorkspaceHelper(path string, optimizer *optimizer, matcher *ospath.Matcher, minMtime time.Time, watcher wmNotify) (WatchOpsEvent, *scanner, error) {
	return newWorkspaceHelperWithLimit(path, optimizer, matcher, minMtime, watcher, defaultMaxFilesWatched)
}

func newWorkspaceHelperWithLimit(path string, optimizer *optimizer, matcher *ospath.Matcher, minMTime time.Time, watcher wmNotify, maxFiles int) (WatchOpsEvent, *scanner, error) {
	if !ospath.IsDir(path) {
		return WatchOpsEvent{}, nil, fmt.Errorf("Not a directory: %s", path)
	}

	root, err := ospath.RealAbs(path)
	if err != nil {
		return WatchOpsEvent{}, nil, err
	}

	s := newScanner(root, matcher, watcher, minMTime, maxFiles)

	update, err := s.scanNewDir(root)
	if err != nil {
		return WatchOpsEvent{}, nil, err
	}
	opsEvent, err := optimizer.updateToEvent(update, true)
	if err != nil {
		return WatchOpsEvent{}, nil, err
	}

	return opsEvent, s, nil
}

func (ww *WorkspaceWatcher) loop(initialOpsEvent WatchOpsEvent, eventsCh chan fsnotify.Event) (outerErr error) {
	defer func() {
		if outerErr != nil {
			ww.closeErr = outerErr
		}
		close(ww.closeChan)

		if outerErr != nil {
			ww.Events <- WatchErrEvent{Err: outerErr}
		}
		close(ww.Events)
	}()

	if !initialOpsEvent.IsEmpty() {
		ww.Events <- initialOpsEvent
	}

	for {
		var event fsnotify.Event
		select {
		case err := <-ww.watcher.Errors():
			return err
		case event = <-eventsCh:
		}

		if strings.HasPrefix(event.Name, ww.syncDir.Path()) {
			ww.noteSync(event.Name)
			ww.Events <- WatchSyncEvent{Token: filepath.Base(event.Name)}
			continue
		}

		update, err := ww.s.eventToUpdate(event)
		if err != nil {
			return err
		}

		watchEvent, err := ww.o.updateToEvent(update, false)
		if err != nil {
			return err
		}

		for _, op := range watchEvent.Ops {
			fileOp, ok := op.(data.FileOp)
			if !ok {
				continue
			}
			if c, ok := ww.notifyChans[fileOp.FilePath()]; ok {
				go func() { c <- struct{}{} }()
			}
		}

		// NB(dbentley): the line below is helpful for testing if you're using sync correctly
		// time.Sleep(100 * time.Millisecond)
		if !watchEvent.IsEmpty() {
			ww.Events <- watchEvent
		}
	}
}

// Watches a directory of files and emits Ops for each change.
type WorkspaceWatcher struct {
	watcher wmNotify
	s       *scanner
	o       *optimizer
	Events  chan WatchEvent

	syncMu      sync.Mutex
	syncChans   map[string]chan error
	syncDir     *temp.TempDir
	notifyChans map[string]chan struct{}

	// A channel and error if the watcher has to stop watching
	closeChan chan struct{}
	closeErr  error
}

func (ww *WorkspaceWatcher) NotifyOnChange(path string, c chan struct{}) {
	ww.notifyChans[path] = c
}

func (ww *WorkspaceWatcher) Root() string {
	return ww.s.root
}

func (ww *WorkspaceWatcher) Matcher() *ospath.Matcher {
	return ww.s.matcher
}

// Wait for WorkspaceWatcher to catch up
// We accomplish this by writing to a file in a sync directory and waiting to notice.
// Sends a SyncOp on the ops channel with the given token.
// Cf. https://facebook.github.io/watchman/docs/cookies.html
// TODO(dbentley): what if they end up on different filesystems?
func (ww *WorkspaceWatcher) FSync(ctx context.Context, token string) error {
	select {
	case err := <-ww.fsync(token):
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-ww.closeChan:
		return ww.closeErr
	}
}

func (ww *WorkspaceWatcher) fsync(token string) chan error {
	ww.syncMu.Lock()
	defer ww.syncMu.Unlock()

	if token == "" {
		token = fmt.Sprintf("sync-file-%d", rand.Uint64())
	}

	ch := make(chan error)
	ww.syncChans[token] = ch

	path := filepath.Join(ww.syncDir.Path(), token)
	f, err := os.Create(path)
	if err != nil {
		delete(ww.syncChans, token)

		ch := make(chan error, 1)
		ch <- errors.Propagatef(err, "watcher#fsync#create(%s)", path)
		return ch
	}

	defer f.Close()
	return ch
}

func (ww *WorkspaceWatcher) noteSync(path string) {
	ww.syncMu.Lock()
	defer ww.syncMu.Unlock()

	syncName, _ := filepath.Rel(ww.syncDir.Path(), path)
	if ch, ok := ww.syncChans[syncName]; ok {
		close(ch)
		delete(ww.syncChans, syncName)
	}
}

func (ww *WorkspaceWatcher) Close() error {
	defer ww.syncDir.TearDown()
	// TODO(dmiller) this deadlocks on macOS. There are several PRs open on fsnotify that claim to resolve this
	// but none of them do
	return ww.watcher.Close()
}
