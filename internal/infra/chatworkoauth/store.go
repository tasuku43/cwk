package chatworkoauth

import (
	"context"
	"errors"

	keyring "github.com/zalando/go-keyring"
)

const (
	credentialService = "cwk.chatwork.oauth2" // #nosec G101 -- public OS credential-store service index, not credential material.
	credentialAccount = "default"             // #nosec G101 -- public single-profile store index, not an account secret or identifier.
	// Windows Credential Manager has the lowest documented payload ceiling of
	// the supported go-keyring backends. Keep a small margin below it.
	maxStoredCredentialBytes = 2400
)

var (
	errCredentialNotFound = errors.New("Chatwork OAuth credential not found")
	errCredentialInvalid  = errors.New("Chatwork OAuth credential is invalid")
)

// Store is the credential-store boundary. Implementations receive only the
// encrypted-at-rest credential payload and never expose an OS keychain handle
// outside infrastructure.
type Store interface {
	Load(context.Context) ([]byte, error)
	Save(context.Context, []byte) error
	Delete(context.Context) error
}

// OSStore persists the OAuth credential in the platform credential manager.
// go-keyring uses macOS Keychain, Secret Service, or Windows Credential
// Manager depending on the current platform.
type OSStore struct{}

func (OSStore) Load(ctx context.Context) ([]byte, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, contextError(ctx)
	}
	value, err := keyring.Get(credentialService, credentialAccount)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, errCredentialNotFound
	}
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return []byte(value), nil
}

func (OSStore) Save(ctx context.Context, value []byte) error {
	if ctx == nil || ctx.Err() != nil {
		return contextError(ctx)
	}
	if len(value) == 0 || len(value) > maxStoredCredentialBytes {
		return keyring.ErrSetDataTooBig
	}
	if err := keyring.Set(credentialService, credentialAccount, string(value)); err != nil {
		return err
	}
	return ctx.Err()
}

func (OSStore) Delete(ctx context.Context) error {
	if ctx == nil || ctx.Err() != nil {
		return contextError(ctx)
	}
	err := keyring.Delete(credentialService, credentialAccount)
	if errors.Is(err, keyring.ErrNotFound) {
		return errCredentialNotFound
	}
	if err != nil {
		return err
	}
	return ctx.Err()
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return context.Canceled
	}
	return ctx.Err()
}
