package kit

import (
	"strconv"

	"github.com/crgimenes/filo-term-go/screen"
)

// Col is a table column: its header and its width in cells.
type Col struct {
	Header string
	Width  int
}

// Table is headers in soft over a dim rule, the row under the arrows in
// orange, "+N more" when the rest does not fit. The window keeps still
// until the cursor would leave it.
type Table struct {
	Cursor int
	first  int
}

// FillCols gives the last column what inner leaves.
func FillCols(cols []Col, inner int) []Col {
	if len(cols) == 0 {
		return cols
	}
	used := 0
	for _, c := range cols[:len(cols)-1] {
		used += c.Width + 1
	}
	out := append([]Col(nil), cols...)
	out[len(out)-1].Width = max(4, inner-used)
	return out
}

// Draw paints total rows from (top, left) in height rows (header and rule
// not counted); rowOf gives row i's cells, one per column.
func (t *Table) Draw(c *screen.Canvas, top, left int, cols []Col, total int, rowOf func(int) []string, height int) {
	wide := -1
	at := left
	for _, col := range cols {
		c.Print(top, at, Cut(col.Header, col.Width), Soft)
		at += col.Width + 1
		wide += col.Width + 1
	}
	c.Fill(top+1, left, 1, wide, '─', Dim)

	if height <= 0 || total <= 0 {
		return
	}
	shown := height
	if total > shown && height > 1 {
		shown-- // the last row says how many more there are
	}
	t.Cursor = max(0, min(t.Cursor, total-1))
	t.first = min(t.first, t.Cursor)
	if t.Cursor >= t.first+shown {
		t.first = t.Cursor - shown + 1
	}
	t.first = max(0, min(t.first, total-shown))
	shown = min(shown, total-t.first)
	for r := range shown {
		i := t.first + r
		row := top + 2 + r
		pen := Text
		if i == t.Cursor {
			pen = Focus
			c.Fill(row, left, 1, wide, ' ', pen)
		}
		cells := rowOf(i)
		at := left
		for j, col := range cols {
			if j < len(cells) {
				c.Print(row, at, Cut(cells[j], col.Width), pen)
			}
			at += col.Width + 1
		}
	}
	if more := total - t.first - shown; more > 0 && height > shown {
		c.Print(top+2+shown, left, "+"+strconv.Itoa(more)+" more", Dim)
	}
}

// Move takes the cursor over n rows with the arrows, a page at a time with
// PgUp and PgDn; false means the key is not the table's.
func (t *Table) Move(k screen.Key, n, page int) bool {
	if n <= 0 {
		t.Cursor = 0
		return false
	}
	switch k {
	case screen.KeyUp:
		t.Cursor = mod(t.Cursor-1, n)
	case screen.KeyDown:
		t.Cursor = mod(t.Cursor+1, n)
	case screen.KeyPgUp:
		t.Cursor = max(0, t.Cursor-page)
	case screen.KeyPgDn:
		t.Cursor = min(n-1, t.Cursor+page)
	case screen.KeyHome:
		t.Cursor = 0
	case screen.KeyEnd:
		t.Cursor = n - 1
	default:
		return false
	}
	return true
}
