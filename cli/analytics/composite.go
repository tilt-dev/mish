package analytics

import (
	"sync"
	"time"
)

type CompositeEventStore struct {
	mu      sync.Mutex
	temp    EventStore
	persist EventStore
}

func newCompositeEventStore(temp EventStore, persist EventStore) *CompositeEventStore {
	return &CompositeEventStore{
		temp:    temp,
		persist: persist,
	}
}

func (s *CompositeEventStore) Write(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.temp.Write(e)
}

func (s *CompositeEventStore) WriteMany(es []Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.temp.WriteMany(es)
}

func (s *CompositeEventStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.temp.Flush(); err != nil {
		return err
	}

	es, err := s.temp.Get(time.Unix(0, 0))
	if err != nil {
		return err
	}

	s.persist.WriteMany(es)

	// TODO(dbentley): we should trim from temp, but that only matters if we're
	// doing it more than once per process

	return s.persist.Flush()
}

func (s *CompositeEventStore) Trim(upTo time.Time) error {
	if err := s.temp.Trim(upTo); err != nil {
		return err
	}

	// TODO(dbentley): what are the right semantics for composite trim when one fails?

	return s.persist.Trim(upTo)
}

func (s *CompositeEventStore) Get(since time.Time) ([]Event, error) {
	r1, err := s.temp.Get(since)
	if err != nil {
		return nil, err
	}

	r2, err := s.persist.Get(since)
	if err != nil {
		return nil, err
	}

	var r []Event
	r = append(r, r1...)
	r = append(r, r2...)

	return r, nil
}
