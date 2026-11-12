//go:build readiness_eval && !(darwin || linux)

package readinessio

import (
	"errors"
	"os"
)

// LockJournal fails closed until this platform has a verified lock.
func LockJournal(_ string) (*os.File, error) {
	return nil, errors.New("bounded readiness journal unsupported on this platform")
}

func AppendLocked(_ *os.File, _ []byte, _ int64) error {
	return errors.New("bounded readiness journal unsupported on this platform")
}

func AppendBounded(_ string, _ []byte, _ int64) error {
	return errors.New("bounded readiness journal unsupported on this platform")
}
