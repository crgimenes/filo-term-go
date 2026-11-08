//go:build !unix

package screen

import (
	"os"
	"sync"
	"time"
)

type chunk struct {
	b   []byte
	err error
}

var (
	readerOnce sync.Once
	chunks     = make(chan chunk, 16)
)

// readQuiet without select: a goroutine blocks on the console and hands
// over what it read. yagni: it keeps reading while Suspend has handed the
// console to an editor; a cancelable console read fixes that when a Windows
// user needs it.
func readQuiet(f *os.File, buf []byte, d time.Duration) (int, error) {
	readerOnce.Do(func() {
		go func() {
			for {
				b := make([]byte, len(buf))
				n, err := f.Read(b)
				chunks <- chunk{b[:n], err}
				if err != nil {
					return
				}
			}
		}()
	})
	select {
	case c := <-chunks:
		return copy(buf, c.b), c.err
	case <-time.After(d):
		return 0, nil
	}
}
