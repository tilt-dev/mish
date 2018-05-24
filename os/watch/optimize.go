package watch

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/windmilleng/go-diff/diffmatchpatch"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/dbpath"
)

var differ = diffmatchpatch.New()

type optimizer struct {
	db    dbint.DB2
	head  data.SnapshotID
	owner data.UserID
	tagFn RecipeTagProvider
}

type RecipeTagProvider func() data.RecipeWTag

func NewRecipeTagProvider(tag data.RecipeWTag) RecipeTagProvider {
	return func() data.RecipeWTag { return tag }
}

func newOptimizer(db dbint.DB2, head data.SnapshotID, owner data.UserID, tagFn RecipeTagProvider) *optimizer {
	return &optimizer{db: db, head: head, owner: owner, tagFn: tagFn}
}

func (o *optimizer) updateToEvent(new map[string]*trackedFile, deleteMissing bool) (WatchOpsEvent, error) {
	keys := make([]string, 0, len(new))
	for k := range new {
		keys = append(keys, k)
	}

	var ops []data.Op

	sort.Strings(keys)
	keyAncestors := ancestors(keys)

	// We're all right with using context.Background() here because this should always
	// be a pure local DB, but we don't have a type for it.
	ctx := context.Background()

	matcher := dbpath.NewAllMatcher()
	if len(keys) < 20 && !deleteMissing {
		var err error

		// Use a pattern matcher that matches subdirs.
		patterns := make([]string, len(keys)+len(keyAncestors))
		for i, k := range keys {
			patterns[i] = path.Join(k, "**")
		}
		for i, k := range keyAncestors {
			patterns[len(keys)+i] = k
		}
		matcher, err = dbpath.NewMatcherFromPatterns(patterns)
		if err != nil {
			return WatchOpsEvent{}, err
		}
	}

	oldS, err := o.db.SnapshotFilesMatching(ctx, o.head, matcher)
	oldMap := oldS.AsMap()
	if err != nil {
		return WatchOpsEvent{}, err
	}

	// Check to see if any files are being deleted and replaced with directories.
	for _, k := range keyAncestors {
		oldF := oldMap[k]
		if oldF != nil {
			ops = append(ops, &data.RemoveFileOp{Path: oldF.Path})
		}
	}

	for _, k := range keys {
		oldF := oldMap[k]
		newF := new[k]
		newOps, err := o.toOps(oldF, newF)
		if err != nil {
			return WatchOpsEvent{}, err
		}
		ops = append(ops, newOps...)

		// When deleting a directory recursively, sometimes inotify doesn't get
		// REMOVE events for the inner files, so we have to infer them.
		if newF == nil && oldF == nil {
			// Something is being deleted but it's not a file!
			// Check if there are any files under it.
			for oldK, oldF := range oldMap {
				if oldF == nil {
					continue
				}

				_, isChild := dbpath.Child(k, oldK)
				if isChild {
					ops = append(ops, &data.RemoveFileOp{Path: oldF.Path})
				}
			}
		}
	}

	if deleteMissing {
		for k, _ := range oldMap {
			if _, ok := new[k]; !ok {
				// it was present in previous, but not in newFiles
				ops = append(ops, &data.RemoveFileOp{Path: k})
			}
		}
	}

	tag := o.tagFn()
	newHead, err := dbint.CreateLinear(ctx, o.db, ops, o.head, o.owner, tag)
	if err != nil {
		return WatchOpsEvent{}, err
	}
	o.head = newHead
	return WatchOpsEvent{SnapID: newHead, Ops: ops}, nil
}

