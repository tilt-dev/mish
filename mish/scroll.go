package mish

import (
	"fmt"
)

type action int

const (
	downAction action = iota
	upAction
	pgDnAction
	pgUpAction
)

func scroll(c Cursor, blockSizes []int, viewHeight int, a action) Cursor {
	switch a {
	case downAction:
		return down(c, blockSizes, viewHeight)
	case upAction:
		return up(c, blockSizes, viewHeight)
	case pgDnAction:
		return pgDn(c, blockSizes, viewHeight)
	case pgUpAction:
		return pgUp(c, blockSizes, viewHeight)
	default:
		panic(fmt.Errorf("unexpected action %v", a))
	}
}

func down(c Cursor, blockSizes []int, viewHeight int) Cursor {
	c.Line++
	c.LineInView++

	// paging
	if c.LineInView >= viewHeight {
		c.LineInView = viewHeight / 2
	}

	// advance to next block
	if c.Line >= blockSizes[c.Block] {
		c.Block++
		c.Line = 0
	}

	// don't go past end of output
	if c.Block >= len(blockSizes) {
		lastBlock := len(blockSizes) - 1
		c.Block = lastBlock
		c.Line = blockSizes[c.Block] - 1
		c.LineInView--
	}

	return c
}

func up(c Cursor, blockSizes []int, viewHeight int) Cursor {
	if c.Line <= 0 {
		if c.Block < 1 {
			c.Block = 0
			c.Line = 0
			c.LineInView = 0
			return c
		}

		c.Line = blockSizes[c.Block-1]
		c.Block--
	}

	c.Line--
	c.LineInView--

	// paging
	if c.LineInView <= 0 {
		bufferIdx := getBufferIdx(c, blockSizes)

		if bufferIdx > (viewHeight / 2) {
			c.LineInView = viewHeight / 2
		} else {
			c.LineInView = bufferIdx
		}
	}

	return c
}

func pgDn(c Cursor, blockSizes []int, viewHeight int) Cursor {
	pgAmt := viewHeight - 2 // Leave a little room to find our place

	// check if there's enough output left to page down
	bufferIdx := getBufferIdx(c, blockSizes)

	bufferTotal := 0
	for _, l := range blockSizes {
		bufferTotal += l
	}

	startLine := bufferIdx - c.LineInView
	if (bufferTotal - startLine) < viewHeight {
		return c
	}

	// page down
	c.Line += (pgAmt - c.LineInView)
	c.LineInView = 0

	return successiveBlock(c, blockSizes)
}

// increment to a successive block based on the size of the current block
func successiveBlock(c Cursor, blockSizes []int) Cursor {
	if c.Line > blockSizes[c.Block] {
		c.Line = c.Line - blockSizes[c.Block]
		c.Block++
		return successiveBlock(c, blockSizes)
	}

	return c
}

func pgUp(c Cursor, blockSizes []int, viewHeight int) Cursor {
	pgAmt := viewHeight - 2 // Leave a little room to find our place
	c.Line -= (pgAmt - c.LineInView)
	c.LineInView = 0

	bufferIdx := getBufferIdx(c, blockSizes)

	if bufferIdx < pgAmt {
		c.Block = 0
		c.Line = 0
		c.LineInView = 0
	}

	return previousBlock(c, blockSizes)
}

func previousBlock(c Cursor, blockSizes []int) Cursor {
	// decrement to a previous block based on size of previous block
	if c.Line < 0 {
		if c.Block < 1 {
			c.Block = 0
			c.Line = 0
			return c
		}
		c.Line = c.Line + blockSizes[c.Block-1]
		c.Block--
		return previousBlock(c, blockSizes)
	}

	return c
}

func getBufferIdx(c Cursor, blockSizes []int) int {
	bufferIdx := 0
	for i, l := range blockSizes {
		if i < c.Block {
			bufferIdx += l
		}
	}

	bufferIdx += c.Line

	return bufferIdx
}
