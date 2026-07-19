package cli

import (
	"context"

	"github.com/tasuku43/cwk/internal/domain/fault"
)

// emit performs exactly one checked write after a command has rendered and
// validated its complete output in memory.
func (c *CLI) emit(ctx context.Context, output []byte) int {
	if err := ctx.Err(); err != nil {
		return c.fail(ctx, err)
	}
	return c.emitComplete(ctx, output)
}

// emitMutationResult writes a result after a mutation action has returned
// confirmed success. It deliberately does not reclassify that success as a
// retryable cancellation if the context becomes done after the action.
func (c *CLI) emitMutationResult(ctx context.Context, output []byte) int {
	return c.emitComplete(ctx, output)
}

func (c *CLI) emitComplete(ctx context.Context, output []byte) int {
	if _, err := writeOnce(c.Out, output); err != nil {
		return c.fail(ctx, fault.Wrap(
			fault.KindInternal,
			"output_write_failed",
			"コマンド出力を完全に書き込めませんでした。",
			true,
			err,
			fault.NextAction{Command: invocationCommandPath(ctx), Reason: "書き込み可能な出力ストリームで再試行してください。"},
		))
	}
	return ExitOK
}
