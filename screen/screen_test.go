package screen

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func decode(in string, flush bool) []Key {
	var d Decoder
	var out []Key
	emit := func(k Key) { out = append(out, k) }
	for i := range len(in) {
		d.Feed(in[i], emit)
	}
	if flush {
		d.Flush(emit)
	}
	return out
}

func TestDecoder(t *testing.T) {
	cases := []struct {
		in    string
		flush bool
		want  []Key
	}{
		{"a", false, []Key{'a'}},
		{"\x1b[A\x1b[B\x1b[C\x1b[D", false, []Key{KeyUp, KeyDown, KeyRight, KeyLeft}},
		{"\x1bOA", false, []Key{KeyUp}},
		{"\x1b[1;5C", false, []Key{KeyRight | ModCtrl}},
		{"\x1b[1;2A", false, []Key{KeyUp | ModShift}},
		{"\x1b[5~\x1b[6~\x1b[3~\x1b[H\x1b[F", false, []Key{KeyPgUp, KeyPgDn, KeyDel, KeyHome, KeyEnd}},
		{"\x1b[15~\x1b[24~\x1bOP", false, []Key{KeyF1 + 4, KeyF1 + 11, KeyF1}},
		{"\x1b[Z", false, []Key{KeyBackTab}},
		{"ação 日本 🎉", false, []Key{'a', 'ç', 'ã', 'o', ' ', '日', '本', ' ', '🎉'}},
		{"\x1b", false, nil},
		{"\x1b", true, []Key{KeyEsc}},
		{"\x1bx", false, []Key{KeyEsc, 'x'}},
		{"\x1b]11;rgb:0/0/0\x07q", false, []Key{'q'}},
		{"\x1b[?99x", false, nil},
		{"\xff" + "b", false, []Key{'b'}},
		{"\r\t\x7f\x17", false, []Key{KeyEnter, KeyTab, KeyBackspace, Ctrl('w')}},
	}
	for _, c := range cases {
		got := decode(c.in, c.flush)
		if !slicesEqual(got, c.want) {
			t.Errorf("decode(%q) = %x, want %x", c.in, got, c.want)
		}
	}
}

func TestDecoderSplitSequence(t *testing.T) {
	var d Decoder
	var out []Key
	emit := func(k Key) { out = append(out, k) }
	d.Feed(0x1b, emit)
	if !d.Pending() {
		t.Fatal("ESC alone should be pending")
	}
	d.Feed('[', emit)
	d.Feed('A', emit)
	if len(out) != 1 || out[0] != KeyUp {
		t.Fatalf("got %x", out)
	}
}

func slicesEqual(a, b []Key) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPrintWide(t *testing.T) {
	c := New(6, 1)
	n := c.Print(0, 0, "日x本", Plain)
	if n != 5 {
		t.Fatalf("width %d, want 5", n)
	}
	if got := c.String(); got != "日x本\n" {
		t.Fatalf("%q", got)
	}
	c.Clear()
	c.Print(0, 4, "ab日", Plain) // the wide glyph lands on the edge
	if got := c.String(); got != "    ab\n" {
		t.Fatalf("%q", got)
	}
}

// replay applies the renderer's output to a grid of runes and fails on any
// sequence it does not know: what the renderer emits must be what it means.
func replay(t *testing.T, out []byte, w, h int) *Canvas {
	t.Helper()
	c := New(w, h)
	row, col := 0, 0
	pen := Plain
	for len(out) > 0 {
		if out[0] != 0x1b {
			r, size := utf8.DecodeRune(out)
			out = out[size:]
			c.Set(row, col, Cell{R: r, Pen: pen})
			if runeWidth(r) == 2 {
				c.Set(row, col+1, Cell{R: continuation, Pen: pen})
			}
			col += runeWidth(r)
			continue
		}
		end := bytes.IndexFunc(out[2:], func(r rune) bool { return r >= 0x40 && r <= 0x7e })
		if out[1] != '[' || end < 0 {
			t.Fatalf("unknown escape %q", out)
		}
		params, final := string(out[2:2+end]), out[2+end]
		out = out[3+end:]
		switch {
		case final == 'H':
			r, cc, _ := strings.Cut(params, ";")
			row, _ = strconv.Atoi(r)
			col, _ = strconv.Atoi(cc)
			row, col = row-1, col-1
		case final == 'm':
			pen = Plain
			f := strings.Split(params, ";")
			for i := 0; i < len(f); i++ {
				switch f[i] {
				case "0":
				case "1":
					pen.Bold = true
				case "38", "48":
					n, _ := strconv.Atoi(f[i+2])
					if f[i] == "38" {
						pen.FG = Color(n)
					} else {
						pen.BG = Color(n)
					}
					i += 2
				default:
					t.Fatalf("unknown SGR %q", params)
				}
			}
		case final == 'J' && params == "2":
			c.Clear()
		case (final == 'l' || final == 'h') && params == "?25":
		default:
			t.Fatalf("unknown sequence %q%c", params, final)
		}
	}
	return c
}

func TestRendererDiff(t *testing.T) {
	var out bytes.Buffer
	r := NewRenderer(&out)
	back := New(20, 3)
	back.Print(0, 0, "hello 日本", Pen{FG: 214, BG: Default, Bold: true})
	back.Print(2, 3, "world", Pen{FG: 0, BG: 236})
	err := r.Draw(back)
	if err != nil {
		t.Fatal(err)
	}
	first := out.Bytes()
	got := replay(t, first, 20, 3)
	if got.String() != back.String() {
		t.Fatalf("replay\n%s\nwant\n%s", got, back)
	}
	for i := range back.Cells {
		if got.Cells[i] != back.Cells[i] {
			t.Fatalf("cell %d: %+v, want %+v", i, got.Cells[i], back.Cells[i])
		}
	}

	out.Reset()
	back.Print(2, 3, "W", Pen{FG: 0, BG: 236})
	err = r.Draw(back)
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() > 40 {
		t.Fatalf("one changed cell cost %d bytes: %q", out.Len(), out.Bytes())
	}
	if !bytes.Contains(out.Bytes(), []byte("\x1b[3;4H")) || !bytes.Contains(out.Bytes(), []byte("W")) {
		t.Fatalf("diff missed the cell: %q", out.Bytes())
	}
}

func TestRendererWideReplacedByNarrow(t *testing.T) {
	var out bytes.Buffer
	r := NewRenderer(&out)
	back := New(4, 1)
	back.Print(0, 0, "日", Plain)
	_ = r.Draw(back)
	out.Reset()
	back.Clear()
	back.Print(0, 0, "ab", Plain)
	_ = r.Draw(back)
	if !bytes.Contains(out.Bytes(), []byte("ab")) {
		t.Fatalf("right half of the old glyph not rewritten: %q", out.Bytes())
	}
}

func TestRendererIdleFrameIsFree(t *testing.T) {
	var out bytes.Buffer
	r := NewRenderer(&out)
	back := New(10, 2)
	back.Print(0, 0, "x", Plain)
	back.ShowCursor(1, 2)
	_ = r.Draw(back)
	out.Reset()
	_ = r.Draw(back)
	if out.Len() != 0 {
		t.Fatalf("an unchanged frame sent %q", out.Bytes())
	}
}
