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

type wmNotifyFixture struct {
	t      *testing.T
	tmp    *temp.TempDir
	notify wmNotify
	events []fsnotify.Event
}

func newWMNotifyFixture(t *testing.T) *wmNotifyFixture {
	notify, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}

	tmp, err := temp.NewDir(t.Name())
	if err != nil {
		t.Fatal(err)
	}

	err = notify.Add(tmp.Path())
	if err != nil {
		t.Fatal(err)
	}
	return &wmNotifyFixture{
		t:      t,
		tmp:    tmp,
		notify: notify,
	}
}

func (f *wmNotifyFixture) assertNoEvents() {
	if len(f.events) != 0 {
		f.t.Fatalf("Expected no events. Actual: %+v", f.events)
	}
}

func (f *wmNotifyFixture) fsync() {
	syncPath := filepath.Join(f.tmp.Path(), "sync.txt")
	eventsDoneCh := make(chan struct{})
	timeout := time.After(time.Second)
	go func() {
		defer close(eventsDoneCh)
		for {
			select {
			case err := <-f.notify.Errors():
				f.t.Fatal(err)

			case event := <-f.notify.Events():
				if strings.Contains(event.Name, "sync.txt") {
					return
				}
				f.events = append(f.events, event)

			case <-timeout:
				f.t.Fatal("fsync: timeout")
			}
		}
	}()

	err := ioutil.WriteFile(syncPath, []byte(fmt.Sprintf("%s", time.Now())), os.FileMode(777))
	if err != nil {
		f.t.Fatal(err)
	}
	<-eventsDoneCh
}

func (f *wmNotifyFixture) tearDown() {
	err := f.tmp.TearDown()
	if err != nil {
		f.t.Fatal(err)
	}

	err = f.notify.Close()
	if err != nil {
		f.t.Fatal(err)
	}
}
