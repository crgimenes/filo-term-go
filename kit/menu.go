package kit

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/crgimenes/filo-term-go/screen"
)

// Item is a menu entry: the key that opens it and its label; what follows
// " — " in the label is a complement, drawn dim.
type Item struct {
	Key   string
	Label string
}

// Menu is the editor's menu: the hot letter in orange, the word in text, a
// pointer › on the item the arrows are on. Items fill the first column
// down, then the next.
type Menu struct {
	At int
}

func menuRows(n, ncols int) int {
	return max(1, (n+ncols-1)/ncols)
}

// Draw lays the items out from (top, left) in ncols columns of colw cells.
func (m *Menu) Draw(c *screen.Canvas, top, left int, items []Item, ncols, colw int) {
	rows := menuRows(len(items), ncols)
	for i, it := range items {
		row := top + i%rows
		col := left + (i/rows)*colw
		if i == m.At {
			c.Print(row, col-2, "›", Hot)
		}
		hot(c, row, col, it, colw-2)
	}
}

// hot draws one item and returns the cells it took.
func hot(c *screen.Canvas, row, col int, it Item, width int) int {
	word, rest, _ := strings.Cut(it.Label, "—")
	word = strings.TrimSpace(word)
	first, size := utf8.DecodeRuneInString(word)
	key, _ := utf8.DecodeRuneInString(it.Key)
	at := col
	if unicode.ToLower(first) == unicode.ToLower(key) {
		at += c.Print(row, at, word[:size], Hot)
		at += c.Print(row, at, word[size:], Text)
	} else {
		at += c.Print(row, at, it.Key, Hot)
		at += c.Print(row, at, " "+word, Text)
	}
	room := width - (at - col) - 3
	rest = strings.TrimSpace(rest)
	if rest != "" && room > 3 {
		at += c.Print(row, at, " — "+Cut(rest, room), Dim)
	}
	return at - col
}

// Key returns the item a key opens; -1 when it only moved the pointer, -2
// when it meant nothing to the menu. A letter opens its item at once.
func (m *Menu) Key(k screen.Key, items []Item, ncols int) int {
	n := len(items)
	if n == 0 {
		return -2
	}
	m.At = min(m.At, n-1)
	switch k {
	case screen.KeyUp:
		m.At = m.step(n, ncols, -1, 0)
		return -1
	case screen.KeyDown:
		m.At = m.step(n, ncols, 1, 0)
		return -1
	case screen.KeyLeft:
		m.At = m.step(n, ncols, 0, -1)
		return -1
	case screen.KeyRight:
		m.At = m.step(n, ncols, 0, 1)
		return -1
	case screen.KeyEnter, '\n':
		return m.At
	}
	if !k.Printable() {
		return -2
	}
	for i, it := range items {
		key, _ := utf8.DecodeRuneInString(it.Key)
		if unicode.ToLower(key) == unicode.ToLower(rune(k)) {
			m.At = i
			return i
		}
	}
	return -2
}

func mod(a, n int) int {
	if n == 0 {
		return 0
	}
	return ((a % n) + n) % n
}

// step walks drow inside the pointer's column, wrapping over its own items,
// or dcol to another column, same row or that column's last.
func (m *Menu) step(n, ncols, drow, dcol int) int {
	rows := menuRows(n, ncols)
	r, col := m.At%rows, m.At/rows
	columnLen := func(c int) int { return min(rows, n-c*rows) }
	if dcol == 0 {
		return col*rows + mod(r+drow, columnLen(col))
	}
	c2 := mod(col+dcol, ncols)
	l := columnLen(c2)
	if l <= 0 {
		return m.At
	}
	return c2*rows + min(r, l-1)
}
