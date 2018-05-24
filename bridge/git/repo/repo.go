package repo

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/windmilleng/mish/bridge/git"
	"github.com/windmilleng/mish/logging"
	"github.com/windmilleng/mish/os/env"
	"github.com/windmilleng/mish/os/temp"
)

type Repo struct {
	dir           string
	repoURL       string
	tmp           *temp.TempDir
	lastUpdate    time.Time
	mu            sync.Mutex
	parsedCommits map[string]git.CommitSHA
}

func CloneRepo(ctx context.Context, repoURL string, tmp *temp.TempDir) (*Repo, error) {
	d, err := tmp.NewDir("repo")
	if err != nil {
		return nil, err
	}
	c := exec.CommandContext(ctx, "git", "clone", "--bare", repoURL, d.Path())
	if err := c.Run(); err != nil {
		return nil, err
	}
	return &Repo{
		dir:           d.Path(),
		tmp:           tmp,
		repoURL:       repoURL,
		lastUpdate:    time.Now(),
		parsedCommits: make(map[string]git.CommitSHA),
	}, nil
}

func (r *Repo) run(ctx context.Context, args ...string) (string, error) {
	return RunGit(ctx, r.dir, args...)
}

func (r *Repo) runSHA(ctx context.Context, args ...string) (string, error) {
	s, err := r.run(ctx, args...)
	if err != nil {
		return "", err
	}

	return ValidateSHA(s)
}

func ValidateSHA(s string) (string, error) {
	if !(len(s) == 40 || len(s) == 41 && s[40] == '\n') {
		return "", fmt.Errorf("sha not 40 or 41 (with a \\n) characters: %q", s)
	}

	_, err := hex.DecodeString(s[0:40])
	if err != nil {
		return "", fmt.Errorf("sha not valid hex: %v", err)
	}

	return s[0:40], nil
}

func (r *Repo) ParseCommit(ctx context.Context, commitish string) (git.CommitSHA, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result, ok := r.parsedCommits[commitish]
	if ok {
		return result, nil
	}

	s, err := r.runSHA(ctx, "rev-parse", fmt.Sprintf("%s^{commit}", commitish))
	if err != nil {
		return "", err
	}

	result = git.CommitSHA(s)
	r.parsedCommits[commitish] = result
	return result, nil
}

func (r *Repo) NewCheckout(ctx context.Context, treeish git.CommitSHA) (Checkout, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, err := r.tmp.NewDir("co")
	if err != nil {
		return Checkout{}, err
	}

	if _, err := r.run(ctx, "worktree", "add", "-f", d.Path(), string(treeish)); err != nil {
		return Checkout{}, err
	}

	if err := os.Remove(path.Join(d.Path(), ".git")); err != nil {
		return Checkout{}, err
	}

	return Checkout{tmp: d}, nil
}

func (r *Repo) RefreshIfNecessary(ctx context.Context, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if time.Since(r.lastUpdate) < ttl {
		return nil
	}

	_, err := r.run(ctx, "fetch", r.repoURL)
	if err != nil {
		return err
	}

	head, err := r.runSHA(ctx, "rev-parse", "FETCH_HEAD")
	if err != nil {
		return err
	}

	_, err = r.run(ctx, "update-ref", "HEAD", string(head))
	if err != nil {
		return err
	}

	r.lastUpdate = time.Now()
	r.parsedCommits = make(map[string]git.CommitSHA)
	return err
}

type Checkout struct {
	tmp *temp.TempDir
}

func (c Checkout) Path() string {
	return c.tmp.Path()
}

func (c Checkout) TearDown() error {
	return c.tmp.TearDown()
}

func RunGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	data, err := cmd.Output()
	if err != nil {
		if env.IsDebug() {
			if err := err.(*exec.ExitError); err != nil {
				logging.With(ctx).Errorf("Failure: %s: %v %s", dir, args, err.Stderr)
			}
		}
		return "", err
	}

	return string(data), err
}
