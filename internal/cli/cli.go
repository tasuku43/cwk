// Package cli owns command routing and presentation.
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/configcmd"
	"github.com/tasuku43/cwk/internal/app/doctorcmd"
	"github.com/tasuku43/cwk/internal/app/samplecmd"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
	"github.com/tasuku43/cwk/internal/infra/commandconfig"
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

	baseCatalog      Catalog
	catalog          Catalog
	commandSelection *configcmd.Service
	doctor           *doctorcmd.Service
	samples          *samplecmd.Service
	chatwork         *chatworkcmd.Service
	chatworkAuth     *appauthn.Gate
	chatworkInitErr  error
	chatworkFactory  func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error)
}

// New builds the production CLI with a lazy PAT adapter. Help and local
// commands remain available without reading CWK_API_TOKEN; the token is
// resolved only when a Chatwork API task actually executes.
func New(in io.Reader, out, errOut io.Writer) *CLI {
	cli := newCLI(in, out, errOut, DefaultCatalog(), systemdoctor.New())
	cli.commandSelection = configcmd.New(commandconfig.NewFileStore())
	cli.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		client, clientErr := chatworkapi.NewFromEnvironment()
		if clientErr != nil {
			return nil, nil, clientErr
		}
		return chatworkcmd.New(client), appauthn.New(client), nil
	}
	return cli
}

func (c *CLI) ensureChatwork(ctx context.Context) error {
	if c == nil {
		return fault.New(fault.KindContract, "missing_chatwork_port", "Chatwork task adapter is not configured", false)
	}
	if c.chatwork != nil && c.chatworkAuth != nil {
		return nil
	}
	if c.chatworkInitErr != nil {
		return c.chatworkInitErr
	}
	if c.chatworkFactory == nil {
		return fault.New(fault.KindContract, "missing_chatwork_port", "Chatwork task adapter is not configured", false)
	}
	service, gate, err := c.chatworkFactory(ctx)
	if err != nil {
		c.chatworkInitErr = err
		return err
	}
	c.chatwork = service
	c.chatworkAuth = gate
	return nil
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
		Version:     "dev",
		baseCatalog: catalog,
		catalog:     catalog,
		doctor:      doctorcmd.New(inspector),
		samples:     samplecmd.New(repository),
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
	baseCatalog := c.baseCatalog
	if len(baseCatalog.commands) == 0 {
		baseCatalog = c.catalog
	}
	if err := baseCatalog.Validate(); err != nil {
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
	activeCatalog, activeErr := c.resolveActiveCatalog(ctx, baseCatalog)
	if activeErr != nil {
		if !commandViewControlInvocation(commandArgs) {
			return c.fail(ctx, commandSelectionDispatchFault(activeErr))
		}
		activeCatalog, _, err = baseCatalog.ActiveView([]string{})
		if err != nil {
			return c.fail(ctx, fault.Wrap(
				fault.KindContract,
				"invalid_catalog",
				"The command catalog control plane is invalid.",
				false,
				err,
				fault.NextAction{Command: "help", Reason: "Repair the catalog before dispatch."},
			))
		}
	}
	c.catalog = activeCatalog
	commandArgs = normalizeTrailingHelpAlias(c.catalog, commandArgs)
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

func (c *CLI) resolveActiveCatalog(ctx context.Context, base Catalog) (Catalog, error) {
	if c == nil || c.commandSelection == nil {
		return base, nil
	}
	profile, configured, err := c.commandSelection.Load(ctx)
	if err != nil {
		return Catalog{}, err
	}
	enabled := make([]string, 0)
	if configured {
		enabled = profile.EnabledCommands()
	} else {
		for _, command := range base.ConfigurableCommands() {
			enabled = append(enabled, command.Path)
		}
	}
	view, _, err := base.ActiveView(enabled)
	if err != nil {
		return Catalog{}, fault.Wrap(
			fault.KindInvalidInput,
			"command_selection_invalid",
			"The saved command selection is invalid.",
			false,
			err,
		)
	}
	return view, nil
}

func commandViewControlInvocation(args []string) bool {
	return len(args) > 0 && (args[0] == "help" || args[0] == "config")
}

func commandSelectionDispatchFault(err error) error {
	if public, ok := fault.PublicCopy(err); ok {
		return fault.New(
			public.Kind,
			public.Code,
			public.Message,
			public.Retryable,
			fault.NextAction{Command: "config edit", Reason: "Inspect and repair the tool-local command selection."},
		)
	}
	return err
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

// normalizeTrailingHelpAlias maps the conventional `<selector> --help` form
// onto the catalog help task. Only selectors already proven by the catalog are
// rewritten, so an unknown namespace retains the normal unknown-command fault
// instead of changing error taxonomy inside the help parser.
func normalizeTrailingHelpAlias(catalog Catalog, args []string) []string {
	if len(args) < 2 || !isHelpFlag(args[len(args)-1]) {
		return args
	}
	selectorWords := args[:len(args)-1]
	selector := strings.Join(selectorWords, " ")
	commands, _ := catalog.Select(selector)
	if len(commands) == 0 {
		return args
	}
	normalized := make([]string, 1, len(args))
	normalized[0] = "help"
	return append(normalized, selectorWords...)
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
