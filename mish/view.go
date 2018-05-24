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
	status := "Waiting for " + pathutil.WMShMill
	if m.Shmill != nil {
		status = "BUSY…"
		if m.Shmill.Done {
			dur := m.Shmill.Duration.Truncate(time.Millisecond).String()
			status = fmt.Sprintf("Done in %s", dur)
		}
	}
	p.text(status)

	// Logo
	p.at(c.maxX-len(logo), 0)
	p.text(logo)

	// Queued Files
	p.at(len(rev)+len(status), 0)
	if len(m.QueuedFiles) > 0 {
		s := fmt.Sprint(m.QueuedFiles[0])
		if len(m.QueuedFiles) > 1 {
			s = fmt.Sprintf("%s (+%d)", m.QueuedFiles[0], len(m.QueuedFiles)-1)
		}

		msg := fmt.Sprintf(" %s Queued: %s", div, s)
		if len(msg) > (c.maxX - len(rev) - len(status) - len(logo)) {
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
	evals := sh.Evals
	if sh.Err != nil {
		evals = append(evals, &ExecError{
			err:      sh.Err,
			duration: sh.Duration,
		})
	}
	for i, ev := range evals {
		numLines := r.renderBlock(p, m, ev, m.Collapsed[i])
		blocks = append(blocks, numLines)
	}

	scrollY := 0
	for i, l := range blocks {
		if i < m.Cursor.Block {
			scrollY += l
		}
	}
	scrollY += m.Cursor.Line

	// Separator at end of cmds
	for i := 0; i < r.maxX; i++ {
		p.ch('━')
	}

	c.RenderAt(0, r.maxY-footerHeight, scrollY)

	return blocks
}

// renderEval renders an eval to the canvas as block, returning the number of lines
// it takes up
func (r *Render) renderBlock(p *pen, m *Model, ev Eval, collapsed bool) (numLines int) {
	startY := p.posY
	p.newlineMaybe()

	// HEADER - top separator
	for i := 0; i < r.maxX; i++ {
		p.ch('━')
	}
	split := strings.SplitN(ev.Headline(), "\n", 2)
	toggle := '▼'
	if collapsed {
		toggle = '▶'
	}

	headline := fmt.Sprintf("%c %s", toggle, split[0])
	if len(split) > 1 {
		// multi-line headline -- just print the first line with an ellipse
		headline += "…"
	}
	p.text(headline)

	// STATUS
	// TODO: color indicating pass/fail?
	state := "pending …"
	if ev.Done() {
		if ev.Err() == nil {
			state = fmt.Sprintf("(%s) ✔", ev.DurStr())
		} else {
			state = fmt.Sprintf("(%s) ✖", ev.DurStr())
		}
	}
	gap := p.maxX - len(state) - len(headline) + 1
	for i := 0; i < gap; i++ {
		p.ch(' ')
	}
	p.text(state)
	p.newlineMaybe()
	if collapsed {
		return p.posY - startY
	}

	if ev.Output() == "" && ev.Err() == nil {
		// Nothing else left to print, we don't want an extra divider
		// HACK(maia): sloppy, but we're under deadline.
		return p.posY - startY
	}
	// HEADER - bottom separator
	for i := 0; i < p.maxX; i++ {
		p.ch('╌')
	}

	p.text(ev.Output())
	p.newline()

	if ev.Err() != nil {
		p.text(fmt.Sprintf("err: %v", ev.Err()))
		p.newlineMaybe()
	}

	return p.posY - startY
}
