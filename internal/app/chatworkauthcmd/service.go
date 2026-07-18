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
	Login(context.Context, RedirectReceiver) (chatworkauth.CredentialStatus, error)
	Status(context.Context) (chatworkauth.CredentialStatus, error)
	Logout(context.Context) error
}

// LogoutResult makes the local-only scope of logout explicit.
type LogoutResult struct {
	Ref              chatworkauth.ProfileReference
	Acknowledged     bool
	RemoteRevocation bool
}

// Service implements the four authentication outcomes without depending on
// OAuth or credential-store implementation types.
type Service struct {
	manager ManagerPort
}

func New(manager ManagerPort) *Service {
	return &Service{manager: manager}
}

// Profiles discovers the one fixed public-client profile without touching the
// credential store.
func (s *Service) Profiles(ctx context.Context) ([]chatworkauth.Profile, error) {
	if ctx == nil {
		return nil, missingContextFault()
	}
	if err := ctx.Err(); err != nil {
		return nil, canceledFault()
	}
	ref, err := chatworkauth.NewProfileReference(chatworkauth.PublicClientProfileReference)
	if err != nil {
		return nil, fault.New(fault.KindContract, "output_encoding_failed", "The fixed Chatwork OAuth profile is invalid.", false)
	}
	return []chatworkauth.Profile{{Ref: ref, Method: "oauth2", State: chatworkauth.ProfileStateUnconfigured}}, nil
}

func (s *Service) Login(ctx context.Context, ref chatworkauth.ProfileReference, receive RedirectReceiver) (chatworkauth.Profile, error) {
	if ctx == nil {
		return chatworkauth.Profile{}, missingContextFault()
	}
	if err := validateReference(ref); err != nil {
		return chatworkauth.Profile{}, err
	}
	if receive == nil {
		return chatworkauth.Profile{}, fault.New(fault.KindContract, "oauth_login_receiver_missing", "The OAuth callback receiver is not configured.", false)
	}
	if s == nil || portcheck.IsNil(s.manager) {
		return chatworkauth.Profile{}, storeUnavailableFault()
	}
	if err := ctx.Err(); err != nil {
		return chatworkauth.Profile{}, canceledFault()
	}
	status, err := s.manager.Login(ctx, receive)
	if err != nil {
		return chatworkauth.Profile{}, err
	}
	if !status.Authenticated || status.ExpiresAt.IsZero() {
		return chatworkauth.Profile{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "The OAuth login result is invalid.", false)
	}
	return profileFromStatus(ref, status)
}

func (s *Service) Status(ctx context.Context, ref chatworkauth.ProfileReference) (chatworkauth.Profile, error) {
	if ctx == nil {
		return chatworkauth.Profile{}, missingContextFault()
	}
	if err := validateReference(ref); err != nil {
		return chatworkauth.Profile{}, err
	}
	if s == nil || portcheck.IsNil(s.manager) {
		return chatworkauth.Profile{}, storeUnavailableFault()
	}
	if err := ctx.Err(); err != nil {
		return chatworkauth.Profile{}, canceledFault()
	}
	status, err := s.manager.Status(ctx)
	if err != nil {
		return chatworkauth.Profile{}, err
	}
	return profileFromStatus(ref, status)
}

func (s *Service) Logout(ctx context.Context, ref chatworkauth.ProfileReference) (LogoutResult, error) {
	if ctx == nil {
		return LogoutResult{}, missingContextFault()
	}
	if err := validateReference(ref); err != nil {
		return LogoutResult{}, err
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
	return LogoutResult{Ref: ref, Acknowledged: true, RemoteRevocation: false}, nil
}

func profileFromStatus(ref chatworkauth.ProfileReference, status chatworkauth.CredentialStatus) (chatworkauth.Profile, error) {
	state := chatworkauth.ProfileStateUnconfigured
	expiresAt := int64(0)
	if !status.ExpiresAt.IsZero() {
		expiresAt = status.ExpiresAt.Unix()
		state = chatworkauth.ProfileStateExpired
	}
	if status.Authenticated {
		if status.ExpiresAt.IsZero() {
			return chatworkauth.Profile{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "The OAuth profile status is invalid.", false)
		}
		state = chatworkauth.ProfileStateReady
	}
	profile := chatworkauth.Profile{Ref: ref, Method: "oauth2", State: state, ExpiresAt: expiresAt}
	if err := profile.Validate(); err != nil {
		return chatworkauth.Profile{}, fault.Wrap(fault.KindAuthentication, "invalid_authentication_session", "The OAuth profile status is invalid.", false, err)
	}
	return profile, nil
}

func validateReference(ref chatworkauth.ProfileReference) error {
	if !ref.Valid() {
		return fault.New(fault.KindInvalidInput, "oauth_profile_reference_invalid", "The Chatwork OAuth profile reference is invalid.", false)
	}
	return nil
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
