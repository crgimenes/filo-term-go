package kit

import (
	"slices"
	"strings"

	"github.com/crgimenes/filo-term-go/screen"
)

// FieldKind is what a form field holds.
type FieldKind int

const (
	TextField FieldKind = iota // one line, at most Max runes
	PickField                  // one of Options, turned with ←→ or space
	// yagni: kit.filo's "area" (a field of N lines, broken at words) comes
	// when a Go screen needs to take a paragraph.
)

type Field struct {
	Kind    FieldKind
	Label   string
	Width   int
	Max     int
	Options []string
	Value   string // a pick's current option; a text field's, at Open
	input   Input
}

// Form is the Clipper way in the editor's colours: every field on screen at
// once; ↑↓ and Tab move between fields and on to the buttons, Enter in a
// field goes to the next, ←→ move inside a field or turn a pick, a button
// fires with Enter or space; Ctrl-W is the first button, ESC cancels.
type Form struct {
	Fields  []Field
	Buttons []string
	Focus   int
	Note    string // the form's own word on what was typed (a refusal)
}

// Form.Key answers with one of these, or a button's label in lower case.
const (
	Moved  = "moved"
	Cancel = "cancel"
	None   = "none"
)

func NewForm(fields []Field, buttons ...string) *Form {
	f := &Form{Fields: fields, Buttons: buttons}
	for i := range f.Fields {
		fd := &f.Fields[i]
		fd.input.Max = fd.Max
		fd.input.Set(fd.Value)
		if fd.Kind == PickField && !slices.Contains(fd.Options, fd.Value) && len(fd.Options) > 0 {
			fd.Value = fd.Options[0]
		}
	}
	return f
}

// Value is what field i holds now.
func (f *Form) Value(i int) string {
	fd := &f.Fields[i]
	if fd.Kind == TextField {
		return fd.input.String()
	}
	return fd.Value
}

// Fail points at the field at fault and says why.
func (f *Form) Fail(i int, text string) {
	f.Focus = i
	f.Note = text
}

func (f *Form) count() int {
	return len(f.Fields) + len(f.Buttons)
}

func (f *Form) onField() bool {
	return f.Focus < len(f.Fields)
}

func (f *Form) move(d int) string {
	f.Focus = mod(f.Focus+d, f.count())
	return Moved
}

func (f *Form) button() string {
	return strings.ToLower(f.Buttons[f.Focus-len(f.Fields)])
}

func (f *Form) Key(k screen.Key) string {
	switch k {
	case screen.KeyEsc:
		return Cancel
	case screen.Ctrl('w'):
		if len(f.Buttons) == 0 {
			return None
		}
		return strings.ToLower(f.Buttons[0])
	case screen.KeyUp, screen.KeyBackTab:
		return f.move(-1)
	case screen.KeyDown, screen.KeyTab:
		return f.move(1)
	}
	if !f.onField() {
		if k == screen.KeyEnter || k == ' ' {
			return f.button()
		}
		switch k {
		case screen.KeyLeft:
			return f.move(-1)
		case screen.KeyRight:
			return f.move(1)
		}
		return None
	}
	fd := &f.Fields[f.Focus]
	if k == screen.KeyEnter {
		return f.move(1)
	}
	if fd.Kind == PickField {
		return f.turn(fd, k)
	}
	if fd.input.Key(k) {
		f.Note = ""
		return Moved
	}
	return None
}

func (f *Form) turn(fd *Field, k screen.Key) string {
	d := 0
	switch k {
	case screen.KeyLeft:
		d = -1
	case screen.KeyRight, ' ':
		d = 1
	}
	if d == 0 || len(fd.Options) == 0 {
		return None
	}
	at := slices.Index(fd.Options, fd.Value)
	fd.Value = fd.Options[mod(at+d, len(fd.Options))]
	return Moved
}

// Height is the rows a form takes: its fields, a gap, the buttons.
func (f *Form) Height() int {
	if len(f.Fields) == 0 {
		return 1
	}
	return len(f.Fields) + 2
}

// Draw paints the form from (top, left): labels right-aligned in labelw
// cells, fields beside them, the buttons a row under the last field, and
// the note under the buttons. Returns the row after the buttons.
func (f *Form) Draw(c *screen.Canvas, top, left, labelw int) int {
	col := left + labelw
	if labelw > 0 {
		col += 2
	}
	row := top
	for i := range f.Fields {
		fd := &f.Fields[i]
		c.Print(row, left+labelw-screen.Width(fd.Label), fd.Label, Soft)
		focused := i == f.Focus
		pen := Rest
		if focused {
			pen = Focus
		}
		switch fd.Kind {
		case TextField:
			fd.input.Draw(c, row, col, fd.Width, pen, focused)
		case PickField:
			c.Fill(row, col, 1, fd.Width, ' ', pen)
			c.Print(row, col, Cut("‹ "+fd.Value+" ›", fd.Width), pen)
		}
		row++
	}
	if len(f.Fields) > 0 {
		row++
	}
	at := col
	for i, b := range f.Buttons {
		pen := Rest
		if len(f.Fields)+i == f.Focus {
			pen = Focus
			pen.Bold = true
		}
		at += c.Print(row, at, "  "+b+"  ", pen) + 2
	}
	return row + 1
}
