package mish

import (
	"time"

	"fmt"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
)

// Model of our MVC
type Model struct {
	File     string
	Now      time.Time
	Rev      int
	HeadSnap data.SnapshotID

	// files that have changed since we started running
	QueuedFiles []string

	// result of shmill'ing
	Shmill     *Shmill
	BlockSizes []int // block i has BlockSizes[i] many lines

	Autorun *dbpath.Matcher

	// modified by keys
	Cursor    Cursor
	Collapsed map[int]bool
}

type Cursor struct {
	Block int
	Line  int // line index within this block (not over the whole doc)
}

type Shmill struct {
	Evals    []Eval
	Start    time.Time
	Duration time.Duration
	Err      error // top-level error (unexpected; killed the whole exec)
	Done     bool

	ChangedFiles []string
}

func NewShmill() *Shmill {
	return &Shmill{Start: time.Now()}
}

// Eval is an Evaluation the user might care about.
// E.g. Run, Watch, Print.
type Eval interface {
	Headline() string
	Done() bool
	DurStr() string
	Err() error
	Output() string
}

type Run struct {
	cmd      string
	output   string
	start    time.Time
	duration time.Duration
	done     bool
	err      error
}

func (r *Run) Headline() string {
	return r.cmd
}

func (r *Run) Done() bool {
	return r.done
}

func (r *Run) DurStr() string {
	return r.duration.Truncate(time.Millisecond).String()
}

func (r *Run) Err() error {
	return r.err
}

func (r *Run) Output() string {
	return r.output
}

type Watch struct {
	output   string
	patterns []string
	start    time.Time
	duration time.Duration
	done     bool
}

func (w *Watch) Headline() string {
	return fmt.Sprintf("watch %s", w.patterns)
}

func (w *Watch) Done() bool {
	return w.done
}

func (w *Watch) DurStr() string {
	return w.duration.Truncate(time.Millisecond).String()
}

func (w *Watch) Err() error {
	return nil
}

func (w *Watch) Output() string {
	return w.output
}

type ExecError struct {
	err      error
	duration time.Duration
}

func (e *ExecError) Headline() string {
	return "Mill error"
}

func (e *ExecError) Done() bool {
	return true
}

func (e *ExecError) DurStr() string {
	return e.duration.Truncate(time.Millisecond).String()
}

func (e *ExecError) Err() error {
	return e.err
}

func (e *ExecError) Output() string {
	return ""
}
