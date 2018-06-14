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

func (v *View) Render(m *Model) ([]int, int) {
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
func (r *Render) Render(m *Model) (blockSizes []int, ShmillHeight int) {
	r.maxX, r.maxY = termbox.Size()

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	if m.Shmill != nil {
		blockSizes = r.renderShmill(m)
	}

	r.renderFooter(m)

	r.maybeRenderFlowChooser(m)

	termbox.Flush()

	// Store terminal size on the model, because we need it for paging
	return blockSizes, r.maxY - footerHeight
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

	// Current Flow
	p.at(1, 0)
	flow := m.SelectedFlow
	if m.SelectedFlow == "" {
		flow = "None"
	}
	if m.ShowFlowChooser {
		flow += "…"
	}

	p.text(fmt.Sprintf("Flow: %s %s ", flow, div))

	// Revision
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

	// Hotkeys
	keys := fmt.Sprintf("BROWSE: [j] next, [k] prev, [o] show/hide  %s  SCROLL: [↑] Up, [↓] Down, [PageUp], [PageDown]  %s  (r)erun  %s  choose (f)low  %s  (q)uit", divDot, divDot, divDot, divDot)
	if m.ShowFlowChooser {
		keys = fmt.Sprintf("BROWSE: [↑] Up, [↓] Down  %s  (r)un", divDot)
	}
	p.at(1, 1)
	p.setColor(termbox.ColorDefault, termbox.ColorDefault)
	p.text(keys)

	c.RenderAt(0, r.maxY-footerHeight)
}

func (r *Render) maybeRenderFlowChooser(m *Model) {
	if !m.ShowFlowChooser {
		return
	}

	chooserHeight := len(m.Flows) + 2
	c := newBoxCanvas(r.maxX, chooserHeight)
	p := newPen(c)

	// header
	p.setColor(termbox.ColorBlack, termbox.ColorWhite)
	for x := 0; x < c.maxX; x++ {
		p.ch(0)
	}
	p.at(1, 0)
	p.text("Choose Flow:")

	cur := fmt.Sprintf("(%d/%d)", m.FlowChooserPos, len(m.Flows))
	p.at(r.maxX-len(cur)-1, 0)
	p.text(cur)

	// list flows
	p.resetColor()
	p.at(3, 1)
	p.text("(None)")
	if m.FlowChooserPos == 0 {
		p.at(1, 1)
		p.text("▸")
	}

	for i, f := range m.Flows {
		gap := 2 // make room for header and (none) option
		if i == m.FlowChooserPos-1 {
			p.at(1, i+gap)
			p.text("▸")
		}

		p.at(3, i+gap)
		p.text(f)
	}

	c.RenderAt(0, r.maxY-chooserHeight-footerHeight)
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

		if e, ok := ev.(*Run); ok {
			b.headline = e.cmd
			b.done = e.done
			b.dur = e.duration.Truncate(time.Millisecond).String()
			b.err = e.err
			b.output = e.output
			bs = append(bs, b)
		}
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
