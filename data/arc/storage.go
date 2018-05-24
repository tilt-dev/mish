package arc

import (
	"context"
	"time"

	"github.com/windmilleng/mish/data"
)

type Topic string

type Sequence int

// ArcAtSequence represents an Arc at a specific sequence number
type ArcAtSequence struct {
	Topic    Topic
	Sequence Sequence
}

// Entry is an sequenced log of bytes, intended to be used for recording "narrow" data.
type Entry struct {
	Topic     Topic
	Sequence  Sequence
	CreatedAt time.Time
	Bytes     data.Bytes
}

type Arcs interface {
	// Create a new Arc, starting at sequence 0, with a topic that begings with the prefix and is followed by a number, separated by a dash
	Create(ctx context.Context, prefix string, initial data.Bytes) (Entry, error)

	// Set a new entry in to the Arc iff it is a valid next sequence
	// (incremented sequence, increasing CreatedAt)
	Append(ctx context.Context, next Entry) error

	// Read all of the entries since a sequence number
	Read(ctx context.Context, since ArcAtSequence) ([]Entry, error)
}
