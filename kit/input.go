package kit

import "github.com/crgimenes/filo-term-go/screen"

// Input is one line of text being typed: runes, not bytes, so a multi-byte
// character is never split. Max caps the runes; 0 is no cap.
type Input struct {
	Max  int
	text []rune
	pos  int
	off  int // first rune shown, so the caret stays in view
}

func (in *Input) Set(s string) {
	in.text = []rune(s)
	if in.Max > 0 && len(in.text) > in.Max {
		in.text = in.text[:in.Max]
	}
	in.pos = len(in.text)
	in.off = 0
}

func (in *Input) String() string {
	return string(in.text)
}

// Key edits the line; false means the key is not the field's (Enter, Tab,
// ESC, the vertical arrows), for the screen to act on.
func (in *Input) Key(k screen.Key) bool {
	switch k {
	case screen.KeyLeft, screen.Ctrl('b'):
		in.pos = max(0, in.pos-1)
	case screen.KeyRight, screen.Ctrl('f'):
		in.pos = min(len(in.text), in.pos+1)
	case screen.KeyHome, screen.Ctrl('a'):
		in.pos = 0
	case screen.KeyEnd, screen.Ctrl('e'):
		in.pos = len(in.text)
	case screen.KeyBackspace, screen.Ctrl('h'):
		if in.pos > 0 {
			in.text = append(in.text[:in.pos-1], in.text[in.pos:]...)
			in.pos--
		}
	case screen.KeyDel, screen.Ctrl('d'):
		if in.pos < len(in.text) {
			in.text = append(in.text[:in.pos], in.text[in.pos+1:]...)
		}
	case screen.Ctrl('u'):
		in.text = in.text[in.pos:]
		in.pos = 0
	case screen.Ctrl('k'):
		in.text = in.text[:in.pos]
	default:
		if !k.Printable() {
			return false
		}
		if in.Max > 0 && len(in.text) >= in.Max {
			return true
		}
		in.text = append(in.text, 0)
		copy(in.text[in.pos+1:], in.text[in.pos:])
		in.text[in.pos] = rune(k)
		in.pos++
	}
	return true
}

// Draw paints the line in width cells on the field's ground; with focus it
// scrolls to keep the caret in view and shows the terminal cursor there.
func (in *Input) Draw(c *screen.Canvas, row, col, width int, p screen.Pen, focused bool) {
	c.Fill(row, col, 1, width, ' ', p)
	if width <= 0 {
		return
	}
	in.off = min(in.off, in.pos)
	for in.width(in.off, in.pos) >= width {
		in.off++
	}
	at := col
	for _, r := range in.text[in.off:] {
		w := screen.RuneWidth(r)
		if at+w > col+width {
			break
		}
		at += c.Print(row, at, string(r), p)
	}
	if focused {
		c.ShowCursor(row, col+in.width(in.off, in.pos))
	}
}

func (in *Input) width(from, to int) int {
	n := 0
	for _, r := range in.text[from:to] {
		n += screen.RuneWidth(r)
	}
	return n
}