func (o *optimizer) toOps(old *dbint.SnapshotFile, new *trackedFile) ([]data.Op, error) {
	if new == nil {
		if old == nil {
			// double delete; skip
			return nil, nil
		}
		return []data.Op{&data.RemoveFileOp{Path: old.Path}}, nil
	}

	if new.ignore {
		// If we've been told to ignore the new fle, don't compare it to the previous
		// file contents.
		return nil, nil
	}

	newF, err := new.toSnapshotFile()

	// Handle the race condition where the file has been deleted since we first
	// scanned it.
	if os.IsNotExist(err) {
		if old == nil {
			// double delete; skip
			return nil, nil
		}
		return []data.Op{&data.RemoveFileOp{Path: old.Path}}, nil
	} else if err == symlinkOutsideWorkspaceErr {
		// Sadly, it's common for links to point outside the workspace.
		// A common example is IDE temp files (like Emacs scratch files).
		// For now, we silently ignore these symlinks. In the future,
		// we may need to more intelligently distinguish between
		// symlinks we can silently ignore and symlinks we should error about.
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	writeOp := &data.WriteFileOp{
		Path:       newF.Path,
		Data:       newF.Contents,
		Executable: newF.Executable,
		Type:       newF.Type,
	}

	if old != nil {
		diffOps := o.diff(old, newF)

		if o.scoreOps(diffOps) < o.scoreOp(writeOp) {
			return diffOps, nil
		}
	}

	return []data.Op{writeOp}, nil
}

func (o *optimizer) diff(old *dbint.SnapshotFile, new *dbint.SnapshotFile) []data.Op {
	diffs := differ.DiffMainBytes(old.Contents.InternalByteSlice(), new.Contents.InternalByteSlice())
	ops := []data.Op{}

	editOp := o.diffsToEditOp(diffs, new.Path)
	if len(editOp.Splices) > 0 {
		ops = append(ops, editOp)
	}

	if new.Executable != old.Executable {
		ops = append(ops, &data.ChmodFileOp{
			Path:       new.Path,
			Executable: new.Executable,
		})
	}

	return ops
}

func (o *optimizer) diffsToEditOp(diffs []diffmatchpatch.Diff, relPath string) *data.EditFileOp {
	op := &data.EditFileOp{Path: relPath, Splices: []data.EditFileSplice{}}
	index := 0
	for _, diff := range diffs {
		b := []byte(diff.Text)
		if diff.Type == diffmatchpatch.DiffDelete {
			op.Splices = append(op.Splices, &data.DeleteBytesSplice{
				Index:       int64(index),
				DeleteCount: int64(len(b)),
			})
		} else if diff.Type == diffmatchpatch.DiffInsert {
			op.Splices = append(op.Splices, &data.InsertBytesSplice{
				Index: int64(index),
				Data:  data.NewBytesWithBacking(b),
			})
			index += len(b)
		} else if diff.Type == diffmatchpatch.DiffEqual {
			index += len(b)
		}
	}

	return op
}

// Generate a heuristic score for a list of ops, for when we have two equivalent
// lists and need to decide which to use.
func (o *optimizer) scoreOps(ops []data.Op) int {
	cost := 0
	for _, op := range ops {
		cost += o.scoreOp(op)
	}
	return cost
}

// Generate a heuristic score for an op, helper for scoreops
func (o *optimizer) scoreOp(op data.Op) int {
	cost := 10 // baseline
	switch op := op.(type) {
	case (*data.WriteFileOp):
		cost += op.Data.Len()
	case (*data.InsertBytesFileOp):
		cost += op.Data.Len()
	case (*data.EditFileOp):
		for _, s := range op.Splices {
			cost += 1
			if insert, ok := s.(*data.InsertBytesSplice); ok {
				cost += insert.Data.Len()
			}
		}
	}
	return cost
}

// Given a list of paths, generate a list of all parents of those paths that aren't
// included in the path list. For example,
// Inputs: ["a/b/c.txt"] -> Outputs: ["a", "a/b"]
// Inputs: ["a/b/c.txt", "a"] -> Outputs: ["a/b"]
func ancestors(paths []string) []string {
	pathSet := dbint.PathsToPathSet(paths)
	outSet := make(map[string]bool)
	for _, p := range paths {
		current := filepath.Dir(p)
		for current != "." && current != "" {
			if !pathSet[current] {
				outSet[current] = true
			}
			current = filepath.Dir(current)
		}
	}
	return dbint.PathSetToPaths(outSet)
}
