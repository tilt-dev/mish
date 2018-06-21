package analytics_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/windmilleng/mish/cli/analytics"
	"github.com/windmilleng/mish/os/temp"
)

func TestOpt(t *testing.T) {
	oldWindmillDir := os.Getenv("WINDMILL_DIR")
	defer os.Setenv("WINDMILL_DIR", oldWindmillDir)
	tmpdir, err := temp.NewDir("TestOpt")
	if err != nil {
		t.Fatalf("Error making temp dir: %v", err)
	}
	fmt.Println(tmpdir)

	f := setup(t)

	os.Setenv("WINDMILL_DIR", tmpdir.Path())

	f.assertOptStatus("default")

	analytics.SetOpt("opt-in")
	f.assertOptStatus("opt-in")

	analytics.SetOpt("opt-out")
	f.assertOptStatus("opt-out")

	analytics.SetOpt("foo")
	f.assertOptStatus("default")
}

type fixture struct {
	t *testing.T
}

func setup(t *testing.T) *fixture {
	return &fixture{t: t}
}

func (f *fixture) assertOptStatus(expected string) {
	actual, err := analytics.OptStatus()
	if err != nil {
		f.t.Fatal(err)
	}
	if actual != expected {
		f.t.Errorf("got opt status %s, expected %s", actual, expected)
	}
}
