// Package chatworkauthcmd owns the task-level Chatwork authentication
// lifecycle. OAuth protocol values and credential-store handles remain behind
// its infrastructure-owned ManagerPort.
package chatworkauthcmd

import (
	"context"

	"github.com/tasuku43/cwk/internal/app/portcheck"
	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

// RedirectReceiver transiently presents a consent URL and returns one complete
// callback URL. It must never persist or render the callback as task output.
type RedirectReceiver = func(context.Context, string) (string, error)

// CredentialStatus is retained as a source-compatible name for the shared
// secret-free domain value used by ManagerPort.
type CredentialStatus = chatworkauth.CredentialStatus

// ManagerPort owns the private OAuth lifecycle used by these public tasks.
type ManagerPort interface {
	Login(context.Context, string, RedirectReceiver) (chatworkauth.CredentialStatus, error)
	Status(context.Context) (chatworkauth.CredentialStatus, error)
	Logout(context.Context) error
}

// LogoutResult makes the local-only scope of logout explicit.
type LogoutResult struct {
	Acknowledged     bool
	RemoteRevocation bool
}

// Service implements the three authentication outcomes without depending on
// OAuth or credential-store implementation types.
type Service struct {
	manager ManagerPort
}

func New(manager ManagerPort) *Service {
	return &Service{manager: manager}
}

func (s *Service) Login(ctx context.Context, clientID string, receive RedirectReceiver) (chatworkauth.Summary, error) {
	if ctx == nil {
		return chatworkauth.Summary{}, missingContextFault()
	}
	if receive == nil {
		return chatworkauth.Summary{}, fault.New(fault.KindContract, "oauth_login_receiver_missing", "The OAuth callback receiver is not configured.", false)
	}
	if s == nil || portcheck.IsNil(s.manager) {
		return chatworkauth.Summary{}, storeUnavailableFault()
	}
	if err := ctx.Err(); err != nil {
		return chatworkauth.Summary{}, canceledFault()
	}
	status, err := s.manager.Login(ctx, clientID, receive)
	if err != nil {
		return chatworkauth.Summary{}, err
	}
	if !status.Authenticated || status.ExpiresAt.IsZero() {
		return chatworkauth.Summary{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "The OAuth login result is invalid.", false)
	}
	return summaryFromStatus(status)
}

func (s *Service) Status(ctx context.Context) (chatworkauth.Summary, error) {
	if ctx == nil {
		return chatworkauth.Summary{}, missingContextFault()
	}
	if s == nil || portcheck.IsNil(s.manager) {
		return chatworkauth.Summary{}, storeUnavailableFault()
	}
	if err := ctx.Err(); err != nil {
		return chatworkauth.Summary{}, canceledFault()
	}
	status, err := s.manager.Status(ctx)
	if err != nil {
		return chatworkauth.Summary{}, err
	}
	return summaryFromStatus(status)
}

func (s *Service) Logout(ctx context.Context) (LogoutResult, error) {
	if ctx == nil {
		return LogoutResult{}, missingContextFault()
	}
	if s == nil || portcheck.IsNil(s.manager) {
		return LogoutResult{}, storeUnavailableFault()
	}
	if err := ctx.Err(); err != nil {
		return LogoutResult{}, canceledFault()
	}
	if err := s.manager.Logout(ctx); err != nil {
		return LogoutResult{}, err
	}
	return LogoutResult{Acknowledged: true, RemoteRevocation: false}, nil
}

func summaryFromStatus(status chatworkauth.CredentialStatus) (chatworkauth.Summary, error) {
	state := chatworkauth.StateUnconfigured
	expiresAt := int64(0)
	if !status.ExpiresAt.IsZero() {
		expiresAt = status.ExpiresAt.Unix()
		state = chatworkauth.StateExpired
	}
	if status.Authenticated {
		if status.ExpiresAt.IsZero() {
			return chatworkauth.Summary{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "The OAuth status is invalid.", false)
		}
		state = chatworkauth.StateReady
	}
	summary := chatworkauth.Summary{Method: "oauth2", State: state, ExpiresAt: expiresAt}
	if err := summary.Validate(); err != nil {
		return chatworkauth.Summary{}, fault.Wrap(fault.KindAuthentication, "invalid_authentication_session", "The OAuth status is invalid.", false, err)
	}
	return summary, nil
}

func missingContextFault() error {
	return fault.New(fault.KindContract, "missing_authentication_context", "The authentication task context is not configured.", false)
}

func canceledFault() error {
	return fault.New(fault.KindCanceled, "authentication_canceled", "The authentication task was canceled.", false)
}

func storeUnavailableFault() error {
	return fault.New(fault.KindUnavailable, "oauth_credential_store_unavailable", "The OAuth credential store is unavailable.", true)
}
