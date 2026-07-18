// Package chatworkconfig owns the narrow process-environment boundary used by
// the Chatwork composition root. It never returns PAT or OAuth token material.
package chatworkconfig

import (
	"os"

	"github.com/tasuku43/cwk/internal/infra/chatworkoauth"
)

const AuthMethodEnvironment = "CWK_AUTH_METHOD"

// AuthMethod returns only the explicit non-secret authenticator selector.
func AuthMethod() string {
	return os.Getenv(AuthMethodEnvironment)
}

// OAuthRegistrationComplete reports whether both non-secret public-client
// registration values are present. Their syntax remains owned by the OAuth
// adapter and neither value crosses this boundary.
func OAuthRegistrationComplete() bool {
	return os.Getenv(chatworkoauth.ClientIDEnvironment) != "" && os.Getenv(chatworkoauth.RedirectEnvironment) != ""
}
