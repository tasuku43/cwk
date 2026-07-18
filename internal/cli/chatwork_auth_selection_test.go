package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
	"github.com/tasuku43/cwk/internal/infra/chatworkconfig"
)

func TestSelectedChatworkClientRequiresOneExactMethodWithoutFallback(t *testing.T) {
	t.Setenv(chatworkapi.TokenEnvironment, "synthetic-valid-token")
	public := chatworkconfig.NewFileStoreAt(t.TempDir())

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
			client, err := selectedChatworkClient(context.Background(), test.method, public, selectionCredentialStore{})
			if client != nil {
				t.Fatal("unexpected client")
			}
			assertPublicFaultCode(t, err, test.code)
		})
	}
}

func TestSelectedPATDoesNotFallBackToOAuth(t *testing.T) {
	t.Setenv(chatworkapi.TokenEnvironment, "")
	public := chatworkconfig.NewFileStoreAt(t.TempDir())
	config, err := chatworkconfig.NewOAuthPublicConfig("public-client")
	if err != nil {
		t.Fatal(err)
	}
	if err := public.Save(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	client, err := selectedChatworkClient(context.Background(), "pat", public, selectionCredentialStore{})
	if client != nil {
		t.Fatal("PAT selection unexpectedly produced a client")
	}
	assertPublicFaultCode(t, err, "chatwork_token_missing")
}

func TestSelectedOAuthDoesNotFallBackToPAT(t *testing.T) {
	t.Setenv(chatworkapi.TokenEnvironment, "synthetic-valid-token")
	client, err := selectedChatworkClient(context.Background(), "oauth2", chatworkconfig.NewFileStoreAt(t.TempDir()), selectionCredentialStore{})
	if client != nil {
		t.Fatal("OAuth selection unexpectedly produced a client")
	}
	assertPublicFaultCode(t, err, "oauth_client_configuration_missing")
}

func TestStoredOAuthSelectionNeedsNoMethodExport(t *testing.T) {
	public := chatworkconfig.NewFileStoreAt(t.TempDir())
	config, err := chatworkconfig.NewOAuthPublicConfig("public-client")
	if err != nil {
		t.Fatal(err)
	}
	if err := public.Save(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	client, err := selectedChatworkClient(context.Background(), "", public, selectionCredentialStore{})
	if err != nil || client == nil {
		t.Fatalf("client/error = %v/%v", client, err)
	}
}

func TestProductionAuthHelpWorksBeforeMethodSelection(t *testing.T) {
	t.Setenv(chatworkconfig.AuthMethodEnvironment, "")
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := command.RunContext(context.Background(), []string{"auth", "login", "--help"}); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "cwk auth login [--client-id <public-client-id>]") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

type selectionCredentialStore struct{}

func (selectionCredentialStore) Load(context.Context) ([]byte, error) {
	return nil, fault.New(fault.KindAuthentication, "unused", "unused", false)
}
func (selectionCredentialStore) Save(context.Context, []byte) error { return nil }
func (selectionCredentialStore) Delete(context.Context) error       { return nil }

func assertPublicFaultCode(t *testing.T, err error, code string) {
	t.Helper()
	public, ok := fault.PublicCopy(err)
	if !ok || public.Code != code {
		t.Fatalf("fault = %#v, want code %q", public, code)
	}
}
