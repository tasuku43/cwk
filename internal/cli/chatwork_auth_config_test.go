package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tasuku43/cwk/internal/app/chatworkauthcmd"
	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/infra/chatworkconfig"
	"github.com/tasuku43/cwk/internal/infra/chatworkoauth"
)

type configuredLoginStub struct {
	status chatworkauth.CredentialStatus
	err    error
	calls  int
}

func (s *configuredLoginStub) Login(ctx context.Context, receive chatworkauthcmd.RedirectReceiver) (chatworkauth.CredentialStatus, error) {
	s.calls++
	if s.err != nil {
		return chatworkauth.CredentialStatus{}, s.err
	}
	if _, err := receive(ctx, "https://www.chatwork.com/packages/oauth2/login.php?client_id=public&state=synthetic"); err != nil {
		return chatworkauth.CredentialStatus{}, err
	}
	return s.status, nil
}

func TestOAuthManagerFirstLoginPersistsPublicConfigBeforeCredentialFlow(t *testing.T) {
	public := chatworkconfig.NewFileStoreAt(t.TempDir())
	login := &configuredLoginStub{status: chatworkauth.CredentialStatus{Authenticated: true, ExpiresAt: time.Unix(1_800_000_000, 0)}}
	manager := &chatworkOAuthManager{
		public: public, credentials: selectionCredentialStore{},
		configured: func(config chatworkconfig.PublicConfig, _ chatworkoauth.Store) (oauthLoginManager, error) {
			stored, err := public.Load(context.Background())
			if err != nil || stored != config {
				t.Fatalf("stored/config/error = %+v/%+v/%v", stored, config, err)
			}
			return login, nil
		},
	}
	status, err := manager.Login(context.Background(), "public-client", func(context.Context, string) (string, error) {
		return "cwk://oauth/callback?code=synthetic&state=synthetic", nil
	})
	if err != nil || !status.Authenticated || login.calls != 1 {
		t.Fatalf("status/error/calls = %+v/%v/%d", status, err, login.calls)
	}
	stored, err := public.Load(context.Background())
	if err != nil || stored.ClientID != "public-client" || stored.AuthMethod != chatworkconfig.AuthMethodOAuth2 || stored.RedirectURI != chatworkconfig.FixedRedirectURI {
		t.Fatalf("stored/error = %+v/%v", stored, err)
	}
}

func TestOAuthManagerReusesStoredConfigWithoutClientID(t *testing.T) {
	public := chatworkconfig.NewFileStoreAt(t.TempDir())
	config, err := chatworkconfig.NewOAuthPublicConfig("stored-client")
	if err != nil || public.Save(context.Background(), config) != nil {
		t.Fatal(err)
	}
	login := &configuredLoginStub{status: chatworkauth.CredentialStatus{Authenticated: true, ExpiresAt: time.Unix(1_800_000_000, 0)}}
	manager := &chatworkOAuthManager{
		public: public, credentials: selectionCredentialStore{},
		configured: func(received chatworkconfig.PublicConfig, _ chatworkoauth.Store) (oauthLoginManager, error) {
			if received != config {
				t.Fatalf("config = %+v", received)
			}
			return login, nil
		},
	}
	if _, err := manager.Login(context.Background(), "", func(context.Context, string) (string, error) { return "callback", nil }); err != nil {
		t.Fatal(err)
	}
	if login.calls != 1 {
		t.Fatalf("login calls = %d", login.calls)
	}
}

func TestOAuthManagerRejectsMissingOrChangedClientIDBeforeCredentialFlow(t *testing.T) {
	public := chatworkconfig.NewFileStoreAt(t.TempDir())
	var factoryCalls int
	manager := &chatworkOAuthManager{
		public: public, credentials: selectionCredentialStore{},
		configured: func(chatworkconfig.PublicConfig, chatworkoauth.Store) (oauthLoginManager, error) {
			factoryCalls++
			return &configuredLoginStub{}, nil
		},
	}
	_, err := manager.Login(context.Background(), "", func(context.Context, string) (string, error) { return "", nil })
	assertPublicFaultCode(t, err, "oauth_client_configuration_missing")
	config, configErr := chatworkconfig.NewOAuthPublicConfig("stored-client")
	if configErr != nil || public.Save(context.Background(), config) != nil {
		t.Fatal(configErr)
	}
	_, err = manager.Login(context.Background(), "other-client", func(context.Context, string) (string, error) { return "", nil })
	assertPublicFaultCode(t, err, "oauth_client_configuration_invalid")
	if factoryCalls != 0 {
		t.Fatalf("credential flow calls = %d", factoryCalls)
	}
}

func TestOAuthManagerRetainsSafePublicConfigWhenConsentFails(t *testing.T) {
	public := chatworkconfig.NewFileStoreAt(t.TempDir())
	private := errors.New("authorization-code-canary")
	login := &configuredLoginStub{err: private}
	manager := &chatworkOAuthManager{
		public: public, credentials: selectionCredentialStore{},
		configured: func(chatworkconfig.PublicConfig, chatworkoauth.Store) (oauthLoginManager, error) { return login, nil },
	}
	_, err := manager.Login(context.Background(), "public-client", func(context.Context, string) (string, error) { return "", nil })
	if !errors.Is(err, private) {
		t.Fatalf("login error = %v", err)
	}
	stored, loadErr := public.Load(context.Background())
	if loadErr != nil || stored.ClientID != "public-client" {
		t.Fatalf("stored/error = %+v/%v", stored, loadErr)
	}
}

func TestPublicConfigurationFaultMappingIsSecretFree(t *testing.T) {
	for _, test := range []struct {
		err  error
		kind fault.Kind
		code string
	}{
		{chatworkconfig.ErrConfigNotFound, fault.KindInvalidInput, "oauth_client_configuration_missing"},
		{chatworkconfig.ErrConfigInvalid, fault.KindInvalidInput, "oauth_client_configuration_invalid"},
		{chatworkconfig.ErrConfigUnavailable, fault.KindUnavailable, "oauth_public_configuration_unavailable"},
	} {
		public, ok := fault.PublicCopy(publicConfigurationFault(test.err))
		if !ok || public.Kind != test.kind || public.Code != test.code {
			t.Fatalf("fault = %+v/%t", public, ok)
		}
	}
}
