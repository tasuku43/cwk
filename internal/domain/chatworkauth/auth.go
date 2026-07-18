// Package chatworkauth defines the secret-free public vocabulary for managing
// Chatwork authentication. OAuth protocol values and credential-store handles
// deliberately remain infrastructure details.
package chatworkauth

import (
	"fmt"
	"time"
)

const (
	// TargetKind and TargetStableID identify the one tool-owned authentication
	// target. They are policy identity, never a credential-store key or token.
	TargetKind     = "chatwork-authentication"
	TargetStableID = "single-account"
)

// Task names authentication outcomes rather than OAuth protocol endpoints.
type Task string

const (
	TaskLogin  Task = "auth.login"
	TaskStatus Task = "auth.status"
	TaskLogout Task = "auth.logout"
)

func (t Task) Valid() bool {
	switch t {
	case TaskLogin, TaskStatus, TaskLogout:
		return true
	default:
		return false
	}
}

// State is a deliberately reduced, secret-free credential projection.
type State string

const (
	StateUnconfigured State = "unconfigured"
	StateReady        State = "ready"
	StateExpired      State = "expired"
)

func (s State) Valid() bool {
	switch s {
	case StateUnconfigured, StateReady, StateExpired:
		return true
	default:
		return false
	}
}

// Summary is safe to render. Expiry is provider-advertised metadata only.
type Summary struct {
	Method    string
	State     State
	ExpiresAt int64
}

// CredentialStatus is the minimal secret-free result shared by the OAuth
// manager and authentication use case. It cannot carry protocol or store data.
type CredentialStatus struct {
	Authenticated bool
	ExpiresAt     time.Time
}

func (s Summary) Validate() error {
	if s.Method != "oauth2" {
		return fmt.Errorf("Chatwork authentication method must be oauth2")
	}
	if !s.State.Valid() {
		return fmt.Errorf("Chatwork authentication state is missing or invalid")
	}
	if s.ExpiresAt < 0 {
		return fmt.Errorf("Chatwork authentication expiry must not be negative")
	}
	if s.State == StateUnconfigured && s.ExpiresAt != 0 {
		return fmt.Errorf("unconfigured Chatwork authentication cannot declare expiry")
	}
	return nil
}
