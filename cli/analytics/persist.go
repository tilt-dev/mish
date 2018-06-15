package analytics

import (
	"time"
)

type ByteEvent struct {
	Key   string
	Value []byte
	T     time.Time
}

type ByteEventStore interface {
	WriteMany(es []ByteEvent)
	Get(since time.Time) ([]ByteEvent, error)
	Trim(upTo time.Time) error
	Flush() error
}
