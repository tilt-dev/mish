package mish

import (
	"fmt"
	"strings"
	"time"

	"github.com/nsf/termbox-go"

	"github.com/windmilleng/mish/data/pathutil"
)

// the View is long-lived
// eventually, it should have other attributes
type View struct {
}

func (v *View) Render(m *Model) []int {
	r := &Render{}
	return r.Render(m)
}

// a Render is short-lived (for one render loop), and holds mutable state as we
// render down the screen.
// It's a helper for the View
type Render struct {
	maxX int
	maxY int
}

// Render renders the Model, and returns layout info the Controller
// will use to scroll.
func (r *Render) Render(m *Model) (blockSizes []int) {
	r.maxX, r.maxY = termbox.Size()

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	if m.Shmill != nil {
		blockSizes = r.renderShmill(m)
	}

	r.renderFooter(m)

	termbox.Flush()
	return blockSizes
}

const footerHeight = 2

func (r *Render) renderFooter(m *Model) {
	c := newBoxCanvas(r.maxX, footerHeight)
	p := newPen(c)
	div := "│"
	divDot := "┊"
	logo := "✧ mish"

	// Footer background
	p.setColor(termbox.ColorBlack, termbox.ColorWhite)
	for x := 0; x < c.maxX; x++ {
		p.ch(0)
	}

	// Revision
	p.at(1, 0)
	rev := fmt.Sprintf("Revision %d %s ", m.Rev, div)
	p.text(rev)

	// Exec status
	exec := "Waiting for " + pathutil.WMShMill
	if m.Shmill != nil {
		exec = fmt.Sprintf("%c Busy", m.Spinner.Cur())
		if m.Shmill.Done {
			state := "⠿ Done"
			if m.Shmill.Err != nil {
				state = "Error"
			}
			exec = fmt.Sprintf("%s (%s)", state, m.Shmill.Duration.Truncate(time.Millisecond).String())
		}
	}
	p.text(exec)

	// Logo
	p.at(c.maxX-len(logo), 0)
	p.text(logo)

	// Queued Files
	p.at(len(rev)+len(exec), 0)
	if len(m.QueuedFiles) > 0 {
		s := fmt.Sprint(m.QueuedFiles[0])
		if len(m.QueuedFiles) > 1 {
			s = fmt.Sprintf("%s (+%d)", m.QueuedFiles[0], len(m.QueuedFiles)-1)
		}

		msg := fmt.Sprintf(" %s Queued: %s", div, s)
		if len(msg) > (c.maxX - len(rev) - len(exec) - len(logo)) {
			msg = fmt.Sprintf(" %s %s", div, s)
		}

		p.text(msg)
	}

	// Hotkeys
	keys := fmt.Sprintf("BROWSE: [j] next, [k] prev, [o] show/hide  %s  SCROLL: [↑] Up, [↓] Down  %s  (r)erun  %s  (q)uit", divDot, divDot, divDot)
	p.at(c.maxX-len(keys)+8, 1) // Unicode runes mess up the count
	p.setColor(termbox.ColorDefault, termbox.ColorDefault)
	p.text(keys)

	c.RenderAt(0, r.maxY-footerHeight)
}

func (r *Render) renderShmill(m *Model) []int {
	sh := m.Shmill

	var blocks []int

	c := newScrollCanvas(r.maxX)
	p := newPen(c)
	var bs []blockView
	for i, ev := range sh.Evals {
		b := blockView{
			collapsed: m.Collapsed[i],
		}

		switch ev := ev.(type) {
		case *Run:
			b.headline = ev.cmd
			b.done = ev.done
			b.dur = ev.duration.Truncate(time.Millisecond).String()
			b.err = ev.err
			b.output = ev.output
		case *Watch:
			b.headline = fmt.Sprintf("watch %s", ev.patterns)
			b.done = ev.done
			b.dur = ev.duration.Truncate(time.Millisecond).String()
			b.err = nil
			b.output = ev.output
		}

		bs = append(bs, b)
	}

	if sh.Err != nil {
		bs = append(bs, blockView{
			headline:  "Mill error",
			done:      true,
			dur:       sh.Duration.Truncate(time.Millisecond).String(),
			err:       sh.Err,
			collapsed: m.Collapsed[len(sh.Evals)],
		})
	}

	for _, b := range bs {
		numLines := r.renderBlock(p, b)
		blocks = append(blocks, numLines)
	}

	scrollY := 0
	for i, l := range blocks {
		if i < m.Cursor.Block {
			scrollY += l
		}
	}
	scrollY += m.Cursor.Line
	c.RenderAt(0, r.maxY-footerHeight, scrollY)

	return blocks
}

type blockView struct {
	headline  string
	done      bool
	dur       string
	err       error
	output    string
	collapsed bool
}

// renderEval renders a blockView to the canvas, returning the number of lines it takes up
func (r *Render) renderBlock(p *pen, b blockView) (numLines int) {
	startY := p.posY
	p.newlineMaybe()

	toggle := '▼'
	if b.collapsed {
		toggle = '▶'
	}

	// multi-line headline -- just print the first line with an ellipse
	split := strings.SplitN(b.headline, "\n", 2)
	headline := fmt.Sprintf("%c %s", toggle, split[0])
	if len(split) > 1 {
		headline += "..."
	}
	p.text(headline)

	// STATUS
	// TODO: color indicating pass/fail?
	state := "pending …"
	if b.done {
		if b.err == nil {
			state = fmt.Sprintf("(%s) ✔", b.dur)
		} else {
			state = fmt.Sprintf("(%s) ✖", b.dur)
		}
	}
	gap := p.maxX - len(state) - len(headline) + 1
	for i := 0; i < gap; i++ {
		p.ch(' ')
	}
	p.text(state)
	p.newlineMaybe()
	if !b.collapsed {
		if b.output == "" && b.err == nil {
			// Nothing else left to print, we don't want an extra divider
			// HACK(maia): sloppy, but we're under deadline.
			for i := 0; i < r.maxX; i++ {
				p.ch('━')
			}
			return p.posY - startY
		}

		// HEADER - bottom separator
		for i := 0; i < p.maxX; i++ {
			p.ch('╌')
		}

		p.text(b.output)
		p.newline()

		if b.err != nil {
			p.text(fmt.Sprintf("err: %v", b.err))
			p.newlineMaybe()
		}
	}

	// HEADER - top separator for next command
	for i := 0; i < r.maxX; i++ {
		p.ch('━')
	}

	return p.posY - startY
}
