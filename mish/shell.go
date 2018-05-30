package mish

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/nsf/termbox-go"

	"github.com/windmilleng/mish/bridge/fs"
	"github.com/windmilleng/mish/bridge/fs/fss"
	"github.com/windmilleng/mish/daemon"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/dbpath"
	"github.com/windmilleng/mish/data/db/storage/storages"
	"github.com/windmilleng/mish/data/pathutil"
	"github.com/windmilleng/mish/logging"
	"github.com/windmilleng/mish/mish/shmill"
	"github.com/windmilleng/mish/os/ospath"
	"github.com/windmilleng/mish/os/temp"
)

// the shell is the controller of our MVC
type Shell struct {
	ctx    context.Context
	dir    string // TODO(dbentley): support a different Millfile
	db     dbint.DB2
	fs     fs.FSBridge
	shmill *shmill.Shmill
	model  *Model
	view   *View

	editCh       chan data.PointerAtRev
	editErrCh    chan error
	termEventCh  chan termbox.Event
	timeCh       <-chan time.Time
	panicCh      chan error
	shmillCh     chan shmill.Event
	shmillCancel context.CancelFunc
}

var ptrID = data.MustNewPointerID(data.AnonymousID, "mirror", data.UserPtr)

func Setup() (*Shell, error) {
	ctx := context.Background()

	wmDir, err := daemon.WindmillDir()
	if err != nil {
		return nil, err
	}
	if err := logging.SetupLogger(filepath.Join(wmDir, "mish")); err != nil {
		return nil, err
	}

	dir, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	recipes := storages.NewTestMemoryRecipeStore()
	ptrs := storages.NewMemoryPointers()
	db := db2.NewDB2(recipes, ptrs)
	tmp, err := temp.NewDir("mish")
	if err != nil {
		return nil, err
	}
	opt := db2.NewOptimizer(db, recipes)
	fs := fss.NewLocalFSBridge(ctx, db, opt, tmp)

	_, err = db.AcquirePointer(ctx, ptrID)
	if err != nil {
		return nil, err
	}

	if err := setupMirror(ctx, fs, dir, ptrID); err != nil {
		return nil, err
	}

	if err := termbox.Init(); err != nil {
		return nil, err
	}

	panicCh := make(chan error)

	return &Shell{
		ctx:    ctx,
		dir:    dir,
		db:     db,
		fs:     fs,
		shmill: shmill.NewShmill(fs, ptrID, dir, panicCh),
		model: &Model{
			File:      filepath.Join(dir, pathutil.WMShMill),
			Now:       time.Now(),
			HeadSnap:  data.EmptySnapshotID,
			Autorun:   dbpath.NewFileMatcherOrPanic(pathutil.WMShMill),
			Collapsed: make(map[int]bool),
			Spinner:   &Spinner{Chars: []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}},
		},
		view: &View{},

		editCh:      make(chan data.PointerAtRev),
		editErrCh:   make(chan error),
		termEventCh: make(chan termbox.Event),
		panicCh:     panicCh,
	}, nil
}

func setupMirror(ctx context.Context, fs fs.FSBridge, dir string, ptrID data.PointerID) error {
	// TODO(dbentley): allow specifying a different file on the command-line
	matcher, err := ospath.NewFileMatcher(pathutil.WMShMill)
	if err != nil {
		return err
	}
	return fs.ToWMStart(ctx, dir, ptrID, matcher)
}

func (sh *Shell) Run() error {
	defer termbox.Close()
	go sh.waitForEdits()
	go sh.waitForTermEvents()
	sh.timeCh = time.Tick(time.Second)
	for {
		select {
		case head := <-sh.editCh:
			if err := sh.handleEdit(head); err != nil {
				return err
			}
		case err := <-sh.editErrCh:
			return err
		case event, ok := <-sh.shmillCh:
			if !ok {
				sh.shmillCh = nil
			}
			if err := sh.handleShmill(event); err != nil {
				return err
			}
		case event := <-sh.termEventCh:
			if event.Type == termbox.EventKey && event.Ch == 'q' {
				return nil
			}
			sh.handleTerminal(event)
		case t := <-sh.timeCh:
			sh.model.Now = t
			sh.model.Spinner.Incr()
		case err := <-sh.panicCh:
			return err
		}
		sh.model.BlockSizes = sh.view.Render(sh.model)
	}
}

func concatenateAndDedupe(old, new []string) []string {
	for _, n := range new {
		dupe := false
		for _, o := range old {
			if o == n {
				dupe = true
				break
			}
		}
		if dupe {
			continue
		}
		old = append(old, n)
	}
	return old
}

func (sh *Shell) handleEdit(head data.PointerAtRev) error {
	sh.model.Rev = int(head.Rev)

	ptsAtSnap, err := sh.db.Get(sh.ctx, head)
	if err != nil {
		return err
	}

	pathsChanged, err := sh.db.PathsChanged(sh.ctx, sh.model.HeadSnap, ptsAtSnap.SnapID, data.RecipeRTagForPointer(ptsAtSnap.ID), dbpath.NewAllMatcher())
	if err != nil {
		return err
	}

	sh.model.HeadSnap = ptsAtSnap.SnapID

	sh.model.QueuedFiles = concatenateAndDedupe(sh.model.QueuedFiles, pathsChanged)

	if sh.shouldAutorun() {
		sh.startRun()
	}

	return nil
}

func (sh *Shell) shouldAutorun() bool {
	for _, f := range sh.model.QueuedFiles {
		if sh.model.Autorun.Match(f) {
			return true
		}
	}
	return false
}

