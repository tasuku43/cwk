package chatworkoauth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tasuku43/cwk/internal/domain/fault"
	"golang.org/x/oauth2"
)

const (
	syntheticAccess  = "synthetic-access-token"
	syntheticRefresh = "synthetic-refresh-token"
	secretCanary     = "oauth-secret-canary"
)

type memoryStore struct {
	mu        sync.Mutex
	value     []byte
	loadErr   error
	saveErr   error
	deleteErr error
	saves     int
	deletes   int
}

func (store *memoryStore) Load(ctx context.Context) ([]byte, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if store.loadErr != nil {
		return nil, store.loadErr
	}
	if len(store.value) == 0 {
		return nil, errCredentialNotFound
	}
	return append([]byte(nil), store.value...), nil
}

func (store *memoryStore) Save(ctx context.Context, value []byte) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if store.saveErr != nil {
		return store.saveErr
	}
	store.value = append([]byte(nil), value...)
	store.saves++
	return nil
}

func (store *memoryStore) Delete(ctx context.Context) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if store.deleteErr != nil {
		return store.deleteErr
	}
	if len(store.value) == 0 {
		return errCredentialNotFound
	}
	store.value = nil
	store.deletes++
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func oauthTestConfig() Config {
	return Config{ClientID: "synthetic-client-id", RedirectURI: "cwk://oauth/callback", Scopes: RequiredScopes()}
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestLoginUsesStatePKCES256PublicClientAndPersistsCredential(t *testing.T) {
	store := &memoryStore{}
	var tokenCalls int
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == AccountEndpoint {
			if request.Header.Get("Authorization") != "Bearer "+syntheticAccess {
				t.Fatalf("identity authorization = %q", request.Header.Get("Authorization"))
			}
			return response(http.StatusOK, `{"account_id":42,"name":"synthetic"}`), nil
		}
		tokenCalls++
		if request.URL.String() != "https://oauth.example.test/token" || request.Method != http.MethodPost {
			t.Fatalf("token request = %s %s", request.Method, request.URL)
		}
		if user, password, ok := request.BasicAuth(); ok || user != "" || password != "" {
			t.Fatal("public client sent basic authentication")
		}
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if request.Form.Get("grant_type") != "authorization_code" || request.Form.Get("client_id") != "synthetic-client-id" || request.Form.Get("code") != "synthetic-code" {
			t.Fatalf("token form = %#v", request.Form)
		}
		if len(request.Form.Get("code_verifier")) < 43 {
			t.Fatal("PKCE verifier is missing")
		}
		return response(http.StatusOK, `{"access_token":"`+syntheticAccess+`","refresh_token":"`+syntheticRefresh+`","token_type":"Bearer","expires_in":1800,"scope":"users.all:read rooms.all:read_write contacts.all:read_write"}`), nil
	})}
	manager, err := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", client, time.Now, strings.NewReader(strings.Repeat("a", 32)))
	if err != nil {
		t.Fatal(err)
	}
	var challenge string
	status, err := manager.Login(context.Background(), func(_ context.Context, authorizationURL string) (string, error) {
		parsed, parseErr := url.Parse(authorizationURL)
		if parseErr != nil {
			return "", parseErr
		}
		query := parsed.Query()
		challenge = query.Get("code_challenge")
		if parsed.Scheme != "https" || parsed.Host != "auth.example.test" || query.Get("response_type") != "code" ||
			query.Get("state") == "" || challenge == "" || query.Get("code_challenge_method") != "S256" ||
			strings.Contains(query.Get("scope"), "offline_access") {
			t.Fatalf("authorization URL = %s", authorizationURL)
		}
		return "cwk://oauth/callback?code=synthetic-code&state=" + url.QueryEscape(query.Get("state")), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if challenge == "" || !status.Authenticated || tokenCalls != 1 || store.saves != 1 {
		t.Fatalf("status=%+v tokenCalls=%d saves=%d challenge=%q", status, tokenCalls, store.saves, challenge)
	}
	if !strings.Contains(string(store.value), syntheticAccess) || !strings.Contains(string(store.value), syntheticRefresh) {
		t.Fatal("credential store did not receive the complete rotated token pair")
	}
	if strings.Contains(toJSON(t, status), syntheticAccess) || strings.Contains(toJSON(t, status), syntheticRefresh) {
		t.Fatal("status exposed OAuth token")
	}
}

func TestLoginVerifierMatchesAuthorizationChallenge(t *testing.T) {
	store := &memoryStore{}
	var challenge string
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == AccountEndpoint {
			return response(http.StatusOK, `{"account_id":42}`), nil
		}
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := oauth2.S256ChallengeFromVerifier(request.Form.Get("code_verifier")); got != challenge {
			t.Fatalf("PKCE challenge = %q, want %q", got, challenge)
		}
		return response(http.StatusOK, `{"access_token":"`+syntheticAccess+`","refresh_token":"`+syntheticRefresh+`","token_type":"Bearer","expires_in":1800}`), nil
	})}
	manager, err := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", client, time.Now, strings.NewReader(strings.Repeat("b", 32)))
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Login(context.Background(), func(_ context.Context, authorizationURL string) (string, error) {
		parsed, _ := url.Parse(authorizationURL)
		challenge = parsed.Query().Get("code_challenge")
		return "cwk://oauth/callback?code=synthetic-code&state=" + url.QueryEscape(parsed.Query().Get("state")), nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoginRejectsStateAndRedirectMismatchBeforeTokenExchange(t *testing.T) {
	for name, redirect := range map[string]string{
		"state":  "cwk://oauth/callback?code=synthetic-code&state=wrong",
		"origin": "other://oauth/callback?code=synthetic-code&state=STATE",
		"http":   "http://oauth/callback?code=synthetic-code&state=STATE",
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			store := &memoryStore{}
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected") })}
			manager, err := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", client, time.Now, strings.NewReader(strings.Repeat("c", 32)))
			if err != nil {
				t.Fatal(err)
			}
			_, err = manager.Login(context.Background(), func(_ context.Context, authorizationURL string) (string, error) {
				parsed, _ := url.Parse(authorizationURL)
				return strings.ReplaceAll(redirect, "STATE", url.QueryEscape(parsed.Query().Get("state"))), nil
			})
			if err == nil || calls != 0 || store.saves != 0 {
				t.Fatalf("error=%v calls=%d saves=%d", err, calls, store.saves)
			}
		})
	}
}

