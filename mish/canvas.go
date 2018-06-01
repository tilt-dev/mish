package mish

import (
	"math"

	"github.com/nsf/termbox-go"
)

func highlightCells(cells []termbox.Cell, fg, bg termbox.Attribute) {
	for i, c := range cells {
		cells[i] = termbox.Cell{
			Ch: c.Ch,
			Fg: fg,
			Bg: bg,
		}
	}
}

type canvas interface {
	MaxX() int
	MaxY() int

	SetCell(x, y int, ch rune, fg, bg termbox.Attribute)
}

// scrollCanvas is a canvas with a fixed x but an infinitely growable buffer of y's
type scrollCanvas struct {
	maxX  int
	lines [][]termbox.Cell
}

func newScrollCanvas(maxX int) *scrollCanvas {
	return &scrollCanvas{
		maxX: maxX,
	}
}

func (c *scrollCanvas) MaxX() int {
	return c.maxX
}

func (c *scrollCanvas) MaxY() int {
	return math.MaxInt32
}

func (c *scrollCanvas) SetCell(x, y int, ch rune, fg, bg termbox.Attribute) {
	if x < 0 || x >= c.maxX || y < 0 {
		return
	}

	// why 2?
	// if y = 3, and len(c.lines) = 3, we want to add 1 line,
	// so we need y-len(c.lines)+x = 1.
	for i := 0; i < y-len(c.lines)+2; i++ {
		// this is a new y, so add lines
		c.lines = append(c.lines, make([]termbox.Cell, c.maxX))
	}

	c.lines[y][x] = termbox.Cell{Ch: ch, Fg: fg, Bg: bg}
}

// RenderAt renders onto the screen starting at line screenY,
// taking up numLines lines, beginning at line startLine in the canvas's buffer
func (c *scrollCanvas) RenderAt(screenY, numLines, startLine int) {
	if len(c.lines) == 0 {
		return
	}

	highlightCells(c.lines[startLine], termbox.ColorBlue, termbox.ColorDefault)

	// Don't keep scrolling if there is no content
	if len(c.lines)-startLine < numLines || len(c.lines)-startLine < 0 {
		startLine = len(c.lines) - numLines
	}

	if startLine < 0 {
		startLine = 0
	}

	for i, line := range c.lines[startLine:] {
		if i >= numLines {
			return
		}
		for j, cell := range line {
			termbox.SetCell(j, screenY+i, cell.Ch, cell.Fg, cell.Bg)
		}
	}
}

func newBoxCanvas(maxX, maxY int) *boxCanvas {
	lines := make([][]termbox.Cell, 0)
	for i := 0; i < maxY; i++ {
		lines = append(lines, make([]termbox.Cell, maxX))
	}
	return &boxCanvas{
		maxX:  maxX,
		maxY:  maxY,
		lines: lines,
	}
}

// boxCanvas is a canvas with a fixed size.
type boxCanvas struct {
	maxX  int
	maxY  int
	lines [][]termbox.Cell
}

func (c *boxCanvas) MaxX() int {
	return c.maxX
}

func (c *boxCanvas) MaxY() int {
	return c.maxY
}

func (c *boxCanvas) SetCell(x, y int, ch rune, fg, bg termbox.Attribute) {
	if x < 0 || x >= c.maxX || y < 0 || y >= c.maxY {
		return
	}

	c.lines[y][x] = termbox.Cell{Ch: ch, Fg: fg, Bg: bg}
}

func (c *boxCanvas) RenderAt(x, y int) {
	for i, line := range c.lines {
		for j, cell := range line {
			termbox.SetCell(x+j, y+i, cell.Ch, cell.Fg, cell.Bg)
		}
	}
}

func newPen(c canvas) *pen {
	return &pen{
		c:    c,
		maxX: c.MaxX(),
		maxY: c.MaxY(),
	}
}

type pen struct {
	c    canvas
	maxX int
	maxY int

	posX int
	posY int

	startX int

	fg termbox.Attribute
	bg termbox.Attribute
}

func (p *pen) setColor(fg, bg termbox.Attribute) {
	p.fg = fg
	p.bg = bg
}

func (p *pen) resetColor() {
	p.setColor(termbox.ColorDefault, termbox.ColorDefault)
}

func (p *pen) indent() {
}

func (p *pen) dedent() {
}

func (p *pen) at(x, y int) {
	p.posX = x
	p.posY = y
}

func (p *pen) text(s string) {
	for _, c := range s {
		p.ch(c)
	}
}

func (p *pen) newline() {
	p.posX = 0
	p.posY += 1
}

func (p *pen) newlineMaybe() {
	if p.posX != 0 {
		p.newline()
	}
}

func (p *pen) ch(c rune) {
	// Respect indentation
	if p.posX < p.startX {
		p.posX = p.startX
	}

	switch c {
	// handle some escape sequences
	case '\n':
		p.newline()
	case '\t':
		p.tab()
	default:
		p.c.SetCell(p.posX, p.posY, c, p.fg, p.bg)

		// TODO(dbentley): use runewidth library
		p.posX += 1
		if p.posX >= p.maxX {
			p.newline()
		}
	}
}

func (p *pen) tab() {
	// Placeholder for Han to do stuff with, then use to replace `textIndent`
	// etc. with \t escape seq. in the text itself
	p.posX += 4
	if p.posX >= p.maxX {
		p.newline()
	}
}

func (p *pen) space() {
	p.ch(' ')
}
