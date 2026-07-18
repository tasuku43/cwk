//go:build darwin

package browseropen

import (
	"context"
	"os/exec"
)

func platformLaunch(ctx context.Context, raw string) error {
	// #nosec G204 -- the executable is fixed and raw passed the exact Chatwork
	// authorization URL validator before this unexported boundary is called.
	return exec.CommandContext(ctx, "/usr/bin/open", raw).Run()
}