func TestLoginRefusesToOverwriteExistingCredentialBeforeConsent(t *testing.T) {
	store := &memoryStore{value: []byte(`{"existing":"credential"}`)}
	calls := 0
	manager, err := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("unexpected")
	})}, time.Now, strings.NewReader(strings.Repeat("z", 32)))
	if err != nil {
		t.Fatal(err)
	}
	receiverCalls := 0
	_, err = manager.Login(context.Background(), func(context.Context, string) (string, error) {
		receiverCalls++
		return "", errors.New("unexpected")
	})
	assertFault(t, err, fault.KindRejected, "oauth_credential_already_present")
	if receiverCalls != 0 || calls != 0 || store.saves != 0 {
		t.Fatalf("receiver=%d token/me calls=%d saves=%d", receiverCalls, calls, store.saves)
	}
}

func TestConcurrentLoginCreatesExactlyOneStoredCredential(t *testing.T) {
	store := &memoryStore{}
	tokenCalls, identityCalls := 0, 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == AccountEndpoint {
			identityCalls++
			return response(http.StatusOK, `{"account_id":42}`), nil
		}
		tokenCalls++
		return response(http.StatusOK, `{"access_token":"`+syntheticAccess+`","refresh_token":"`+syntheticRefresh+`","token_type":"Bearer","expires_in":1800}`), nil
	})}
	manager, _ := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", client, time.Now, strings.NewReader(strings.Repeat("m", 64)))
	entered := make(chan struct{})
	release := make(chan struct{})
	receiverCalls := 0
	receiver := func(_ context.Context, authorizationURL string) (string, error) {
		receiverCalls++
		close(entered)
		<-release
		parsed, _ := url.Parse(authorizationURL)
		return "cwk://oauth/callback?code=synthetic-code&state=" + url.QueryEscape(parsed.Query().Get("state")), nil
	}
	first := make(chan error, 1)
	second := make(chan error, 1)
	go func() { _, err := manager.Login(context.Background(), receiver); first <- err }()
	<-entered
	go func() { _, err := manager.Login(context.Background(), receiver); second <- err }()
	close(release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	err := <-second
	assertFault(t, err, fault.KindRejected, "oauth_credential_already_present")
	if receiverCalls != 1 || tokenCalls != 1 || identityCalls != 1 || store.saves != 1 {
		t.Fatalf("receiver=%d token=%d identity=%d saves=%d", receiverCalls, tokenCalls, identityCalls, store.saves)
	}
}

