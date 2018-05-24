package dbint

import (
	"context"
	"time"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/storage"
	"github.com/windmilleng/mish/errors"
	"github.com/windmilleng/mish/logging"
	"golang.org/x/sync/errgroup"
)

func HeadSnap(ctx context.Context, db Reader, ptr data.PointerID) (data.PointerAtSnapshot, error) {
	head, err := db.Head(ctx, ptr)
	if err != nil {
		return data.PointerAtSnapshot{}, err
	}

	return db.Get(ctx, head)
}

func AcquireSnap(ctx context.Context, db Reader, ptr data.PointerID) (data.PointerAtSnapshot, error) {
	head, err := db.AcquirePointer(ctx, ptr)
	if err != nil {
		return data.PointerAtSnapshot{}, err
	}

	return db.Get(ctx, head)
}

func EditPointerAtSnapshot(ctx context.Context, db DB2, head data.PointerAtSnapshot, op data.Op) (data.PointerAtSnapshot, error) {
	recipe := data.Recipe{Op: op, Inputs: []data.SnapshotID{head.SnapID}}
	tag := data.RecipeWTagForPointer(head.ID)
	newSnap, _, err := db.Create(ctx, recipe, head.Owner(), tag)
	if err != nil {
		return data.PointerAtSnapshot{}, err
	}

	next := data.PtrEdit(head, newSnap)
	err = db.Set(ctx, next)
	if err != nil {
		return data.PointerAtSnapshot{}, err
	}

	return next, nil
}

func EditPointer(ctx context.Context, db DB2, ptr data.PointerID, op data.Op) (data.PointerAtSnapshot, error) {
	head, err := AcquireSnap(ctx, db, ptr)
	if err != nil {
		return data.PointerAtSnapshot{}, err
	}

	return EditPointerAtSnapshot(ctx, db, head, op)
}

func ReadSnapFileAtPtr(ctx context.Context, db Reader, ptr data.PointerID, path string) (*SnapshotFile, error) {
	ptrAtSnap, err := HeadSnap(ctx, db, ptr)
	if err != nil {
		return nil, err
	}

	f, err := db.SnapshotFile(ctx, ptrAtSnap.SnapID, path)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// Lookup the snapshot id for a list of PointerAtRevs, returning
// the PointerAtSnapshots in the same order. Fails if any lookup fails.
func ResolvePointerAtRevs(ctx context.Context, db DB2, ptrAtRevs []data.PointerAtRev) ([]data.PointerAtSnapshot, error) {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]data.PointerAtSnapshot, len(ptrAtRevs))

	for i, ptrAtRev := range ptrAtRevs {
		i, ptrAtRev := i, ptrAtRev
		g.Go(func() error {
			ptrAtSnapshot, err := db.Get(ctx, ptrAtRev)
			if err != nil {
				return err
			}
			results[i] = ptrAtSnapshot
			return nil
		})
	}

	err := g.Wait()
	if err != nil {
		return nil, err
	}
	return results, nil
}

func PathSet(ctx context.Context, db DB2, snap data.SnapshotID) (map[string]bool, error) {
	s, err := db.Snapshot(ctx, snap)
	if err != nil {
		return nil, err
	}

	return s.PathSet(), nil
}

// SetFrozen is a helper function to freeze a pointer.
func SetFrozen(ctx context.Context, db DB2, ptrAtSnap data.PointerAtSnapshot) error {
	next := data.PtrEdit(ptrAtSnap, ptrAtSnap.SnapID)
	next.Frozen = true
	return db.Set(ctx, next)
}

// WaitForFrozen is a helper function to wait for a Pointer to Freeze
func WaitForFrozen(ctx context.Context, db Reader, monitor storage.PointerLockMonitor, ptr data.PointerID) (data.PointerAtSnapshot, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		pollInterval := 200 * time.Millisecond

		for {
			lost, err := monitor.IsLockLost(ctx, ptr)
			if lost {
				cancel()
				return
			}

			if err == nil {
				// also check if the context has an error
				err = ctx.Err()
			}

			if err != nil {
				if errors.IsCanceled(err) || errors.IsDeadlineExceeded(err) {
					return
				}

				logging.With(ctx).Warnf("WaitForFrozen: IsLockLost: %v", err)
			}

			time.Sleep(pollInterval)
		}
	}()

	last := data.PointerAtSnapshot{ID: ptr, Rev: 0}
	for !last.Frozen {
		err := db.Wait(ctx, last.AsPointerAtRev())
		if err != nil {
			return last, errors.Propagatef(err, "WaitForFrozen(%s)", ptr)
		}
		last, err = HeadSnap(ctx, db, last.ID)
		if err != nil {
			return last, errors.Propagatef(err, "WaitForFrozen(%s)", ptr)
		}
	}
	return last, nil
}

func CreateLinear(ctx context.Context, db DB2, ops []data.Op, input data.SnapshotID, owner data.UserID, tag data.RecipeWTag) (data.SnapshotID, error) {
	var err error
	for _, op := range ops {
		input, _, err = db.Create(ctx, data.Recipe{
			Op:     op,
			Inputs: []data.SnapshotID{input},
		}, owner, tag)
		if err != nil {
			return data.SnapshotID{}, err
		}
	}
	return input, nil
}

// Fetches all the recipes between the given rev and the previous rev.
func RecipesFromPreviousRev(ctx context.Context, db Reader, want data.PointerAtSnapshot) ([]data.StoredRecipe, error) {
	if want.Rev == 0 {
		return []data.StoredRecipe{}, nil
	}

	have, err := db.Get(ctx, data.PointerAtRev{ID: want.ID, Rev: want.Rev - 1})
	if err != nil {
		return nil, err
	}

	return db.RecipesNeeded(ctx, have.SnapID, want.SnapID, want.ID)
}
