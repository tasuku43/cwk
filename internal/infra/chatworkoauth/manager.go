// Package chatworkoauth implements Chatwork's public-client Authorization Code
// flow. It owns PKCE, callback verification, tokens, refresh, and credential
// storage; only secret-free identity metadata crosses into chatworkapi.
package chatworkoauth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
	"golang.org/x/oauth2"
)

const (
	AuthorizationEndpoint = "https://www.chatwork.com/packages/oauth2/login.php"
	TokenEndpoint         = "https://oauth.chatwork.com/token"
	ClientIDEnvironment   = "CWK_OAUTH_CLIENT_ID"
	RedirectEnvironment   = "CWK_OAUTH_REDIRECT_URI"

	AccountEndpoint        = "https://api.chatwork.com/v2/me"
	identityRequestTimeout = 20 * time.Second
	maxIdentityBodyBytes   = 64 * 1024
	maxRedirectBytes       = 8192
)

var requiredScopes = []string{
	"users.all:read",
	"rooms.all:read_write",
	"contacts.all:read_write",
}

// Config contains public OAuth client registration metadata. It deliberately
// has no client secret because cwk supports only Chatwork public clients.
type Config struct {
	ClientID    string
	RedirectURI string
	Scopes      []string
}

// RedirectReceiver shows the authorization URL to the user and reads one full
// redirected custom-scheme URL. The authorization code itself never becomes a
// CLI argument or a return value.
type RedirectReceiver = func(context.Context, string) (string, error)

type Manager struct {
	mu     sync.Mutex
	config oauth2.Config
	store  Store
	http   *http.Client
	now    func() time.Time
	random io.Reader
}

type storedCredential struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
	Scopes       []string  `json:"scopes"`
	AccountID    string    `json:"account_id"`
}

// New constructs the production OAuth manager with fixed official endpoints
// and a redirect-disabled bounded token client.
func New(config Config, store Store) (*Manager, error) {
	return newManager(config, store, AuthorizationEndpoint, TokenEndpoint, productionTokenClient(), time.Now, rand.Reader)
}

// NewLifecycle constructs the store-only profile lifecycle. Status and Logout
// remain available after client registration environment variables are
// removed; Login and credential use still fail closed without New.
func NewLifecycle(store Store) (*Manager, error) {
	if isNilStore(store) {
		return nil, oauthFault(fault.KindContract, "oauth_credential_store_missing", "Chatwork OAuth credential store is not configured", false)
	}
	return &Manager{store: store, http: productionTokenClient(), now: time.Now, random: rand.Reader}, nil
}

// NewFromEnvironment reads only non-secret public registration metadata. The
// access and refresh tokens remain in the supplied credential store.
func NewFromEnvironment(store Store) (*Manager, error) {
	return New(Config{
		ClientID:    os.Getenv(ClientIDEnvironment),
		RedirectURI: os.Getenv(RedirectEnvironment),
		Scopes:      RequiredScopes(),
	}, store)
}

// RequiredScopes returns the fixed public-client scopes needed by the complete
// Chatwork API task surface. offline_access is deliberately absent.
func RequiredScopes() []string {
	return append([]string(nil), requiredScopes...)
}