func TestLoginRejectsNarrowedProviderScopeBeforeIdentityOrSave(t *testing.T) {
	store := &memoryStore{}
	identityCalls := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == AccountEndpoint {
			identityCalls++
			return response(http.StatusOK, `{"account_id":42}`), nil
		}
		return response(http.StatusOK, `{"access_token":"`+syntheticAccess+`","refresh_token":"`+syntheticRefresh+`","token_type":"Bearer","expires_in":1800,"scope":"users.all:read"}`), nil
	})}
	manager, _ := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", client, time.Now, strings.NewReader(strings.Repeat("j", 32)))
	_, err := manager.Login(context.Background(), func(_ context.Context, authorizationURL string) (string, error) {
		parsed, _ := url.Parse(authorizationURL)
		return "cwk://oauth/callback?code=synthetic-code&state=" + url.QueryEscape(parsed.Query().Get("state")), nil
	})
	assertFault(t, err, fault.KindPermission, "insufficient_authentication_capability")
	if identityCalls != 0 || store.saves != 0 {
		t.Fatalf("identity calls=%d saves=%d", identityCalls, store.saves)
	}
}

func TestRefreshRotationPersistsBeforeAuthorizing(t *testing.T) {
	now := time.Now()
	store := &memoryStore{}
	stored := storedCredential{AccessToken: "expired-access", RefreshToken: syntheticRefresh, TokenType: "Bearer", Expiry: now.Add(-time.Hour), Scopes: RequiredScopes(), AccountID: "42"}
	store.value, _ = json.Marshal(stored)
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == AccountEndpoint {
			if request.Header.Get("Authorization") != "Bearer rotated-access" {
				t.Fatalf("identity authorization = %q", request.Header.Get("Authorization"))
			}
			return response(http.StatusOK, `{"account_id":42}`), nil
		}
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if request.Form.Get("grant_type") != "refresh_token" || request.Form.Get("refresh_token") != syntheticRefresh || request.Form.Get("client_id") != "synthetic-client-id" {
			t.Fatalf("refresh form = %#v", request.Form)
		}
		return response(http.StatusOK, `{"access_token":"rotated-access","refresh_token":"rotated-refresh","token_type":"Bearer","expires_in":1800,"scope":"users.all:read rooms.all:read_write contacts.all:read_write"}`), nil
	})}
	manager, err := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", client, time.Now, strings.NewReader(strings.Repeat("d", 32)))
	if err != nil {
		t.Fatal(err)
	}
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.chatwork.com/v2/me", nil)
	identity, err := manager.Authorize(request)
	if err != nil {
		t.Fatal(err)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer rotated-access" {
		t.Fatalf("authorization = %q", got)
	}
	if store.saves != 1 || !strings.Contains(string(store.value), "rotated-refresh") || identity.ExpiresAt.IsZero() {
		t.Fatalf("saves=%d stored=%s identity=%+v", store.saves, store.value, identity)
	}
}

