package chatworkconfig

import (
	"testing"

	"github.com/tasuku43/cwk/internal/infra/chatworkoauth"
)

func TestEnvironmentProjectionExposesOnlySelectionAndRegistrationPresence(t *testing.T) {
	t.Setenv(AuthMethodEnvironment, "oauth2")
	t.Setenv(chatworkoauth.ClientIDEnvironment, "public-client")
	t.Setenv(chatworkoauth.RedirectEnvironment, "cwk://oauth/callback")
	if got := AuthMethod(); got != "oauth2" {
		t.Fatalf("AuthMethod() = %q", got)
	}
	if !OAuthRegistrationComplete() {
		t.Fatal("OAuthRegistrationComplete() = false")
	}
	t.Setenv(chatworkoauth.RedirectEnvironment, "")
	if OAuthRegistrationComplete() {
		t.Fatal("incomplete OAuth registration was accepted")
	}
}
