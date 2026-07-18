// Package cli owns command routing and presentation.
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkauthcmd"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/doctorcmd"
	"github.com/tasuku43/cwk/internal/app/samplecmd"
	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
	"github.com/tasuku43/cwk/internal/infra/chatworkconfig"
	"github.com/tasuku43/cwk/internal/infra/chatworkoauth"
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
	chatworkOAuth   *chatworkauthcmd.Service
	chatworkInitErr error
}

// New builds the production CLI with the fixed Chatwork adapters. OAuth
// lifecycle commands remain available even when an API authentication method
// has not yet been selected or configured.
func New(in io.Reader, out, errOut io.Writer) *CLI {
	cli := newCLI(in, out, errOut, DefaultCatalog(), systemdoctor.New())
	store := chatworkoauth.OSStore{}
	lifecycle, err := chatworkoauth.NewLifecycle(store)
	configured, configuredErr := chatworkoauth.NewFromEnvironment(store)
	manager := &chatworkOAuthManager{configured: configured, lifecycle: lifecycle}
	if configuredErr != nil {
		manager.configErr = oauthClientConfigurationFault()
	}
	if err != nil {
		cli.chatworkOAuth = chatworkauthcmd.New(nil)
	} else {
		cli.chatworkOAuth = chatworkauthcmd.New(manager)
	}

	client, err := selectedChatworkClient(chatworkconfig.AuthMethod(), configured, manager.configErr)
	if err != nil {
		cli.chatworkInitErr = err
		return cli
	}
	cli.chatwork = chatworkcmd.New(client)
	cli.chatworkAuth = appauthn.New(client)
	return cli
}

// chatworkOAuthManager keeps registration-dependent login separate from the
// store-only status/logout path, so a credential can always be inspected and
// removed after registration environment variables are cleared.
type chatworkOAuthManager struct {
	configured *chatworkoauth.Manager
	lifecycle  *chatworkoauth.Manager
	configErr  error
}

func (m *chatworkOAuthManager) Login(ctx context.Context, receive chatworkauthcmd.RedirectReceiver) (chatworkauth.CredentialStatus, error) {
	if m == nil || m.configured == nil {
		if m != nil && m.configErr != nil {
			return chatworkauth.CredentialStatus{}, m.configErr
		}
		return chatworkauth.CredentialStatus{}, oauthClientConfigurationFault()
	}
	return m.configured.Login(ctx, receive)
}

func (m *chatworkOAuthManager) Status(ctx context.Context) (chatworkauth.CredentialStatus, error) {
	if m == nil || m.lifecycle == nil {
		return chatworkauth.CredentialStatus{}, fault.New(fault.KindUnavailable, "oauth_credential_store_unavailable", "The OAuth credential store is unavailable.", true)
	}
	return m.lifecycle.Status(ctx)
}

func (m *chatworkOAuthManager) Logout(ctx context.Context) error {
	if m == nil || m.lifecycle == nil {
		return fault.New(fault.KindUnavailable, "oauth_credential_store_unavailable", "The OAuth credential store is unavailable.", true)
	}
	return m.lifecycle.Logout(ctx)
}

func selectedChatworkClient(method string, oauth *chatworkoauth.Manager, oauthErr error) (*chatworkapi.Client, error) {
	switch method {
	case "pat":
		return chatworkapi.NewFromEnvironment()
	case "oauth2":
		if oauthErr != nil || oauth == nil {
			if oauthErr != nil {
				return nil, oauthErr
			}
			return nil, oauthClientConfigurationFault()
		}
		return chatworkapi.NewWithOAuth(oauth)
	case "":
		return nil, fault.New(fault.KindAuthentication, "chatwork_auth_method_missing", "Chatwork authentication method is not selected.", false)
	default:
		return nil, fault.New(fault.KindAuthentication, "chatwork_auth_method_invalid", "Chatwork authentication method must be pat or oauth2.", false)
	}
}

func oauthClientConfigurationFault() error {
	if !chatworkconfig.OAuthRegistrationComplete() {
		return fault.New(fault.KindInvalidInput, "oauth_client_configuration_missing", "Chatwork OAuth public-client configuration is missing.", false)
	}
	return fault.New(fault.KindInvalidInput, "oauth_client_configuration_invalid", "Chatwork OAuth public-client configuration is invalid.", false)
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
