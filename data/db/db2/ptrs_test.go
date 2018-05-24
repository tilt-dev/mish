package db2

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
)

func TestPointers(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	id1 := f.create(wf("foo.txt", "foo"))
	id2 := f.create(wf("bar.txt", "bar"), id1)
	id3a := f.create(wf("foo.txt", "foo2"), id2)

	id3b := f.create(wf("foo.txt", "alterna-third"), id2)

	pID := "foo"

	h := makePointerHelper(t, f.db)
	defer h.Close()

	h.assertHead(pID, 0)

	h.acquire(pID)
	h.update(pID, 0, id1)

	// h.update also does these checks
	h.assertHead(pID, 1)
	h.assertRev(pID, 1, id1)

	h.update(pID, 1, id2)
	h.update(pID, 2, id3a)

	// now try a stale write
	h.updateError(pID, 2, id3b)

	// We can update it if we really mean to overwrite (by increasing the cur revision)
	h.update(pID, 3, id3b)

	// We can update backwards and in all directions
	h.update(pID, 4, id1)
	h.update(pID, 5, id3a)

	// We can freeze from and only from the head
	h.freezeError(pID, 5)
	h.freeze(pID, 6, id3a)

	// Now just test history
	h.assertRev(pID, 1, id1)
	h.assertRev(pID, 2, id2)
	h.assertRev(pID, 3, id3a)
	h.assertRev(pID, 4, id3b)
	h.assertRev(pID, 5, id1)
	h.assertRev(pID, 6, id3a)
	h.assertRev(pID, 7, id3a)

	// TODO(dbentley): test Wait
}

func TestPointersTmp(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	id1 := f.create(wf("foo.txt", "first"))
	id2 := f.create(wf("foo.txt", "second"), id1)

	h := makePointerHelper(t, f.db)
	defer h.Close()

	p1 := h.tmp()
	p2 := h.tmp()

	h.assertHead(p1, 1)
	h.assertRev(p1, 1, data.EmptySnapshotID)
	h.assertHead(p2, 1)
	h.assertRev(p2, 1, data.EmptySnapshotID)

	h.update(p1, 1, id1)
	h.assertHead(p2, 1)
	h.assertRev(p2, 1, data.EmptySnapshotID)

	h.update(p2, 1, id2)
	h.assertHead(p1, 2)
	h.assertRev(p1, 2, id1)
}

type pointerHelper struct {
	db  *DB2
	t   *testing.T
	ctx context.Context
}

func makePointerHelper(t *testing.T, db *DB2) *pointerHelper {
	return &pointerHelper{
		db:  db,
		t:   t,
		ctx: context.Background(),
	}
}

func (h *pointerHelper) Close() {}

func (h *pointerHelper) tmp() string {
	head, err := h.db.MakeTemp(h.ctx, data.UserTestID, "db_test", data.UserPtr)
	if err != nil {
		h.t.Fatal(err)
	}

	return head.ID.String()
}

func (h *pointerHelper) acquire(pointerID string) {
	_, err := h.db.AcquirePointer(h.ctx, data.MustParsePointerID(pointerID))
	if err != nil {
		h.t.Fatal(err)
	}
}

func (h *pointerHelper) assertHead(pointerID string, expected int64) {
	h.assertHeadWithFreeze(pointerID, expected, false)
}

func (h *pointerHelper) assertHeadWithFreeze(pointerID string, expected int64, exFrozen bool) {
	expectedMissing := expected == 0
	actualAtRev, err := dbint.HeadSnap(h.ctx, h.db, data.MustParsePointerID(pointerID))
	isMissing := err != nil && grpc.Code(err) == codes.NotFound
	if expectedMissing && !isMissing {
		h.t.Fatalf("Expected missing: %s", pointerID)
	} else if !expectedMissing && err != nil {
		h.t.Fatalf("assertHead(%s, %d): %v", pointerID, expected, err)
	}

	actual := actualAtRev.Rev
	if actualAtRev.Frozen != exFrozen {
		h.t.Fatalf("pointer %v at rev %v's frozen is %v, expected %v", pointerID, actual, actualAtRev.Frozen, exFrozen)
	}

	ex := data.PointerRev(expected)
	if ex != actual {
		h.t.Fatalf("pointer %v is at rev %v; expected %v", pointerID, actual, ex)
	}
}

func (h *pointerHelper) assertRev(pointerID string, rev int64, expected data.SnapshotID) {
	reply, err := h.db.Get(h.ctx, data.PointerAtRev{ID: data.MustParsePointerID(pointerID), Rev: data.PointerRev(rev)})
	if err != nil {
		h.t.Fatal(err)
	}

	actual := reply.SnapID
	if expected != actual {
		h.t.Fatalf("pointer %v at rev %v points to %v; expected %v", pointerID, rev, actual, expected)
	}
}

func (h *pointerHelper) update(pointerID string, curRev int64, snap data.SnapshotID) {
	next := data.PointerAtSnapshot{
		ID:     data.MustParsePointerID(pointerID),
		Rev:    data.PointerRev(curRev + 1),
		SnapID: snap,
	}
	if err := h.db.Set(h.ctx, next); err != nil {
		h.t.Fatal(err)
	}

	h.assertHead(pointerID, curRev+1)
	h.assertRev(pointerID, curRev+1, snap)
}

func (h *pointerHelper) updateError(pointerID string, curRev int64, snap data.SnapshotID) {
	next := data.PointerAtSnapshot{
		ID:     data.MustParsePointerID(pointerID),
		Rev:    data.PointerRev(curRev + 1),
		SnapID: snap,
	}
	if err := h.db.Set(h.ctx, next); err == nil {
		h.t.Fatalf("expected update of pointer %v at rev %v to %v to fail; it succeeded", pointerID, curRev, snap)
	}

}

func (h *pointerHelper) freeze(pointerID string, curRev int64, snap data.SnapshotID) {
	next := data.PointerAtSnapshot{
		ID:     data.MustParsePointerID(pointerID),
		Rev:    data.PointerRev(curRev + 1),
		SnapID: snap,
		Frozen: true,
	}
	if err := h.db.Set(h.ctx, next); err != nil {
		h.t.Fatal(err)
	}

	h.assertHeadWithFreeze(pointerID, curRev+1, true)
	h.assertRev(pointerID, curRev, snap)
	h.assertRev(pointerID, curRev+1, snap)
}

func (h *pointerHelper) freezeError(pointerID string, curRev int64) {
	next := data.PointerAtSnapshot{
		ID:     data.MustParsePointerID(pointerID),
		Rev:    data.PointerRev(curRev + 1),
		SnapID: data.EmptySnapshotID,
		Frozen: true,
	}
	if err := h.db.Set(h.ctx, next); err == nil {
		h.t.Fatalf("expected freeze of pointer %v at rev %v to fail; it succeeded", pointerID, curRev)
	}
}
