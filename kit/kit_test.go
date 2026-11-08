package kit

import (
	"strings"
	"testing"

	"github.com/crgimenes/filo-term-go/screen"
)

func keys(s string) []screen.Key {
	var out []screen.Key
	for _, r := range s {
		out = append(out, screen.Key(r))
	}
	return out
}

func TestCut(t *testing.T) {
	cases := []struct {
		in   string
		w    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"日本語", 4, "日…"},
		{"ação", 3, "aç…"},
		{"x", 0, ""},
	}
	for _, c := range cases {
		got := Cut(c.in, c.w)
		if got != c.want || screen.Width(got) > max(c.w, 0) {
			t.Errorf("Cut(%q, %d) = %q, want %q", c.in, c.w, got, c.want)
		}
	}
}

func TestFrameAndLegend(t *testing.T) {
	c := screen.New(30, 5)
	Frame(c, 0, 0, 4, 30, "Tables")
	Legend(c, 4, 1, 28, Hint{"↑↓", "move"}, Hint{"enter", "open"}, Hint{"esc", "back"})
	want := "" +
		"╭─ Tables ───────────────────╮\n" +
		"│                            │\n" +
		"│                            │\n" +
		"╰────────────────────────────╯\n" +
		" ↑↓ move • enter open\n"
	if got := c.String(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if c.At(0, 0).FG != CDim || c.At(0, 3).Pen != Title {
		t.Fatal("frame colours")
	}
}

func TestLegendDropsWholeEntries(t *testing.T) {
	c := screen.New(20, 1)
	Legend(c, 0, 0, 14, Hint{"q", "quit"}, Hint{"enter", "open"})
	if got := c.String(); got != "q quit\n" {
		t.Fatalf("%q", got)
	}
}

func TestInputRunes(t *testing.T) {
	var in Input
	for _, k := range keys("açã") {
		in.Key(k)
	}
	in.Key(screen.KeyLeft)
	in.Key(screen.KeyBackspace)
	in.Key('x')
	if in.String() != "axã" {
		t.Fatalf("%q", in.String())
	}
	in.Key(screen.Ctrl('u'))
	if in.String() != "ã" {
		t.Fatalf("ctrl-u: %q", in.String())
	}
	if in.Key(screen.KeyEnter) || in.Key(screen.KeyUp) {
		t.Fatal("Enter and Up belong to the screen")
	}
	in.Max = 2
	in.Set("abcd")
	if in.String() != "ab" {
		t.Fatalf("max: %q", in.String())
	}
}

func TestInputScrollsToCaret(t *testing.T) {
	var in Input
	in.Set("0123456789abcdef")
	c := screen.New(8, 1)
	in.Draw(c, 0, 0, 8, Rest, true)
	if got := c.String(); got != "9abcdef\n" || c.CursorCol != 7 {
		t.Fatalf("%q cursor %d", got, c.CursorCol)
	}
}

func TestMenu(t *testing.T) {
	items := []Item{{"1", "First — the one"}, {"2", "Second"}, {"q", "Quit"}}
	var m Menu
	c := screen.New(40, 3)
	m.Draw(c, 0, 2, items, 1, 36)
	want := "› 1 First — the one\n  2 Second\n  Quit\n"
	if got := c.String(); got != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
	if c.At(2, 2).Pen != Hot {
		t.Fatal("the hot letter of a joined word is orange")
	}
	if m.Key(screen.KeyDown, items, 1) != -1 || m.At != 1 {
		t.Fatal("down moves")
	}
	if m.Key(screen.KeyEnter, items, 1) != 1 {
		t.Fatal("enter opens the pointed item")
	}
	if m.Key('Q', items, 1) != 2 {
		t.Fatal("a letter opens its item, any case")
	}
	if m.Key('z', items, 1) != -2 {
		t.Fatal("a stray letter means nothing")
	}
	if m.Key(screen.KeyUp, items, 1) != -1 || m.At != 1 {
		t.Fatal("up from the last")
	}
}

func TestForm(t *testing.T) {
	f := NewForm([]Field{
		{Kind: TextField, Label: "Name", Width: 10, Max: 20, Value: "ana"},
		{Kind: PickField, Label: "Level", Width: 12, Options: []string{"user", "sysop"}},
	}, "Save", "Cancel")
	for _, k := range keys("!") {
		f.Key(k)
	}
	if f.Value(0) != "ana!" {
		t.Fatalf("typing: %q", f.Value(0))
	}
	if f.Key(screen.KeyEnter) != Moved || f.Focus != 1 {
		t.Fatal("enter goes to the next field")
	}
	f.Key(screen.KeyRight)
	if f.Value(1) != "sysop" {
		t.Fatalf("pick: %q", f.Value(1))
	}
	f.Key(screen.KeyEnter)
	if f.Key(screen.KeyRight) != Moved || f.Focus != 3 {
		t.Fatal("arrows walk the buttons")
	}
	if f.Key(screen.KeyEnter) != "cancel" {
		t.Fatal("enter fires the focused button")
	}
	if f.Key(screen.Ctrl('w')) != "save" || f.Key(screen.KeyEsc) != Cancel {
		t.Fatal("ctrl-w is the first button, esc cancels")
	}

	f.Focus = 0
	c := screen.New(40, 5)
	end := f.Draw(c, 0, 0, 6)
	lines := strings.Split(c.String(), "\n")
	if lines[0] != "  Name  ana!" || lines[1] != " Level  ‹ sysop ›" || lines[3] != "          Save      Cancel" {
		t.Fatalf("draw:\n%s", c)
	}
	if end != 4 || f.Height() != 4 {
		t.Fatalf("end %d height %d", end, f.Height())
	}
	if c.At(0, 8).Pen != Focus || c.At(1, 8).Pen != Rest || !c.CursorOn {
		t.Fatal("focus colours and caret")
	}
}

func TestTableWindow(t *testing.T) {
	var tb Table
	cols := []Col{{"#", 3}, {"name", 6}}
	rowOf := func(i int) []string { return []string{string(rune('a' + i)), "row"} }
	draw := func() string {
		c := screen.New(12, 6)
		tb.Draw(c, 0, 0, cols, 10, rowOf, 4)
		return c.String()
	}
	got := draw()
	if !strings.HasPrefix(got, "#   name\n──────────\na   row\nb   row\nc   row\n+7 more") {
		t.Fatalf("first page:\n%s", got)
	}
	for range 3 {
		tb.Move(screen.KeyDown, 10, 3)
	}
	got = draw()
	if !strings.Contains(got, "d   row\n+6 more") || !strings.HasPrefix(strings.Split(got, "\n")[2], "b") {
		t.Fatalf("cursor on d must be on screen:\n%s", got)
	}
	tb.Move(screen.KeyEnd, 10, 3)
	got = draw()
	if !strings.Contains(got, "j   row") || strings.Contains(got, "more") {
		t.Fatalf("end:\n%s", got)
	}
}

func TestDegenerateInputs(t *testing.T) {
	c := screen.New(20, 4)
	var m Menu
	m.Draw(c, 0, 2, []Item{{"a", "Alpha"}}, 0, 10)
	m.Key(screen.KeyDown, []Item{{"a", "Alpha"}}, 0)
	m.Draw(c, 0, 2, nil, 0, 10)

	f := NewForm(nil)
	for _, k := range []screen.Key{screen.KeyEnter, ' ', screen.KeyDown, screen.KeyRight, screen.Ctrl('w')} {
		if f.Key(k) == "" {
			t.Fatal("empty answer")
		}
	}

	var tb Table
	rowOf := func(i int) []string { return []string{"r"} }
	c = screen.New(10, 4)
	tb.Draw(c, 0, 0, []Col{{"h", 3}}, 2, rowOf, 1)
	if strings.Count(c.String(), "r") != 1 || strings.Contains(c.String(), "more") {
		t.Fatalf("height 1 must take one row:\n%s", c)
	}
	c = screen.New(10, 4)
	tb.Draw(c, 0, 0, []Col{{"h", 3}}, 2, rowOf, 0)
	if strings.Contains(c.String(), "r\n") {
		t.Fatalf("height 0 draws no rows:\n%s", c)
	}
}

func TestFormDrawsItsNote(t *testing.T) {
	f := NewForm([]Field{{Kind: TextField, Label: "Name", Width: 8}}, "Save")
	f.Fail(0, "required")
	c := screen.New(30, 6)
	end := f.Draw(c, 0, 0, 4)
	if !strings.Contains(c.String(), "required") || end != f.Height() {
		t.Fatalf("end %d height %d\n%s", end, f.Height(), c)
	}
}

func TestMenuLabelKeepsToItsColumn(t *testing.T) {
	c := screen.New(20, 1)
	var m Menu
	m.Draw(c, 0, 2, []Item{{"a", "abcdefghij"}, {"b", "bb"}}, 2, 6)
	if got := c.String(); got != "› abc…  bb\n" {
		t.Fatalf("%q", got)
	}
}
