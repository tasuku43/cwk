//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows && !zos

package terminalui

import (
	"context"
	"errors"
)

func readTerminal(ctx context.Context, _ int, _ []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return 0, errors.New("terminalui: context-aware terminal input is unsupported on this platform")
}
