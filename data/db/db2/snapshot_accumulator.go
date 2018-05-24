package db2

import (
	"fmt"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
)

// An evaluation engine that produces a list of snapshot ids where the matched
// path has been edited by the previous recipe.
//
// The snapshot ids are returned in pre-order (newer ids first).
type snapshotAccumulator struct {
	matcher *dbpath.Matcher
}

func (d *snapshotAccumulator) empty() opEvalResult {
	return emptyList
}

// Adjust the path as we pre-visit ops.
func (d *snapshotAccumulator) visitBackwards(op data.Op, input data.SnapshotID, inputNum int) (opEvaluator, opEvalResult, error) {
	matcher, err := visitBackwardsAdjustingPath(d.matcher, op, inputNum)
	if err != nil || matcher.Empty() {
		return nil, nil, err
	}
	return &snapshotAccumulator{matcher: matcher}, nil, nil
}

func (d *snapshotAccumulator) applyOp(id data.SnapshotID, op data.Op, genericInputs []opEvalResult) (opEvalResult, error) {
	var inputIDList *snapshotIDList
	if len(genericInputs) == 1 {
		inputIDList = genericInputs[0].(*snapshotIDList)
	} else {
		inputIDList = emptyList
		for _, input := range genericInputs {
			inputIDList = inputIDList.concat(input.(*snapshotIDList))
		}
	}

	switch op := op.(type) {
	case data.FileOp:
		return d.maybeConcatIDList(d.matcher.Match(op.FilePath()), id, inputIDList), nil

	case *data.IdentityOp:
		return inputIDList, nil

	case *data.RmdirOp:
		applies := !d.matcher.Child(op.Path).Empty()
		return d.maybeConcatIDList(applies, id, inputIDList), nil

	case *data.SubdirOp:
		return d.maybeConcatIDList(true, id, inputIDList), nil

	case *data.DirOp:
		return d.maybeConcatIDList(true, id, inputIDList), nil

	case *data.OverlayOp:
		return d.maybeConcatIDList(true, id, inputIDList), nil

	case *data.FailureOp:
		return nil, fmt.Errorf("applyOp: %s", op.Message)

	default:
		return nil, fmt.Errorf("applyOp: unexpected op type %T %v", op, op)

	}
}

// A helper function for a common idiom where we prepend `id` to `inputIDList` if
// it passes the test, and return inputIDList untouched if it doesn't.
func (d *snapshotAccumulator) maybeConcatIDList(applies bool, id data.SnapshotID, inputIDList *snapshotIDList) *snapshotIDList {
	if applies {
		return prepend(id, inputIDList)
	} else {
		return inputIDList
	}
}

// A simple linked list implementation, to make it easier to append one at a time.
type snapshotIDList struct {
	first data.SnapshotID
	next  *snapshotIDList
}

var emptyList = &snapshotIDList{}

func prepend(first data.SnapshotID, next *snapshotIDList) *snapshotIDList {
	return &snapshotIDList{first: first, next: next}
}

func (l *snapshotIDList) concat(other *snapshotIDList) *snapshotIDList {
	if l == emptyList {
		return other
	}
	return prepend(l.first, l.next.concat(other))
}

func (l *snapshotIDList) toSlice() []data.SnapshotID {
	result := []data.SnapshotID{}
	current := l
	for current != emptyList {
		result = append(result, current.first)
		current = current.next
	}
	return result
}

func (l *snapshotIDList) toSet() map[data.SnapshotID]bool {
	result := map[data.SnapshotID]bool{}
	current := l
	for current != emptyList {
		result[current.first] = true
		current = current.next
	}
	return result
}

func (l *snapshotIDList) opEvalResult() {}
