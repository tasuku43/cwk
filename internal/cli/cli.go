// Package cli owns command routing and presentation.
package cli

import (
	"context"
	"errors"
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
	"github.com/tasuku43/cwk/internal/infra/browseropen"
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
	chatworkFactory func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error)
	authBrowser     authBrowserOpener
}

// New builds the production CLI with the fixed Chatwork adapters. OAuth
// lifecycle commands remain available even when an API authentication method
// has not yet been selected or configured.
func New(in io.Reader, out, errOut io.Writer) *CLI {
	cli := newCLI(in, out, errOut, DefaultCatalog(), systemdoctor.New())
	credentialStore := chatworkoauth.OSStore{}
	publicStore := chatworkconfig.NewFileStore()
	lifecycle, err := chatworkoauth.NewLifecycle(credentialStore)
	manager := &chatworkOAuthManager{lifecycle: lifecycle, credentials: credentialStore, public: publicStore}
	if err != nil {
		cli.chatworkOAuth = chatworkauthcmd.New(nil)
	} else {
		cli.chatworkOAuth = chatworkauthcmd.New(manager)
	}
	cli.authBrowser = browseropen.New()
	cli.chatworkFactory = func(ctx context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		client, clientErr := selectedChatworkClient(ctx, chatworkconfig.AuthMethod(), publicStore, credentialStore)
		if clientErr != nil {
			return nil, nil, clientErr
		}
		return chatworkcmd.New(client), appauthn.New(client), nil
	}
	return cli
}

// chatworkOAuthManager keeps registration-dependent login separate from the
// store-only status/logout path, so a credential can always be inspected and
// removed even when public configuration storage is unavailable.
type chatworkOAuthManager struct {
	lifecycle   *chatworkoauth.Manager
	credentials chatworkoauth.Store
	public      *chatworkconfig.FileStore
	configured  func(chatworkconfig.PublicConfig, chatworkoauth.Store) (oauthLoginManager, error)
}

type oauthLoginManager interface {
	Login(context.Context, chatworkauthcmd.RedirectReceiver) (chatworkauth.CredentialStatus, error)
}

func (m *chatworkOAuthManager) Login(ctx context.Context, clientID string, receive chatworkauthcmd.RedirectReceiver) (chatworkauth.CredentialStatus, error) {
	if m == nil || m.public == nil || m.credentials == nil {
		return chatworkauth.CredentialStatus{}, fault.New(fault.KindUnavailable, "oauth_public_configuration_unavailable", "The OAuth public configuration is unavailable.", true)
	}
	config, err := m.public.Load(ctx)
	switch {
	case err == nil:
		if clientID != "" && clientID != config.ClientID {
			return chatworkauth.CredentialStatus{}, fault.New(fault.KindInvalidInput, "oauth_client_configuration_invalid", "The supplied OAuth client ID does not match stored public configuration.", false)
		}
	case errors.Is(err, chatworkconfig.ErrConfigNotFound):
		if clientID == "" {
			return chatworkauth.CredentialStatus{}, fault.New(fault.KindInvalidInput, "oauth_client_configuration_missing", "The first OAuth login requires a public client ID.", false)
		}
		config, err = chatworkconfig.NewOAuthPublicConfig(clientID)
		if err != nil {
			return chatworkauth.CredentialStatus{}, fault.New(fault.KindInvalidInput, "oauth_client_configuration_invalid", "The OAuth public client configuration is invalid.", false)
		}
		// Store non-secret selection before the credential flow. A later login
		// failure therefore leaves a safe, inspectable unconfigured state and
		// never leaves a usable token without its exact public configuration.
		if err := m.public.Save(ctx, config); err != nil {
			return chatworkauth.CredentialStatus{}, publicConfigurationFault(err)
		}
	case err != nil:
		return chatworkauth.CredentialStatus{}, publicConfigurationFault(err)
	}
	factory := m.configured
	if factory == nil {
		factory = func(config chatworkconfig.PublicConfig, store chatworkoauth.Store) (oauthLoginManager, error) {
			return newOAuthManager(config, store)
		}
	}
	configured, err := factory(config, m.credentials)
	if err != nil {
		return chatworkauth.CredentialStatus{}, err
	}
	return configured.Login(ctx, receive)
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

func selectedChatworkClient(ctx context.Context, method string, public *chatworkconfig.FileStore, credentials chatworkoauth.Store) (*chatworkapi.Client, error) {
	switch method {
	case "pat":
		return chatworkapi.NewFromEnvironment()
	case "oauth2", "":
		if public == nil || credentials == nil {
			return nil, fault.New(fault.KindUnavailable, "oauth_public_configuration_unavailable", "The OAuth public configuration is unavailable.", true)
		}
		config, err := public.Load(ctx)
		if errors.Is(err, chatworkconfig.ErrConfigNotFound) && method == "" {
			return nil, fault.New(fault.KindAuthentication, "chatwork_auth_method_missing", "Chatwork authentication method is not selected.", false)
		}
		if err != nil {
			return nil, publicConfigurationFault(err)
		}
		oauth, err := newOAuthManager(config, credentials)
		if err != nil {
			return nil, err
		}
		return chatworkapi.NewWithOAuth(oauth)
	default:
		return nil, fault.New(fault.KindAuthentication, "chatwork_auth_method_invalid", "Chatwork authentication method must be pat or oauth2.", false)
	}
}

func newOAuthManager(config chatworkconfig.PublicConfig, store chatworkoauth.Store) (*chatworkoauth.Manager, error) {
	manager, err := chatworkoauth.New(chatworkoauth.Config{
		ClientID: config.ClientID, RedirectURI: config.RedirectURI,
		Scopes: chatworkoauth.RequiredScopes(),
	}, store)
	if err != nil {
		return nil, fault.New(fault.KindInvalidInput, "oauth_client_configuration_invalid", "Chatwork OAuth public-client configuration is invalid.", false)
	}
	return manager, nil
}

func publicConfigurationFault(err error) error {
	switch {
	case errors.Is(err, chatworkconfig.ErrConfigNotFound):
		return fault.New(fault.KindInvalidInput, "oauth_client_configuration_missing", "Chatwork OAuth public-client configuration is missing.", false)
	case errors.Is(err, chatworkconfig.ErrConfigInvalid):
		return fault.New(fault.KindInvalidInput, "oauth_client_configuration_invalid", "Chatwork OAuth public-client configuration is invalid.", false)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return fault.New(fault.KindCanceled, "authentication_canceled", "The authentication task was canceled.", false)
	default:
		return fault.New(fault.KindUnavailable, "oauth_public_configuration_unavailable", "The OAuth public configuration is unavailable.", true)
	}
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
