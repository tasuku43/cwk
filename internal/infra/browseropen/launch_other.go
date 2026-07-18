//go:build !darwin && !linux && !windows

package browseropen

import "context"

func platformLaunch(context.Context, string) error {
	return ErrUnavailable
}
