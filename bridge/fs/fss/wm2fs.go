package fss

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/windmilleng/mish/bridge/fs"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/dbpath"
	"github.com/windmilleng/mish/logging"
	"github.com/windmilleng/mish/os/temp"
	"github.com/windmilleng/mish/os/watch"
)

// ctx: A context that should live for the lifetime of this object.
func newWM2FS(ctx context.Context, db dbint.DB2, tmp *temp.TempDir) *wm2fs {
	return &wm2fs{
		ctx:    ctx,
		db:     db,
		tmp:    tmp,
		active: make(map[string]*checkout),
	}
}

type wm2fs struct {
	ctx context.Context
	db  dbint.DB2
	tmp *temp.TempDir

	active map[string]*checkout
	mu     sync.Mutex
}

func (m *wm2fs) Checkout(ctx context.Context, id data.SnapshotID, path string) (co fs.CheckoutStatus, outerErr error) {
	if path == "" {
		checkoutTmp, err := m.tmp.NewDir("co-")
		if err != nil {
			return co, err
		}
		path, err = filepath.EvalSymlinks(checkoutTmp.Path())
		if err != nil {
			return co, err
		}
		defer func() {
			if outerErr != nil {
				os.RemoveAll(path)
			}
		}()
	} else if filepath.IsAbs(path) {
		fs, err := ioutil.ReadDir(path)
		if err != nil {
			return co, fmt.Errorf("Checkout: can't read path %q", path)
		}

		if len(fs) != 0 {
			return co, fmt.Errorf("Checkout: path %q isn't empty; e.g. %s", path, fs[0].Name())
		}
	} else {
		return co, fmt.Errorf("Checkout: only absolute paths allowed: %s", path)
	}

	return checkoutAt(ctx, m.db, id, path)
}

func (m *wm2fs) ResetCheckout(ctx context.Context, co fs.CheckoutStatus) error {
	id := co.SnapID
	path := co.Path
	opsEvent, err := watch.ChangesSinceModTimeToOpsEvent(path, m.db, id, id.Owner(), co.Mtime)
	if err != nil {
		return err
	}
	ops := opsEvent.Ops

	pathsChanged := make(map[string]bool, len(ops))
	for _, op := range ops {
		_, ok := op.(*data.SyncOp)
		if ok {
			continue
		}

		fileOp, ok := op.(data.FileOp)
		if ok {
			pathsChanged[fileOp.FilePath()] = true
		} else {
			// DirectoryToOpsEvent should only produce file ops
			return fmt.Errorf("ResetCheckout: unexpected op %T", op)
		}
	}

	for k, _ := range pathsChanged {
		f, err := m.db.SnapshotFile(ctx, id, k)
		missing := err != nil && grpc.Code(err) == codes.NotFound
		if missing {
			err = os.Remove(filepath.Join(path, k))
			if err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}

		err = applyWrite(&data.WriteFileOp{
			Path:       k,
			Data:       f.Contents,
			Executable: f.Executable,
			Type:       f.Type,
		}, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *wm2fs) Start(ptr data.PointerID, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logging.With(m.ctx).Infoln("Starting", ptr, path)

	if c := m.active[path]; c != nil {
		// TODO(dbentley): check for overlapping mirrors. if path is /foo/bar, we should error
		// if we're already mirroring into either /foo or /foo/bar/baz
		return fmt.Errorf("Start: already mirroring into path %s", path)
	}

	fs, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("Start: can't read path %q", path)
	}

	if len(fs) != 0 {
		return fmt.Errorf("Start: path %q isn't empty; e.g. %s", path, fs[0].Name())
	}

	ctx, stopFunc := context.WithCancel(m.ctx)
	c := &checkout{
		db: m.db,

		stopFunc: stopFunc,
		ctx:      ctx,

		path:        path,
		currentHead: data.PointerAtSnapshotZero(ptr),

		err:  nil,
		done: make(chan struct{}),
	}
	m.active[path] = c
	go c.loop(ptr)
	return nil
}

func (m *wm2fs) Stop(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	c := m.active[path]
	if c == nil {
		return fmt.Errorf("Stop: not mirroring into path %s", path)
	}

	c.stopFunc()
	select {
	case <-c.done:
		delete(m.active, path)
		return c.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *wm2fs) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.active {
		c.stopFunc()

		select {
		case <-c.done:
			// Shutdown shouldn't affect serialization state,
			// so don't delete the checkin from the map.
			err := c.err
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

type checkout struct {
	db   dbint.DB2
	ptrs data.Pointers

	stopFunc context.CancelFunc
	ctx      context.Context

	path        string
	currentHead data.PointerAtSnapshot

	err  error
	done chan struct{}
}

func (c *checkout) loop(ptr data.PointerID) {
	defer close(c.done)

	for {
		ptrUpdateCh := make(chan error, 1)

		go func() {
			ptrUpdateCh <- c.db.Wait(c.ctx, c.currentHead.AsPointerAtRev())
		}()

		select {
		case <-c.ctx.Done():
			return
		case err := <-ptrUpdateCh:
			if err != nil {
				// there was an error waiting for an update!
				c.err = err
				return
			}
			if err := c.update(); err != nil {
				c.err = err
				return
			}
		}
	}
}

func (c *checkout) update() error {
	newHead, err := dbint.HeadSnap(c.ctx, c.db, c.currentHead.ID)
	if err != nil {
		return err
	}

	newSnap := newHead.SnapID
	logging.With(c.ctx).Infof("Updating path %s from %v to %v", c.path, c.currentHead.SnapID, newSnap)

	recipes, err := c.db.RecipesNeeded(c.ctx, c.currentHead.SnapID, newHead.SnapID, c.currentHead.ID)
	if err != nil {
		return err
	}

	incremental := true
	for _, r := range recipes {
		if !canApply(r.Op) {
			incremental = false
		}
	}

	if incremental {
		logging.With(c.ctx).Infof("Applying %d ops", len(recipes))
		for i, _ := range recipes {
			op := recipes[len(recipes)-i-1].Op
			if err := applyToFS(op, c.path); err != nil {
				return err
			}
		}
	} else {
		logging.With(c.ctx).Infof("Clear and checkout")
		if err := applyClear(&data.ClearOp{}, c.path); err != nil {
			return err
		}

		if _, err := checkoutAt(c.ctx, c.db, newSnap, c.path); err != nil {
			return err
		}
	}

	c.currentHead = newHead

	return nil
}

func (c *checkout) stop() {
	c.stopFunc()
}

func checkoutAt(ctx context.Context, db dbint.DB2, snap data.SnapshotID, checkout string) (fs.CheckoutStatus, error) {
	s, err := db.SnapshotFilesMatching(ctx, snap, dbpath.NewAllMatcher())
	if err != nil {
		return fs.CheckoutStatus{}, err
	}

	for _, file := range s {
		if err := applyWrite(&data.WriteFileOp{
			Path:       file.Path,
			Executable: file.Executable,
			Data:       file.Contents,
			Type:       file.Type,
		}, checkout); err != nil {
			return fs.CheckoutStatus{}, err
		}

	}

	mTime := time.Time{}
	if len(s) > 0 {
		lastFile := s[len(s)-1]
		info, err := os.Stat(filepath.Join(checkout, lastFile.Path))
		if err != nil {
			return fs.CheckoutStatus{}, err
		}
		mTime = info.ModTime()
	}
	return fs.CheckoutStatus{
		Path:   checkout,
		Mtime:  mTime,
		SnapID: snap,
	}, nil
}
