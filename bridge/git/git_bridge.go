package git

import (
	"context"
	"time"

	"github.com/windmilleng/mish/data"
)

// a SHA identifying a commit; hex-encoded, 40 bytes long
type CommitSHA string

type GitBridge interface {
	// ResolveBranch resolves a branch in the git repo repoURL to a Commit SHA.
	// ttl: If the repo is older than this ttl, do a fetch.
	ResolveBranch(ctx context.Context, repoURL string, branch string, ttl time.Duration) (CommitSHA, error)

	// SnapshotCommit snapshots a commit into a Windmill Snapshot.
	SnapshotCommit(ctx context.Context, repoURL string, commit CommitSHA) (data.SnapshotID, error)
}
