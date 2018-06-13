package mish

import (
	"time"

	"github.com/windmilleng/mish/data"
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
	Shmill *Shmill

	// byproducts of shmill
	BlockSizes   []int // block i has BlockSizes[i] many lines
	ShmillHeight int

	// modified by keys
	Cursor    Cursor
	Collapsed map[int]bool

	// select flow to exec
	ShowFlowChooser bool
	FlowChooserPos  int
	Targets         []string
	SelectedTarget  string

	Spinner *Spinner
}

type Spinner struct {
	Chars []rune
	Index int
}

func (s *Spinner) Incr() {
	s.Index = (s.Index + 1) % len(s.Chars)
}

func (s *Spinner) Cur() rune {
	return s.Chars[s.Index]
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
// E.g. Run, Print.
type Eval interface {
	eval()
}

type Run struct {
	cmd      string
	output   string
	start    time.Time
	duration time.Duration
	done     bool
	err      error
}

func (*Run) eval() {}
