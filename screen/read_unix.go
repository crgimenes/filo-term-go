//go:build unix

package screen

import (
	"errors"
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// readQuiet waits up to d for input and reads what came; 0 bytes means the
// line was quiet. select, not poll: macOS poll refuses terminal devices.
func readQuiet(f *os.File, buf []byte, d time.Duration) (int, error) {
	fd := int(f.Fd())
	var set unix.FdSet
	set.Set(fd)
	tv := unix.NsecToTimeval(d.Nanoseconds())
	_, err := unix.Select(fd+1, &set, nil, nil, &tv)
	if errors.Is(err, unix.EINTR) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !set.IsSet(fd) {
		return 0, nil
	}
	n, err := f.Read(buf)
	if n == 0 && err == nil {
		err = io.EOF
	}
	return n, err
}
