package storages

import (
	"context"
	"testing"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/arc"
)

func TestCreate(t *testing.T) {
	m := NewMemoryArcs()
	ctx := context.Background()

	a, err := m.Create(ctx, "hello", data.BytesFromString("tilters"))
	if err != nil {
		t.Fatal(err)
	}

	if a.Bytes.Equal(data.BytesFromString("tilters")) != true || a.Topic != "hello-0" || a.Sequence != 0 {
		t.Errorf("expected an empty initial arc entry to be returned from create, got %+v", a)
	}

	a2, err := m.Create(ctx, "hello", data.BytesFromString("millers"))

	if a2.Topic != "hello-1" {
		t.Errorf("expected the next topic name to be hello-1, got %s", a2.Topic)
	}
}

func TestAppend(t *testing.T) {
	m := NewMemoryArcs()
	ctx := context.Background()

	_, err := m.Create(ctx, "hello", data.BytesFromString("tilters"))
	if err != nil {
		t.Fatal(err)
	}

	err = m.Append(ctx, arc.Entry{Bytes: data.BytesFromString("millers"), Topic: "hello-0", Sequence: 1})

	if err != nil {
		t.Fatal(err)
	}

	err = m.Append(ctx, arc.Entry{Bytes: data.BytesFromString("boo"), Topic: "bad", Sequence: 1})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	err = m.Append(ctx, arc.Entry{Bytes: data.BytesFromString("boo"), Topic: "hello-0", Sequence: 1})
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestRead(t *testing.T) {
	m := NewMemoryArcs()
	ctx := context.Background()
	_, err := m.Create(ctx, "hello", data.BytesFromString("tilters"))
	if err != nil {
		t.Fatal(err)
	}

	err = m.Append(ctx, arc.Entry{Bytes: data.BytesFromString("millers"), Topic: "hello-0", Sequence: 1})
	if err != nil {
		t.Fatal(err)
	}
	err = m.Append(ctx, arc.Entry{Bytes: data.BytesFromString("fillers"), Topic: "hello-0", Sequence: 2})
	if err != nil {
		t.Fatal(err)
	}

	since := arc.ArcAtSequence{
		Topic:    "hello-0",
		Sequence: 0,
	}

	results, err := m.Read(ctx, since)
	if err != nil {
		t.Fatal(err)
	}

	expected := []arc.Entry{
		arc.Entry{Bytes: data.BytesFromString("millers"), Topic: "hello-0", Sequence: 1},
		arc.Entry{Bytes: data.BytesFromString("fillers"), Topic: "hello-0", Sequence: 2},
	}

	if entriesEq(results, expected) != true {
		t.Errorf("Expected arc entries to be equal. Expected %+v, got %+v", expected, results)
	}
}

func entriesEq(a []arc.Entry, b []arc.Entry) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i, e := range a {
		if e.Bytes.Equal(b[i].Bytes) != true || e.Sequence != b[i].Sequence || e.Topic != b[i].Topic {
			return false
		}
	}

	return true
}
