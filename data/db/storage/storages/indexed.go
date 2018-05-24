package storages

import (
	"context"
	"crypto/sha256"
	"sync"

	"github.com/golang/protobuf/proto"

	"github.com/windmilleng/mish/data"
	dbProto "github.com/windmilleng/mish/data/db/proto"
	"github.com/windmilleng/mish/data/db/storage"
)

type MemoryRecipeIndex struct {
	mu       sync.Mutex
	byRecipe map[string]data.StoredRecipe
}

func NewMemoryRecipeIndex() *MemoryRecipeIndex {
	return &MemoryRecipeIndex{
		byRecipe: make(map[string]data.StoredRecipe),
	}
}

func (i *MemoryRecipeIndex) LookupRecipe(ctx context.Context, r data.Recipe) (data.StoredRecipe, error) {
	key, err := hashRecipe(r)
	if err != nil {
		return data.StoredRecipe{}, err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	return i.byRecipe[key], nil
}

func (i *MemoryRecipeIndex) Replicate(ctx context.Context, sr data.StoredRecipe) error {
	key, err := hashRecipe(sr.Recipe)
	if err != nil {
		return err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if _, ok := i.byRecipe[key]; !ok {
		i.byRecipe[key] = sr
	}

	return nil
}

func hashRecipe(r data.Recipe) (string, error) {
	p, err := dbProto.RecipeD2P(r)
	if err != nil {
		return "", err
	}

	bytes, err := proto.Marshal(p)
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	if _, err = hash.Write(bytes); err != nil {
		return "", err
	}

	bytes = hash.Sum(nil)

	return string(bytes), nil
}

// RecipeIndex that, when a Recipe is looked up, replicates it.
// This is needed for Mill recycling. It needs to know which StoredRecipe's to write to this execlog.
// TODO(dbentley) We should probably change Creator and DB2 to have Create return the Recipes it needed to write? Or something?
type ReplicateOnLookupRecipeIndex struct {
	idx  storage.RecipeIndex
	sink storage.RecipeSink
}

func NewReplicateOnLookupRecipeIndex(idx storage.RecipeIndex, sink storage.RecipeSink) *ReplicateOnLookupRecipeIndex {
	return &ReplicateOnLookupRecipeIndex{idx: idx, sink: sink}
}

func (i *ReplicateOnLookupRecipeIndex) LookupRecipe(ctx context.Context, r data.Recipe) (data.StoredRecipe, error) {
	sr, err := i.idx.LookupRecipe(ctx, r)
	if err != nil {
		return data.StoredRecipe{}, err
	}

	if !sr.Snap.Nil() {
		if err := i.sink.Replicate(ctx, sr); err != nil {
			return data.StoredRecipe{}, err
		}
	}

	return sr, nil
}
