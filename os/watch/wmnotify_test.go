package watch

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/windmilleng/fsnotify"
	"github.com/windmilleng/mish/os/temp"
)

// Each implementation of the wmNotify interface should have the same basic
// behavior.

func TestNoEvents(t *testing.T) {
	f := newWMNotifyFixture(t)
	defer f.tearDown()
	f.fsync()
	f.assertNoEvents()
}

func TestEventOrdering(t *testing.T) {
	f := newWMNotifyFixture(t)
	defer f.tearDown()

	count := 10
	dirs := make([]string, count)
	for i, _ := range dirs {
		dir, err := f.root.NewDir("watched")
		if err != nil {
			t.Fatal(err)
		}
		dirs[i] = dir.Path()
		err = f.notify.Add(dir.Path())
		if err != nil {
			t.Fatal(err)
		}
	}

	f.fsync()
	f.events = nil

	for i, dir := range dirs {
		base := fmt.Sprintf("%d.txt", i)
		p := filepath.Join(dir, base)
		err := ioutil.WriteFile(p, []byte(base), os.FileMode(0777))
		if err != nil {
			t.Fatal(err)
		}
	}

	f.fsync()

	// Check to make sure that the files appeared in the right order.
	createEvents := make([]fsnotify.Event, 0, count)
	for _, e := range f.events {
		if e.Op == fsnotify.Create {
			createEvents = append(createEvents, e)
		}
	}

	if len(createEvents) != count {
		t.Fatalf("Expected %d create events. Actual: %+v", count, createEvents)
	}

	for i, event := range createEvents {
		base := fmt.Sprintf("%d.txt", i)
		p := filepath.Join(dirs[i], base)
		if event.Name != p {
			t.Fatalf("Expected event %q at %d. Actual: %+v", base, i, createEvents)
		}
	}
}

type wmNotifyFixture struct {
	t       *testing.T
	root    *temp.TempDir
	watched *temp.TempDir
	notify  wmNotify
	events  []fsnotify.Event
}

func newWMNotifyFixture(t *testing.T) *wmNotifyFixture {
	notify, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}

	root, err := temp.NewDir(t.Name())
	if err != nil {
		t.Fatal(err)
	}

	watched, err := root.NewDir("watched")
	if err != nil {
		t.Fatal(err)
	}

	err = notify.Add(watched.Path())
	if err != nil {
		t.Fatal(err)
	}
	return &wmNotifyFixture{
		t:       t,
		root:    root,
		watched: watched,
		notify:  notify,
	}
}

func (f *wmNotifyFixture) assertNoEvents() {
	if len(f.events) != 0 {
		f.t.Fatalf("Expected no events. Actual: %+v", f.events)
	}
}

func (f *wmNotifyFixture) fsync() {
	syncPath := filepath.Join(f.watched.Path(), "sync.txt")
	eventsDoneCh := make(chan error)
	timeout := time.After(time.Second)
	go func() {
		var exitErr error
		defer func() {
			eventsDoneCh <- exitErr
			close(eventsDoneCh)
		}()

		for {
			select {
			case exitErr = <-f.notify.Errors():
				return

			case event := <-f.notify.Events():
				if strings.Contains(event.Name, "sync.txt") {
					if event.Op == fsnotify.Write {
						return
					}
					continue
				}
				f.events = append(f.events, event)

			case <-timeout:
				exitErr = fmt.Errorf("fsync: timeout")
				return
			}
		}
	}()

	err := ioutil.WriteFile(syncPath, []byte(fmt.Sprintf("%s", time.Now())), os.FileMode(0777))
	if err != nil {
		f.t.Fatal(err)
	}
	err = <-eventsDoneCh
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f *wmNotifyFixture) tearDown() {
	err := f.root.TearDown()
	if err != nil {
		f.t.Fatal(err)
	}

	err = f.notify.Close()
	if err != nil {
		f.t.Fatal(err)
	}
}
