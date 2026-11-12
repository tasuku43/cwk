//go:build readiness_eval && (darwin || linux)

package readinessio

import (
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// LockJournal grants an exclusive advisory lock until the returned file closes.
// It serializes cooperating helpers; it does not prevent an untrusted writer.
func LockJournal(path string) (*os.File, error) {
	file, err := openRegular(path, unix.O_RDWR|unix.O_APPEND)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

// AppendLocked checks and appends using a caller-held journal lock.
func AppendLocked(file *os.File, data []byte, limit int64) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if limit < 0 || info.Size() > limit || int64(len(data)) > limit-info.Size() {
		return errors.New("invocation journal byte limit exceeded")
	}
	n, err := file.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

// AppendBounded serializes cooperating writers before checking and appending.
func AppendBounded(path string, data []byte, limit int64) error {
	file, err := LockJournal(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return AppendLocked(file, data, limit)
}
