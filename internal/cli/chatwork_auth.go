package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/tasuku43/cwk/internal/app/chatworkauthcmd"
	"github.com/tasuku43/cwk/internal/app/execution"
	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

const (
	maxOAuthCallbackBytes = 8192
	maxAuthOutputBytes    = 4096
)

type authBrowserOpener interface {
	Open(context.Context, string) error
}

// withChatworkAuthHandlers attaches runtime behavior to catalog-owned command
// declarations without creating a second public registry. The service remains
// captured by the composition root and never becomes process-global state.
func withChatworkAuthHandlers(specs []CommandSpec, service *chatworkauthcmd.Service) []CommandSpec {
	bound := make([]CommandSpec, len(specs))
	copy(bound, specs)
	for index := range bound {
		switch bound[index].Path {
		case "auth login":
			bound[index].handler = func(ctx context.Context, c *CLI, command CommandSpec, intent operation.Intent, args []string) int {
				return runChatworkAuthLogin(ctx, c, command, intent, args, authLifecycleService(c, service))
			}
		case "auth status":
			bound[index].handler = func(ctx context.Context, c *CLI, command CommandSpec, intent operation.Intent, args []string) int {
				return runChatworkAuthStatus(ctx, c, command, intent, args, authLifecycleService(c, service))
			}
		case "auth logout":
			bound[index].handler = func(ctx context.Context, c *CLI, command CommandSpec, intent operation.Intent, args []string) int {
				return runChatworkAuthLogout(ctx, c, command, intent, args, authLifecycleService(c, service))
			}
		}
	}
	return bound
}

func authLifecycleService(c *CLI, injected *chatworkauthcmd.Service) *chatworkauthcmd.Service {
	if injected != nil {
		return injected
	}
	if c != nil && c.chatworkOAuth != nil {
		return c.chatworkOAuth
	}
	return chatworkauthcmd.New(nil)
}

func runChatworkAuthLogin(ctx context.Context, c *CLI, command CommandSpec, base operation.Intent, args []string, service *chatworkauthcmd.Service) int {
	clientID, err := parseOAuthClientID(args)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error()+"; usage: "+command.Usage(), "help auth login", "Pass --client-id only when public OAuth configuration is absent.")
	}
	request, err := buildAuthExecutionRequest(command, base)
	if err != nil {
		return c.fail(ctx, err)
	}
	var summary chatworkauth.Summary
	err = execution.New(chatworkauthcmd.ExactTargetPolicy{}).Invoke(ctx, request, func(actionContext context.Context, _ operation.Intent) error {
		result, loginErr := service.Login(actionContext, clientID, oauthCallbackReceiver(c))
		if loginErr == nil {
			summary = result
		}
		return loginErr
	})
	if err != nil {
		return c.fail(ctx, err)
	}
	output, err := renderAuthSummary(summary)
	if err != nil {
		return c.fail(ctx, err)
	}
	return c.emit(ctx, output)
}

func runChatworkAuthStatus(ctx context.Context, c *CLI, command CommandSpec, _ operation.Intent, args []string, service *chatworkauthcmd.Service) int {
	if len(args) != 0 {
		return c.failUsage(ctx, "invalid_arguments", "auth status accepts no arguments; usage: "+command.Usage(), "help auth status", "Remove undeclared arguments.")
	}
	summary, err := service.Status(ctx)
	if err != nil {
		return c.fail(ctx, err)
	}
	output, err := renderAuthSummary(summary)
	if err != nil {
		return c.fail(ctx, err)
	}
	return c.emit(ctx, output)
}

func runChatworkAuthLogout(ctx context.Context, c *CLI, command CommandSpec, base operation.Intent, args []string, service *chatworkauthcmd.Service) int {
	if len(args) != 0 {
		return c.failUsage(ctx, "invalid_arguments", "auth logout accepts no arguments; usage: "+command.Usage(), "help auth logout", "Remove undeclared arguments.")
	}
	request, err := buildAuthExecutionRequest(command, base)
	if err != nil {
		return c.fail(ctx, err)
	}
	var result chatworkauthcmd.LogoutResult
	err = execution.New(chatworkauthcmd.ExactTargetPolicy{}).Invoke(ctx, request, func(actionContext context.Context, _ operation.Intent) error {
		value, logoutErr := service.Logout(actionContext)
		if logoutErr == nil {
			result = value
		}
		return logoutErr
	})
	if err != nil {
		return c.fail(ctx, err)
	}
	output, err := renderAuthLogout(result)
	if err != nil {
		return c.fail(ctx, err)
	}
	return c.emit(ctx, output)
}

func parseOAuthClientID(args []string) (string, error) {
	var value string
	seen := false
	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--client-id":
			if seen {
				return "", fmt.Errorf("--client-id may be specified only once")
			}
			seen = true
			if index+1 >= len(args) || strings.HasPrefix(args[index+1], "-") {
				return "", fmt.Errorf("--client-id requires a value")
			}
			index++
			value = args[index]
		case strings.HasPrefix(argument, "--client-id="):
			if seen {
				return "", fmt.Errorf("--client-id may be specified only once")
			}
			seen = true
			value = strings.TrimPrefix(argument, "--client-id=")
		case strings.HasPrefix(argument, "-"):
			return "", fmt.Errorf("unknown flag %q", argument)
		default:
			return "", fmt.Errorf("auth login accepts the public client ID only through --client-id")
		}
	}
	if !seen {
		return "", nil
	}
	if value == "" || len(value) > 512 || strings.TrimSpace(value) != value {
		return "", fmt.Errorf("--client-id is invalid")
	}
	for _, character := range value {
		if unicode.Is(unicode.C, character) || character == '\u2028' || character == '\u2029' {
			return "", fmt.Errorf("--client-id is invalid")
		}
	}
	return value, nil
}

