package cmds

import (
	"context"

	"github.com/windmilleng/mish/data/cmd"
)

type EmptyRunStore struct {
}

func NewEmptyRunStore() EmptyRunStore {
	return EmptyRunStore{}
}

func (s EmptyRunStore) GetRunInfo(ctx context.Context, key cmd.Key) (cmd.RunInfo, error) {
	return cmd.RunInfo{}, nil
}
