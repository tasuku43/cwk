//go:build readiness_eval && !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos)

// Package readinessio contains test-only bounded-evaluation I/O helpers.
package readinessio

import (
	"errors"
	"os"
)

// OpenRegular rejects observed links and non-regular files. On this platform
// the check and open are not atomic, so the runner must remain disabled until
// a platform-specific no-follow, non-blocking implementation is verified.
func OpenRegular(path string) (*os.File, error) {
	return openRegular(path, os.O_RDONLY)
}

// OpenAppendRegular opens an existing regular file for append.
func OpenAppendRegular(path string) (*os.File, error) {
	return openRegular(path, os.O_WRONLY|os.O_APPEND)
}

func openRegular(path string, flags int) (*os.File, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() {
		return nil, errors.New("readiness input is not a regular file")
	}
	file, err := os.OpenFile(path, flags, 0)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		_ = file.Close()
		return nil, errors.New("readiness input changed while opening")
	}
	return file, nil
}
