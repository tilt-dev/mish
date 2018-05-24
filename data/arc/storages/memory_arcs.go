package storages

import (
	"context"
	"fmt"
	"sync"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/arc"
)

type MemoryArcs struct {
	mu sync.RWMutex
	// NOTE(dmiller) @nicks mentioned that there might be some problems with this approach. If there are lots of conflicts with
	// these autoincrementing numbers we'll have to switch to a random ID generation similar to what pointers does.
	nextTopicID int
	data        map[arc.Topic][]arc.Entry
}

func NewMemoryArcs() *MemoryArcs {
	return &MemoryArcs{
		data:        make(map[arc.Topic][]arc.Entry),
		nextTopicID: 0,
	}
}

func (m *MemoryArcs) Create(c context.Context, prefix string, initial data.Bytes) (arc.Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	candidate := arc.Topic(fmt.Sprintf("%s-%d", prefix, m.nextTopicID))
	if _, ok := m.data[candidate]; !ok {
		entry := arc.Entry{Topic: candidate, Bytes: initial}
		m.nextTopicID++
		m.data[candidate] = append(m.data[candidate], entry)
		return entry, nil
	}

	return arc.Entry{}, fmt.Errorf("Unexpected topic ID. Expected %s-%d to be available, but it was already taken", prefix, m.nextTopicID)
}

func (m *MemoryArcs) Append(ctx context.Context, next arc.Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.data[next.Topic]; !ok {
		return fmt.Errorf("Topic %s does not exist", next.Topic)
	}

	entries := m.data[next.Topic]
	lastEntry := entries[len(entries)-1]
	expectedSequence := int(lastEntry.Sequence) + 1
	if int(next.Sequence) != expectedSequence {
		return fmt.Errorf("Expected next sequence number to be %d, got %d", expectedSequence, next.Sequence)
	}
	m.data[next.Topic] = append(m.data[next.Topic], next)

	return nil
}

func (m *MemoryArcs) Read(ctx context.Context, since arc.ArcAtSequence) ([]arc.Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.data[since.Topic]; !ok {
		return []arc.Entry{}, fmt.Errorf("Topic %s does not exist", since.Topic)
	}

	entries := m.data[since.Topic]
	for i, e := range entries {
		if e.Sequence == since.Sequence {
			return entries[i+1:], nil
		}
	}

	return []arc.Entry{}, fmt.Errorf("Given sequence (%d) does not exist in topic %s", since.Sequence, since.Topic)
}
