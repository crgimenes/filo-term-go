package screen

import (
	"errors"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// Event is a key, or Resize with no key when the window changed size.
type Event struct {
	Key    Key
	Resize bool
}

// How long the line stays quiet before a lone ESC counts as the key. Long
// enough for an arrow's bytes to arrive apart over ssh.
const quiet = 50 * time.Millisecond

// Terminal owns the controlling terminal while a program runs: raw mode,
// the alternate screen, autowrap off, the keys as events.
type Terminal struct {
	in     *os.File
	out    *os.File
	old    *term.State
	r      *Renderer
	events chan Event
	done   chan struct{}

	mu     sync.Mutex // the reader holds it across one wait and one read
	paused bool

	sizeMu sync.Mutex // apart from mu, so Size never waits out a read
	w, h   int
}

var ErrNotTerminal = errors.New("not a terminal")

func Open() (*Terminal, error) {
	t := &Terminal{
		in:     os.Stdin,
		out:    os.Stdout,
		events: make(chan Event, 64),
		done:   make(chan struct{}),
	}
	if !term.IsTerminal(int(t.in.Fd())) || !term.IsTerminal(int(t.out.Fd())) {
		return nil, ErrNotTerminal
	}
	t.r = NewRenderer(t.out)
	err := t.enter()
	if err != nil {
		return nil, err
	}
	t.w, t.h = t.size()
	go t.loop()
	return t, nil
}

func (t *Terminal) enter() error {
	old, err := term.MakeRaw(int(t.in.Fd()))
	if err != nil {
		return err
	}
	t.old = old
	_, err = t.out.WriteString("\x1b[?1049h\x1b[?7l\x1b[?25l")
	t.r.Invalidate()
	return err
}

func (t *Terminal) leave() error {
	_, err := t.out.WriteString("\x1b[0m\x1b[?25h\x1b[?7h\x1b[?1049l")
	if t.old != nil {
		err = errors.Join(err, term.Restore(int(t.in.Fd()), t.old))
	}
	return err
}

// Close gives the terminal back as it was found.
func (t *Terminal) Close() error {
	close(t.done)
	t.mu.Lock() // the reader is between reads, and will not read again
	defer t.mu.Unlock()
	return t.leave()
}

// Suspend hands the terminal to another program (an editor, a pager) until
// Resume; no key is read meanwhile, so none is stolen from it.
func (t *Terminal) Suspend() error {
	t.mu.Lock()
	t.paused = true
	t.mu.Unlock()
	return t.leave()
}

// Resume takes the terminal back; the next Draw repaints everything.
func (t *Terminal) Resume() error {
	err := t.enter()
	t.mu.Lock()
	t.paused = false
	t.mu.Unlock()
	return err
}

// Events closes when the terminal is gone (EOF, a hangup).
func (t *Terminal) Events() <-chan Event {
	return t.events
}

func (t *Terminal) Size() (w, h int) {
	t.sizeMu.Lock()
	defer t.sizeMu.Unlock()
	return t.w, t.h
}

func (t *Terminal) Draw(c *Canvas) error {
	return t.r.Draw(c)
}

func (t *Terminal) size() (w, h int) {
	w, h, err := term.GetSize(int(t.out.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

func (t *Terminal) loop() {
	defer close(t.events)
	var d Decoder
	var keys []Key
	collect := func(k Key) { keys = append(keys, k) }
	buf := make([]byte, 256)
	for {
		select {
		case <-t.done:
			return
		default:
		}
		t.mu.Lock()
		if t.paused {
			t.mu.Unlock()
			time.Sleep(quiet)
			continue
		}
		n, err := readQuiet(t.in, buf, quiet)
		t.mu.Unlock()
		if err != nil {
			return
		}
		w, h := t.size()
		t.sizeMu.Lock()
		resized := w != t.w || h != t.h
		t.w, t.h = w, h
		t.sizeMu.Unlock()

		keys = keys[:0]
		for _, b := range buf[:n] {
			d.Feed(b, collect)
		}
		if n == 0 && d.Pending() {
			d.Flush(collect)
		}
		if resized {
			keys = append(keys, 0)
		}
		for _, k := range keys {
			ev := Event{Key: k, Resize: k == 0}
			select {
			case t.events <- ev:
			case <-t.done:
				return
			}
		}
	}
}
