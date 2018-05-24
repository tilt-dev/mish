package watch

import (
	"fmt"
	"testing"
	"time"

	"github.com/windmilleng/fsnotify"
)

type expectedCoalesce struct {
	e1 fsnotify.Event
	e2 fsnotify.Event

	coalesce bool
	expected fsnotify.Event
}

func TestCoalesce(t *testing.T) {
	expects := []expectedCoalesce{
		{e("foo.txt", fsnotify.Create), e("foo.txt", fsnotify.Create),
			true, e("foo.txt", fsnotify.Create)},
		{e("foo.txt", fsnotify.Create), e("foo.txt", fsnotify.Write),
			true, e("foo.txt", fsnotify.Create)},
		{e("foo.txt", fsnotify.Create), e("foo.txt", fsnotify.Remove),
			true, e("foo.txt", fsnotify.Create)},
		{e("foo.txt", fsnotify.Create), e("foo.txt", fsnotify.Rename),
			true, e("foo.txt", fsnotify.Create)},
		{e("foo.txt", fsnotify.Create), e("foo.txt", fsnotify.Chmod),
			true, e("foo.txt", fsnotify.Create)},
		{e("foo.txt", fsnotify.Write), e("foo.txt", fsnotify.Create),
			true, e("foo.txt", fsnotify.Create)},
		{e("foo.txt", fsnotify.Write), e("foo.txt", fsnotify.Write),
			true, e("foo.txt", fsnotify.Write)},
		{e("foo.txt", fsnotify.Write), e("foo.txt", fsnotify.Remove),
			true, e("foo.txt", fsnotify.Write)},
		{e("foo.txt", fsnotify.Write), e("foo.txt", fsnotify.Rename),
			true, e("foo.txt", fsnotify.Write)},
		{e("foo.txt", fsnotify.Write), e("foo.txt", fsnotify.Chmod),
			true, e("foo.txt", fsnotify.Write)},
		{e("foo.txt", fsnotify.Create), e("bar.txt", fsnotify.Create),
			false, fsnotify.Event{}},
		{e("foo.txt", fsnotify.Write), e("bar.txt", fsnotify.Write),
			false, fsnotify.Event{}},
	}

	for _, expect := range expects {
		t.Run(fmt.Sprintf("%v %v", expect.e1, expect.e2), func(t *testing.T) {
			actual, ok := coalesce(simplify(expect.e1), simplify(expect.e2))

			if ok != expect.coalesce {
				t.Fatalf("coalesce did not match expectation: %v %v %v %v", expect.e1, expect.e2, ok, expect.coalesce)
			}

			if ok {
				if actual.Name != expect.expected.Name || actual.Op != expect.expected.Op {
					t.Errorf("bad coalesce: %v != %v", actual, expect.expected)
				}
			}

		})
	}
}

type expectedCoalesceChannel struct {
	in  []*fsnotify.Event
	out []fsnotify.Event
}

func TestCoalesceChannel(t *testing.T) {
	expects := []expectedCoalesceChannel{
		{
			in: []*fsnotify.Event{
				ep("foo.txt", fsnotify.Create),
			},
			out: []fsnotify.Event{
				e("foo.txt", fsnotify.Create),
			},
		},
		{
			in: []*fsnotify.Event{
				ep("foo.txt", fsnotify.Create),
				ep("foo.txt", fsnotify.Remove),
			},
			out: []fsnotify.Event{
				e("foo.txt", fsnotify.Create),
			},
		},
		{
			in: []*fsnotify.Event{
				ep("foo.txt", fsnotify.Create),
				nil,
				ep("foo.txt", fsnotify.Remove),
			},
			out: []fsnotify.Event{
				e("foo.txt", fsnotify.Create),
				e("foo.txt", fsnotify.Write),
			},
		},
		{
			in: []*fsnotify.Event{
				ep("foo.txt", fsnotify.Create),
				ep("bar.txt", fsnotify.Create),
				ep("bar.txt", fsnotify.Create),
			},
			out: []fsnotify.Event{
				e("foo.txt", fsnotify.Create),
				e("bar.txt", fsnotify.Create),
			},
		},
		{
			in: []*fsnotify.Event{
				ep("foo.txt", fsnotify.Create),
				nil,
				ep("foo.txt", fsnotify.Create),
				ep("bar.txt", fsnotify.Create),
				ep("bar.txt", fsnotify.Create),
			},
			out: []fsnotify.Event{
				e("foo.txt", fsnotify.Create),
				e("foo.txt", fsnotify.Create),
				e("bar.txt", fsnotify.Create),
			},
		},
		{
			in: []*fsnotify.Event{
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Remove),
			},
			out: []fsnotify.Event{
				e("foo.txt", fsnotify.Write),
			},
		},
		{
			in: []*fsnotify.Event{
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
			},
			out: []fsnotify.Event{
				e("foo.txt", fsnotify.Write),
			},
		},
		{
			in: []*fsnotify.Event{
				ep("foo.txt", fsnotify.Remove),
				ep("foo.txt", fsnotify.Remove),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
			},
			out: []fsnotify.Event{
				e("foo.txt", fsnotify.Write),
			},
		},
		{
			in: []*fsnotify.Event{
				ep("foo.txt", fsnotify.Write),
				ep("bar.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
			},
			out: []fsnotify.Event{
				e("foo.txt", fsnotify.Write),
				e("bar.txt", fsnotify.Write),
				e("foo.txt", fsnotify.Write),
			},
		},
		{
			in: []*fsnotify.Event{
				// test the coalesce cap
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write),
				ep("foo.txt", fsnotify.Write), // 10
				ep("foo.txt", fsnotify.Write),
			},
			out: []fsnotify.Event{
				e("foo.txt", fsnotify.Write),
				e("foo.txt", fsnotify.Write),
			},
		},
	}

	for i, expect := range expects {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			inCh := make(chan fsnotify.Event)
			outCh := make(chan fsnotify.Event)

			doneCh := make(chan struct{})
			actual := []fsnotify.Event(nil)

			go func() {
				for e := range outCh {
					actual = append(actual, e)
				}
				close(doneCh)
			}()
			go coalesceEvents(inCh, outCh)

			for _, e := range expect.in {
				if e == nil {
					time.Sleep(5 * time.Millisecond)
				} else {
					inCh <- *e
				}
			}
			close(inCh)
			<-doneCh
			if len(actual) != len(expect.out) {
				t.Fatalf("got wrong number of events: %d vs %d (%v %v)", len(actual), len(expect.out), actual, expect.out)
			}
			for i, a := range actual {
				ex := expect.out[i]
				if a.Name != ex.Name || a.Op != ex.Op {
					t.Errorf("bad coalesce channel: %v %v", a, ex)
				}
			}
		})
	}

}

func e(p string, op fsnotify.Op) fsnotify.Event {
	return fsnotify.Event{Name: p, Op: op}
}

func ep(p string, op fsnotify.Op) *fsnotify.Event {
	return &fsnotify.Event{Name: p, Op: op}
}
