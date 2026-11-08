# filo-term-go

The screen kit of [filo-term](https://github.com/crgimenes/filo-term) for Go
programs: the same palette, frame, menu, form, table and legend that `edt`
and the BBS draw with, called as a Go API instead of from Filo.

- `screen` — the host: raw mode and the alternate screen, keys decoded from
  the terminal's bytes (the key codes are filo-term's `keys.h`), a grid of
  cells, and a renderer that sends only the cells that changed.
- `kit` — what a screen draws with: `Frame`, `Legend`, `Note`, `Menu`,
  `Form` (Clipper-style: every field on screen, ↑↓/Tab between fields, Enter
  advances, Ctrl-W saves, ESC cancels), `Table`, `Input`.

Dependencies: `golang.org/x/term` and `golang.org/x/sys`. No cgo.

```go
t, err := screen.Open()
if err != nil {
	log.Fatal(err)
}
defer t.Close()

c := screen.New(t.Size())
menu := kit.Menu{}
items := []kit.Item{{Key: "o", Label: "Open — a file"}, {Key: "q", Label: "Quit"}}
for {
	w, h := t.Size()
	c.Resize(w, h)
	kit.Frame(c, 0, 0, h-1, w, "Example")
	menu.Draw(c, 2, 4, items, 1, 30)
	kit.Legend(c, h-1, 1, w-2, kit.Hint{Key: "↑↓", What: "move"}, kit.Hint{Key: "enter", What: "open"})
	_ = t.Draw(c)

	ev, ok := <-t.Events()
	if !ok || ev.Key == screen.KeyEsc {
		return
	}
	if menu.Key(ev.Key, items, 1) == 1 {
		return
	}
}
```

`Canvas.String()` returns the screen as text, so a test (or an agent) reads
a screen without a terminal.

`Terminal.Suspend` and `Resume` hand the terminal to another program, such
as `$EDITOR`, without the reader stealing its keys (on Unix).

## License

MIT