func buildAuthExecutionRequest(command CommandSpec, base operation.Intent) (execution.Request, error) {
	if command.Agent.Mutation == nil || command.Agent.FixedTarget == nil {
		return execution.Request{}, fault.New(fault.KindContract, "invalid_mutation_contract", "The authentication mutation contract is invalid.", false)
	}
	mutation := *command.Agent.Mutation
	target := *command.Agent.FixedTarget
	if target.Scope != FixedTargetScopeToolLocal || target.Kind != chatworkauth.TargetKind || target.StableID != chatworkauth.TargetStableID || mutation.TargetKind != target.Kind || mutation.TargetInputs == nil || len(mutation.TargetInputs) != 0 || mutation.ParentInput != "" || mutation.TargetIDInput != "" {
		return execution.Request{}, fault.New(fault.KindContract, "invalid_mutation_contract", "The authentication fixed-target binding is invalid.", false)
	}
	intent := base
	intent.Target = operation.TargetRef{Kind: mutation.TargetKind}
	intent.Impact = mutation.Impact
	switch command.Effect {
	case operation.EffectCreate:
		intent.Target.ParentID = target.StableID
	case operation.EffectWrite:
		intent.Target.ID = target.StableID
	default:
		return execution.Request{}, fault.New(fault.KindContract, "invalid_mutation_contract", "The authentication mutation effect is invalid.", false)
	}
	return execution.Request{
		Intent: intent, ExpectedCommand: command.Path, ExpectedEffect: command.Effect,
		ExpectedTarget: intent.Target, ExpectedImpact: mutation.Impact,
	}, nil
}

func oauthCallbackReceiver(c *CLI) chatworkauthcmd.RedirectReceiver {
	return func(ctx context.Context, authorizationURL string) (string, error) {
		if ctx == nil || ctx.Err() != nil {
			return "", fault.New(fault.KindCanceled, "authentication_canceled", "The authentication task was canceled.", false)
		}
		if c == nil || c.In == nil || c.Err == nil {
			return "", fault.New(fault.KindContract, "oauth_login_receiver_missing", "The OAuth callback receiver is not configured.", false)
		}
		if authorizationURL == "" || len(authorizationURL) > maxOAuthCallbackBytes || strings.ContainsAny(authorizationURL, "\r\n") {
			return "", fault.New(fault.KindInvalidInput, "oauth_configuration_invalid", "The OAuth authorization request is invalid.", false)
		}
		browserOpened := c.authBrowser != nil && c.authBrowser.Open(ctx, authorizationURL) == nil
		prompt := []byte("browser_opened: true\ncallback_url: ")
		if !browserOpened {
			prompt = []byte("browser_opened: false\nauthorization_url: " + authorizationURL + "\ncallback_url: ")
		}
		if _, err := writeOnce(c.Err, prompt); err != nil {
			return "", fault.Wrap(fault.KindInternal, "output_write_failed", "The OAuth authorization prompt could not be written.", true, err)
		}
		callback, err := readBoundedLine(c.In, maxOAuthCallbackBytes)
		if err != nil {
			return "", fault.Wrap(fault.KindAuthentication, "oauth_redirect_receive_failed", "The OAuth redirect could not be received.", false, err)
		}
		if callback == "" {
			return "", fault.New(fault.KindInvalidInput, "oauth_callback_missing", "The complete OAuth callback URL is required.", false)
		}
		return callback, nil
	}
}

func readBoundedLine(reader io.Reader, limit int) (string, error) {
	if reader == nil || limit <= 0 {
		return "", fmt.Errorf("callback reader is unavailable")
	}
	value := make([]byte, 0, min(limit, 512))
	var next [1]byte
	for {
		count, err := reader.Read(next[:])
		if count == 1 {
			if next[0] == '\n' {
				return string(value), nil
			}
			if len(value) == limit {
				return "", fmt.Errorf("callback exceeds the declared byte limit")
			}
			value = append(value, next[0])
		}
		if err == io.EOF {
			return string(value), nil
		}
		if err != nil {
			return "", err
		}
		if count == 0 {
			return "", io.ErrNoProgress
		}
	}
}

func renderAuthSummary(summary chatworkauth.Summary) ([]byte, error) {
	if err := summary.Validate(); err != nil {
		return nil, fault.Wrap(fault.KindContract, "output_encoding_failed", "The OAuth status result is invalid.", false, err)
	}
	var output strings.Builder
	output.WriteString("cwk-auth/1\n")
	output.WriteString("method: oauth2\n")
	fmt.Fprintf(&output, "status: %s\n", summary.State)
	fmt.Fprintf(&output, "expires_at: %d\n", summary.ExpiresAt)
	return boundedAuthOutput(output.String())
}

func renderAuthLogout(result chatworkauthcmd.LogoutResult) ([]byte, error) {
	if !result.Acknowledged || result.RemoteRevocation {
		return nil, fault.New(fault.KindContract, "output_encoding_failed", "The OAuth logout result is invalid.", false)
	}
	var output strings.Builder
	output.WriteString("cwk-auth-logout/1\n")
	fmt.Fprintf(&output, "acknowledged: %t\n", result.Acknowledged)
	fmt.Fprintf(&output, "remote_revocation: %t\n", result.RemoteRevocation)
	return boundedAuthOutput(output.String())
}

func boundedAuthOutput(value string) ([]byte, error) {
	if value == "" || len(value) > maxAuthOutputBytes {
		return nil, fault.New(fault.KindContract, "output_contract_exceeded", "The authentication result exceeds its output contract.", false)
	}
	return []byte(value), nil
}
