package storages

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/storage"
)

type MemoryRecipeStore struct {
	mu   sync.RWMutex
	ider IDer

	recipes map[data.SnapshotID][]data.TaggedRecipe
}

func NewTestMemoryRecipeStore() *MemoryRecipeStore {
	return NewMemoryRecipeStore(NewContentIDer(NewIncrementingIDer(data.NewTestSnapshotPrefix())))
}

func NewMemoryRecipeStore(ider IDer) *MemoryRecipeStore {
	recipes := make(map[data.SnapshotID][]data.TaggedRecipe)

	// EmptySnapshotID is the alpha and the omega. It can be neither created nor destroyed.
	recipes[data.EmptySnapshotID] = []data.TaggedRecipe{
		{Tag: data.RecipeWTagOptimal, Recipe: data.Recipe{Op: &data.DirOp{}}},
	}

	return &MemoryRecipeStore{
		recipes: recipes,
		ider:    ider,
	}
}

func (s *MemoryRecipeStore) AllPathsToSnapshot(ctx context.Context, snapID data.SnapshotID) ([]data.StoredRecipe, error) {
	recipes, err := s.TaggedPathsToSnapshot(ctx, snapID)
	if err != nil {
		return nil, err
	}
	storedRecipes := make([]data.StoredRecipe, len(recipes))
	for i, r := range recipes {
		storedRecipes[i] = data.StoredRecipe{Snap: snapID, Tag: r.Tag, Recipe: r.Recipe}
	}
	return storedRecipes, nil
}

// Returns all the tagged recipes that meet a snapshot.
func (s *MemoryRecipeStore) TaggedPathsToSnapshot(ctx context.Context, id data.SnapshotID) ([]data.TaggedRecipe, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recipes, ok := s.recipes[id]
	if !ok {
		return nil, grpc.Errorf(codes.NotFound, "Snapshot not found %q", id)
	}
	return append([]data.TaggedRecipe{}, recipes...), nil
}

func (s *MemoryRecipeStore) LookupPathToSnapshot(ctx context.Context, tag data.RecipeRTag, snapID data.SnapshotID) (data.StoredRecipe, error) {
	taggedRecipes, err := s.TaggedPathsToSnapshot(ctx, snapID)
	if err != nil {
		return data.StoredRecipe{}, err
	}

	best := taggedRecipes[0]
	for _, tr := range taggedRecipes {
		if tr.Tag.Type == tag.Type {
			if tr.Tag.ID == tag.ID {
				return data.StoredRecipe{Snap: snapID, Recipe: tr.Recipe, Tag: tr.Tag}, nil
			} else {
				best = tr
			}
		}
	}

	return data.StoredRecipe{Snap: snapID, Recipe: best.Recipe, Tag: best.Tag}, nil
}

func (s *MemoryRecipeStore) Create(ctxt context.Context, r data.Recipe, owner data.UserID, t data.RecipeWTag) (data.SnapshotID, bool, error) {
	// Only temp recipes can be written without an owner.
	if owner == 0 && t != data.RecipeWTagTemp {
		return data.SnapshotID{}, false, fmt.Errorf("Create: Invalid owner: %d", owner)
	}

	id := s.ider.NewID(owner, r)

	s.mu.Lock()
	defer s.mu.Unlock()

	rs := s.recipes[id]
	if id.IsContentID() && rs != nil {
		isOnlyRecipeTemp := len(rs) == 1 && rs[0].Tag.Type == data.RecipeTagTypeTemp
		canReuse := !isOnlyRecipeTemp || t.Type == data.RecipeTagTypeTemp
		if canReuse {
			// By definition, the only way to reach a content ID is with optimal recipes.
			return id, false, nil
		}
	}

	if t.Type == data.RecipeTagTypeOptimal && !id.IsContentID() {
		return data.SnapshotID{}, false, fmt.Errorf("Non-optimal recipe cannot be written with an optimal tag: %v", r)
	}

	s.recipes[id] = append(rs, data.TaggedRecipe{Tag: t, Recipe: r})

	return id, true, nil
}

func (s *MemoryRecipeStore) CreatePath(ctx context.Context, r data.StoredRecipe) error {
	return s.createOrReplicatePath(ctx, r, true)
}

func (s *MemoryRecipeStore) createOrReplicatePath(ctx context.Context, r data.StoredRecipe, mustExist bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Snap.IsTempID() {
		return data.TempWriteError{ID: r.Snap}
	}

	t := r.Tag

	recipes := s.recipes[r.Snap]

	for _, tr := range recipes {
		if tr.Tag == t {
			equal := tr.Op.OpEqual(r.Op) && data.SnapshotIDsEqual(r.Inputs, tr.Inputs)
			if equal {
				// Because hub overfetches, sometimes we get duplicate recipes. This is ok.
				return nil
			} else {
				// Something really bad has happened.
				return fmt.Errorf("Recipe already exists for snapshot %q with tag %+v", r.Snap, t)
			}
		}
	}

	if r.Tag.Type == data.RecipeTagTypeOptimal {
		// Optimal recipes must lead to an optimal snapshot ID or be an identity op.
		_, isIdentityOp := r.Recipe.Op.(*data.IdentityOp)
		isContentID := r.Snap.IsContentID()
		if !(isIdentityOp || isContentID) {
			return fmt.Errorf("Non-optimal recipe cannot be written with an optimal tag: %v", r.Recipe)
		}
	}

	if mustExist && len(recipes) == 0 {
		return grpc.Errorf(codes.NotFound, "CreatePath: Snapshot does not exist: %s", r.Snap)
	}

	s.recipes[r.Snap] = append(s.recipes[r.Snap], data.TaggedRecipe{Recipe: r.Recipe, Tag: r.Tag})
	return nil
}

func (s *MemoryRecipeStore) Replicate(ctx context.Context, r data.StoredRecipe) error {
	return s.createOrReplicatePath(ctx, r, false)
}

func (s *MemoryRecipeStore) WriteQueueSize(ctx context.Context) (data.WriteQueueSize, error) {
	return data.WriteQueueSize{}, nil
}

func (s *MemoryRecipeStore) HasSnapshots(ctx context.Context, snapIDs []data.SnapshotID) ([]data.SnapshotID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]data.SnapshotID, 0, len(snapIDs))
	for _, id := range snapIDs {
		if len(s.recipes[id]) > 0 {
			result = append(result, id)
		}
	}
	return result, nil
}

var _ storage.RecipeStore = &MemoryRecipeStore{}