func newManager(config Config, store Store, authURL, tokenURL string, client *http.Client, now func() time.Time, random io.Reader) (*Manager, error) {
	if err := validateConfig(config); err != nil {
		return nil, oauthFault(fault.KindInvalidInput, "oauth_configuration_invalid", "Chatwork OAuth configuration is invalid", false)
	}
	if isNilStore(store) || client == nil || now == nil || random == nil {
		return nil, oauthFault(fault.KindContract, "oauth_manager_contract_invalid", "Chatwork OAuth manager dependencies are invalid", false)
	}
	if err := validateEndpoint(authURL, true); err != nil || validateEndpoint(tokenURL, false) != nil {
		return nil, oauthFault(fault.KindContract, "oauth_endpoint_contract_invalid", "Chatwork OAuth endpoints are invalid", false)
	}
	return &Manager{
		config: oauth2.Config{
			ClientID:    config.ClientID,
			RedirectURL: config.RedirectURI,
			Scopes:      append([]string(nil), config.Scopes...),
			Endpoint: oauth2.Endpoint{
				AuthURL:   authURL,
				TokenURL:  tokenURL,
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		store: store, http: client, now: now, random: random,
	}, nil
}

// Login performs one state-bound PKCE S256 Authorization Code exchange and
// atomically replaces the credential stored by this manager.
func (m *Manager) Login(ctx context.Context, receive RedirectReceiver) (chatworkauth.CredentialStatus, error) {
	if ctx == nil {
		return chatworkauth.CredentialStatus{}, oauthFault(fault.KindContract, "missing_authentication_context", "OAuth login context is not configured", false)
	}
	if err := ctx.Err(); err != nil {
		return chatworkauth.CredentialStatus{}, canceledFault()
	}
	if m == nil || receive == nil {
		return chatworkauth.CredentialStatus{}, oauthFault(fault.KindContract, "oauth_login_receiver_missing", "OAuth redirect receiver is not configured", false)
	}
	if !m.configured() {
		return chatworkauth.CredentialStatus{}, oauthFault(fault.KindInvalidInput, "oauth_configuration_invalid", "Chatwork OAuth client configuration is invalid", false)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.store.Load(ctx)
	switch {
	case err == nil:
		return chatworkauth.CredentialStatus{}, oauthFault(fault.KindRejected, "oauth_credential_already_present", "A Chatwork OAuth credential already exists; log out before logging in again", false)
	case errors.Is(err, errCredentialNotFound):
		// Login is a create operation. Only an absent credential may proceed.
	case ctx.Err() != nil:
		return chatworkauth.CredentialStatus{}, canceledFault()
	default:
		return chatworkauth.CredentialStatus{}, storeFault(ctx)
	}
	state, err := randomState(m.random)
	if err != nil {
		return chatworkauth.CredentialStatus{}, oauthFault(fault.KindInternal, "oauth_state_generation_failed", "OAuth login could not be initialized", false)
	}
	verifier := oauth2.GenerateVerifier()
	authorizationURL := m.config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	redirected, err := receive(ctx, authorizationURL)
	if err != nil {
		if ctx.Err() != nil {
			return chatworkauth.CredentialStatus{}, canceledFault()
		}
		return chatworkauth.CredentialStatus{}, oauthFault(fault.KindAuthentication, "oauth_redirect_receive_failed", "OAuth redirect could not be received", false)
	}
	code, err := m.verifyRedirect(redirected, state)
	if err != nil {
		return chatworkauth.CredentialStatus{}, err
	}
	exchangeCtx := context.WithValue(ctx, oauth2.HTTPClient, m.http)
	token, err := m.config.Exchange(exchangeCtx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		if ctx.Err() != nil {
			return chatworkauth.CredentialStatus{}, canceledFault()
		}
		return chatworkauth.CredentialStatus{}, oauthFault(fault.KindAuthentication, "oauth_code_exchange_failed", "Chatwork rejected the OAuth authorization code", false)
	}
	if err := validateOAuthToken(token, m.now()); err != nil {
		return chatworkauth.CredentialStatus{}, err
	}
	grantedScopes, err := tokenScopes(token, m.config.Scopes)
	if err != nil || !sameStringSet(grantedScopes, m.config.Scopes) {
		return chatworkauth.CredentialStatus{}, oauthFault(fault.KindPermission, "insufficient_authentication_capability", "Chatwork OAuth did not grant the required scopes", false)
	}
	accountID, err := m.verifyAccount(ctx, token)
	if err != nil {
		return chatworkauth.CredentialStatus{}, err
	}
	err = m.saveTokenLocked(ctx, token, grantedScopes, accountID)
	if err != nil {
		return chatworkauth.CredentialStatus{}, err
	}
	return statusFromToken(token), nil
}

func isNilStore(store Store) bool {
	if store == nil {
		return true
	}
	value := reflect.ValueOf(store)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// Identity resolves or refreshes the stored token and returns only metadata
// suitable for an ephemeral application authentication session.
func (m *Manager) Identity(ctx context.Context) (chatworkapi.OAuthIdentity, error) {
	token, stored, err := m.currentToken(ctx)
	if err != nil {
		return chatworkapi.OAuthIdentity{}, err
	}
	return m.identity(token, stored), nil
}

// Authorize resolves or refreshes at the last possible boundary, verifies the
// fixed Chatwork API destination, and sets one Bearer header.
func (m *Manager) Authorize(request *http.Request) (chatworkapi.OAuthIdentity, error) {
	if request == nil || request.URL == nil || request.Context() == nil {
		return chatworkapi.OAuthIdentity{}, oauthFault(fault.KindAuthentication, "invalid_authentication_session", "Chatwork OAuth session is invalid", false)
	}
	if request.URL.Scheme != "https" || request.URL.Host != "api.chatwork.com" || (request.URL.Path != "/v2" && !strings.HasPrefix(request.URL.Path, "/v2/")) {
		return chatworkapi.OAuthIdentity{}, oauthFault(fault.KindAuthentication, "authentication_context_mismatch", "OAuth credential destination does not match Chatwork API", false)
	}
	token, stored, err := m.currentToken(request.Context())
	if err != nil {
		return chatworkapi.OAuthIdentity{}, err
	}
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	return m.identity(token, stored), nil
}

func (m *Manager) Status(ctx context.Context) (chatworkauth.CredentialStatus, error) {
	if ctx == nil {
		return chatworkauth.CredentialStatus{}, oauthFault(fault.KindContract, "missing_authentication_context", "OAuth status context is not configured", false)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	token, _, err := m.loadTokenLocked(ctx)
	if errors.Is(err, errCredentialNotFound) {
		return chatworkauth.CredentialStatus{Authenticated: false}, nil
	}
	if err != nil {
		if errors.Is(err, errCredentialInvalid) {
			return chatworkauth.CredentialStatus{}, oauthFault(fault.KindAuthentication, "invalid_authentication_session", "Stored Chatwork OAuth credential is invalid", false)
		}
		return chatworkauth.CredentialStatus{}, storeFault(ctx)
	}
	if err := validateOAuthToken(token, m.now()); err != nil {
		return chatworkauth.CredentialStatus{Authenticated: false, ExpiresAt: token.Expiry}, nil
	}
	return statusFromToken(token), nil
}

func (m *Manager) Logout(ctx context.Context) error {
	if ctx == nil {
		return oauthFault(fault.KindContract, "missing_authentication_context", "OAuth logout context is not configured", false)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	err := m.store.Delete(ctx)
	if errors.Is(err, errCredentialNotFound) {
		return nil
	}
	if err != nil {
		return storeFault(ctx)
	}
	return nil
}

func (m *Manager) currentToken(ctx context.Context) (*oauth2.Token, storedCredential, error) {
	if ctx == nil || m == nil {
		return nil, storedCredential{}, oauthFault(fault.KindContract, "missing_authentication_context", "OAuth authentication context is not configured", false)
	}
	if ctx.Err() != nil {
		return nil, storedCredential{}, canceledFault()
	}
	if !m.configured() {
		return nil, storedCredential{}, oauthFault(fault.KindInvalidInput, "oauth_configuration_invalid", "Chatwork OAuth client configuration is invalid", false)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	token, stored, err := m.loadTokenLocked(ctx)
	if errors.Is(err, errCredentialNotFound) {
		return nil, storedCredential{}, oauthFault(fault.KindAuthentication, "oauth_credential_missing", "Chatwork OAuth login is required", false)
	}
	if err != nil {
		if errors.Is(err, errCredentialInvalid) {
			return nil, storedCredential{}, oauthFault(fault.KindAuthentication, "invalid_authentication_session", "Stored Chatwork OAuth credential is invalid", false)
		}
		return nil, storedCredential{}, storeFault(ctx)
	}
	if !sameStringSet(stored.Scopes, m.config.Scopes) || !validAccountID(stored.AccountID) {
		return nil, storedCredential{}, oauthFault(fault.KindAuthentication, "invalid_authentication_session", "Stored Chatwork OAuth identity is invalid", false)
	}
	if token.Expiry.IsZero() {
		return nil, storedCredential{}, oauthFault(fault.KindAuthentication, "invalid_authentication_session", "Stored Chatwork OAuth expiry is missing", false)
	}
	if !token.Expiry.After(m.now().Add(10*time.Second)) && token.RefreshToken == "" {
		return nil, storedCredential{}, oauthFault(fault.KindAuthentication, "authentication_expired", "Chatwork OAuth authentication expired", false)
	}
	tokenCtx := context.WithValue(ctx, oauth2.HTTPClient, m.http)
	refreshed, err := m.config.TokenSource(tokenCtx, token).Token()
	if err != nil {
		if ctx.Err() != nil {
			return nil, storedCredential{}, canceledFault()
		}
		return nil, storedCredential{}, oauthFault(fault.KindAuthentication, "oauth_refresh_failed", "Chatwork OAuth credential could not be refreshed", false)
	}
	if err := validateOAuthToken(refreshed, m.now()); err != nil {
		return nil, storedCredential{}, err
	}
	if !sameToken(token, refreshed) {
		grantedScopes, err := tokenScopes(refreshed, stored.Scopes)
		if err != nil || !sameStringSet(grantedScopes, m.config.Scopes) {
			return nil, storedCredential{}, oauthFault(fault.KindPermission, "insufficient_authentication_capability", "Refreshed Chatwork OAuth credential lost a required scope", false)
		}
		accountID, err := m.verifyAccount(ctx, refreshed)
		if err != nil {
			return nil, storedCredential{}, err
		}
		if subtle.ConstantTimeCompare([]byte(accountID), []byte(stored.AccountID)) != 1 {
			return nil, storedCredential{}, oauthFault(fault.KindAuthentication, "authentication_context_mismatch", "Refreshed Chatwork OAuth identity changed", false)
		}
		if err := m.saveTokenLocked(ctx, refreshed, grantedScopes, accountID); err != nil {
			return nil, storedCredential{}, err
		}
		stored.AccessToken = refreshed.AccessToken
		stored.RefreshToken = refreshed.RefreshToken
		stored.TokenType = refreshed.Type()
		stored.Expiry = refreshed.Expiry.UTC()
		stored.Scopes = append([]string(nil), grantedScopes...)
	}
	return refreshed, stored, nil
}

func (m *Manager) configured() bool {
	if m == nil || m.store == nil || m.http == nil || m.now == nil || m.random == nil {
		return false
	}
	config := Config{ClientID: m.config.ClientID, RedirectURI: m.config.RedirectURL, Scopes: append([]string(nil), m.config.Scopes...)}
	return validateConfig(config) == nil && m.config.Endpoint.AuthURL != "" && m.config.Endpoint.TokenURL != ""
}

func (m *Manager) identity(token *oauth2.Token, stored storedCredential) chatworkapi.OAuthIdentity {
	return chatworkapi.OAuthIdentity{
		Method:              authn.MethodOAuth2,
		SubjectID:           stored.AccountID,
		AccountID:           stored.AccountID,
		GrantedCapabilities: []string{chatwork.AuthenticationCapability},
		ExpiresAt:           token.Expiry,
	}
}

func (m *Manager) loadTokenLocked(ctx context.Context) (*oauth2.Token, storedCredential, error) {
	value, err := m.store.Load(ctx)
	if err != nil {
		return nil, storedCredential{}, err
	}
	if len(value) == 0 || len(value) > maxStoredCredentialBytes {
		return nil, storedCredential{}, errCredentialInvalid
	}
	var stored storedCredential
	if err := json.Unmarshal(value, &stored); err != nil {
		return nil, storedCredential{}, errCredentialInvalid
	}
	token := &oauth2.Token{AccessToken: stored.AccessToken, RefreshToken: stored.RefreshToken, TokenType: stored.TokenType, Expiry: stored.Expiry}
	return token, stored, nil
}

func (m *Manager) saveTokenLocked(ctx context.Context, token *oauth2.Token, scopes []string, accountID string) error {
	if !validAccountID(accountID) || !sameStringSet(scopes, m.config.Scopes) {
		return oauthFault(fault.KindAuthentication, "invalid_authentication_session", "Chatwork OAuth identity is invalid", false)
	}
	stored := storedCredential{
		AccessToken: token.AccessToken, RefreshToken: token.RefreshToken,
		TokenType: token.Type(), Expiry: token.Expiry.UTC(), Scopes: append([]string(nil), scopes...), AccountID: accountID,
	}
	value, err := json.Marshal(stored)
	if err != nil || len(value) > maxStoredCredentialBytes {
		return oauthFault(fault.KindContract, "oauth_credential_too_large", "Chatwork OAuth credential exceeds the credential-store limit", false)
	}
	if err := m.store.Save(ctx, value); err != nil {
		return storeFault(ctx)
	}
	return nil
}

func (m *Manager) verifyAccount(ctx context.Context, token *oauth2.Token) (string, error) {
	if ctx == nil || token == nil {
		return "", oauthFault(fault.KindContract, "missing_authentication_context", "OAuth identity verification context is not configured", false)
	}
	callCtx, cancel := context.WithTimeout(ctx, identityRequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(callCtx, http.MethodGet, AccountEndpoint, nil)
	if err != nil {
		return "", oauthFault(fault.KindContract, "oauth_identity_request_invalid", "Chatwork OAuth identity request is invalid", false)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err := m.http.Do(request)
	if err != nil {
		if ctx.Err() != nil || callCtx.Err() != nil {
			return "", canceledFault()
		}
		return "", oauthFault(fault.KindUnavailable, "oauth_identity_verification_unavailable", "Chatwork OAuth identity could not be verified", true)
	}
	if response == nil || response.Body == nil {
		return "", oauthFault(fault.KindContract, "oauth_identity_response_invalid", "Chatwork OAuth identity response is invalid", false)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return "", oauthFault(fault.KindAuthentication, "oauth_identity_verification_failed", "Chatwork rejected the OAuth credential during identity verification", false)
	}
	if response.StatusCode == http.StatusForbidden {
		return "", oauthFault(fault.KindPermission, "insufficient_authentication_capability", "Chatwork OAuth credential cannot read its account identity", false)
	}
	if response.StatusCode >= 500 {
		return "", oauthFault(fault.KindUnavailable, "oauth_identity_verification_unavailable", "Chatwork OAuth identity could not be verified", true)
	}
	if response.StatusCode != http.StatusOK {
		return "", oauthFault(fault.KindContract, "oauth_identity_response_invalid", "Chatwork OAuth identity response is invalid", false)
	}
	limited := io.LimitReader(response.Body, maxIdentityBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil || len(body) > maxIdentityBodyBytes {
		return "", oauthFault(fault.KindContract, "oauth_identity_response_invalid", "Chatwork OAuth identity response exceeded its contract", false)
	}
	var payload struct {
		AccountID json.Number `json:"account_id"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil || !validAccountID(string(payload.AccountID)) {
		return "", oauthFault(fault.KindContract, "oauth_identity_response_invalid", "Chatwork OAuth identity response is invalid", false)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return "", oauthFault(fault.KindContract, "oauth_identity_response_invalid", "Chatwork OAuth identity response is invalid", false)
	}
	return string(payload.AccountID), nil
}

func tokenScopes(token *oauth2.Token, fallback []string) ([]string, error) {
	if token == nil {
		return nil, errors.New("OAuth token is missing")
	}
	extra := token.Extra("scope")
	if extra == nil || extra == "" {
		return append([]string(nil), fallback...), nil
	}
	value, ok := extra.(string)
	if !ok || value == "" || containsUnsafe(value) {
		return nil, errors.New("OAuth scope response is invalid")
	}
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return nil, errors.New("OAuth scope response is empty")
	}
	seen := make(map[string]struct{}, len(parts))
	for _, scope := range parts {
		if _, exists := seen[scope]; exists {
			return nil, errors.New("OAuth scope response contains duplicates")
		}
		seen[scope] = struct{}{}
	}
	return parts, nil
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]struct{}, len(left))
	for _, value := range left {
		if value == "" {
			return false
		}
		seen[value] = struct{}{}
	}
	if len(seen) != len(left) {
		return false
	}
	seenRight := make(map[string]struct{}, len(right))
	for _, value := range right {
		if _, exists := seen[value]; !exists {
			return false
		}
		if _, exists := seenRight[value]; exists {
			return false
		}
		seenRight[value] = struct{}{}
	}
	return true
}

func validAccountID(value string) bool {
	if value == "" || len(value) > 20 || value[0] == '0' {
		return false
	}
	number, err := strconv.ParseUint(value, 10, 64)
	return err == nil && number > 0
}

func (m *Manager) verifyRedirect(raw, expectedState string) (string, error) {
	if raw == "" || len(raw) > maxRedirectBytes || containsUnsafe(raw) {
		return "", oauthFault(fault.KindInvalidInput, "oauth_redirect_invalid", "OAuth redirect URL is invalid", false)
	}
	redirected, err := url.Parse(raw)
	if err != nil || redirected.Fragment != "" || redirected.User != nil {
		return "", oauthFault(fault.KindInvalidInput, "oauth_redirect_invalid", "OAuth redirect URL is invalid", false)
	}
	registered, _ := url.Parse(m.config.RedirectURL)
	if redirected.Scheme != registered.Scheme || redirected.Host != registered.Host || redirected.Path != registered.Path {
		return "", oauthFault(fault.KindAuthentication, "oauth_redirect_mismatch", "OAuth redirect URL does not match the registered callback", false)
	}
	query := redirected.Query()
	states, codes, providerErrors := query["state"], query["code"], query["error"]
	if len(states) != 1 || subtle.ConstantTimeCompare([]byte(states[0]), []byte(expectedState)) != 1 {
		return "", oauthFault(fault.KindAuthentication, "oauth_state_mismatch", "OAuth callback state did not match", false)
	}
	if len(providerErrors) == 1 && providerErrors[0] == "access_denied" && len(codes) == 0 {
		return "", oauthFault(fault.KindRejected, "oauth_authorization_denied", "Chatwork OAuth authorization was denied", false)
	}
	if len(providerErrors) != 0 || len(codes) != 1 || codes[0] == "" || len(codes[0]) > 4096 || containsUnsafe(codes[0]) {
		return "", oauthFault(fault.KindInvalidInput, "oauth_redirect_invalid", "OAuth redirect did not contain one valid authorization code", false)
	}
	return codes[0], nil
}

func validateConfig(config Config) error {
	if config.ClientID == "" || len(config.ClientID) > 512 || containsUnsafe(config.ClientID) || !sameStringSet(config.Scopes, requiredScopes) {
		return errors.New("invalid OAuth client configuration")
	}
	redirect, err := url.Parse(config.RedirectURI)
	if err != nil || redirect.Scheme == "" || redirect.Scheme == "http" || redirect.Scheme == "https" || redirect.RawQuery != "" || redirect.Fragment != "" || redirect.User != nil {
		return errors.New("public OAuth redirect must use a custom scheme")
	}
	seen := make(map[string]struct{}, len(config.Scopes))
	for _, scope := range config.Scopes {
		if scope == "" || scope == "offline_access" || containsUnsafe(scope) {
			return errors.New("invalid public-client OAuth scope")
		}
		if _, exists := seen[scope]; exists {
			return errors.New("duplicate OAuth scope")
		}
		seen[scope] = struct{}{}
	}
	return nil
}

func validateEndpoint(raw string, authorization bool) error {
	value, err := url.Parse(raw)
	if err != nil || value.Scheme == "" || value.Host == "" || value.RawQuery != "" || value.Fragment != "" || value.User != nil {
		return errors.New("invalid OAuth endpoint")
	}
	if authorization && value.Path == "" {
		return errors.New("invalid authorization endpoint")
	}
	return nil
}

func validateOAuthToken(token *oauth2.Token, now time.Time) error {
	if token == nil || token.AccessToken == "" || len(token.AccessToken) > 8192 || containsUnsafe(token.AccessToken) || !strings.EqualFold(token.Type(), "Bearer") {
		return oauthFault(fault.KindAuthentication, "invalid_authentication_session", "Chatwork OAuth token response is invalid", false)
	}
	if token.Expiry.IsZero() || !token.Expiry.After(now) {
		return oauthFault(fault.KindAuthentication, "authentication_expired", "Chatwork OAuth authentication expired", false)
	}
	if len(token.RefreshToken) > 8192 || containsUnsafe(token.RefreshToken) {
		return oauthFault(fault.KindAuthentication, "invalid_authentication_session", "Chatwork OAuth token response is invalid", false)
	}
	return nil
}

func productionTokenClient() *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
	}
	return &http.Client{Timeout: 20 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func randomState(reader io.Reader) (string, error) {
	raw := make([]byte, 32)
	if _, err := io.ReadFull(reader, raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func statusFromToken(token *oauth2.Token) chatworkauth.CredentialStatus {
	return chatworkauth.CredentialStatus{Authenticated: true, ExpiresAt: token.Expiry}
}

func sameToken(left, right *oauth2.Token) bool {
	return left.AccessToken == right.AccessToken && left.RefreshToken == right.RefreshToken && left.Type() == right.Type() && left.Expiry.Equal(right.Expiry)
}

func containsUnsafe(value string) bool {
	if strings.TrimSpace(value) != value {
		return true
	}
	for _, r := range value {
		if unicode.Is(unicode.C, r) || r == '\u2028' || r == '\u2029' {
			return true
		}
	}
	return false
}

func canceledFault() error {
	return oauthFault(fault.KindCanceled, "authentication_canceled", "Chatwork OAuth authentication was canceled", false)
}

func storeFault(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return canceledFault()
	}
	return oauthFault(fault.KindUnavailable, "oauth_credential_store_unavailable", "Chatwork OAuth credential store is unavailable", true)
}

func oauthFault(kind fault.Kind, code, message string, retryable bool) error {
	return fault.New(kind, code, message, retryable)
}
