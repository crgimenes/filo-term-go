//go:build !unix && !windows

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

func checkFd(int) error { return nil }

// readQuiet without select, for platforms with neither select nor a
// waitable console: a goroutine blocks on the input and hands over what it
// read. It keeps reading while Suspend has handed the terminal over; no
// supported target uses this file.
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
