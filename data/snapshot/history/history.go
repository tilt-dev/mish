package history

import (
	"context"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
)

type SnapshotHistoryService struct {
	db dbint.DB2
}

func NewSnapshotHistoryService(db dbint.DB2) *SnapshotHistoryService {
	return &SnapshotHistoryService{db: db}
}

type PointerHistoryQuery struct {
	MaxRev      data.PointerRev
	MinRev      data.PointerRev
	MaxRevCount int64
}

// Returns a list of pointer revisions based on the given query.
func (h *SnapshotHistoryService) GetPointerHistory(ctx context.Context, ptrID data.PointerID, query PointerHistoryQuery) ([]data.PointerAtRev, error) {
	head, err := h.db.Head(ctx, ptrID)
	if err != nil {
		return nil, err
	}

	maxRevCount := query.MaxRevCount
	if maxRevCount == 0 {
		maxRevCount = 10
	}

	maxRev := query.MaxRev
	if query.MinRev != 0 && maxRev == 0 {
		maxRev = data.PointerRev(int64(query.MinRev) + maxRevCount - 1)
	}

	return data.NewPointerHistoryIter(ptrID, head.Rev, data.PointerHistoryIterParams{
		MaxRev: maxRev,
		MinRev: query.MinRev,
	}).Take(int(maxRevCount))
}

// Given a sequence of snapshot ids, find the op directly before each snapshot. Order-preserving.
func (s *SnapshotHistoryService) LookupOps(ctx context.Context, snapIDs []data.SnapshotID) ([]data.Op, error) {
	type result struct {
		err error
		op  data.Op
	}
	resultChans := make([]chan result, 0, len(snapIDs))
	for _, snapID := range snapIDs {
		c := make(chan result)
		go func(snapID data.SnapshotID, c chan result) {
			defer close(c)

			recipe, err := s.db.LookupPathToSnapshot(ctx, data.RecipeRTagEdit, snapID)
			if err != nil {
				c <- result{err: err}
				return
			}

			c <- result{op: recipe.Recipe.Op}
		}(snapID, c)
		resultChans = append(resultChans, c)
	}

	opList := make([]data.Op, 0, len(snapIDs))
	var err error = nil
	for _, c := range resultChans {
		r := <-c
		if r.err == nil {
			opList = append(opList, r.op)
		} else {
			err = r.err
		}
	}

	if err != nil {
		return nil, err
	}

	return opList, nil
}
