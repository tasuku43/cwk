// Package chatworkconfig owns secret-free Chatwork authentication selection
// and public-client registration metadata. It never represents or returns PAT
// or OAuth token material.
package chatworkconfig

import (
	"os"
)

const AuthMethodEnvironment = "CWK_AUTH_METHOD"

// AuthMethod returns only the explicit non-secret authenticator selector.
func AuthMethod() string {
	return os.Getenv(AuthMethodEnvironment)
}
