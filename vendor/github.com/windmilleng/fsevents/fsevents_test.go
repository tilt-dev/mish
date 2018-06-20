// +build darwin

package fsevents

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBasicExamplePerDevice(t *testing.T) {
	f := newPerDeviceFixture(t)
	defer f.tearDown()
	RunBasicExampleTest(t, f)
}

func TestBasicExamplePerHost(t *testing.T) {
	f := newPerHostFixture(t)
	defer f.tearDown()
	RunBasicExampleTest(t, f)
}

func RunBasicExampleTest(t *testing.T, f fixture) {
	es := f.es
	es.Start()

	wait := f.startEventLoop()

	err := ioutil.WriteFile(filepath.Join(f.path, "example.txt"), []byte("example"), 0700)
	if err != nil {
		t.Fatal(err)
	}

	ev1, err := nextEvent(wait)
	if err != nil {
		t.Fatal(err)
	}

	if (ev1.Flags&ItemIsDir) != ItemIsDir ||
		(ev1.Flags&ItemCreated) != ItemCreated {
		t.Errorf("Expected dir create event. Actual: %+v", ev1)
	}

	ev2, err := nextEvent(wait)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(ev2.Path, "example.txt") ||
		(ev2.Flags&ItemIsFile) != ItemIsFile ||
		(ev2.Flags&ItemCreated) != ItemCreated {
		t.Errorf("Expected file create event. Actual: %+v", ev2)
	}
}

func TestCreateFileBeforeStart(t *testing.T) {
	f := newPerDeviceFixture(t)
	defer f.tearDown()

	err := ioutil.WriteFile(filepath.Join(f.path, "example.txt"), []byte("example"), 0700)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)
	es := f.es
	es.Start()

	wait := f.startEventLoop()

	err = ioutil.WriteFile(filepath.Join(f.path, "example2.txt"), []byte("example"), 0700)
	if err != nil {
		t.Fatal(err)
	}

	ev1, err := nextEvent(wait)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(ev1.Path, "example2.txt") ||
		(ev1.Flags&ItemIsFile) != ItemIsFile ||
		(ev1.Flags&ItemCreated) != ItemCreated {
		t.Errorf("Expected file create event. Actual: %+v", ev1)
	}
}
func TestRestart(t *testing.T) {
	f := newPerDeviceFixture(t)
	defer f.tearDown()

	err := ioutil.WriteFile(filepath.Join(f.path, "example.txt"), []byte("example"), 0700)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	es := f.es
	es.Start()
	es.Restart()

	wait := f.startEventLoop()

	err = ioutil.WriteFile(filepath.Join(f.path, "example2.txt"), []byte("example"), 0700)
	if err != nil {
		t.Fatal(err)
	}

	ev1, err := nextEvent(wait)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(ev1.Path, "example2.txt") ||
		(ev1.Flags&ItemIsFile) != ItemIsFile ||
		(ev1.Flags&ItemCreated) != ItemCreated {
		t.Errorf("Expected file create event. Actual: %+v", ev1)
	}
}

type fixture struct {
	path string
	es   *EventStream
	t    *testing.T
}

func newPerHostFixture(t *testing.T) fixture {
	path, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatal(err)
	}

	es := &EventStream{
		Paths:   []string{path},
		Latency: 500 * time.Millisecond,
		Flags:   FileEvents,
	}
	return fixture{
		t:    t,
		path: path,
		es:   es,
	}
}

func newPerDeviceFixture(t *testing.T) fixture {
	path, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatal(err)
	}

	dev, err := DeviceForPath(path)
	if err != nil {
		t.Fatal(err)
	}

	es := &EventStream{
		Paths:   []string{path},
		Latency: 500 * time.Millisecond,
		Device:  dev,
		Flags:   FileEvents,
	}
	return fixture{
		t:    t,
		path: path,
		es:   es,
	}
}

func (f fixture) startEventLoop() chan Event {
	wait := make(chan Event, 100)
	go func() {
		for msg := range f.es.Events {
			for _, event := range msg {
				f.t.Logf("Event: %#v", event)
				wait <- event
			}
		}
	}()
	return wait
}

func (f fixture) tearDown() {
	f.es.Stop()
	os.RemoveAll(f.path)
}

func nextEvent(wait chan Event) (Event, error) {
	select {
	case ev, ok := <-wait:
		if !ok {
			return Event{}, fmt.Errorf("channel closed")
		}
		return ev, nil
	case <-time.After(time.Second):
		return Event{}, fmt.Errorf("timed out waiting for event")
	}
}
