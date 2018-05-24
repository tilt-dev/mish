package watch

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/windmilleng/fsnotify"
	"github.com/windmilleng/mish/os/sysctl"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const enospc = "no space left on device"
const inotifyErrMsg = "The user limit on the total number of inotify watches was reached; increase the fs.inotify.max_user_watches sysctl. See here for more information: https://facebook.github.io/watchman/docs/install.html#linux-inotify-limits"
const inotifyMin = 8192

type linuxNotify struct {
	watcher *fsnotify.Watcher
	events  chan fsnotify.Event
	errors  chan error
}

func (d *linuxNotify) Add(name string) error {
	err := d.watcher.Add(name)
	if err != nil {
		if strings.Contains(err.Error(), enospc) {
			return fmt.Errorf("error watching path %s. %s (%s)", name, inotifyErrMsg, err.Error())
		}

		return err
	}

	return nil
}

func (d *linuxNotify) Close() error {
	return d.watcher.Close()
}

func (d *linuxNotify) Events() chan fsnotify.Event {
	return d.events
}

func (d *linuxNotify) Errors() chan error {
	return d.errors
}

func newWMWatcher() (wmNotify, error) {
	err := checkInotifyLimits()
	if err != nil {
		return nil, err
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	wmw := &linuxNotify{
		watcher: fsw,
		events:  fsw.Events,
		errors:  fsw.Errors,
	}

	return wmw, nil
}

func checkInotifyLimits() error {
	if !LimitChecksEnabled() {
		return nil
	}
	uw, err := sysctl.Get("fs.inotify.max_user_watches")
	if err != nil {
		return err
	}
	i, err := strconv.Atoi(uw)
	if err != nil {
		return err
	}

	if i < inotifyMin {
		return grpc.Errorf(
			codes.ResourceExhausted,
			"The user limit on the total number of inotify watches is too low (%d); increase the fs.inotify.max_user_watches sysctl. See here for more information: https://facebook.github.io/watchman/docs/install.html#linux-inotify-limits",
			i,
		)
	}

	return nil
}
