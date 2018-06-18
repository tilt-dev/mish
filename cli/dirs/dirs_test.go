package dirs

import (
	"os"
	"path"
	"path/filepath"
	"testing"
)

func TestWindmillDir(t *testing.T) {
	emptyPath := ""
	oldWmdaemonHome := os.Getenv("WMDAEMON_HOME")
	oldHome := os.Getenv("HOME")
	oldWindmillDir := os.Getenv("WINDMILL_DIR")
	defer os.Setenv("WMDAEMON_HOME", oldWmdaemonHome)
	defer os.Setenv("HOME", oldHome)
	defer os.Setenv("WINDMILL_DIR", oldWindmillDir)
	tmpHome := os.TempDir()

	f := setup(t)

	os.Setenv("HOME", tmpHome)

	os.Setenv("WMDAEMON_HOME", emptyPath)
	f.assertWindmillDir(path.Join(tmpHome, ".windmill"), "empty WMDAEMON_HOME")

	tmpWmdaemonHome := os.TempDir()
	os.Setenv("WMDAEMON_HOME", tmpWmdaemonHome)
	f.assertWindmillDir(tmpWmdaemonHome, "tmp WMDAEMON_HOME")

	nonExistentWmdaemonHome := path.Join(tmpWmdaemonHome, "foo")
	os.Setenv("WMDAEMON_HOME", nonExistentWmdaemonHome)
	f.assertWindmillDir(nonExistentWmdaemonHome, "nonexistent WMDAEMON_HOME")

	wmDir := os.TempDir()
	os.Setenv("WINDMILL_DIR", wmDir)
	f.assertWindmillDir(nonExistentWmdaemonHome, "prefer WMDAEMON_HOME") // prefer WMDAEMON_HOME

	os.Unsetenv("WMDAEMON_HOME")
	f.assertWindmillDir(wmDir, "no WMDAEMON_HOME")
}

type fixture struct {
	t *testing.T
}

func setup(t *testing.T) *fixture {
	return &fixture{t: t}
}

func (f *fixture) assertWindmillDir(expected, testCase string) {
	actual, err := GetWindmillDir()
	if err != nil {
		f.t.Error(err)
	}

	// NOTE(maia): filepath behavior is weird on macOS, use abs path to mitigate
	absExpected, err := filepath.Abs(expected)
	if err != nil {
		f.t.Error("[filepath.Abs]", err)
	}

	if actual != absExpected {
		f.t.Errorf("[TEST CASE: %s] got windmill dir %q; expected %q", testCase, actual, absExpected)
	}
}
