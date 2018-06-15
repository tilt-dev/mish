package analytics

import (
	"encoding/json"
	"sync"
	"time"
)

func NewMarshalingEventStore(bs ByteEventStore) (*MarshalingEventStore, error) {
	return &MarshalingEventStore{del: bs}, nil
}

type MarshalingEventStore struct {
	mu  sync.Mutex
	del ByteEventStore
	err error
}

func (s *MarshalingEventStore) Write(e Event) {
	s.WriteMany([]Event{e})
}

func (s *MarshalingEventStore) WriteMany(es []Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return
	}

	ebs := make([]ByteEvent, len(es))
	for i, e := range es {
		bs, err := json.Marshal(e.Value)
		if err != nil {
			s.err = err
			return
		}
		ebs[i] = ByteEvent{Key: e.Key, Value: bs, T: e.T}
	}

	s.del.WriteMany(ebs)
}

func (s *MarshalingEventStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.err
	s.err = nil
	err2 := s.del.Flush()
	if err != nil {
		return err
	}
	return err2
}

func (s *MarshalingEventStore) Get(since time.Time) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}

	evs, err := s.del.Get(since)
	if err != nil {
		return nil, err
	}

	r := make([]Event, len(evs))

	for i, ev := range evs {
		var v interface{}
		if err := json.Unmarshal(ev.Value, &v); err != nil {
			return nil, err
		}
		r[i] = Event{Key: ev.Key, Value: v, T: ev.T}
	}

	return r, nil
}

func (s *MarshalingEventStore) Trim(upTo time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	return s.del.Trim(upTo)
}
