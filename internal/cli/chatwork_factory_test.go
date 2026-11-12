package cli

import (
	"context"
	"io"
	"testing"

	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
)

func TestNewBuildsChatworkAdapterLazilyWithInjectedVersion(t *testing.T) {
	command := New(nil, io.Discard, io.Discard)
	command.Version = "1.2.3-rc.4"
	var calls int
	var gotVersion string
	command.chatworkClientFactory = func(version string) (*chatworkapi.Client, error) {
		calls++
		gotVersion = version
		return &chatworkapi.Client{}, nil
	}
	if calls != 0 {
		t.Fatal("New initialized the Chatwork adapter eagerly")
	}
	if err := command.ensureChatwork(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || gotVersion != command.Version {
		t.Fatalf("adapter factory calls/version = %d/%q, want one call with CLI version %q", calls, gotVersion, command.Version)
	}
	if command.chatwork == nil || command.chatworkAuth == nil {
		t.Fatal("lazy Chatwork task adapter was not composed")
	}
}
