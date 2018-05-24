package io

// Bridges for forwarding IO streams to pointers.

import (
	"context"
	"io"
	"sync"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
)

// A helper for writing linear ops to a pointer.
// Intended to avoid bookkeeping in cases where we're the only one
// writing to the pointer.
//
// In the long term, we will want some sort of pointer-acquiring mutex,
// and this struct should encapsulate the logic for aquiring that lock.
type PointerWriter struct {
	ctx     context.Context
	current data.PointerAtSnapshot
	tag     data.RecipeWTag
	db      dbint.DB2
	mu      sync.Mutex
	wg      sync.WaitGroup
}

func NewPointerWriter(ctx context.Context, db dbint.DB2, id data.PointerID) (*PointerWriter, error) {
	headSnap, err := dbint.AcquireSnap(ctx, db, id)
	if err != nil {
		return nil, err
	}
	return &PointerWriter{
		ctx:     ctx,
		current: headSnap,
		tag:     data.RecipeWTagForPointer(id),
		db:      db,
	}, nil
}

func (b *PointerWriter) CurrentHead() data.PointerAtSnapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.current
}

func (b *PointerWriter) Write(op data.Op) error {
	b.wg.Add(1)
	defer b.wg.Done()

	// TODO(nick): It would be nice to buffer writes instead of locking.
	b.mu.Lock()
	defer b.mu.Unlock()

	newSnap, _, err := b.db.Create(b.ctx, data.Recipe{
		Op:     op,
		Inputs: []data.SnapshotID{b.current.SnapID},
	}, b.current.Owner(), b.tag)
	if err != nil {
		return err
	}

	newHead := data.PtrEdit(b.current, newSnap)
	err = b.db.Set(b.ctx, newHead)
	b.current = newHead
	return err
}

func (b *PointerWriter) WriteSlice(ops []data.Op) error {
	if len(ops) == 0 {
		return nil
	}

	b.wg.Add(1)
	defer b.wg.Done()

	b.mu.Lock()
	defer b.mu.Unlock()

	newSnap, err := dbint.CreateLinear(b.ctx, b.db, ops, b.current.SnapID, b.current.Owner(), b.tag)
	if err != nil {
		return err
	}

	newHead := data.PtrEdit(b.current, newSnap)
	err = b.db.Set(b.ctx, newHead)
	b.current = newHead
	return err
}

// Freezes the pointer. If anyone tries to write to the pointerWriter after this
// Close(), we will panic. All PointerWriters must eventually Close() to free
// resources.
func (b *PointerWriter) Close(ctx context.Context) error {
	b.wg.Wait()

	b.mu.Lock()
	defer b.mu.Unlock()

	err := dbint.SetFrozen(ctx, b.db, b.current)
	return err
}

// Creates an io.Writer that sends its output to a path of a pointer.
//
// TODO(dbentley): consider using a bufio.Writer to buffer writes
// This could improve perf, but at the cost of latency (and confusion when writes don't come)
func (b *PointerWriter) CreateStream(ctx context.Context, path string) (io.Writer, error) {
	err := b.Write(&data.WriteFileOp{Path: path, Data: data.NewEmptyBytes()})
	if err != nil {
		return nil, err
	}
	return &streamWriter{bridge: b, ctx: ctx, path: path}, nil
}

type streamWriter struct {
	bridge *PointerWriter
	ctx    context.Context
	path   string
	nextAt int64
}

func (w *streamWriter) Write(p []byte) (int, error) {
	op := &data.InsertBytesFileOp{Path: w.path, Index: w.nextAt, Data: data.NewBytes(p)}
	err := w.bridge.Write(op)
	if err != nil {
		return 0, err
	}

	w.nextAt += int64(len(p))

	return len(p), nil
}
