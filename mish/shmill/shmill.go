package shmill

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/windmilleng/skylurk"

	"github.com/windmilleng/mish/bridge/fs"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/pathutil"
	"github.com/windmilleng/mish/errors"
	"github.com/windmilleng/mish/os/ospath"
)

type Event interface {
	shmillEvent()
}

type WatchStartEvent struct {
	Patterns []string
	Output   string
}

type WatchDoneEvent struct {
}

type AutorunEvent struct {
	Patterns []string
}

type CmdStartedEvent struct {
	// TODO(dbentley): UID
	Cmd string
}

type CmdOutputEvent struct {
	Output string
}

type CmdDoneEvent struct {
	Err error
}

type ExecDoneEvent struct {
	Err error
}

func (WatchStartEvent) shmillEvent() {}
func (WatchDoneEvent) shmillEvent()  {}
func (AutorunEvent) shmillEvent()    {}
func (CmdStartedEvent) shmillEvent() {}
func (CmdOutputEvent) shmillEvent()  {}
func (CmdDoneEvent) shmillEvent()    {}
func (ExecDoneEvent) shmillEvent()   {}

func NewShmill(fs fs.FSBridge, ptrID data.PointerID, dir string, panicCh chan error) *Shmill {
	return &Shmill{fs: fs, ptrID: ptrID, dir: dir, panicCh: panicCh}
}

type Shmill struct {
	fs      fs.FSBridge
	ptrID   data.PointerID
	dir     string
	panicCh chan error
}

func (sh *Shmill) Start(ctx context.Context) chan Event {
	ch := make(chan Event)
	e := &ex{
		ch:      ch,
		ctx:     ctx,
		shmill:  sh,
		panicCh: sh.panicCh,
	}

	go e.exec()
	return ch
}

const (
	shN      = "sh"
	watchN   = "watch"
	autorunN = "autorun"
)

type ex struct {
	ch      chan Event
	ctx     context.Context
	shmill  *Shmill
	panicCh chan error

	watchCalled   bool
	autorunCalled bool
	runAlready    bool // whether a run has already happened

	// exec encountered an expected error (i.e. from a failed command), don't
	// report it as a top-level error
	expectedErr bool
}

func (e *ex) exec() (outerErr error) {
	defer func() {
		e.ch <- ExecDoneEvent{Err: outerErr}
		close(e.ch)
		if r := recover(); r != nil {
			e.panicCh <- fmt.Errorf("exec panic: %v", r)
		}
	}()

	text, err := ioutil.ReadFile(pathutil.WMShMill)
	if err != nil {
		return err
	}
	t := &skylurk.Thread{}
	globals := e.builtins()
	_, err = skylurk.ExecFile(t, pathutil.WMShMill, text, globals)
	if e.expectedErr {
		// a command failed, but it's nbd and that information was captured elsewhere.
		return nil
	}
	return err
}

func (e *ex) builtins() skylurk.StringDict {
	return skylurk.StringDict{
		shN:      skylurk.NewBuiltin(shN, e.Sh),
		watchN:   skylurk.NewBuiltin(watchN, e.Watch),
		autorunN: skylurk.NewBuiltin(autorunN, e.Autorun),
	}
}

func (e *ex) Sh(thread *skylurk.Thread, fn *skylurk.Builtin, args skylurk.Tuple, kwargs []skylurk.Tuple) (skylurk.Value, error) {
	if !e.runAlready {
		if err := e.watch([]string{pathutil.WMShMill}, true); err != nil {
			return nil, err
		}
		e.runAlready = true
	}
	var cmd string
	var tolerateFailure bool

	if err := skylurk.UnpackArgs(shN, args, kwargs,
		"cmd", &cmd, "tolerate_failure?", &tolerateFailure); err != nil {
		return nil, errors.Propagatef(err, "Millfile error")
	}

	w := &exWriter{ch: e.ch}
	c := exec.CommandContext(e.ctx, "bash", "-c", cmd)
	c.Stdout = w
	c.Stderr = w

	e.ch <- CmdStartedEvent{Cmd: cmd}
	err := c.Run()
	e.ch <- CmdDoneEvent{Err: err}

	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// NOT just a failed command, something has actually gone wrong.
			return skylurk.None, err
		}
		if !tolerateFailure {
			// The cmd exited with non-zero code. This cmd doesn't tolerate failure, so
			// return this err back up the stack (which kills this exec) -- but set
			// expectedErr flag so we don't freak out.
			e.expectedErr = true
			return skylurk.None, err
		}
	}

	return skylurk.None, nil
}

func (e *ex) Watch(thread *skylurk.Thread, fn *skylurk.Builtin, args skylurk.Tuple, kwargs []skylurk.Tuple) (skylurk.Value, error) {
	var patterns []string

	for i, p := range args {
		s, ok := p.(skylurk.String)
		if !ok {
			return nil, fmt.Errorf("argument %d to `watch` is not a string: %v %T", i, p, p)
		}
		patterns = append(patterns, string(s))
	}

	if err := e.watch(patterns, false); err != nil {
		return nil, err
	}

	return skylurk.None, nil
}

func (e *ex) watch(patterns []string, implicit bool) error {
	if e.runAlready {
		return fmt.Errorf("watch must be called before the first run")
	}

	if e.watchCalled {
		if implicit {
			return nil
		}
		return fmt.Errorf("watch may only be called once in your Millfile")
	}
	e.watchCalled = true

	m, err := ospath.NewMatcherFromPatterns(patterns)
	if err != nil {
		return err
	}

	output := ""
	if implicit {
		output = fmt.Sprintf("Implicit watch(%s)\n", pathutil.WMShMill)
	}
	if !m.Match(pathutil.WMShMill) {
		output = fmt.Sprintf("%sWarning: you are not watching %s\n", output, pathutil.WMShMill)
	}

	e.ch <- WatchStartEvent{
		Patterns: patterns,
		Output:   output,
	}

	err = e.shmill.fs.ToWMStart(e.ctx, e.shmill.dir, e.shmill.ptrID, m)
	e.ch <- WatchDoneEvent{}

	return err
}

func (e *ex) Autorun(thread *skylurk.Thread, fn *skylurk.Builtin, args skylurk.Tuple, kwargs []skylurk.Tuple) (skylurk.Value, error) {
	var patterns []string

	if e.runAlready {
		return nil, fmt.Errorf("autorun must be called before the first run")
	}

	if e.autorunCalled {
		return nil, fmt.Errorf("autorun may only be called once in your Millfile")
	}
	e.autorunCalled = true

	for i, p := range args {
		s, ok := p.(skylurk.String)
		if !ok {
			return nil, fmt.Errorf("argument %d to `autorun` is not a string: %v %T", i, p, p)
		}
		patterns = append(patterns, string(s))
	}

	e.ch <- AutorunEvent{Patterns: patterns}
	return skylurk.None, nil
}

type exWriter struct {
	ch chan Event
}

func (w *exWriter) Write(p []byte) (int, error) {
	w.ch <- CmdOutputEvent{Output: string(p)}
	return len(p), nil
}
