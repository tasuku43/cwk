// Package chatworkauth defines the secret-free public vocabulary for managing
// Chatwork authentication. OAuth protocol values and credential-store handles
// deliberately remain infrastructure details.
package chatworkauth

import "fmt"

const (
	// OAuthProfileReferenceKind joins profile discovery to every action without
	// exposing a credential or accepting a reconstructed method name.
	OAuthProfileReferenceKind = "chatwork-oauth-profile"

	// PublicClientProfileReference is the one fixed profile supported by the
	// first implementation. It is an opaque workflow value, not a store key.
	PublicClientProfileReference = "cwk_chatwork_oauth_public_v1"
)

// Task names authentication outcomes rather than OAuth protocol endpoints.
type Task string

const (
	TaskProfilesList Task = "auth.profiles.list"
	TaskLogin        Task = "auth.login"
	TaskStatus       Task = "auth.status"
	TaskLogout       Task = "auth.logout"
)

func (t Task) Valid() bool {
	switch t {
	case TaskProfilesList, TaskLogin, TaskStatus, TaskLogout:
		return true
	default:
		return false
	}
}

// ProfileReference preserves the exact discovery value accepted by login,
// status, and logout. It does not identify an OS credential-store entry.
type ProfileReference struct {
	value string
}

func NewProfileReference(value string) (ProfileReference, error) {
	if err := ValidateProfileReference(value); err != nil {
		return ProfileReference{}, err
	}
	return ProfileReference{value: value}, nil
}

func ValidateProfileReference(value string) error {
	if value != PublicClientProfileReference {
		return fmt.Errorf("Chatwork OAuth profile reference is not a supported exact value")
	}
	return nil
}

func (r ProfileReference) Value() string {
	return r.value
}

func (r ProfileReference) Valid() bool {
	return ValidateProfileReference(r.value) == nil
}

// ProfileState is a deliberately reduced, secret-free credential projection.
type ProfileState string

const (
	ProfileStateUnconfigured ProfileState = "unconfigured"
	ProfileStateReady        ProfileState = "ready"
	ProfileStateExpired      ProfileState = "expired"
)

func (s ProfileState) Valid() bool {
	switch s {
	case ProfileStateUnconfigured, ProfileStateReady, ProfileStateExpired:
		return true
	default:
		return false
	}
}

// Profile is safe to render. Expiry is provider-advertised metadata only.
type Profile struct {
	Ref       ProfileReference
	Method    string
	State     ProfileState
	ExpiresAt int64
}

func (p Profile) Validate() error {
	if !p.Ref.Valid() {
		return fmt.Errorf("Chatwork OAuth profile reference is missing or invalid")
	}
	if p.Method != "oauth2" {
		return fmt.Errorf("Chatwork OAuth profile method must be oauth2")
	}
	if !p.State.Valid() {
		return fmt.Errorf("Chatwork OAuth profile state is missing or invalid")
	}
	if p.ExpiresAt < 0 {
		return fmt.Errorf("Chatwork OAuth profile expiry must not be negative")
	}
	if p.State == ProfileStateUnconfigured && p.ExpiresAt != 0 {
		return fmt.Errorf("an unconfigured Chatwork OAuth profile cannot declare expiry")
	}
	return nil
}
