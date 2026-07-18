package cli

import (
	"context"
	"fmt"

	"github.com/tasuku43/agentic-cli-foundry/internal/domain/operation"
)

func runVersion(ctx context.Context, c *CLI, command CommandSpec, _ operation.Intent, args []string) int {
	if len(args) != 0 {
		return c.failUsage(ctx, "invalid_arguments", "usage: "+command.Usage(), "help version", "Run version without command arguments.")
	}
	if c.Commit == "" {
		return c.emit(ctx, []byte(fmt.Sprintf("%s %s\n", ProgramName, c.Version)))
	}
	return c.emit(ctx, []byte(fmt.Sprintf("%s %s (%s)\n", ProgramName, c.Version, c.Commit)))
}
