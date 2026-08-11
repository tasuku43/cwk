package cli

import (
	"context"
	"io"

	"github.com/tasuku43/cwk/internal/infra/streaminput"
)

// readContextBounded stops at the exact caller-owned byte ceiling and makes
// production pipe/terminal reads interruptible without abandoning a goroutine
// that could consume a later invocation's stdin.
func readContextBounded(ctx context.Context, reader io.Reader, limit int) ([]byte, error) {
	return streaminput.ReadBounded(ctx, reader, limit)
}
