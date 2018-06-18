package fs

import (
	"context"
	"fmt"
	"time"

	"github.com/windmilleng/mish/bridge/fs/proto"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/acl"
	"github.com/windmilleng/mish/os/ospath"
	ospathConv "github.com/windmilleng/mish/os/ospath/convert"
)

// A Hint describes how to Snapshot a Directory.
// For now, it's just the Base Snapshot that we expect this to be based off of.
// But it might grow to include, e.g., an output we expect it to resemble.
// (If we ran this test before, include both the base input and the output from last time)
type Hint struct {
	Base data.SnapshotID
}

type FSBridge interface {
	// Checkout a snapshot into an empty directory.
	//
	// If path is empty, we check out to a random tmp directory
	// in the bridge's workspace.
	//
	// If the path is relative, we throw an error.
	Checkout(ctx context.Context, id data.SnapshotID, path string) (CheckoutStatus, error)

	// Set the contents of the directory to match a snapshot.
	//
	// Optimized for the case when a small set of arbitrary changes has been made
	// to a directory, and we need to reset it to the previous state.
	//
	// The mtime of CheckoutStatus is optional.  If mtime is provided, then as an
	// optimization we will only look at files modified after the mtime.
	ResetCheckout(ctx context.Context, status CheckoutStatus) error

	// Capture the contents of a directory and write it to the DB.
	SnapshotDir(ctx context.Context, path string, matcher *ospath.Matcher, owner data.UserID, tag data.RecipeWTag, hint Hint) (data.SnapshotID, error)

	ToWMStart(ctx context.Context, path string, ptr data.PointerID, matcher *ospath.Matcher) error
	ToWMStatus(ctx context.Context, path string) (*proto.FsToWmState, error)
	ToWMPointerStatus(ctx context.Context, ptr data.PointerID) (*proto.FsToWmState, error)
	ToWMFSync(ctx context.Context, ptr data.PointerID) (data.PointerAtSnapshot, error)
	ToWMStop(ctx context.Context, ptr data.PointerID) error

	FromWMStart(ctx context.Context, ptr data.PointerID, path string) error
	FromWMStop(ctx context.Context, path string) error

	Shutdown(ctx context.Context) error
}

// Initializes the bridge from a state protobuf.
// user: The currently authenticated user. If the user doesn't
//   have write access to the pointer, we should quietly skip it.
func InitFromState(ctx context.Context, bridge FSBridge, state *proto.FsBridgeState, acl acl.Checker, user data.User) error {
	for _, s := range state.FsToWmMirrors {
		path := string(s.Path)
		ptr, err := data.ParsePointerID(s.Pointer)
		if err != nil {
			return err
		}

		if !acl.CanWrite(ctx, ptr, user.UserID) {
			continue
		}

		if len(s.Matcher.Patterns) == 0 {
			return fmt.Errorf("Ignore patterns should not be empty")
		}

		matcher, err := ospathConv.MatcherP2D(s.Matcher)
		if err != nil {
			return err
		}
		err = bridge.ToWMStart(ctx, path, ptr, matcher)
		if err != nil {
			return err
		}
	}
	return nil
}

type CheckoutStatus struct {
	// The path on disk where the checkout lives.
	Path string

	// The initial snapshot that was checked out.
	SnapID data.SnapshotID

	// The modification time of the initial checkout.
	// Any files modified after this time were not part of the checkout.
	Mtime time.Time
}

// Configuration for runners and compositions that need to capture file system state.
type SnapshotConfig struct {
	OSMatcher *ospath.Matcher
}

func (c *SnapshotConfig) Matcher() *ospath.Matcher {
	if c.OSMatcher.Matcher == nil {
		return ospath.NewEmptyMatcher()
	}
	return c.OSMatcher
}

func NewSnapshotConfig(matcher *ospath.Matcher) *SnapshotConfig {
	return &SnapshotConfig{OSMatcher: matcher}
}