func TestRefreshStoreFailureDoesNotAuthorizeWithUnpersistedToken(t *testing.T) {
	now := time.Now()
	store := &memoryStore{saveErr: errors.New(secretCanary)}
	stored := storedCredential{AccessToken: "expired-access", RefreshToken: syntheticRefresh, TokenType: "Bearer", Expiry: now.Add(-time.Hour), Scopes: RequiredScopes(), AccountID: "42"}
	store.value, _ = json.Marshal(stored)
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == AccountEndpoint {
			return response(http.StatusOK, `{"account_id":42}`), nil
		}
		return response(http.StatusOK, `{"access_token":"rotated-access","refresh_token":"rotated-refresh","token_type":"Bearer","expires_in":1800}`), nil
	})}
	manager, _ := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", client, time.Now, strings.NewReader(strings.Repeat("e", 32)))
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.chatwork.com/v2/me", nil)
	_, err := manager.Authorize(request)
	assertFault(t, err, fault.KindUnavailable, "oauth_credential_store_unavailable")
	if request.Header.Get("Authorization") != "" || strings.Contains(err.Error(), secretCanary) {
		t.Fatal("failed refresh authorized request or exposed store error")
	}
}

func TestRefreshIdentityChangeFailsBeforeSaveOrAuthorization(t *testing.T) {
	store := &memoryStore{}
	stored := storedCredential{AccessToken: "expired-access", RefreshToken: syntheticRefresh, TokenType: "Bearer", Expiry: time.Now().Add(-time.Hour), Scopes: RequiredScopes(), AccountID: "42"}
	store.value, _ = json.Marshal(stored)
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == AccountEndpoint {
			return response(http.StatusOK, `{"account_id":43}`), nil
		}
		return response(http.StatusOK, `{"access_token":"rotated-access","refresh_token":"rotated-refresh","token_type":"Bearer","expires_in":1800}`), nil
	})}
	manager, _ := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", client, time.Now, strings.NewReader(strings.Repeat("k", 32)))
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.chatwork.com/v2/me", nil)
	_, err := manager.Authorize(request)
	assertFault(t, err, fault.KindAuthentication, "authentication_context_mismatch")
	if store.saves != 0 || request.Header.Get("Authorization") != "" || strings.Contains(string(store.value), "rotated-access") {
		t.Fatal("identity-changing refresh was persisted or authorized")
	}
}

func TestIdentityVerificationIsBoundedAndSecretFree(t *testing.T) {
	store := &memoryStore{}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == AccountEndpoint {
			return response(http.StatusOK, strings.Repeat(secretCanary, maxIdentityBodyBytes)), nil
		}
		return response(http.StatusOK, `{"access_token":"`+syntheticAccess+`","refresh_token":"`+syntheticRefresh+`","token_type":"Bearer","expires_in":1800}`), nil
	})}
	manager, _ := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", client, time.Now, strings.NewReader(strings.Repeat("l", 32)))
	_, err := manager.Login(context.Background(), func(_ context.Context, authorizationURL string) (string, error) {
		parsed, _ := url.Parse(authorizationURL)
		return "cwk://oauth/callback?code=synthetic-code&state=" + url.QueryEscape(parsed.Query().Get("state")), nil
	})
	assertFault(t, err, fault.KindContract, "oauth_identity_response_invalid")
	if strings.Contains(err.Error(), secretCanary) || store.saves != 0 {
		t.Fatal("oversized identity response was exposed or persisted")
	}
}

func TestLoginAndRefreshErrorsAreSecretFree(t *testing.T) {
	store := &memoryStore{}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusBadRequest, `{"error":"invalid_grant","error_description":"`+secretCanary+`"}`), nil
	})}
	manager, _ := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", client, time.Now, strings.NewReader(strings.Repeat("f", 32)))
	_, err := manager.Login(context.Background(), func(_ context.Context, authorizationURL string) (string, error) {
		parsed, _ := url.Parse(authorizationURL)
		return "cwk://oauth/callback?code=" + secretCanary + "&state=" + url.QueryEscape(parsed.Query().Get("state")), nil
	})
	assertFault(t, err, fault.KindAuthentication, "oauth_code_exchange_failed")
	if strings.Contains(err.Error(), secretCanary) {
		t.Fatal("OAuth exchange error exposed authorization code or provider body")
	}
}

