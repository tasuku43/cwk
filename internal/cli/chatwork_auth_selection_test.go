package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
	"github.com/tasuku43/cwk/internal/infra/chatworkconfig"
	"github.com/tasuku43/cwk/internal/infra/chatworkoauth"
)

func TestSelectedChatworkClientRequiresOneExactMethodWithoutFallback(t *testing.T) {
	t.Setenv(chatworkapi.TokenEnvironment, "synthetic-valid-token")

	for _, test := range []struct {
		name   string
		method string
		code   string
	}{
		{name: "missing", method: "", code: "chatwork_auth_method_missing"},
		{name: "case is not normalized", method: "PAT", code: "chatwork_auth_method_invalid"},
		{name: "unknown", method: "automatic", code: "chatwork_auth_method_invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, err := selectedChatworkClient(test.method, nil, nil)
			if client != nil {
				t.Fatal("unexpected client")
			}
			assertPublicFaultCode(t, err, test.code)
		})
	}
}

func TestSelectedPATDoesNotFallBackToOAuth(t *testing.T) {
	t.Setenv(chatworkapi.TokenEnvironment, "")
	t.Setenv(chatworkoauth.ClientIDEnvironment, "public-client")
	t.Setenv(chatworkoauth.RedirectEnvironment, "cwk://oauth/callback")
	manager, err := chatworkoauth.NewFromEnvironment(chatworkoauth.OSStore{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := selectedChatworkClient("pat", manager, nil)
	if client != nil {
		t.Fatal("PAT selection unexpectedly produced a client")
	}
	assertPublicFaultCode(t, err, "chatwork_token_missing")
}

func TestSelectedOAuthDoesNotFallBackToPAT(t *testing.T) {
	t.Setenv(chatworkapi.TokenEnvironment, "synthetic-valid-token")
	client, err := selectedChatworkClient("oauth2", nil, fault.New(fault.KindInvalidInput, "oauth_client_configuration_missing", "OAuth configuration is missing.", false))
	if client != nil {
		t.Fatal("OAuth selection unexpectedly produced a client")
	}
	assertPublicFaultCode(t, err, "oauth_client_configuration_missing")
}

func TestProductionAuthDiscoveryWorksBeforeMethodSelection(t *testing.T) {
	t.Setenv(chatworkconfig.AuthMethodEnvironment, "")
	t.Setenv(chatworkoauth.ClientIDEnvironment, "")
	t.Setenv(chatworkoauth.RedirectEnvironment, "")
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := command.RunContext(context.Background(), []string{"auth", "profiles"}); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "profile_ref: "+chatworkauth.PublicClientProfileReference) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func assertPublicFaultCode(t *testing.T, err error, code string) {
	t.Helper()
	public, ok := fault.PublicCopy(err)
	if !ok || public.Code != code {
		t.Fatalf("fault = %#v, want code %q", public, code)
	}
}
