//go:build readiness_eval && (aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos)

// Package readinessio contains test-only bounded-evaluation I/O helpers.
package readinessio

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// OpenRegular opens only a regular file without following a final symlink or
// blocking on a FIFO. The caller is responsible for bounding its reads.
func OpenRegular(path string) (*os.File, error) {
	return openRegular(path, unix.O_RDONLY)
}

// OpenAppendRegular opens an existing regular file for nonblocking append.
func OpenAppendRegular(path string) (*os.File, error) {
	return openRegular(path, unix.O_WRONLY|unix.O_APPEND)
}

func openRegular(path string, flags int) (*os.File, error) {
	fd, err := unix.Open(path, flags|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, errors.New("cannot construct readiness file")
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, errors.New("readiness input is not a regular file")
	}
	return file, nil
}
