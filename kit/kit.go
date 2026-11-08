// Package kit is filo-term's screen kit (kit/kit.filo) in Go: the palette,
// the frame, the legend, the note, the menu, the form, the table and the
// one-line field. A screen draws with these and nothing else decides a
// colour, so a Go program looks like edt and the BBS.
package kit

import (
	"strings"

	"github.com/crgimenes/filo-term-go/screen"
)

// The palette: the editor's. The ground is the terminal's own; colour is
// spent on what is read, and one colour, orange, on what acts.
const (
	CText  screen.Color = 252 // what is read
	CSoft  screen.Color = 250 // labels, headers: read second
	CDim   screen.Color = 244 // frames, legends: there, not asked for
	CPanel screen.Color = 236 // a field or a button at rest
	CHot   screen.Color = 214 // the one colour that acts
	COK    screen.Color = 2
	CWarn  screen.Color = 214
	CErr   screen.Color = 203
)

var (
	Text  = screen.Pen{FG: CText, BG: screen.Default}
	Soft  = screen.Pen{FG: CSoft, BG: screen.Default}
	Dim   = screen.Pen{FG: CDim, BG: screen.Default}
	Title = screen.Pen{FG: CText, BG: screen.Default, Bold: true}
	// Focus is a field, a button or a row under the arrows.
	Focus = screen.Pen{FG: 0, BG: CHot}
	// Rest is a field or a button without the focus.
	Rest = screen.Pen{FG: CText, BG: CPanel}
	Hot  = screen.Pen{FG: CHot, BG: screen.Default, Bold: true}
)

// Cut fits s in width cells, with an ellipsis when it had to cut.
func Cut(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if screen.Width(s) <= width {
		return s
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		w := screen.RuneWidth(r)
		if used+w > width-1 {
			break
		}
		b.WriteRune(r)
		used += w
	}
	b.WriteRune('…')
	return b.String()
}

// Pad is Cut, then spaces to exactly width cells.
func Pad(s string, width int) string {
	s = Cut(s, width)
	return s + strings.Repeat(" ", max(0, width-screen.Width(s)))
}

// PadLeft aligns s to the right of width cells: numbers.
func PadLeft(s string, width int) string {
	s = Cut(s, width)
	return strings.Repeat(" ", max(0, width-screen.Width(s))) + s
}

// BoxWidth is how wide a framed box is: what the screen wants, less when the
// window is narrower, never under 30 columns.
func BoxWidth(want, w int) int {
	return max(30, min(want, w-4))
}

// Frame draws a rounded, dim box with the title in its top edge.
func Frame(c *screen.Canvas, top, left, rows, cols int, title string) {
	if rows < 2 || cols < 2 {
		return
	}
	right, bottom := left+cols-1, top+rows-1
	c.Fill(top, left+1, 1, cols-2, '─', Dim)
	c.Fill(bottom, left+1, 1, cols-2, '─', Dim)
	c.Fill(top+1, left, rows-2, 1, '│', Dim)
	c.Fill(top+1, right, rows-2, 1, '│', Dim)
	c.Set(top, left, screen.Cell{R: '╭', Pen: Dim})
	c.Set(top, right, screen.Cell{R: '╮', Pen: Dim})
	c.Set(bottom, left, screen.Cell{R: '╰', Pen: Dim})
	c.Set(bottom, right, screen.Cell{R: '╯', Pen: Dim})
	if title != "" && cols > 6 {
		c.Print(top, left+2, " "+Cut(title, cols-6)+" ", Title)
	}
}

// Hint is one entry of a legend: the key, and what it does there.
type Hint struct {
	Key  string
	What string
}

// Legend draws the keys that work on this screen: key soft, meaning dim, a
// dot between. Entries that do not fit in width are left out whole.
// Returns the cells used.
func Legend(c *screen.Canvas, row, col, width int, hints ...Hint) int {
	at := col
	for i, h := range hints {
		sep := ""
		if i > 0 {
			sep = " • "
		}
		need := screen.Width(sep) + screen.Width(h.Key) + 1 + screen.Width(h.What)
		if at-col+need > width {
			break
		}
		at += c.Print(row, at, sep, Dim)
		at += c.Print(row, at, h.Key, Soft)
		at += c.Print(row, at, " "+h.What, Dim)
	}
	return at - col
}

// Tone says how a note reads.
type Tone int

const (
	Plain Tone = iota
	OK
	Warn
	Err
	Wait // a spinner in front; Spin picks its frame
)

var spinner = []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

// Spin is the spinner's frame, advanced by whoever ticks it.
var Spin int

// Note is the one line that says how it went. An empty text says nothing.
func Note(c *screen.Canvas, row, col, width int, tone Tone, text string) {
	if text == "" {
		return
	}
	pen := Text
	switch tone {
	case OK:
		pen.FG = COK
	case Warn:
		pen.FG = CWarn
	case Err:
		pen.FG = CErr
	case Wait:
		c.Set(row, col, screen.Cell{R: spinner[Spin%len(spinner)], Pen: Hot})
		c.Print(row, col+2, Cut(text, width-2), Soft)
		return
	}
	c.Print(row, col, Cut(text, width), pen)
}
