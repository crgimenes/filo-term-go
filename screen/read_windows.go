//go:build windows

package screen

import (
	"io"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	peekConsoleInput = kernel32.NewProc("PeekConsoleInputW")
	readConsoleInput = kernel32.NewProc("ReadConsoleInputW")
)

// inputRecord is INPUT_RECORD: an event type and a 16-byte union whose
// first field, for a KEY_EVENT, is bKeyDown.
type inputRecord struct {
	eventType uint16
	_         uint16
	event     [16]byte
}

func checkFd(int) error { return nil }

// readQuiet waits up to d on the console handle and reads only when a key
// press is queued, so the read cannot block past the wait: the console
// queues focus, mouse and key-release events too, which signal the handle
// but give a read nothing, and those are taken off the queue here.
//
// PeekConsoleInputW: https://learn.microsoft.com/windows/console/peekconsoleinput
func readQuiet(f *os.File, buf []byte, d time.Duration) (int, error) {
	h := windows.Handle(f.Fd())
	ev, err := windows.WaitForSingleObject(h, uint32(d.Milliseconds())) // #nosec G115 -- d is the 50 ms quiet window
	if err != nil {
		return 0, err
	}
	if ev == uint32(windows.WAIT_TIMEOUT) {
		return 0, nil
	}
	var recs [16]inputRecord
	var n uint32
	// #nosec G103 -- the documented way to pass INPUT_RECORD buffers to kernel32
	ok, _, err := peekConsoleInput.Call(uintptr(h), uintptr(unsafe.Pointer(&recs[0])), uintptr(len(recs)), uintptr(unsafe.Pointer(&n)))
	if ok == 0 {
		return 0, err
	}
	for _, r := range recs[:n] {
		keyDown := *(*int32)(unsafe.Pointer(&r.event[0])) != 0 // #nosec G103 -- bKeyDown, the union's first field
		if r.eventType == windows.KEY_EVENT && keyDown {
			got, err := f.Read(buf)
			if got == 0 && err == nil {
				err = io.EOF
			}
			return got, err
		}
	}
	// nothing a read would return: drop those events so the handle stops
	// signalling for them
	// #nosec G103 -- as above
	ok, _, err = readConsoleInput.Call(uintptr(h), uintptr(unsafe.Pointer(&recs[0])), uintptr(n), uintptr(unsafe.Pointer(&n)))
	if ok == 0 {
		return 0, err
	}
	return 0, nil
}
