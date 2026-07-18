// Package cli owns command routing and presentation.
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/doctorcmd"
	"github.com/tasuku43/cwk/internal/app/samplecmd"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
	"github.com/tasuku43/cwk/internal/infra/sampledata"
	"github.com/tasuku43/cwk/internal/infra/systemdoctor"
)

// CLI contains injected streams and application services.
type CLI struct {
	In      io.Reader
	Out     io.Writer
	Err     io.Writer
	Version string
	Commit  string

	catalog         Catalog
	doctor          *doctorcmd.Service
	samples         *samplecmd.Service
	chatwork        *chatworkcmd.Service
	chatworkAuth    *appauthn.Gate
	chatworkInitErr error
}

// New builds the production CLI with offline template adapters.
func New(in io.Reader, out, errOut io.Writer) *CLI {
	cli := newCLI(in, out, errOut, DefaultCatalog(), systemdoctor.New())
	client, err := chatworkapi.NewFromEnvironment()
	if err != nil {
		cli.chatworkInitErr = err
		return cli
	}
	cli.chatwork = chatworkcmd.New(client)
	cli.chatworkAuth = appauthn.New(client)
	return cli
}

func newCLI(in io.Reader, out, errOut io.Writer, catalog Catalog, inspector doctorcmd.InspectorPort) *CLI {
	return newCLIWithSamples(in, out, errOut, catalog, inspector, sampledata.New())
}

func newCLIWithSamples(
	in io.Reader,
	out, errOut io.Writer,
	catalog Catalog,
	inspector doctorcmd.InspectorPort,
	repository samplecmd.RepositoryPort,
) *CLI {
	if in == nil {
		in = strings.NewReader("")
	}
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}
	return &CLI{
		In: in, Out: out, Err: errOut,
		Version: "dev",
		catalog: catalog,
		doctor:  doctorcmd.New(inspector),
		samples: samplecmd.New(repository),
	}
}

// RunContext validates global options and the catalog, resolves one command,
// and propagates the same context to the selected application boundary.
func (c *CLI) RunContext(ctx context.Context, args []string) int {
	if c == nil {
		return ExitInternal
	}
	if ctx == nil {
		return c.fail(nil, fault.New(
			fault.KindContract,
			"missing_context",
			"The command context is not configured.",
			false,
			fault.NextAction{Command: "help", Reason: "Retry through a context-aware CLI entry point."},
		))
	}
	options, commandArgs, err := parseRootOptions(args)
	ctx = withErrorFormat(ctx, options.ErrorFormat)
	if err != nil {
		return c.failUsage(ctx, "invalid_root_options", err.Error(), "help", "Correct the global options.")
	}
	if err := ctx.Err(); err != nil {
		return c.fail(ctx, err)
	}
	if err := c.catalog.Validate(); err != nil {
		return c.fail(ctx, fault.Wrap(
			fault.KindContract,
			"invalid_catalog",
			"The command catalog is invalid.",
			false,
			err,
			fault.NextAction{Command: "help", Reason: "Repair the catalog before dispatch."},
		))
	}
	if len(commandArgs) == 0 {
		return c.failUsage(ctx, "missing_command", "A command is required.", "help", "Discover available command outcomes.")
	}

	commandArgs = normalizeRootAlias(commandArgs)
	command, rest, found := c.catalog.Match(commandArgs)
	if !found {
		return c.failUsage(
			ctx,
			"unknown_command",
			fmt.Sprintf("Unknown command %q.", strings.Join(commandArgs, " ")),
			"help",
			"Discover an exact command path or namespace.",
		)
	}
	ctx = withCommandPath(ctx, command.Path)
	if len(rest) == 1 && isHelpFlag(rest[0]) {
		return c.emit(ctx, renderCommandHelp(command))
	}
	if err := ctx.Err(); err != nil {
		return c.fail(ctx, err)
	}

	intent := operation.Intent{Command: command.Path, Effect: command.Effect}
	if command.Effect == operation.EffectRead {
		if err := intent.Validate(); err != nil {
			return c.fail(ctx, fault.Wrap(
				fault.KindContract,
				"invalid_intent",
				"The command intent is invalid.",
				false,
				err,
				fault.NextAction{Command: "help " + command.Path, Reason: "Repair the command declaration."},
			))
		}
	}
	return command.handler(ctx, c, command, intent, rest)
}

func normalizeRootAlias(args []string) []string {
	switch args[0] {
	case "--help", "-h":
		return append([]string{"help"}, args[1:]...)
	case "--version", "-v":
		return append([]string{"version"}, args[1:]...)
	default:
		return args
	}
}

func isHelpFlag(value string) bool {
	return value == "--help" || value == "-h"
}

type rootOptions struct {
	ErrorFormat errorFormat
}

func parseRootOptions(args []string) (rootOptions, []string, error) {
	options := rootOptions{ErrorFormat: errorFormatText}
	seenErrorFormat := false
	index := 0
	for index < len(args) {
		argument := args[index]
		var value string
		switch {
		case argument == "--error-format":
			if index+1 >= len(args) {
				return options, nil, fmt.Errorf("--error-format requires text or json")
			}
			index++
			value = args[index]
		case strings.HasPrefix(argument, "--error-format="):
			value = strings.TrimPrefix(argument, "--error-format=")
		default:
			return options, args[index:], nil
		}
		if seenErrorFormat {
			return options, nil, fmt.Errorf("--error-format may be specified only once")
		}
		parsed, err := parseErrorFormat(value)
		if err != nil {
			return options, nil, err
		}
		options.ErrorFormat = parsed
		seenErrorFormat = true
		index++
	}
	return options, args[index:], nil
}
