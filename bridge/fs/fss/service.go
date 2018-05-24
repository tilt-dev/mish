package fss

import (
	"context"

	"github.com/windmilleng/mish/bridge/fs"
	"github.com/windmilleng/mish/bridge/fs/proto"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/os/ospath"
	"github.com/windmilleng/mish/os/temp"
)

type LocalFSBridge struct {
	wm2fs *wm2fs
	fs2wm *fs2wm
}

// ctx: Context for the lifetime of this bridge.
func NewLocalFSBridge(ctx context.Context, db2 dbint.DB2, optimizer dbint.Optimizer, tmp *temp.TempDir) *LocalFSBridge {
	return &LocalFSBridge{
		wm2fs: newWM2FS(ctx, db2, tmp),
		fs2wm: newFS2WM(ctx, db2, optimizer, tmp),
	}
}

// Add a change listener that stops listening when the context is done.
func (b *LocalFSBridge) AddChangeListener(ctx context.Context, l chan *proto.FsBridgeState) {
	b.fs2wm.AddChangeListener(ctx, l)
}

func (b *LocalFSBridge) SetMirrorErrorHandler(f func(err error)) {
	b.fs2wm.SetMirrorErrorHandler(f)
}

func (b *LocalFSBridge) Checkout(ctx context.Context, id data.SnapshotID, path string) (fs.CheckoutStatus, error) {
	return b.wm2fs.Checkout(ctx, id, path)
}

func (b *LocalFSBridge) ResetCheckout(ctx context.Context, co fs.CheckoutStatus) error {
	return b.wm2fs.ResetCheckout(ctx, co)
}

func (b *LocalFSBridge) SnapshotDir(ctx context.Context, path string, matcher *ospath.Matcher, owner data.UserID, tag data.RecipeWTag, hint fs.Hint) (data.SnapshotID, error) {
	return b.fs2wm.SnapshotDir(ctx, path, matcher, owner, tag, hint)
}

func (b *LocalFSBridge) ToWMStart(ctx context.Context, path string, ptr data.PointerID, matcher *ospath.Matcher) error {
	return b.fs2wm.Start(ctx, path, ptr, matcher)
}

func (b *LocalFSBridge) ToWMStatus(ctx context.Context, path string) (*proto.FsToWmState, error) {
	return b.fs2wm.Status(path)
}

func (b *LocalFSBridge) ToWMPointerStatus(ctx context.Context, id data.PointerID) (*proto.FsToWmState, error) {
	return b.fs2wm.PointerStatus(id)
}

func (b *LocalFSBridge) ToWMFSync(ctx context.Context, ptr data.PointerID) (data.PointerAtSnapshot, error) {
	return b.fs2wm.FSync(ctx, ptr)
}

func (b *LocalFSBridge) ToWMStop(ctx context.Context, ptr data.PointerID) error {
	return b.fs2wm.Stop(ctx, ptr)
}

func (b *LocalFSBridge) FromWMStart(ctx context.Context, ptr data.PointerID, path string) error {
	return b.wm2fs.Start(ptr, path)
}

func (b *LocalFSBridge) FromWMStop(ctx context.Context, path string) error {
	return b.wm2fs.Stop(ctx, path)
}

func (b *LocalFSBridge) Shutdown(ctx context.Context) error {
	err := b.wm2fs.Shutdown(ctx)
	if err != nil {
		return err
	}
	return b.fs2wm.Shutdown(ctx)
}

var _ fs.FSBridge = &LocalFSBridge{}
