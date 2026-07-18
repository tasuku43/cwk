package chatworkconfig

import (
	"testing"
)

func TestEnvironmentProjectionExposesOnlyExplicitAuthenticatorSelection(t *testing.T) {
	t.Setenv(AuthMethodEnvironment, "oauth2")
	if got := AuthMethod(); got != "oauth2" {
		t.Fatalf("AuthMethod() = %q", got)
	}
}
