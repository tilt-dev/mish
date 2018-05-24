package db2

import (
	"context"
	"fmt"
	"path"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
	"github.com/windmilleng/mish/tracing"
)

func (db *DB2) PathsChanged(ctx context.Context, start, end data.SnapshotID, tag data.RecipeRTag, matcher *dbpath.Matcher) ([]string, error) {
	span, ctx := tracing.StartSystemSpanFromContext(ctx, "db2/PathsChanged", opentracing.Tags{
		"patterns": strings.Join(matcher.ToPatterns(), ","),
		"start":    start,
		"end":      end,
		string(ext.Component): "db2",
	})
	defer span.Finish()

	// We have two algorithms for finding paths changed.
	//
	// 1) Traverse the pointer history and accumulates all changes until it reaches the end. It's
	// very fast for common cases where there have been a small number of changes. But if it hits a complex
	// diff history or a content ID, it thinks every file has changed.
	//
	// 2) Playback both snapshots completely and find the diffs. This is slower most of the time,
	// but doesn't have the terrible worst-case condition where it returns overbroad data.
	useContentComparison := tag.Type != data.RecipeTagTypeEdit || start.IsContentID() || end.IsContentID()
	if !useContentComparison {
		result, err := db.eval(ctx, tag, end, &fileAccumulator{stop: start, matcher: matcher})
		if err != nil {
			return nil, err
		}

		filesChanged := result.(fileChangedSet)

		// If we didn't hit the stop ID, that means we have all paths. This isn't quite what we want.
		// Fallback to the diffing case.
		if filesChanged.hitStopSnapID {
			return filesChanged.toSlice(), nil
		}
	}

	resultStart, err := db.eval(ctx, data.RecipeRTagOptimal, start, &snapshotEvaluator{matcher: matcher})
	if err != nil {
		return nil, err
	}

	resultEnd, err := db.eval(ctx, data.RecipeRTagOptimal, end, &snapshotEvaluator{matcher: matcher})
	if err != nil {
		return nil, err
	}

	return diffPaths(resultStart.(snapshotVal), resultEnd.(snapshotVal)), nil
}

// An evaluation engine that accumulates the files that have changed
// since a fixed snapshot ID.
type fileAccumulator struct {
	matcher *dbpath.Matcher
	stop    data.SnapshotID
}

func (a *fileAccumulator) empty() opEvalResult {
	return newFileChangedSet()
}

// Adjust the path as we pre-visit ops.
func (a *fileAccumulator) visitBackwards(op data.Op, input data.SnapshotID, inputNum int) (opEvaluator, opEvalResult, error) {
	if input == a.stop {
		r := newFileChangedSet()
		r.hitStopSnapID = true
		return nil, r, nil
	}
	matcher, err := visitBackwardsAdjustingPath(a.matcher, op, inputNum)
	if err != nil || matcher.Empty() {
		return nil, nil, err
	}
	return &fileAccumulator{matcher: matcher, stop: a.stop}, nil, nil
}

func (a *fileAccumulator) applyOp(id data.SnapshotID, op data.Op, genericInputs []opEvalResult) (opEvalResult, error) {
	var result fileChangedSet = newFileChangedSet()
	if len(genericInputs) == 1 {
		result = genericInputs[0].(fileChangedSet)
	}

	switch op := op.(type) {
	case data.FileOp:
		if a.matcher.Match(op.FilePath()) {
			result.set[op.FilePath()] = true
		}
		return result, nil

	case *data.IdentityOp:
		return result, nil

	case *data.RmdirOp:
		// Per the contract of FilesChanged in dbint, we generally don't
		// look at rmdirs when determining files changed.
		return result, nil

	case *data.SubdirOp:
		subdirResult := newFileChangedSet()
		subdirResult.hitStopSnapID = result.hitStopSnapID
		for k, _ := range result.set {
			newPath, ok := dbpath.Child(op.Path, k)
			if ok {
				subdirResult.set[newPath] = true
			}
		}
		return subdirResult, nil

	case *data.DirOp:
		result := newFileChangedSet()
		for i, input := range genericInputs {
			set := input.(fileChangedSet)
			for p, _ := range set.set {
				result.set[path.Join(op.Names[i], p)] = true
			}
			result.hitStopSnapID = result.hitStopSnapID || set.hitStopSnapID
		}
		return result, nil

	case *data.OverlayOp:
		result := newFileChangedSet()
		for _, input := range genericInputs {
			set := input.(fileChangedSet)
			for p, _ := range set.set {
				result.set[p] = true
			}
			result.hitStopSnapID = result.hitStopSnapID || set.hitStopSnapID
		}
		return result, nil

	case *data.FailureOp:
		return nil, fmt.Errorf("applyOp: %s", op.Message)

	default:
		return nil, fmt.Errorf("applyOp: unexpected op type %T %v", op, op)

	}
}

// A map of files that have changed.
type fileChangedSet struct {
	hitStopSnapID bool
	set           map[string]bool
}

func newFileChangedSet() fileChangedSet {
	return fileChangedSet{set: make(map[string]bool)}
}

func (s fileChangedSet) addAll(other fileChangedSet) {
	for k, _ := range other.set {
		s.set[k] = true
	}
}

func (s fileChangedSet) opEvalResult() {}

func (s fileChangedSet) toSlice() []string {
	result := make([]string, 0, len(s.set))
	for k, _ := range s.set {
		result = append(result, k)
	}
	return result
}
