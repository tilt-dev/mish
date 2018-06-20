package shmill

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/windmilleng/skylurk"

	"github.com/windmilleng/mish/data/pathutil"
	"github.com/windmilleng/mish/errors"
)

type Event interface {
	shmillEvent()
}

type TargetsFoundEvent struct {
	Targets []string
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

func (TargetsFoundEvent) shmillEvent() {}
func (CmdStartedEvent) shmillEvent()   {}
func (CmdOutputEvent) shmillEvent()    {}
func (CmdDoneEvent) shmillEvent()      {}
func (ExecDoneEvent) shmillEvent()     {}

func NewShmill(dir string, panicCh chan error) *Shmill {
	return &Shmill{dir: dir, panicCh: panicCh}
}

type Shmill struct {
	dir     string
	panicCh chan error
}

func (sh *Shmill) Start(ctx context.Context, target string) chan Event {
	ch := make(chan Event)
	e := &ex{
		ch:      ch,
		ctx:     ctx,
		shmill:  sh,
		panicCh: sh.panicCh,
		target:  target,
	}

	go e.exec()
	return ch
}

const (
	shN = "sh"
)

const targetPrefix = "wf_"

type ex struct {
	ch      chan Event
	ctx     context.Context
	shmill  *Shmill
	panicCh chan error

	target string // which target to execute

	runAlready bool // whether a run has already happened

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
	globals, err = skylurk.ExecFile(t, pathutil.WMShMill, text, globals)
	if e.expectedErr {
		// a command failed, but it's nbd and that information was captured elsewhere.
		return nil
	}
	if err != nil {
		return err
	}

	// we've now finished eval of top-level code.
	// We know what targets there are, so report on those, and then run the selected target
	var targets []string

	for k, v := range globals {
		if _, ok := v.(skylurk.Callable); strings.HasPrefix(k, targetPrefix) && ok {
			if len(strings.TrimPrefix(k, targetPrefix)) == 0 {
				return fmt.Errorf("global %v is an empty target name; give it a name", targetPrefix)
			}
			targets = append(targets, strings.TrimPrefix(k, targetPrefix))
		}
	}

	e.ch <- TargetsFoundEvent{Targets: targets}

	if e.target == "" {
		return nil
	}

	globalName := targetPrefix + e.target
	targetV, ok := globals[globalName]
	if !ok {
		return fmt.Errorf("target %s doesn't exist (no global %s)", e.target, globalName)
	}

	targetFunc, ok := targetV.(skylurk.Callable)
	if !ok {
		return fmt.Errorf("global %s is not a function; is a %T: %v", globalName, targetV, targetV)
	}

	_, err = targetFunc.Call(t, nil, nil)

	return err
}

func (e *ex) builtins() skylurk.StringDict {
	return skylurk.StringDict{
		shN: skylurk.NewBuiltin(shN, e.Sh),
	}
}

func (e *ex) Sh(thread *skylurk.Thread, fn *skylurk.Builtin, args skylurk.Tuple, kwargs []skylurk.Tuple) (skylurk.Value, error) {
	var cmd string
	var tolerateFailure bool

	if err := skylurk.UnpackArgs(shN, args, kwargs,
		"cmd", &cmd, "tolerate_failure?", &tolerateFailure); err != nil {
		return nil, errors.Propagatef(err, "Millfile error")
	}

	w := &exWriter{ch: e.ch}
	doneCh := make(chan error, 1)
	c := exec.Command("bash", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Stdout = w
	c.Stderr = w

	// run the command
	go func() {
		e.ch <- CmdStartedEvent{Cmd: cmd}
		doneCh <- c.Run()
	}()

	// wait for command to be finished
	// OR ctx.Done
	select {
	case err := <-doneCh:
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
	case <-e.ctx.Done():
		err := syscall.Kill(c.Process.Pid, syscall.SIGINT)
		if err != nil {
			killPG(c)
		} else {
			// wait and then send SIGKILL to the process group, unless the command finished
			select {
			case <-time.After(50 * time.Millisecond):
				killPG(c)
			case <-doneCh:
			}
		}
	}

	return skylurk.None, nil
}

type exWriter struct {
	ch chan Event
}

func (w *exWriter) Write(p []byte) (int, error) {
	w.ch <- CmdOutputEvent{Output: string(p)}
	return len(p), nil
}

func killPG(c *exec.Cmd) {
	syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
}
