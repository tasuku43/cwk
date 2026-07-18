package cli

import (
	"context"

	"github.com/tasuku43/agentic-cli-foundry/internal/domain/fault"
)

// emit performs exactly one checked write after a command has rendered and
// validated its complete output in memory.
func (c *CLI) emit(ctx context.Context, output []byte) int {
	if err := ctx.Err(); err != nil {
		return c.fail(ctx, err)
	}
	if _, err := writeOnce(c.Out, output); err != nil {
		return c.fail(ctx, fault.Wrap(
			fault.KindInternal,
			"output_write_failed",
			"The command output could not be written completely.",
			true,
			err,
			fault.NextAction{Command: invocationCommandPath(ctx), Reason: "Retry with a writable output stream."},
		))
	}
	return ExitOK
}
