package storages

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/windmilleng/mish/data"
)

type idExpectation struct {
	id string
	r  data.Recipe
}

func TestContentHashes(t *testing.T) {
	s := NewTestMemoryRecipeStore()

	recipes := []idExpectation{
		{
			id: "sha256-2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
			r:  data.Recipe{Op: &data.WriteFileOp{Path: "", Data: data.BytesFromString("foo")}},
		},
		{
			id: "sha256-2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
			r: data.Recipe{
				Op:     &data.WriteFileOp{Path: "", Data: data.BytesFromString("foo")},
				Inputs: []data.SnapshotID{data.EmptySnapshotID},
			},
		},
		{
			id: "test-0",
			r:  data.Recipe{Op: &data.WriteFileOp{Path: "foo.txt", Data: data.BytesFromString("foo")}},
		},
		{
			id: "sha256-baa5a0964d3320fbc0c6a922140453c8513ea24ab8fd0577034804a967248096",
			r:  data.Recipe{Op: &data.WriteFileOp{Path: "", Data: data.BytesFromString("baz")}},
		},
		{
			id: "sha256-b9954a5884f98cce22572ad0f0d8b8425b196dcabac9f7cbec9f1427930d8c44",
			r:  data.Recipe{Op: &data.WriteFileOp{Path: "", Data: data.BytesFromString("foo"), Executable: true}},
		},
		{
			id: "sha256-2de13384d175aa324e4b677607a18a877b2a1d55bd18283d9efb5c6b7f07307d",
			r:  data.Recipe{Op: &data.WriteFileOp{Path: "", Data: data.BytesFromString("foo"), Type: data.FileSymlink}},
		},
		{
			id: "sha256-76069c512f77648af0ca449b5bc20b42978d34b38f8021268bce9723e8feb16d",
			r: data.Recipe{
				Op:     &data.DirOp{Names: []string{"foo.txt"}},
				Inputs: data.SnapshotIDs("sha256-2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"),
			},
		},
		{
			id: "sha256-773ad29d2b932f93ed1f3e21664bee2a190c8aadb85eb23fe96bbabdb6d3ff4d",
			r: data.Recipe{
				Op:     &data.DirOp{Names: []string{"bar.txt"}},
				Inputs: data.SnapshotIDs("sha256-2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"),
			},
		},
		{
			id: "test-1",
			r: data.Recipe{
				Op:     &data.DirOp{Names: []string{"foo.txt"}},
				Inputs: data.SnapshotIDs("test-0"),
			},
		},
		{
			// Not in sorted order, so not content-addressable
			id: "test-2",
			r: data.Recipe{
				Op: &data.DirOp{Names: []string{"foo.txt", "bar.txt"}},
				Inputs: data.SnapshotIDs(
					"sha256-2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
					"sha256-baa5a0964d3320fbc0c6a922140453c8513ea24ab8fd0577034804a967248096"),
			},
		},
		{
			id: "sha256-a3ee13edbdbbd81e35446c7d1761032dac7dc335eed7db67928326b334972bf6",
			r: data.Recipe{
				Op: &data.DirOp{Names: []string{"bar.txt", "foo.txt"}},
				Inputs: data.SnapshotIDs(
					"sha256-baa5a0964d3320fbc0c6a922140453c8513ea24ab8fd0577034804a967248096",
					"sha256-2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"),
			},
		},
		{
			id: "test-3",
			r: data.Recipe{
				Op: &data.DirOp{Names: []string{"bar.txt", "foo.txt"}},
				Inputs: data.SnapshotIDs(
					"3$sha256-baa5a0964d3320fbc0c6a922140453c8513ea24ab8fd0577034804a967248096",
					"sha256-2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"),
			},
		},
		{
			// NOTE(nick): It's not totally clear to me that an empty directory should
			// be valid as an input to a DirOp. I don't know how this even happens. But I've
			// seen it in live systems and I can't think of an easy way to prevent it.
			id: "2$sha256-9e57e2340d4075607f75428c35c04bdb6a967147a683f5b2a2328e3360e113ba",
			r: data.Recipe{
				Op: &data.DirOp{Names: []string{"bar", "foo"}},
				Inputs: data.SnapshotIDs(
					"empty",
					"sha256-2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"),
			},
		},
		{
			// Ensure that rearranging the inputs changes the hash.
			id: "2$sha256-3a2e366d99d2fb36097a762f84736564f01405a6f3b6594d60a2dbeab9addb39",
			r: data.Recipe{
				Op: &data.DirOp{Names: []string{"bar", "foo"}},
				Inputs: data.SnapshotIDs(
					"sha256-2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
					"empty"),
			},
		},
	}
	ctx := context.Background()
	tag := data.RecipeWTagEdit

	for i, e := range recipes {
		t.Run(fmt.Sprintf("TestCreate-%d", i), func(t *testing.T) {
			id, _, err := s.Create(ctx, e.r, data.UserTestID, tag)
			if err != nil {
				t.Fatal(err)
			}

			expected := data.ParseSnapshotID(e.id)
			if id != expected {
				t.Errorf("Wrong ID. Expected %q. Actual %q.", expected, id)
			}
		})
	}
}

