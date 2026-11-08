// Package screen is the host side of a full-screen terminal program: raw
// mode and the alternate screen, keys decoded from the terminal's bytes, a
// grid of cells to draw on, and a renderer that sends only what changed.
package screen

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Color is an xterm 256-colour index, or Default for the terminal's own.
type Color int16

const Default Color = -1

type Pen struct {
	FG   Color
	BG   Color
	Bold bool
}

var Plain = Pen{FG: Default, BG: Default}

type Cell struct {
	R rune
	// Mark is a combining mark drawn over R (the accent of a decomposed
	// "é"). yagni: one per cell, a stack of marks keeps its first; a ZWJ
	// emoji sequence shows as its parts.
	Mark rune
	Pen
}

var Blank = Cell{R: ' ', Pen: Plain}

// Canvas is drawn in (row, col) order, the order a screen is read in. A
// cursor shows only where a field is being typed into.
type Canvas struct {
	W, H      int
	Cells     []Cell
	CursorRow int
	CursorCol int
	CursorOn  bool
}

func New(w, h int) *Canvas {
	c := &Canvas{}
	c.Resize(w, h)
	return c
}

func (c *Canvas) Resize(w, h int) {
	w = max(w, 0)
	h = max(h, 0)
	c.W, c.H = w, h
	n := w * h
	if cap(c.Cells) < n {
		c.Cells = make([]Cell, n)
	}
	c.Cells = c.Cells[:n]
	c.Clear()
}

func (c *Canvas) Clear() {
	for i := range c.Cells {
		c.Cells[i] = Blank
	}
	c.CursorOn = false
}

// At and Set clip to the grid, so drawing can run off an edge.
func (c *Canvas) At(row, col int) Cell {
	if row < 0 || col < 0 || row >= c.H || col >= c.W {
		return Blank
	}
	return c.Cells[row*c.W+col]
}

func (c *Canvas) Set(row, col int, cell Cell) {
	if row < 0 || col < 0 || row >= c.H || col >= c.W {
		return
	}
	c.Cells[row*c.W+col] = cell
}

// Print writes s from (row, col) and returns the cells it took, counting
// those cut off by the right edge. Control characters print as spaces: a
// raw tab or newline would move the terminal's cursor where the grid does
// not think it is.
func (c *Canvas) Print(row, col int, s string, p Pen) int {
	at := col
	for _, r := range s {
		if r < ' ' || r == 0x7f {
			r = ' '
		}
		w := runeWidth(r)
		if w == 0 {
			if unicode.In(r, unicode.Mn, unicode.Me) {
				c.mark(row, at-1, r)
			}
			continue // a format character (ZWJ) would make the terminal join glyphs the grid keeps apart
		}
		if w == 2 && at+1 == c.W {
			c.Set(row, at, Cell{R: ' ', Pen: p}) // half a wide glyph cannot show
			at += 2
			continue
		}
		c.Set(row, at, Cell{R: r, Pen: p})
		if w == 2 {
			c.Set(row, at+1, Cell{R: continuation, Pen: p})
		}
		at += w
	}
	return at - col
}

// mark puts r over the glyph that ends at col, unless it has one already.
func (c *Canvas) mark(row, col int, r rune) {
	if c.At(row, col).R == continuation {
		col--
	}
	if row < 0 || col < 0 || row >= c.H || col >= c.W {
		return
	}
	cell := &c.Cells[row*c.W+col]
	if cell.Mark == 0 && cell.R != ' ' {
		cell.Mark = r
	}
}

// Fill paints a block of rows x cols with r.
func (c *Canvas) Fill(row, col, rows, cols int, r rune, p Pen) {
	for y := row; y < row+rows; y++ {
		for x := col; x < col+cols; x++ {
			c.Set(y, x, Cell{R: r, Pen: p})
		}
	}
}

func (c *Canvas) ShowCursor(row, col int) {
	c.CursorRow, c.CursorCol, c.CursorOn = row, col, true
}

// String is the screen as text, one line per row, trailing blanks cut: what
// a test or an agent reads instead of a picture.
func (c *Canvas) String() string {
	var b strings.Builder
	line := make([]byte, 0, c.W*3)
	for y := range c.H {
		line = line[:0]
		for x := range c.W {
			r := c.Cells[y*c.W+x].R
			if r == continuation {
				continue
			}
			line = utf8.AppendRune(line, r)
			if m := c.Cells[y*c.W+x].Mark; m != 0 {
				line = utf8.AppendRune(line, m)
			}
		}
		b.WriteString(strings.TrimRight(string(line), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// Width is the cells s takes on a terminal.
func Width(s string) int {
	n := 0
	for _, r := range s {
		n += runeWidth(r)
	}
	return n
}

// RuneWidth is the cells r takes: 0, 1 or 2.
func RuneWidth(r rune) int {
	return runeWidth(r)
}