func TestStatusAndLogoutHandleMissingAndUnavailableStore(t *testing.T) {
	manager, _ := newManager(oauthTestConfig(), &memoryStore{}, "https://auth.example.test/login", "https://oauth.example.test/token", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("unused") })}, time.Now, strings.NewReader(strings.Repeat("g", 32)))
	status, err := manager.Status(context.Background())
	if err != nil || status.Authenticated {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	if err := manager.Logout(context.Background()); err != nil {
		t.Fatalf("idempotent logout error = %v", err)
	}

	broken := &memoryStore{loadErr: errors.New(secretCanary), deleteErr: errors.New(secretCanary)}
	manager, _ = newManager(oauthTestConfig(), broken, "https://auth.example.test/login", "https://oauth.example.test/token", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("unused") })}, time.Now, strings.NewReader(strings.Repeat("h", 32)))
	_, err = manager.Status(context.Background())
	assertFault(t, err, fault.KindUnavailable, "oauth_credential_store_unavailable")
	err = manager.Logout(context.Background())
	assertFault(t, err, fault.KindUnavailable, "oauth_credential_store_unavailable")
	if strings.Contains(err.Error(), secretCanary) {
		t.Fatal("credential-store failure exposed private cause")
	}
}

func TestLifecycleStatusAndLogoutDoNotRequireClientRegistration(t *testing.T) {
	store := &memoryStore{}
	manager, err := NewLifecycle(store)
	if err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status(context.Background())
	if err != nil || status.Authenticated {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	if err := manager.Logout(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err = manager.Login(context.Background(), func(context.Context, string) (string, error) { return "", nil })
	assertFault(t, err, fault.KindInvalidInput, "oauth_configuration_invalid")
	_, err = manager.Identity(context.Background())
	assertFault(t, err, fault.KindInvalidInput, "oauth_configuration_invalid")
}

func TestConfigRequiresCustomSchemeAndRejectsOfflineAccess(t *testing.T) {
	for name, config := range map[string]Config{
		"http":    {ClientID: "client", RedirectURI: "http://localhost/callback", Scopes: RequiredScopes()},
		"https":   {ClientID: "client", RedirectURI: "https://example.test/callback", Scopes: RequiredScopes()},
		"offline": {ClientID: "client", RedirectURI: "cwk://oauth/callback", Scopes: []string{"offline_access"}},
		"narrow":  {ClientID: "client", RedirectURI: "cwk://oauth/callback", Scopes: []string{"users.all:read"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := New(config, &memoryStore{}); err == nil {
				t.Fatal("unsafe public-client configuration succeeded")
			}
		})
	}
}

func TestProductionTokenClientIsBoundedAndRejectsRedirects(t *testing.T) {
	client := productionTokenClient()
	if client.Timeout != 20*time.Second || client.CheckRedirect == nil {
		t.Fatal("production OAuth client is not bounded")
	}
	request, _ := http.NewRequest(http.MethodGet, "https://oauth.chatwork.com/token", nil)
	if !errors.Is(client.CheckRedirect(request, nil), http.ErrUseLastResponse) {
		t.Fatal("production OAuth client follows redirects")
	}
}

func TestAuthorizeRejectsNonChatworkDestinationBeforeCredentialUse(t *testing.T) {
	store := &memoryStore{}
	stored := storedCredential{AccessToken: syntheticAccess, RefreshToken: syntheticRefresh, TokenType: "Bearer", Expiry: time.Now().Add(time.Hour), Scopes: RequiredScopes(), AccountID: "42"}
	store.value, _ = json.Marshal(stored)
	manager, _ := newManager(oauthTestConfig(), store, "https://auth.example.test/login", "https://oauth.example.test/token", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("unused") })}, time.Now, strings.NewReader(strings.Repeat("i", 32)))
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test/v2/me", nil)
	_, err := manager.Authorize(request)
	assertFault(t, err, fault.KindAuthentication, "authentication_context_mismatch")
	if request.Header.Get("Authorization") != "" {
		t.Fatal("credential sent to a non-Chatwork destination")
	}
}

func assertFault(t *testing.T, err error, kind fault.Kind, code string) {
	t.Helper()
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Kind != kind || structured.Code != code {
		t.Fatalf("error = %#v, want %s/%s", err, kind, code)
	}
}

func toJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
