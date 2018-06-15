package analytics

import (
	"sync"
	"time"
)

type CompositeEventWriter struct {
	mu      sync.Mutex
	temp    EventStore
	persist EventWriter
}

func newCompositeEventWriter(temp EventStore, persist EventWriter) *CompositeEventWriter {
	return &CompositeEventWriter{
		temp:    temp,
		persist: persist,
	}
}

func (w *CompositeEventWriter) Write(e Event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.temp.Write(e)
}

func (w *CompositeEventWriter) WriteMany(es []Event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.temp.WriteMany(es)
}

func (w *CompositeEventWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.temp.Flush(); err != nil {
		return err
	}

	es, err := w.temp.Get(time.Unix(0, 0))
	if err != nil {
		return err
	}

	w.persist.WriteMany(es)

	// TODO(dbentley): we should trim from temp, but that only matters if we're
	// doing it more than once per process

	return w.persist.Flush()
}
