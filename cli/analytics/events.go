package analytics

import (
	"sync"
	"time"
)

type Event struct {
	Key   string
	Value interface{}
	T     time.Time
}

type EventWriter interface {
	Write(e Event)
	WriteMany(es []Event)
	Flush() error
}

type EventStore interface {
	EventWriter
	Get(since time.Time) ([]Event, error)
	Trim(upTo time.Time) error
}

type MemoryEventStore struct {
	mu     sync.Mutex
	events []Event
}

func newMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{}
}

func (s *MemoryEventStore) Write(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
}

func (s *MemoryEventStore) WriteMany(es []Event) {
	for _, e := range es {
		s.Write(e)
	}
}

func (s *MemoryEventStore) Flush() error {
	return nil
}

func (s *MemoryEventStore) Get(since time.Time) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var r []Event
	for _, e := range s.events {
		if e.T.Before(since) {
			continue
		}
		r = append(r, e)
	}

	return r, nil
}

func (s *MemoryEventStore) Trim(upTo time.Time) error {
	s.mu.Lock()
	s.mu.Unlock()
	var n []Event
	for _, e := range s.events {
		if e.T.After(upTo) {
			n = append(n, e)
		}
	}
	s.events = n
	return nil
}
