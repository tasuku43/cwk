package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

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

// withChatworkAuthHandlers attaches runtime behavior to catalog-owned command
// declarations without creating a second public registry. The service remains
// captured by the composition root and never becomes process-global state.
func withChatworkAuthHandlers(specs []CommandSpec, service *chatworkauthcmd.Service) []CommandSpec {
	bound := make([]CommandSpec, len(specs))
	copy(bound, specs)
	for index := range bound {
		switch bound[index].Path {
		case "auth profiles":
			bound[index].handler = func(ctx context.Context, c *CLI, command CommandSpec, intent operation.Intent, args []string) int {
				return runChatworkAuthProfiles(ctx, c, command, intent, args, authLifecycleService(c, service))
			}
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

func runChatworkAuthProfiles(ctx context.Context, c *CLI, command CommandSpec, _ operation.Intent, args []string, service *chatworkauthcmd.Service) int {
	if len(args) != 0 {
		return c.failUsage(ctx, "invalid_arguments", "auth profiles accepts no arguments; usage: "+command.Usage(), "help auth profiles", "Remove undeclared arguments.")
	}
	profiles, err := service.Profiles(ctx)
	if err != nil {
		return c.fail(ctx, err)
	}
	output, err := renderAuthProfiles(profiles)
	if err != nil {
		return c.fail(ctx, err)
	}
	return c.emit(ctx, output)
}

func runChatworkAuthLogin(ctx context.Context, c *CLI, command CommandSpec, base operation.Intent, args []string, service *chatworkauthcmd.Service) int {
	ref, err := parseOAuthProfile(args)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error()+"; usage: "+command.Usage(), "help auth login", "Pass exactly one profile reference from auth profiles.")
	}
	request, err := buildAuthExecutionRequest(command, base, ref)
	if err != nil {
		return c.fail(ctx, err)
	}
	var profile chatworkauth.Profile
	err = execution.New(chatworkauthcmd.ExactProfilePolicy{Profile: ref}).Invoke(ctx, request, func(actionContext context.Context, _ operation.Intent) error {
		result, loginErr := service.Login(actionContext, ref, oauthCallbackReceiver(c))
		if loginErr == nil {
			profile = result
		}
		return loginErr
	})
	if err != nil {
		return c.fail(ctx, err)
	}
	output, err := renderAuthProfile(profile)
	if err != nil {
		return c.fail(ctx, err)
	}
	return c.emit(ctx, output)
}

func runChatworkAuthStatus(ctx context.Context, c *CLI, command CommandSpec, _ operation.Intent, args []string, service *chatworkauthcmd.Service) int {
	ref, err := parseOAuthProfile(args)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error()+"; usage: "+command.Usage(), "help auth status", "Pass exactly one profile reference from auth profiles.")
	}
	profile, err := service.Status(ctx, ref)
	if err != nil {
		return c.fail(ctx, err)
	}
	output, err := renderAuthProfile(profile)
	if err != nil {
		return c.fail(ctx, err)
	}
	return c.emit(ctx, output)
}

func runChatworkAuthLogout(ctx context.Context, c *CLI, command CommandSpec, base operation.Intent, args []string, service *chatworkauthcmd.Service) int {
	ref, err := parseOAuthProfile(args)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error()+"; usage: "+command.Usage(), "help auth logout", "Pass exactly one profile reference from auth profiles.")
	}
	request, err := buildAuthExecutionRequest(command, base, ref)
	if err != nil {
		return c.fail(ctx, err)
	}
	var result chatworkauthcmd.LogoutResult
	err = execution.New(chatworkauthcmd.ExactProfilePolicy{Profile: ref}).Invoke(ctx, request, func(actionContext context.Context, _ operation.Intent) error {
		value, logoutErr := service.Logout(actionContext, ref)
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

func parseOAuthProfile(args []string) (chatworkauth.ProfileReference, error) {
	var value string
	seen := false
	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--profile":
			if seen {
				return chatworkauth.ProfileReference{}, fmt.Errorf("--profile may be specified only once")
			}
			seen = true
			if index+1 >= len(args) || strings.HasPrefix(args[index+1], "-") {
				return chatworkauth.ProfileReference{}, fmt.Errorf("--profile requires a value")
			}
			index++
			value = args[index]
		case strings.HasPrefix(argument, "--profile="):
			if seen {
				return chatworkauth.ProfileReference{}, fmt.Errorf("--profile may be specified only once")
			}
			seen = true
			value = strings.TrimPrefix(argument, "--profile=")
		case strings.HasPrefix(argument, "-"):
			return chatworkauth.ProfileReference{}, fmt.Errorf("unknown flag %q", argument)
		default:
			return chatworkauth.ProfileReference{}, fmt.Errorf("authentication actions accept a profile only through --profile")
		}
	}
	if !seen || value == "" {
		return chatworkauth.ProfileReference{}, fmt.Errorf("--profile is required")
	}
	ref, err := chatworkauth.NewProfileReference(value)
	if err != nil {
		return chatworkauth.ProfileReference{}, fmt.Errorf("--profile requires the exact reference emitted by auth profiles")
	}
	return ref, nil
}

