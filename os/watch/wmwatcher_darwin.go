// +build !fsevents

package watch

import (
	"fmt"

	"github.com/windmilleng/fsnotify"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const sysctlMinimum = 49152

type darwinNotify struct {
	watcher *fsnotify.Watcher
	events  chan fsnotify.Event
	errors  chan error
}

func (d *darwinNotify) Add(name string) error {
	return d.watcher.Add(name)
}

func (d *darwinNotify) Close() error {
	return d.watcher.Close()
}

func (d *darwinNotify) Events() chan fsnotify.Event {
	return d.events
}

func (d *darwinNotify) Errors() chan error {
	return d.errors
}

func newWMWatcher() (wmNotify, error) {
	err := raiseRLimit()
	if err != nil {
		return nil, err
	}
	err = checkSysctl()
	if err != nil {
		return nil, err
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	wmw := &darwinNotify{
		watcher: fsw,
		events:  fsw.Events,
		errors:  fsw.Errors,
	}

	return wmw, nil
}

func raiseRLimit() error {
	lim := &unix.Rlimit{}
	err := unix.Getrlimit(unix.RLIMIT_NOFILE, lim)
	if err != nil {
		return err
	}

	if lim.Cur != unix.RLIM_INFINITY && lim.Cur < lim.Max {
		lim.Cur = lim.Max
		err = unix.Setrlimit(unix.RLIMIT_NOFILE, lim)
		if err != nil {
			return fmt.Errorf("failed to raise open file limit to %d, %s", lim.Max, err.Error())
		}
	}

	return nil
}

func checkSysctl() error {
	if !LimitChecksEnabled() {
		return nil
	}

	name := "kern.maxfiles"
	maxfiles, err := unix.SysctlUint32(name)
	if err != nil {
		return err
	}

	if maxfiles < sysctlMinimum {
		return fmt.Errorf(
			"%s is too low (want %d, got %d). Please raise the limit. See here for how: https://facebook.github.io/watchman/docs/install.html#mac-os-file-descriptor-limits",
			"kern.maxfiles",
			sysctlMinimum,
			maxfiles,
		)
	}

	name = "kern.maxfilesperproc"
	maxfilesperproc, err := unix.SysctlUint32(name)
	if err != nil {
		return err
	}

	if maxfilesperproc < sysctlMinimum {
		return grpc.Errorf(
			codes.ResourceExhausted,
			"%s is too low (want %d, got %d). Please raise the limit. See here for how: https://facebook.github.io/watchman/docs/install.html#mac-os-file-descriptor-limits",
			"kern.maxfilesperproc",
			sysctlMinimum,
			maxfilesperproc,
		)
	}

	return nil
}
