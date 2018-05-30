package mish

import (
	"time"

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
// E.g. Run, Watch, Print.
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

type Watch struct {
	output   string
	patterns []string
	start    time.Time
	duration time.Duration
	done     bool
}

func (*Run) eval()   {}
func (*Watch) eval() {}
