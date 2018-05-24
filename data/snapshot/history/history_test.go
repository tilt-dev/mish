package history

import (
	"context"
	"fmt"
	"testing"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/storage/storages"
	"github.com/windmilleng/mish/os/temp"
)

var ptr = data.MustParsePointerID("ptr-1")
var ctx = context.Background()

func TestPointerHistoryMaxRev(t *testing.T) {
	f := newFixture(t)

	f.writeRevs(20)
	historyPtrIDs, err := f.history.GetPointerHistory(ctx, ptr, PointerHistoryQuery{
		MaxRev: data.PointerRev(15),
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(historyPtrIDs) != 10 {
		t.Errorf("Unexpected history ptrs %v", historyPtrIDs)
	}

	if int(historyPtrIDs[0].Rev) != 15 {
		t.Errorf("Unexpected history ptr %v", historyPtrIDs[0])
	}

	if int(historyPtrIDs[9].Rev) != 6 {
		t.Errorf("Unexpected history ptr %v", historyPtrIDs[9])
	}
}

func TestPointerHistoryMaxRevCount(t *testing.T) {
	f := newFixture(t)

	f.writeRevs(20)
	historyPtrIDs, err := f.history.GetPointerHistory(ctx, ptr, PointerHistoryQuery{
		MaxRevCount: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(historyPtrIDs) != 5 {
		t.Errorf("Unexpected history ptrs %v", historyPtrIDs)
	}

	if int(historyPtrIDs[0].Rev) != 21 {
		t.Errorf("Unexpected history ptr %v", historyPtrIDs[0])
	}

	if int(historyPtrIDs[4].Rev) != 17 {
		t.Errorf("Unexpected history ptr %v", historyPtrIDs[9])
	}
}

func TestPointerHistoryMinRev(t *testing.T) {
	f := newFixture(t)

	f.writeRevs(20)
	historyPtrIDs, err := f.history.GetPointerHistory(ctx, ptr, PointerHistoryQuery{
		MinRev: data.PointerRev(6),
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(historyPtrIDs) != 10 {
		t.Errorf("Unexpected history ptrs %v", historyPtrIDs)
	}

	if int(historyPtrIDs[0].Rev) != 15 {
		t.Errorf("Unexpected history ptr %v", historyPtrIDs[0])
	}

	if int(historyPtrIDs[9].Rev) != 6 {
		t.Errorf("Unexpected history ptr %v", historyPtrIDs[9])
	}
}

type fixture struct {
	t          *testing.T
	db         *db2.DB2
	history    *SnapshotHistoryService
	tmpDir     *temp.TempDir
	headSnapID data.SnapshotID
}

func (f *fixture) writeRevs(count int) {
	var ptrAtSnap data.PointerAtSnapshot

	for i := 0; i < count; i++ {
		ptrAtSnap, _ = dbint.EditPointer(ctx, f.db, ptr, &data.WriteFileOp{Path: fmt.Sprintf("tmp%d.txt", i), Data: data.BytesFromString("hi")})
	}

	f.headSnapID = ptrAtSnap.SnapID
}

func newFixture(t *testing.T) *fixture {
	ptrs := storages.NewMemoryPointers()
	store := storages.NewTestMemoryRecipeStore()
	db := db2.NewDB2(store, ptrs)
	_, err := db.AcquirePointer(context.Background(), ptr)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Set(context.Background(), data.PointerAtSnapshot{ID: ptr, Rev: 1, SnapID: data.EmptySnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{
		t:          t,
		db:         db,
		history:    NewSnapshotHistoryService(db),
		headSnapID: data.EmptySnapshotID,
	}
}
