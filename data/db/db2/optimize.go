package db2

import (
	"context"
	"fmt"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/dbpath"
	"github.com/windmilleng/mish/data/db/storage"
	"github.com/windmilleng/mish/errors"
	"github.com/windmilleng/mish/logging"
)

type Optimizer struct {
	db     *DB2
	syncer storage.RecipeWriter
}

func NewOptimizer(db *DB2, syncer storage.RecipeWriter) *Optimizer {
	return &Optimizer{db: db, syncer: syncer}
}

func (o *Optimizer) IsOptimized(ctx context.Context, snapID data.SnapshotID) (data.SnapshotID, error) {
	// If this is already a content id, it doesn't need to be optimized.
	if snapID.IsContentID() {
		return snapID, nil
	}

	// A snapshot has already been optimized if there's an identity op recipe
	// pointing to a content id.
	storedRecipe, err := o.db.LookupPathToSnapshot(ctx, data.RecipeRTagOptimal, snapID)
	if err != nil {
		return data.SnapshotID{}, err
	}

	return o.isPathToOptimizedSnapshot(storedRecipe.Recipe), nil
}

func (o *Optimizer) isPathToOptimizedSnapshot(recipe data.Recipe) data.SnapshotID {
	if _, ok := recipe.Op.(*data.IdentityOp); ok {
		if len(recipe.Inputs) == 1 && recipe.Inputs[0].IsContentID() {
			return recipe.Inputs[0]
		}
	}
	return data.SnapshotID{}
}

func (o *Optimizer) OptimizeSnapshot(ctx context.Context, snapID data.SnapshotID) (data.SnapshotID, error) {
	optimizedSnapID, err := o.IsOptimized(ctx, snapID)
	if err != nil || !optimizedSnapID.Nil() {
		return optimizedSnapID, err
	}

	optimizedSnapID, err = o.optimizeSnapshotFastPath(ctx, snapID)
	if err != nil {
		// Log the error and continue down the slow path.
		logging.With(ctx).Errorf("OptimizeSnapshot: %v", err)
	} else if !optimizedSnapID.Nil() {
		return optimizedSnapID, nil
	}

	return o.optimizeSnapshotSlowPath(ctx, snapID)
}

const maxFastPathSearch = 500

// The fast path traverses backwards through the recipes until it reaches an optimized
// snapshot ID. It figured out exactly what paths changed, and recomputes the hashes
// for those paths.
//
// It does the same amount of re-hashing as the slow path, but it does not need to load
// the entire repo into memory.
//
// Returns an empty string if we were not able to optimize this way.
func (o *Optimizer) optimizeSnapshotFastPath(ctx context.Context, snapID data.SnapshotID) (data.SnapshotID, error) {
	numOps := 0
	pathsChanged := make(map[string]bool)
	current := snapID
	closestOptimalSnapID := data.SnapshotID{}
	for numOps < maxFastPathSearch {
		recipe, err := o.db.LookupPathToSnapshot(ctx, data.RecipeRTagOptimal, current)
		if err != nil {
			return data.SnapshotID{}, err
		}

		closestOptimalSnapID = o.isPathToOptimizedSnapshot(recipe.Recipe)
		if !closestOptimalSnapID.Nil() {
			break
		}

		// TODO(nick): Also handle DirOps.
		fileOp, ok := recipe.Recipe.Op.(data.FileOp)
		if !ok || !isLinearOp(fileOp) {
			return data.SnapshotID{}, nil
		}

		if len(recipe.Inputs) > 1 {
			return data.SnapshotID{}, nil
		}

		pathsChanged[fileOp.FilePath()] = true
		numOps++
		if len(recipe.Inputs) == 1 {
			current = recipe.Inputs[0]
		} else {
			closestOptimalSnapID = data.EmptySnapshotID
			break
		}
	}

	if closestOptimalSnapID.Nil() {
		return data.SnapshotID{}, nil
	}

	// Playback all the files that changed.
	matcher, err := dbpath.NewFilesMatcher(dbint.PathSetToPaths(pathsChanged))
	if err != nil {
		return data.SnapshotID{}, err
	}

	filesChanged, err := o.db.eval(ctx, data.RecipeRTagOptimal, snapID, &snapshotEvaluator{matcher: matcher, stripContents: false})
	if err != nil {
		return data.SnapshotID{}, err
	}

	filesChangedDir, ok := filesChanged.(*snapshotDir)
	if !ok {
		return data.SnapshotID{}, err
	}

	fileMap := make(map[string]*snapshotFile, len(pathsChanged))
	for path, _ := range pathsChanged {
		// TODO(nick): Handle the case where the file was removed.
		file, err := lookupFile(filesChangedDir, path, onNotFoundDoNothing)
		if err != nil {
			return data.SnapshotID{}, errors.Propagatef(err, "OptimizeFastPath")
		}

		fileMap[path] = file
	}

	optimizedSnapID, err := o.reoptimizeDirWithFilesChanged(ctx, closestOptimalSnapID, fileMap, snapID.Owner())
	if err != nil {
		return data.SnapshotID{}, err
	}

	return o.link(ctx, snapID, optimizedSnapID)
}

