package screen

import "unicode/utf8"

// Key is a rune, or one of the synthetic codes below with modifiers or-ed
// in. The codes are filo-term's (src/keys.h): one vocabulary for the C host,
// the Filo programs and this package.
type Key rune

const (
	KeyUp      Key = 0xE000
	KeyDown    Key = 0xE001
	KeyRight   Key = 0xE002
	KeyLeft    Key = 0xE003
	KeyPgUp    Key = 0xE004
	KeyPgDn    Key = 0xE005
	KeyHome    Key = 0xE006
	KeyEnd     Key = 0xE007
	KeyDel     Key = 0xE008
	KeyIns     Key = 0xE009
	KeyBackTab Key = 0xE00B // Shift+Tab; not in keys.h, which has no use for it yet
	KeyEsc     Key = 0xE00F
	KeyF1      Key = 0xE080 // F1 to F12 are KeyF1 + 0 to 11

	ModShift Key = 0x10
	ModAlt   Key = 0x20
	ModCtrl  Key = 0x40

	KeyTab       Key = 9
	KeyEnter     Key = 13
	KeyBackspace Key = 127
)

// Ctrl is the key a control chord sends: Ctrl('w') is 23.
func Ctrl(r rune) Key {
	return Key(r & 0x1f)
}

// IsCode says k is a synthetic key and not a rune: the codes live in the
// first 256 of the private use area.
func (k Key) IsCode() bool {
	return k >= 0xE000 && k <= 0xE0FF
}

func (k Key) Base() Key {
	if !k.IsCode() {
		return k
	}
	return k &^ 0x70
}

// Printable says k is text to insert, not a key to obey.
func (k Key) Printable() bool {
	return !k.IsCode() && k >= ' ' && k != 0x7f
}

const csiMax = 32

// Decoder turns terminal bytes into keys. A lone ESC is ambiguous until the
// next byte, or the lack of one, says it was the key: the host calls Flush
// when the line has been quiet.
type Decoder struct {
	state int // 0 ground, 1 after ESC, 2 in CSI/SS3, 3 in a string, 4 ESC inside it
	ss3   bool
	csi   []byte
	utf   []byte
}

func (d *Decoder) Pending() bool {
	return d.state != 0 || len(d.utf) > 0
}

// Flush ends whatever the quiet line left open: a lone ESC is the key, a
// half sequence is dropped.
func (d *Decoder) Flush(emit func(Key)) {
	if d.state == 1 {
		emit(KeyEsc)
	}
	d.state = 0
	d.utf = d.utf[:0]
}

func (d *Decoder) Feed(b byte, emit func(Key)) {
	switch d.state {
	case 1:
		switch b {
		case '[', 'O':
			d.state, d.ss3, d.csi = 2, b == 'O', d.csi[:0]
			return
		case ']', 'P', 'X', '^', '_':
			// OSC/DCS/SOS/PM/APC on input is a terminal answering a query
			// nobody here made: swallowed.
			d.state = 3
			return
		}
		d.state = 0
		emit(KeyEsc)
		// b goes on as ordinary input: Alt+x arrives as ESC, x
	case 2:
		if d.ss3 || (b >= 0x40 && b <= 0x7e) {
			d.state = 0
			k := csiKey(d.csi, b)
			if k != 0 {
				emit(k) // unknown sequences are swallowed
			}
			return
		}
		if len(d.csi) < csiMax {
			d.csi = append(d.csi, b)
		}
		return
	case 3:
		switch b {
		case 0x07:
			d.state = 0
		case 0x1b:
			d.state = 4
		}
		return
	case 4:
		d.state = 3
		if b == '\\' {
			d.state = 0
		}
		return
	}
	if b == 0x1b {
		d.state = 1
		d.utf = d.utf[:0]
		return
	}
	d.rune(b, emit)
}

func (d *Decoder) rune(b byte, emit func(Key)) {
	if len(d.utf) == 0 && b < utf8.RuneSelf {
		emit(Key(b))
		return
	}
	d.utf = append(d.utf, b)
	if !utf8.FullRune(d.utf) {
		return
	}
	r, size := utf8.DecodeRune(d.utf)
	if r == utf8.RuneError && size <= 1 {
		// invalid: drop the first byte and try the rest again
		rest := append([]byte(nil), d.utf[1:]...)
		d.utf = d.utf[:0]
		for _, c := range rest {
			d.rune(c, emit)
		}
		return
	}
	d.utf = d.utf[:0]
	emit(Key(r))
}

// csiKey maps a finished CSI/SS3 sequence to a key; 0 is none. Parameters
// are "n" or "n;mod", mod being 1 + the modifier bits (shift 1, alt 2, ctrl 4).
func csiKey(params []byte, final byte) Key {
	var p [2]int
	np := 0
	for _, c := range params {
		if c == ';' {
			np++
			if np > 1 {
				break
			}
			continue
		}
		if c < '0' || c > '9' {
			return 0
		}
		p[np] = p[np]*10 + int(c-'0')
	}
	var mods Key
	if np >= 1 && p[1] >= 2 && p[1] <= 8 {
		mods = Key(p[1]-1) << 4 // #nosec G115 -- bounded to 1..7 by the test above
	}
	k := finalKey(final, p[0])
	if k == 0 {
		return 0
	}
	return k | mods
}

var fkeys = [14]Key{1, 2, 3, 4, 5, 0, 6, 7, 8, 9, 10, 0, 11, 12} // CSI 11~ to 24~

func finalKey(final byte, n int) Key {
	switch final {
	case 'A':
		return KeyUp
	case 'B':
		return KeyDown
	case 'C':
		return KeyRight
	case 'D':
		return KeyLeft
	case 'H':
		return KeyHome
	case 'F':
		return KeyEnd
	case 'Z':
		return KeyBackTab
	case 'P', 'Q', 'R', 'S':
		return KeyF1 + Key(final-'P')
	case '~':
		return tildeKey(n)
	}
	return 0
}

func tildeKey(n int) Key {
	switch n {
	case 1, 7:
		return KeyHome
	case 2:
		return KeyIns
	case 3:
		return KeyDel
	case 4, 8:
		return KeyEnd
	case 5:
		return KeyPgUp
	case 6:
		return KeyPgDn
	}
	if n >= 11 && n <= 24 && fkeys[n-11] != 0 {
		return KeyF1 + fkeys[n-11] - 1
	}
	return 0
}
