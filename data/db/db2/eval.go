// Helper functions for evaluation.

package db2

import (
	"fmt"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
)

// Adjust the matcher as we pre-visit ops. Returns the adjusted matcher.
//
// If the returned matcher is empty, we can short-circuit: there is no possible way that visiting
// backwards could yield more changes.
func visitBackwardsAdjustingPath(m *dbpath.Matcher, op data.Op, inputNum int) (*dbpath.Matcher, error) {
	switch op := op.(type) {
	case data.FileOp:
		return m, nil

	case *data.IdentityOp:
		return m, nil

	case *data.OverlayOp:
		return m, nil

	case *data.PreserveOp:
		newM, _ := preTraversePreserveOp(m, op)
		return newM, nil

	case *data.RmdirOp:
		return m, nil

	case *data.SubdirOp:
		return m.SubdirDB(op.Path), nil

	case *data.DirOp:
		return m.ChildDB(op.Names[inputNum]), nil

	case *data.FailureOp:
		return m, nil

	default:
		return m, fmt.Errorf("visitBackwards: unexpected op type %T %v", op, op)
	}
}

// In some cases, we can apply the PreserveOp on pre-traversal by creating a new matcher
// (expressed as the Evaluator Matcher AND the PreserveOp Matcher).
//
// Returns false to indicate that we should apply the PresrveOp on post-traversal instead.
func preTraversePreserveOp(m *dbpath.Matcher, op *data.PreserveOp) (*dbpath.Matcher, bool) {
	// TODO(nick): Expand the number of cases when we can apply the preserve op on pre-traversal.
	// If we were willing to support matchers of arbitrary boolean expressions, we would always
	// be able to do this on pre-traversal, but right now we deliberately put limits on matcher
	// complexity.
	if m.All() {
		return op.Matcher, true
	}
	return m, false
}