func (sh *Shell) startRun() {
	sh.model.Shmill = NewShmill()
	sh.model.QueuedFiles = nil
	if sh.shmillCh != nil {
		sh.shmillCancel()
		// wait until this shmill execution tells us it's done (by closing the channel)
		for _ = range sh.shmillCh {
		}
	}

	ctx, cancelFunc := context.WithCancel(sh.ctx)
	sh.shmillCancel = cancelFunc
	sh.shmillCh = sh.shmill.Start(ctx)
}

func stringsEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, ae := range a {
		if b[i] != ae {
			return false
		}
	}
	return true
}

func (sh *Shell) handleShmill(ev shmill.Event) error {
	m := sh.model.Shmill
	switch ev := ev.(type) {
	case shmill.WatchStartEvent:
		m.Evals = append(m.Evals, &Watch{
			patterns: ev.Patterns,
			output:   ev.Output,
			start:    time.Now(),
		})
	case shmill.WatchDoneEvent:
		w := m.Evals[len(m.Evals)-1].(*Watch)
		w.done = true
		w.duration = time.Now().Sub(w.start)
	case shmill.AutorunEvent:
		m, err := dbpath.NewMatcherFromPatterns(append(ev.Patterns, pathutil.WMShMill))
		if err != nil {
			return err
		}
		sh.model.Autorun = m
	case shmill.CmdStartedEvent:
		m.Evals = append(m.Evals, &Run{
			cmd:   ev.Cmd,
			start: time.Now(),
		})
	case shmill.CmdOutputEvent:
		run := m.Evals[len(m.Evals)-1].(*Run)
		run.output += ev.Output
	case shmill.CmdDoneEvent:
		run := m.Evals[len(m.Evals)-1].(*Run)
		run.done = true
		run.err = ev.Err
		run.duration = time.Now().Sub(run.start)
	case shmill.ExecDoneEvent:
		m.Err = ev.Err
		m.Done = true
		m.Duration = time.Now().Sub(m.Start)
	}
	return nil
}

func (sh *Shell) handleTerminal(event termbox.Event) {
	if event.Type != termbox.EventKey {
		return
	}
	switch event.Key {
	case termbox.KeyArrowUp:
		sh.model.Cursor.Line--
		sh.snapCursorToBlock()
	case termbox.KeyArrowDown:
		sh.model.Cursor.Line++
		sh.snapCursorToBlock()
	}

	switch event.Ch {
	case 'r':
		sh.startRun()
	case 'j':
		sh.model.Cursor.Block++
		sh.model.Cursor.Line = 0
		sh.snapCursorToBlock()
	case 'k':
		sh.model.Cursor.Block--
		sh.model.Cursor.Line = 0
		sh.snapCursorToBlock()
	case 'o':
		if sh.model.Collapsed[sh.model.Cursor.Block] {
			delete(sh.model.Collapsed, sh.model.Cursor.Block)
		} else {
			sh.model.Collapsed[sh.model.Cursor.Block] = true
			sh.model.Cursor.Line = 0
		}
	}
}

// snapCursorToBlock makes the cursor point to a sensible position.
// E.g., if the Cursor is (4, -1), it will fix it to point to (3, <last line of block 3>)
func (sh *Shell) snapCursorToBlock() {
	c := sh.model.Cursor
	blocks := sh.model.BlockSizes

	if c.Line < 0 {
		c.Block = sh.prevBlock(c.Block)
		c.Line = sh.lastBlockLine(c.Block)
	}
	if c.Line > sh.lastBlockLine(c.Block) {
		c.Block++
		c.Line = 0
	}
	if c.Block >= len(blocks) {
		// we've fallen off the bottom edge; snap back
		lastBlock := len(blocks) - 1
		c.Block = lastBlock
		c.Line = sh.lastBlockLine(lastBlock)
	}
	if c.Block < 0 {
		c.Block = 0
		c.Line = 0
	}

	sh.model.Cursor = c
}

func (sh *Shell) lastBlockLine(i int) int {
	if i < 0 {
		return 0
	}
	if i >= len(sh.model.BlockSizes) {
		return 0
	}
	l := sh.model.BlockSizes[i] - 1
	if l < 0 {
		return 0
	}
	return l
}

func (sh *Shell) prevBlock(i int) int {
	if i >= len(sh.model.BlockSizes) {
		return len(sh.model.BlockSizes) - 1
	}
	return i - 1
}

// Below here is code that happens on goroutines other than Run()

func (sh *Shell) waitForEdits() (outerErr error) {
	// TODO(dbentley): the Millfile might run a command that edits this dir, which would cause an edit, which would cause us to start rerunning.
	// That is silly; how can we filter out, while not missing intentional user edits?
	defer func() {
		if outerErr != nil {
			sh.editErrCh <- outerErr
		}
		close(sh.editErrCh)
		close(sh.editCh)
		if r := recover(); r != nil {
			sh.panicCh <- fmt.Errorf("edit panic: %v", r)
		}
	}()

	head := data.PointerAtRev{ID: ptrID}
	for {
		if err := sh.db.Wait(sh.ctx, head); err != nil {
			return err
		}

		var err error
		head, err = sh.db.Head(sh.ctx, ptrID)
		if err != nil {
			return err
		}

		sh.editCh <- head
	}
}

func (sh *Shell) waitForTermEvents() {
	defer func() {
		if r := recover(); r != nil {
			sh.panicCh <- fmt.Errorf("term panic: %v", r)
		}
	}()
	for {
		sh.termEventCh <- termbox.PollEvent()
	}
}
