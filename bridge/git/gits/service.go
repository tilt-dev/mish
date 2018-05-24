package gits

import (
	"context"
	"sync"
	"time"

	"github.com/windmilleng/mish/bridge/fs"
	"github.com/windmilleng/mish/bridge/git"
	"github.com/windmilleng/mish/bridge/git/repo"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/os/ospath"
	"github.com/windmilleng/mish/os/temp"
)

type LocalGitBridge struct {
	fs  fs.FSBridge   // FSBridge to read from/write to
	tmp *temp.TempDir // where to store temp files

	mu    sync.Mutex            // control access to repos
	repos map[string]*repo.Repo // map of repoURL to a (bare) repo

	// TODO(dbentley): remove this hack
	// HACK(dbentley): if we've already Snapshot'ed this commit (with the same parameters),
	// we can just return the previous SnapshotID.
	// We should probably do this, but in a more principled way.
	// After the demo.
	lastRepoURL    string
	lastCommit     git.CommitSHA
	lastSnapshotID data.SnapshotID
}

func NewLocalGitBridge(fs fs.FSBridge, tmp *temp.TempDir) *LocalGitBridge {
	return &LocalGitBridge{
		fs:    fs,
		repos: make(map[string]*repo.Repo),
		tmp:   tmp,
	}
}

func (b *LocalGitBridge) ResolveBranch(ctx context.Context, repoURL string, branch string, ttl time.Duration) (git.CommitSHA, error) {
	// TODO(nick): Narrow down the use of this mutex so that we don't lock
	// while doing checkouts.
	b.mu.Lock()
	defer b.mu.Unlock()

	r, err := b.update(ctx, repoURL, ttl)
	if err != nil {
		return "", err
	}

	sha, err := r.ParseCommit(ctx, branch)
	if err != nil {
		return "", err
	}

	return git.CommitSHA(sha), nil
}

func (b *LocalGitBridge) SnapshotCommit(ctx context.Context, repoURL string, commit git.CommitSHA) (data.SnapshotID, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if repoURL == b.lastRepoURL &&
		commit == b.lastCommit {
		return b.lastSnapshotID, nil
	}

	// Assume that if we have a commit SHA, the bare repo is up to date, so just
	// use 1000 hours as a dummy ttl.
	repo, err := b.update(ctx, repoURL, time.Hour*1000)
	if err != nil {
		return data.SnapshotID{}, err
	}

	co, err := repo.NewCheckout(ctx, commit)
	if err != nil {
		return data.SnapshotID{}, err
	}

	defer co.TearDown()

	id, err := b.fs.SnapshotDir(ctx, co.Path(), ospath.NewAllMatcher(), data.PublicID, data.RecipeWTagOptimal, fs.Hint{})
	if err != nil {
		return data.SnapshotID{}, err
	}

	b.lastRepoURL = repoURL
	b.lastCommit = commit
	b.lastSnapshotID = id

	return id, nil
}

// update returns an up-to-date repo for repoURL, either cloning or fetching
func (b *LocalGitBridge) update(ctx context.Context, repoURL string, ttl time.Duration) (*repo.Repo, error) {
	// TODO(dbentley): thread context through so we stop fetching if context is done
	existingRepo := b.repos[repoURL]
	if existingRepo == nil {
		r, err := repo.CloneRepo(ctx, repoURL, b.tmp)
		if err != nil {
			return nil, err
		}
		b.repos[repoURL] = r
		return r, nil
	}

	err := existingRepo.RefreshIfNecessary(ctx, ttl)
	if err != nil {
		return nil, err
	}

	return existingRepo, nil
}

var _ git.GitBridge = &LocalGitBridge{}