func (o *Optimizer) reoptimizeDirWithFilesChanged(ctx context.Context, optimizedSnapID data.SnapshotID, fileMap map[string]*snapshotFile, owner data.UserID) (data.SnapshotID, error) {
	dirRecipe, err := o.db.LookupPathToSnapshot(ctx, data.RecipeRTagOptimal, optimizedSnapID)
	if err != nil {
		return data.SnapshotID{}, err
	}

	dirOp, ok := dirRecipe.Recipe.Op.(*data.DirOp)
	if !ok {
		return data.SnapshotID{}, nil
	}

	nameIndex := make(map[string]int, len(dirOp.Names))
	for i, name := range dirOp.Names {
		nameIndex[name] = i
	}
	oldInputs := dirRecipe.Recipe.Inputs

	shallowFileMap := make(map[string]*snapshotFile)
	deepFileMap := make(map[string]map[string]*snapshotFile)
	for path, file := range fileMap {
		first, rest := dbpath.Split(path)

		_, ok := nameIndex[first]
		if !ok {
			nameIndex[first] = -1
		}

		if rest == "" {
			shallowFileMap[first] = file
		} else {
			innerMap := deepFileMap[first]
			if innerMap == nil {
				innerMap = make(map[string]*snapshotFile)
				deepFileMap[first] = innerMap
			}
			innerMap[rest] = file
		}
	}

	da := dbint.NewDirAssembler(ctx, o.db, owner, data.RecipeWTagOptimal)
	for name, inputIndex := range nameIndex {
		// Case 1) We're replacing a file directly in this directory.
		shallowFile, ok := shallowFileMap[name]
		if ok {
			if shallowFile != nil {
				da.Write(name, shallowFile.data, shallowFile.executable, shallowFile.fileType)
			}
			continue
		}

		// Case 2) We're replacing a tree of files in this directory.
		deepFiles, ok := deepFileMap[name]
		if ok {
			deepSnapID := data.EmptySnapshotID
			if inputIndex != -1 {
				deepSnapID = oldInputs[inputIndex]
			}
			reoptimizedSnapID, err := o.reoptimizeDirWithFilesChanged(ctx, deepSnapID, deepFiles, owner)
			if err != nil {
				return data.SnapshotID{}, err
			}

			// If this got optimized to an empty tree, we can skip it!
			// We don't want phantom trees in our directory structure.
			if reoptimizedSnapID != data.EmptySnapshotID {
				da.WriteSnapshot(name, reoptimizedSnapID)
			}
			continue
		}

		// Case 3) We're reusing the old tree.
		if inputIndex == -1 {
			return data.SnapshotID{}, fmt.Errorf("Optimize error: could not find old tree id")
		}
		da.WriteSnapshot(name, oldInputs[inputIndex])
	}

	return da.Commit()
}

// The slow path plays back the entire snapshot, and loads it into memory.
//
// The playback mechanism keeps track of which directories need to be re-hashed.
//
// After playback, we recurse through the tree, recomputing only the things that had changes
// since the last optimize.
func (o *Optimizer) optimizeSnapshotSlowPath(ctx context.Context, snapID data.SnapshotID) (data.SnapshotID, error) {
	snapshot, err := o.db.evalSnapshotAtPath(ctx, snapID, dbpath.NewAllMatcher(), false)
	if err != nil {
		return data.SnapshotID{}, err
	}

	optimizedSnapID, err := o.optimizeDir(ctx, snapshot.d, snapID.Owner())
	if err != nil {
		return data.SnapshotID{}, err
	}

	return o.link(ctx, snapID, optimizedSnapID)
}

func (o *Optimizer) link(ctx context.Context, snapID data.SnapshotID, optimizedSnapID data.SnapshotID) (data.SnapshotID, error) {
	recipe := data.Recipe{
		Op:     &data.IdentityOp{},
		Inputs: []data.SnapshotID{optimizedSnapID},
	}
	err := o.syncer.CreatePath(ctx, data.StoredRecipe{Snap: snapID, Recipe: recipe, Tag: data.RecipeWTagOptimal})
	return optimizedSnapID, err
}

func (o *Optimizer) optimizeDir(ctx context.Context, dir *snapshotDir, owner data.UserID) (data.SnapshotID, error) {
	// The best part about this optimized format is that every intermediate snapshot
	// is also written in optimal form, which is also expressed as a content-addressable ID.
	da := dbint.NewDirAssembler(ctx, o.db, owner, data.RecipeWTagOptimal)
	for k, v := range dir.files {
		contentID := v.contentID()
		if !contentID.Nil() {
			da.WriteSnapshot(k, contentID)
			continue
		}

		switch v := v.(type) {
		case *snapshotFile:
			da.Write(k, v.data, v.executable, v.fileType)
		case *snapshotDir:
			snapID, err := o.optimizeDir(ctx, v, owner)
			if err != nil {
				return data.SnapshotID{}, err
			}
			da.WriteSnapshot(k, snapID)
		default:
			return data.SnapshotID{}, fmt.Errorf("Unrecognized type %T", v)
		}
	}
	return da.Commit()
}