func buildAuthExecutionRequest(command CommandSpec, base operation.Intent, ref chatworkauth.ProfileReference) (execution.Request, error) {
	if command.Agent.Mutation == nil || !ref.Valid() {
		return execution.Request{}, fault.New(fault.KindContract, "invalid_mutation_contract", "The authentication mutation contract is invalid.", false)
	}
	mutation := *command.Agent.Mutation
	intent := base
	intent.Target = operation.TargetRef{Kind: mutation.TargetKind}
	intent.Impact = mutation.Impact
	switch command.Effect {
	case operation.EffectCreate:
		if mutation.ParentInput != "--profile" || mutation.TargetIDInput != "" {
			return execution.Request{}, fault.New(fault.KindContract, "invalid_mutation_contract", "The authentication create binding is invalid.", false)
		}
		intent.Target.ParentID = ref.Value()
	case operation.EffectWrite:
		if mutation.TargetIDInput != "--profile" || mutation.ParentInput != "" {
			return execution.Request{}, fault.New(fault.KindContract, "invalid_mutation_contract", "The authentication write binding is invalid.", false)
		}
		intent.Target.ID = ref.Value()
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
		prompt := []byte("authorization_url: " + authorizationURL + "\ncallback_url: ")
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

func renderAuthProfiles(profiles []chatworkauth.Profile) ([]byte, error) {
	if len(profiles) != 1 || profiles[0].Ref.Value() != chatworkauth.PublicClientProfileReference || profiles[0].Method != "oauth2" {
		return nil, fault.New(fault.KindContract, "output_encoding_failed", "The OAuth profile discovery result is invalid.", false)
	}
	var output strings.Builder
	output.WriteString("cwk-auth-profiles/1\n")
	fmt.Fprintf(&output, "profile_ref: %s\n", profiles[0].Ref.Value())
	output.WriteString("method: oauth2\n")
	output.WriteString("api_selector: CWK_AUTH_METHOD\n")
	output.WriteString("allowed_api_methods: pat,oauth2\n")
	output.WriteString("callback_model: authorization_code_pkce_s256_manual_callback\n")
	output.WriteString("credential_storage: operating_system\n")
	return boundedAuthOutput(output.String())
}

func renderAuthProfile(profile chatworkauth.Profile) ([]byte, error) {
	if err := profile.Validate(); err != nil {
		return nil, fault.Wrap(fault.KindContract, "output_encoding_failed", "The OAuth profile result is invalid.", false, err)
	}
	var output strings.Builder
	output.WriteString("cwk-auth-profile/1\n")
	fmt.Fprintf(&output, "profile_ref: %s\n", profile.Ref.Value())
	output.WriteString("method: oauth2\n")
	fmt.Fprintf(&output, "status: %s\n", profile.State)
	fmt.Fprintf(&output, "expires_at: %d\n", profile.ExpiresAt)
	return boundedAuthOutput(output.String())
}

func renderAuthLogout(result chatworkauthcmd.LogoutResult) ([]byte, error) {
	if !result.Ref.Valid() || !result.Acknowledged || result.RemoteRevocation {
		return nil, fault.New(fault.KindContract, "output_encoding_failed", "The OAuth logout result is invalid.", false)
	}
	var output strings.Builder
	output.WriteString("cwk-auth-logout/1\n")
	fmt.Fprintf(&output, "profile_ref: %s\n", result.Ref.Value())
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
