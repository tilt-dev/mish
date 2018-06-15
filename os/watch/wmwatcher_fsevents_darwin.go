// +build fsevents

package watch

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsevents"
	_ "github.com/windmilleng/fsevents"
	"github.com/windmilleng/fsnotify"
)

type darwinNotify struct {
	streams []fsevents.EventStream
	sm      *sync.Mutex
	events  chan fsnotify.Event
	errors  chan error
	stop    chan struct{}
}

func (d *darwinNotify) Add(name string) error {
	dev, err := fsevents.DeviceForPath(name)
	if err != nil {
		return err
	}
	es := fsevents.EventStream{
		Latency: 1 * time.Millisecond,
		Flags:   fsevents.FileEvents,
		Device:  dev,
		Paths:   []string{name},
	}

	d.sm.Lock()
	d.streams = append(d.streams, es)
	d.sm.Unlock()

	go func() {
		for {
			select {
			case <-d.stop:
				return
			case x := <-es.Events:
				for _, y := range x {
					y.Path = filepath.Join("/", y.Path)
					op := eventFlagsToOp(y.Flags)

					// ignore events that say the watched directory
					// has been created. these are fired spuriously
					// on initiation.
					if name == y.Path && op == fsnotify.Create {
						continue
					}

					d.events <- fsnotify.Event{
						Name: y.Path,
						Op:   op,
					}
				}
			}
		}
	}()

	es.Start()

	return nil
}

func (d *darwinNotify) Close() error {
	for _, es := range d.streams {
		es.Stop()
	}
	close(d.errors)
	close(d.stop)

	return nil
}

func (d *darwinNotify) Events() chan fsnotify.Event {
	return d.events
}

func (d *darwinNotify) Errors() chan error {
	return d.errors
}

func newWMWatcher() (wmNotify, error) {
	dw := &darwinNotify{
		streams: []fsevents.EventStream{},
		sm:      &sync.Mutex{},
		events:  make(chan fsnotify.Event),
		errors:  make(chan error),
		stop:    make(chan struct{}),
	}

	return dw, nil
}

func eventFlagsToOp(flags fsevents.EventFlags) fsnotify.Op {
	if flags&fsevents.ItemCreated != 0 {
		return fsnotify.Create
	}
	if flags&fsevents.ItemRemoved != 0 {
		return fsnotify.Remove
	}
	if flags&fsevents.ItemRenamed != 0 {
		return fsnotify.Rename
	}
	if flags&fsevents.ItemChangeOwner != 0 {
		return fsnotify.Chmod
	}

	return fsnotify.Write
}
