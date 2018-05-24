package db2

import (
	"context"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/dbpath"
)

func (db *DB2) Cost(ctx context.Context, snapID data.SnapshotID, tag data.RecipeRTag, matcher *dbpath.Matcher) (dbint.SnapshotCost, error) {
	result, err := db.eval(ctx, tag, snapID, &costEvaluator{matcher: matcher})
	if err != nil {
		return dbint.SnapshotCost{}, err
	}

	return result.(costAccumulator).SnapshotCost, nil
}

type costEvaluator struct {
	matcher *dbpath.Matcher
}

func (e *costEvaluator) empty() opEvalResult {
	return costAccumulator{}
}

func (e *costEvaluator) visitBackwards(op data.Op, input data.SnapshotID, inputNum int) (opEvaluator, opEvalResult, error) {
	matcher, err := visitBackwardsAdjustingPath(e.matcher, op, inputNum)
	if err != nil || matcher.Empty() {
		return nil, nil, err
	}
	return &costEvaluator{matcher: matcher}, nil, nil
}

func (e *costEvaluator) applyOp(outputID data.SnapshotID, op data.Op, inputs []opEvalResult) (opEvalResult, error) {
	result := costAccumulator{}
	for _, i := range inputs {
		cost := i.(costAccumulator).SnapshotCost
		result.SnapshotCost.Ops += cost.Ops
		result.SnapshotCost.NonOptimizedOps += cost.NonOptimizedOps
	}
	result.SnapshotCost.Ops += 1
	if !outputID.IsContentID() {
		result.SnapshotCost.NonOptimizedOps += 1
	}
	return result, nil
}

type costAccumulator struct {
	dbint.SnapshotCost
}

func (a costAccumulator) opEvalResult() {}
