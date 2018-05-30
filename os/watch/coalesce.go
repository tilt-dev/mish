package watch

import (
	"time"

	"github.com/windmilleng/fsnotify"
)

// We put a cap on how many sequential events to coalesce for use-cases like log
// files where we see continuous data for a long time.
const MAX_COALESCE = 10

func coalesceEvents(inCh chan fsnotify.Event, outCh chan fsnotify.Event) {
	defer func() {
		close(outCh)
	}()

	var event fsnotify.Event
	var nextEvent fsnotify.Event
	var ok bool
	coalesceCount := 1
	var timerCh <-chan time.Time // timerCh is non-nil iff there is a pending event to send out

	for {
		select {
		case nextEvent, ok = <-inCh:
			if !ok {
				if timerCh != nil {
					outCh <- event
				}
				return
			}
			nextEvent = simplify(nextEvent)

			if timerCh != nil {
				merged := false
				if coalesceCount < MAX_COALESCE {
					event, merged = coalesce(event, nextEvent)
				}

				// If we were able to coalesce, try to coalesce against the next event.
				// Otherwise, emit the event.
				if merged {
					coalesceCount++
				} else {
					outCh <- event
					event = nextEvent
					coalesceCount = 1
				}
			} else {
				event = nextEvent
				timerCh = time.After(time.Millisecond)
				coalesceCount = 1
			}
		case <-timerCh:
			outCh <- event
			timerCh = nil
			coalesceCount = 1
		}
	}
}

// Narrow all events to Create or Write.
//
// We don't need fine-grained events because we're just going to compare
// what we have in the snapshot against what we have on disk anyway.
//
// We only need Create for special handling when a directory is created.
// We could probably get rid of Create too if we had a better API for querying
// a snapshot by directory.
func simplify(e fsnotify.Event) fsnotify.Event {
	if e.Op&fsnotify.Create != 0 {
		return fsnotify.Event{Name: e.Name, Op: fsnotify.Create}
	} else {
		return fsnotify.Event{Name: e.Name, Op: fsnotify.Write}
	}
}

// All events passed into coalesce should be simplified.
func coalesce(old, new fsnotify.Event) (fsnotify.Event, bool) {
	if old.Name != new.Name {
		return old, false
	}

	if new.Op == fsnotify.Create {
		// A create of a Dir means we have to scan the whole dir, so make sure
		// we don't drop that.
		return new, true
	}

	return old, true
}
