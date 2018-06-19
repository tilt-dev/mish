package mish

import (
	"testing"
)

func TestDown(t *testing.T) {
	f := setup(t)
	f.viewHeight = 9
	f.blocks(9, 3)

	// paging
	for i := 0; i < 8; i++ {
		f.down(0, i+1, i+1)
	}
	f.down(1, 0, 4)

	// don't go past end of output
	f.down(1, 1, 5)
	f.down(1, 2, 6)
	f.down(1, 2, 6)
}

func TestDownBlock(t *testing.T) {
	f := setup(t)

	f.blocks(3, 3)
	f.down(0, 1, 1)
	f.down(0, 2, 2)
	f.down(1, 0, 3)
}

func TestUp(t *testing.T) {
	f := setup(t)
	f.viewHeight = 9
	f.blocks(9, 3)

	// don't go above top
	f.up(0, 0, 0)

	// paging
	for i := 0; i < 8; i++ {
		f.down(0, i+1, i+1)
	}
	f.down(1, 0, 4)
	f.up(0, 8, 3)
	f.up(0, 7, 2)
	f.up(0, 6, 1)
	f.up(0, 5, 4)
}

func TestUpBlock(t *testing.T) {
	f := setup(t)

	f.blocks(3, 3)
	f.down(0, 1, 1)
	f.down(0, 2, 2)
	f.down(1, 0, 3)
	f.up(0, 2, 2)
}

func TestPageDown(t *testing.T) {
	f := setup(t)
	f.viewHeight = 9
	f.blocks(3, 11, 6, 3)

	// blocks
	f.pgDn(1, 4, 0)
	f.pgDn(1, 11, 0)
	f.pgDn(3, 1, 0)
	f.pgDn(3, 1, 0)
}

type fixture struct {
	t *testing.T

	cursor Cursor

	viewHeight int
	blockSizes []int
}

func setup(t *testing.T) *fixture {
	return &fixture{
		t:          t,
		viewHeight: 100,
		blockSizes: []int{1000},
	}
}

func (f *fixture) blocks(blockSizes ...int) {
	f.blockSizes = blockSizes
}

func (f *fixture) down(exBlock int, exLine int, exLineInView int) {
	expected := Cursor{exBlock, exLine, exLineInView}
	f.scroll(expected, downAction)
}

func (f *fixture) pgDn(exBlock int, exLine int, exLineInView int) {
	expected := Cursor{exBlock, exLine, exLineInView}
	f.scroll(expected, pgDnAction)
}

func (f *fixture) up(exBlock int, exLine int, exLineInView int) {
	expected := Cursor{exBlock, exLine, exLineInView}
	f.scroll(expected, upAction)
}

func (f *fixture) scroll(expected Cursor, a action) {
	actual := scroll(f.cursor, f.blockSizes, f.viewHeight, a)
	if expected != actual {
		f.t.Fatalf("got cursor %v; expected %v", actual, expected)
	}

	f.cursor = actual
}
