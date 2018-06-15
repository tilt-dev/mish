package analytics

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

func newBoltByteEventStore(filename string) (*BoltByteEventStore, error) {
	return &BoltByteEventStore{filename: filename}, nil
}

type BoltByteEventStore struct {
	filename string

	mu  sync.Mutex
	err error
}

// A note on how we store events:
// we store them in one bucket ("events") keyed by time. So it's just one big stream of events.
// this looks weird, because events have keys, but those are keys for aggregation, not for storage.

func (s *BoltByteEventStore) open() (*bolt.DB, error) {
	if s.err != nil {
		return nil, s.err
	}
	return bolt.Open(s.filename, 0600, nil)
}

func (s *BoltByteEventStore) Write(e ByteEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.open()
	if err != nil {
		s.err = err
		return
	}

	s.err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(eventBucket)
		if err != nil {
			return err
		}

		kBs, err := timeBs(e.T)
		if err != nil {
			return err
		}

		vBs, err := json.Marshal(e)
		if err != nil {
			return err
		}

		if err := b.Put(kBs, vBs); err != nil {
			return err
		}
		return nil
	})
}

func (s *BoltByteEventStore) WriteMany(es []ByteEvent) {
	for _, e := range es {
		s.Write(e)
	}
}

func (s *BoltByteEventStore) Get(since time.Time) ([]ByteEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	since = since.UTC()
	// TODO(dbentley): actually filter by since
	var r []ByteEvent
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(eventBucket)
		if b == nil {
			log.Printf("no bucket")
			return nil
		}

		c := b.Cursor()
		for k, v := c.Seek([]byte{0}); k != nil; k, v = c.Next() {
			var e rawByteEvent
			if err := json.Unmarshal(v, &e); err != nil {
				return err
			}
			t, err := bsTime(k)
			if err != nil {
				return err
			}
			r = append(r, ByteEvent{Key: e.Key, Value: e.Value, T: t})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *BoltByteEventStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.err
	s.err = nil
	return err
}

func (s *BoltByteEventStore) Trim(upTo time.Time) error {
	return fmt.Errorf("BoltByteEventStore.Trim: not yet implemented")
}

func timeBs(t time.Time) ([]byte, error) {
	// TODO(dbentley): t.MarshalBinary is not that great; we should used rfc3339 for sortable time encoding
	return t.MarshalBinary()
}

func bsTime(bs []byte) (time.Time, error) {
	var r time.Time
	err := r.UnmarshalBinary(bs)
	return r, err
}

var eventBucket = []byte("events")

type rawByteEvent struct {
	Key   string
	Value []byte
}
