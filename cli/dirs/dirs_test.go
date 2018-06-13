package dirs

import (
	"os"
	"path"
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
	f.assertWindmillDir(path.Join(tmpHome, ".windmill"))

	tmpWmdaemonHome := os.TempDir()
	os.Setenv("WMDAEMON_HOME", tmpWmdaemonHome)
	f.assertWindmillDir(tmpWmdaemonHome)

	nonExistentWmdaemonHome := path.Join(tmpWmdaemonHome, "foo")
	os.Setenv("WMDAEMON_HOME", nonExistentWmdaemonHome)
	f.assertWindmillDir(nonExistentWmdaemonHome)

	wmDir := os.TempDir()
	os.Setenv("WINDMILL_DIR", wmDir)
	f.assertWindmillDir(nonExistentWmdaemonHome) // prefer WMDAEMON_HOME

	os.Unsetenv("WMDAEMON_HOME")
	f.assertWindmillDir(wmDir)
}

type fixture struct {
	t *testing.T
}

func setup(t *testing.T) *fixture {
	return &fixture{t: t}
}

func (f *fixture) assertWindmillDir(expected string) {
	actual, err := GetWindmillDir()
	if err != nil {
		f.t.Error(err)
	}

	if actual != expected {
		f.t.Errorf("got windmill dir %q; expected %q", actual, expected)
	}
}
