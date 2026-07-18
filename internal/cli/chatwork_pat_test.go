package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
)

func TestProductionHelpDoesNotResolvePAT(t *testing.T) {
	for _, test := range []struct {
		args  []string
		token string
	}{
		{args: []string{"--help"}},
		{args: []string{"rooms", "--help"}, token: "synthetic-valid-token"},
		{args: []string{"rooms", "list", "--help"}, token: "synthetic-valid-token"},
		{args: []string{"help", "rooms", "--format", "agent"}, token: "synthetic-valid-token"},
	} {
		t.Run(strings.Join(test.args, "_"), func(t *testing.T) {
			t.Setenv(chatworkapi.TokenEnvironment, test.token)
			var stdout, stderr bytes.Buffer
			command := New(strings.NewReader(""), &stdout, &stderr)
			command.chatworkFactory = nil
			if command.chatwork != nil || command.chatworkAuth != nil {
				t.Fatal("production CLI eagerly constructed the authenticated adapter")
			}
			if code := command.RunContext(context.Background(), test.args); code != ExitOK {
				t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
			}
			if command.chatwork != nil || command.chatworkAuth != nil {
				t.Fatal("help resolved the process-local PAT")
			}
		})
	}
}

func TestProductionChatworkFactoryUsesPATWithoutMethodSelection(t *testing.T) {
	t.Setenv(chatworkapi.TokenEnvironment, "synthetic-valid-token")
	command := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err := command.ensureChatwork(context.Background()); err != nil {
		t.Fatalf("ensureChatwork() error = %v", err)
	}
	if command.chatwork == nil || command.chatworkAuth == nil {
		t.Fatal("PAT did not construct the Chatwork service and authentication gate")
	}
}

func TestProductionChatworkFactoryFailsClosedForMissingOrInvalidPAT(t *testing.T) {
	for name, test := range map[string]struct {
		token string
		code  string
	}{
		"missing": {token: "", code: "chatwork_token_missing"},
		"invalid": {token: "short", code: "chatwork_token_invalid"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(chatworkapi.TokenEnvironment, test.token)
			command := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
			err := command.ensureChatwork(context.Background())
			public, ok := fault.PublicCopy(err)
			if !ok || public.Code != test.code || command.chatwork != nil || command.chatworkAuth != nil {
				t.Fatalf("fault/service/gate = %#v/%v/%v, want %s/nil/nil", public, command.chatwork, command.chatworkAuth, test.code)
			}
		})
	}
}

func TestDefaultCatalogHasNoAuthenticationLifecycleCommands(t *testing.T) {
	catalog := DefaultCatalog()
	for _, path := range []string{"auth login", "auth status", "auth logout"} {
		if _, found := catalog.Lookup(path); found {
			t.Errorf("removed authentication lifecycle command %q remains public", path)
		}
	}
}