func TestBadCreate(t *testing.T) {
	s := NewTestMemoryRecipeStore()
	ctx := context.Background()
	r := data.Recipe{Op: &data.WriteFileOp{Path: "foo.txt", Data: data.BytesFromString("foo")}}
	_, _, err := s.Create(ctx, r, data.UserTestID, data.RecipeWTagOptimal)
	if err == nil || !strings.Contains(err.Error(), "Non-optimal recipe") {
		t.Errorf("Expected error. Actual: %v", err)
	}

	err = s.CreatePath(ctx, data.StoredRecipe{Snap: data.ParseSnapshotID("sim-0"), Tag: data.RecipeWTagOptimal, Recipe: r})
	if err == nil || !strings.Contains(err.Error(), "Non-optimal recipe") {
		t.Errorf("Expected error. Actual: %v", err)
	}
}

func TestEmptyCreate(t *testing.T) {
	s := NewTestMemoryRecipeStore()
	ctx := context.Background()
	r := data.Recipe{Op: &data.DirOp{}}
	snapID, isNew, err := s.Create(ctx, r, data.UserTestID, data.RecipeWTagEdit)
	if err != nil {
		t.Fatal(err)
	}

	if snapID != data.EmptySnapshotID {
		t.Errorf("Expected %s. Actual %s", data.EmptySnapshotID, snapID)
	}

	if isNew {
		t.Errorf("Writing to EmptySnapshotID should not be considered new")
	}
}

func TestBackfillHistory(t *testing.T) {
	ptrs := NewMemoryPointers()
	ctx := context.Background()

	p := data.MustParsePointerID("ptr")
	s := data.EmptySnapshotID
	ptrs.SetExisting(ctx, data.PointerAtSnapshot{ID: p, Rev: 10, SnapID: s})
	ptrs.SetExisting(ctx, data.PointerAtSnapshot{ID: p, Rev: 5, SnapID: s})
	ptrs.SetExisting(ctx, data.PointerAtSnapshot{ID: p, Rev: 7, SnapID: s})

	head, err := ptrs.Head(ctx, p)
	if err != nil {
		t.Fatal(err)
	}

	if head.Rev != 10 {
		t.Errorf("Expected 10. Actual %d", head.Rev)
	}

	err = ptrs.Set(ctx, data.PointerAtSnapshot{ID: p, Rev: 6, SnapID: s})
	if err == nil || strings.Index(err.Error(), "Stale write") == -1 {
		t.Errorf("Expected stale write error. Actual: %v", err)
	}

	err = ptrs.Set(ctx, data.PointerAtSnapshot{ID: p, Rev: 11, SnapID: s})
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := ptrs.Get(ctx, data.PointerAtRev{ID: p, Rev: 7})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Rev != 7 {
		t.Errorf("Expected 7. Actual %d", resolved.Rev)
	}

	resolved, err = ptrs.Get(ctx, data.PointerAtRev{ID: p, Rev: 8})
	if err == nil || grpc.Code(err) != codes.NotFound {
		t.Errorf("Expected not found error. Actual: %v", err)
	}
}

