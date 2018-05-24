package gits

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/windmilleng/mish/bridge/fs"
	"github.com/windmilleng/mish/bridge/fs/fss"
	"github.com/windmilleng/mish/bridge/git"
	"github.com/windmilleng/mish/bridge/git/repo"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/storage/storages"
	"github.com/windmilleng/mish/os/temp"
)

type gitTestCase struct {
	repo   string
	branch string
}

func TestService(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	cases := []gitTestCase{
		{"repo1", "master"},
		{"repo1", "featureA"},
		{"repo2", "master"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s:%s", c.repo, c.branch), func(t *testing.T) {
			repoURL := f.val(c.repo)
			expected := f.val(fmt.Sprintf("%s_%s", c.repo, c.branch))
			id := f.resolveBranch(c.repo, c.branch, time.Hour)
			if string(id) != expected {
				t.Fatalf("%s:%s resolved to %s; expected %s", repoURL, c.branch, id, expected)
			}

			snapID, err := f.b.SnapshotCommit(f.ctx, repoURL, id)
			if err != nil {
				t.Fatal(err)
			}

			f.assertCheckoutsEqual(snapID, repoURL, string(id))
		})
	}
}

func TestTTL(t *testing.T) {
	f, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer f.tearDown()

	repoURL := f.val("repo1")
	expected := f.val("repo1_master")
	id := f.resolveBranch("repo1", "master", time.Hour)
	if string(id) != expected {
		t.Fatalf("repo1:master resolved to %s; expected %s", id, expected)
	}

	// Change master.
	clone, err := f.tmp.NewDir("clone")
	if err != nil {
		t.Fatal(err)
	}
	defer clone.TearDown()

	_, err = repo.RunGit(f.ctx, f.tmp.Path(), "clone", repoURL, clone.Path())
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.RunGit(f.ctx, clone.Path(), "commit", "--amend", "-m", "new message")
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.RunGit(f.ctx, clone.Path(), "push", "-f", "origin", "master")
	if err != nil {
		t.Fatal(err)
	}

	// Because the TTL is an hour, the expected value will be the same
	id = f.resolveBranch("repo1", "master", time.Hour)
	if string(id) != expected {
		t.Fatalf("repo1:master resolved to %s; expected %s", id, expected)
	}

	// Because the TTL is 0, we will fetch the latest.
	id = f.resolveBranch("repo1", "master", 0)
	if string(id) == expected {
		t.Fatalf("repo1:master resolved to %s; expected not equal to %s", id, expected)
	}
}

type fixture struct {
	ctx context.Context
	b   *LocalGitBridge
	fs  fs.FSBridge
	t   *testing.T
	tmp *temp.TempDir

	vals map[string]string
}

func setup(t *testing.T) (*fixture, error) {
	tmp, err := temp.NewDir("gits-service-test")
	if err != nil {
		return nil, err
	}

	// run the script
	c := exec.Command("testdata/make_repos.sh")
	c.Env = append(os.Environ(), "TMPDIR="+tmp.Path())
	out, err := c.Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			log.Printf("Stderr: %s", err.Stderr)
		} else {
			log.Printf("Err: %v", err)
		}
		return nil, err
	}

	vals := make(map[string]string)
	bs := bytes.NewBuffer(out)
	scanner := bufio.NewScanner(bs)
	for scanner.Scan() {
		l := scanner.Text()
		elems := strings.Split(l, "=")
		if len(elems) != 2 {
			return nil, fmt.Errorf("line %q does not have exactly one =", l)
		}
		vals[elems[0]] = elems[1]
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	store := storages.NewTestMemoryRecipeStore()
	ptrs := storages.NewMemoryPointers()
	db := db2.NewDB2(store, ptrs)
	opt := db2.NewOptimizer(db, store)
	fs := fss.NewLocalFSBridge(ctx, db, opt, tmp)
	bridge := NewLocalGitBridge(fs, tmp)
	return &fixture{
		ctx:  ctx,
		b:    bridge,
		fs:   fs,
		t:    t,
		tmp:  tmp,
		vals: vals,
	}, nil
}

func (f *fixture) tearDown() {
	f.tmp.TearDown()
}

func (f *fixture) resolveBranch(repo string, branch string, ttl time.Duration) git.CommitSHA {
	repoURL := f.val(repo)
	id, err := f.b.ResolveBranch(f.ctx, repoURL, branch, ttl)
	if err != nil {
		if err := err.(*exec.ExitError); err != nil {
			log.Printf("Err: %s", err.Stderr)
		}
		f.t.Fatal(err)
	}

	return id
}

func (f *fixture) val(name string) string {
	r := f.vals[name]
	if r == "" {
		f.t.Fatalf("val %s is unknown; know about %v", name, f.vals)
	}
	return r
}

func (f *fixture) assertCheckoutsEqual(snapID data.SnapshotID, repoURL string, commitID string) {
	// checkout the snapshot
	dbCO, err := f.fs.Checkout(f.ctx, snapID, "")
	if err != nil {
		f.t.Fatal(err)
	}

	// checkout from the repo
	r, err := f.b.update(f.ctx, repoURL, time.Hour)
	if err != nil {
		f.t.Fatal(err)
	}

	gitCO, err := r.NewCheckout(f.ctx, git.CommitSHA(commitID))
	if err != nil {
		f.t.Fatal(err)
	}

	// diff
	out, err := exec.Command("diff", "-r", dbCO.Path, gitCO.Path()).CombinedOutput()
	if err != nil {
		f.t.Fatalf("Err: %v %s", err, out)
	}
}
