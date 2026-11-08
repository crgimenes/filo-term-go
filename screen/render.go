package screen

import (
	"io"
	"strconv"
	"unicode/utf8"
)

// Renderer sends a frame as its difference against the one before: a frame
// costs bytes in proportion to what changed, not to the size of the screen.
// It owns the whole screen (the alternate one) and addresses cells
// absolutely. Adapted from hanabi/canvas, which renders in a region instead.
type Renderer struct {
	w     io.Writer
	front *Canvas
	buf   []byte
	pen   Pen
	row   int
	col   int
	full  bool
	// the cursor as last sent: row -1 hidden, -2 unknown
	curRow int
	curCol int
}

func NewRenderer(w io.Writer) *Renderer {
	return &Renderer{w: w, front: New(0, 0), full: true, curRow: -1}
}

// Invalidate makes the next Draw repaint every cell, for when something
// else wrote to the terminal (an editor run in between, a resize).
func (r *Renderer) Invalidate() {
	r.full = true
}

func (r *Renderer) Draw(back *Canvas) error {
	if r.front.W != back.W || r.front.H != back.H {
		r.front.Resize(back.W, back.H)
		r.full = true
	}
	mendWide(back)

	r.buf = r.buf[:0]
	if r.full {
		r.buf = append(r.buf, "\x1b[0m\x1b[2J"...)
		r.curRow = -2 // unknown after someone else had the terminal
	}
	r.pen = Plain
	r.row, r.col = -1, -1

	for y := range back.H {
		base := y * back.W
		for x := range back.W {
			cell := back.Cells[base+x]
			if cell.R == continuation {
				continue // written by the wide glyph on its left
			}
			if !r.full && cell == r.front.Cells[base+x] && !r.wideNeighbourChanged(back, base, x) {
				continue
			}
			if r.curRow != -1 {
				r.buf = append(r.buf, "\x1b[?25l"...) // a visible cursor would run over the glass
				r.curRow = -1
			}
			r.moveTo(y, x)
			r.setPen(cell.Pen)
			r.buf = utf8.AppendRune(r.buf, cell.R)
			r.col = x + runeWidth(cell.R)
		}
	}
	if r.pen != Plain {
		r.buf = append(r.buf, "\x1b[0m"...)
	}
	wantRow, wantCol := -1, 0
	if back.CursorOn {
		wantRow, wantCol = back.CursorRow, back.CursorCol
	}
	switch {
	case wantRow >= 0 && (wantRow != r.curRow || wantCol != r.curCol):
		r.row, r.col = -1, -1
		r.moveTo(wantRow, wantCol)
		r.buf = append(r.buf, "\x1b[?25h"...)
	case wantRow < 0 && r.curRow != -1:
		r.buf = append(r.buf, "\x1b[?25l"...)
	}
	r.curRow, r.curCol = wantRow, wantCol
	copy(r.front.Cells, back.Cells)
	r.full = false
	if len(r.buf) == 0 {
		return nil // nothing changed: not even a reset is worth sending
	}
	_, err := r.w.Write(r.buf)
	return err
}

// A wide glyph replaced by a narrow one leaves its right half to the cell
// beside it, which looks unchanged in the grid but not on the glass.
func (r *Renderer) wideNeighbourChanged(back *Canvas, base, x int) bool {
	return x > 0 && r.front.Cells[base+x-1] != back.Cells[base+x-1] &&
		runeWidth(r.front.Cells[base+x-1].R) == 2
}

// mendWide blanks whichever half of a wide glyph lost its other half to a
// later write: a terminal cannot draw half a glyph.
func mendWide(c *Canvas) {
	for y := range c.H {
		base := y * c.W
		for x := range c.W {
			cell := c.Cells[base+x]
			switch {
			case cell.R == continuation:
				if x > 0 && runeWidth(c.Cells[base+x-1].R) == 2 {
					continue
				}
			case runeWidth(cell.R) == 2:
				if x+1 < c.W && c.Cells[base+x+1].R == continuation {
					c.Cells[base+x+1].Pen = cell.Pen
					continue
				}
			default:
				continue
			}
			c.Cells[base+x] = Cell{R: ' ', Pen: cell.Pen}
		}
	}
}

func (r *Renderer) moveTo(row, col int) {
	if row == r.row && col == r.col {
		return
	}
	r.buf = append(r.buf, "\x1b["...)
	r.buf = strconv.AppendInt(r.buf, int64(row+1), 10)
	r.buf = append(r.buf, ';')
	r.buf = strconv.AppendInt(r.buf, int64(col+1), 10)
	r.buf = append(r.buf, 'H')
	r.row, r.col = row, col
}

func (r *Renderer) setPen(p Pen) {
	if p == r.pen {
		return
	}
	r.buf = append(r.buf, "\x1b[0"...)
	if p.Bold {
		r.buf = append(r.buf, ";1"...)
	}
	r.buf = appendColor(r.buf, p.FG, "38")
	r.buf = appendColor(r.buf, p.BG, "48")
	r.buf = append(r.buf, 'm')
	r.pen = p
}

func appendColor(b []byte, c Color, set string) []byte {
	if c == Default {
		return b
	}
	b = append(b, ';')
	b = append(b, set...)
	b = append(b, ";5;"...)
	return strconv.AppendInt(b, int64(c), 10)
}