func TestBadCreatePath(t *testing.T) {
	s := NewTestMemoryRecipeStore()
	ctx := context.Background()
	ptr := data.MustParsePointerID("ptr")
	tag := data.RecipeWTagForPointer(ptr)
	r := data.Recipe{Op: &data.WriteFileOp{Path: "foo.txt", Data: data.BytesFromString("foo")}}
	r2 := data.Recipe{Op: &data.WriteFileOp{Path: "foo.txt", Data: data.BytesFromString("foo2")}}
	snapID, _, err := s.Create(ctx, r, data.UserTestID, tag)
	if err != nil {
		t.Fatal(err)
	}

	err = s.CreatePath(ctx, data.StoredRecipe{Snap: snapID, Tag: tag, Recipe: r})
	if err != nil {
		t.Fatal(err)
	}

	err = s.CreatePath(ctx, data.StoredRecipe{Snap: snapID, Tag: tag, Recipe: r2})
	if err == nil || !strings.Contains(err.Error(), "Recipe already exists") {
		t.Errorf("Expected error. Actual: %v", err)
	}
}

func TestUniqueIDs(t *testing.T) {
	s := NewTestMemoryRecipeStore()
	ctx := context.Background()
	ptr := data.MustParsePointerID("ptr")
	tag := data.RecipeWTagForPointer(ptr)

	var mu sync.Mutex
	created := make(map[data.SnapshotID]bool)

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			iters := 100
			local := make([]data.SnapshotID, iters)
			for j := 0; j < iters; j++ {
				id, _, err := s.Create(ctx,
					data.Recipe{
						Op: &data.WriteFileOp{Path: "foo.txt",
							Data: data.BytesFromString(fmt.Sprintf("data-%d-%d", i, j))},
						Inputs: []data.SnapshotID{data.EmptySnapshotID},
					},
					data.UserTestID, tag)
				if err != nil {
					t.Fatal(err)
				}

				local[j] = id
			}

			mu.Lock()
			defer mu.Unlock()
			for j := 0; j < iters; j++ {
				id := local[j]
				if created[id] {
					t.Fatalf("snapshot ID already exists: %v", id)
				}
				created[id] = true
			}
		}()
	}
	wg.Wait()
}

func TestAcquirePointer(t *testing.T) {
	ptrs := NewMemoryPointers()
	ctx := context.Background()
	ptr := data.MustParsePointerID("ptr")
	host1 := data.Host("wmfrontend")
	host2 := data.Host("wmapi")

	_, err := ptrs.AcquirePointerWithHost(ctx, ptr, host1)
	if err != nil {
		t.Fatal(err)
	}

	m1, err := ptrs.PointerMetadata(ctx, ptr)
	if err != nil {
		t.Fatal(err)
	} else if m1.WriteHost != host1 {
		t.Errorf("Expected host %s. Actual %s", host1, m1.WriteHost)
	}

	_, err = ptrs.AcquirePointerWithHost(ctx, ptr, host2)
	if err != nil {
		t.Fatal(err)
	}

	m2, err := ptrs.PointerMetadata(ctx, ptr)
	if err != nil {
		t.Fatal(err)
	} else if m2.WriteHost != host2 {
		t.Errorf("Expected host %s. Actual %s", host2, m2.WriteHost)
	}
}

func TestSetPointerWithoutAcquire(t *testing.T) {
	ptrs := NewMemoryPointers()
	ctx := context.Background()
	ptr := data.MustNewPointerID(data.UserTestID, t.Name(), data.UserPtr)

	err := ptrs.Set(ctx, data.PointerAtSnapshot{ID: ptr, Rev: 1, SnapID: data.EmptySnapshotID})
	if err == nil || grpc.Code(err) != codes.NotFound {
		t.Errorf("Expected NotFound error. Actual: %v", err)
	}
}
