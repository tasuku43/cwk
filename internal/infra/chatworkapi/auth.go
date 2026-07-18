package chatworkapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

const TokenEnvironment = "CWK_API_TOKEN" // #nosec G101 -- environment variable name only; no credential value is embedded.

type credentialRecord struct {
	credential requestCredential
	session    authn.Session
}

type credentialHeader uint8

const (
	credentialHeaderChatworkToken credentialHeader = iota + 1
	credentialHeaderBearer
)

// requestCredential is the sole token-to-request boundary. A future OAuth
// source can create a bearer record behind the same BindingID contract without
// changing task ports or request mapping.
type requestCredential struct {
	method authn.Method
	header credentialHeader
	secret string
	oauth  OAuthCredentialSource
	bound  OAuthIdentity
}

func (credential requestCredential) authorize(request *http.Request) error {
	if request == nil {
		return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork authentication session is invalid", false)
	}
	if credential.oauth != nil {
		identity, err := credential.oauth.Authorize(request)
		if err != nil {
			request.Header.Del("Authorization")
			return publicOAuthFault(err)
		}
		if err := identity.validate(); err != nil || !identity.equal(credential.bound) {
			request.Header.Del("Authorization")
			return fault.New(fault.KindAuthentication, "authentication_context_mismatch", "authentication does not match the Chatwork API context", false)
		}
		header := request.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") || len(header) == len("Bearer ") {
			request.Header.Del("Authorization")
			return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork authentication session is invalid", false)
		}
		return nil
	}
	if credential.secret == "" {
		return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork authentication session is invalid", false)
	}
	switch credential.header {
	case credentialHeaderChatworkToken:
		request.Header.Set("x-chatworktoken", credential.secret)
	case credentialHeaderBearer:
		request.Header.Set("Authorization", "Bearer "+credential.secret)
	default:
		return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork authentication session is invalid", false)
	}
	return nil
}

// OAuthIdentity is the secret-free metadata returned by an infrastructure
// OAuth source. Access and refresh tokens never cross this seam.
type OAuthIdentity struct {
	Method              authn.Method
	SubjectID           string
	AccountID           string
	GrantedCapabilities []string
	ExpiresAt           time.Time
}

func (identity OAuthIdentity) validate() error {
	if identity.Method != authn.MethodOAuth2 || identity.SubjectID == "" || identity.AccountID == "" || identity.SubjectID != identity.AccountID || identity.ExpiresAt.IsZero() {
		return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork OAuth session is invalid", false)
	}
	session := authn.Session{
		Method:              identity.Method,
		Authority:           chatwork.AuthenticationAuthority,
		Audience:            chatwork.AuthenticationAudience,
		SubjectID:           identity.SubjectID,
		GrantedCapabilities: append([]string(nil), identity.GrantedCapabilities...),
		ExpiresAt:           identity.ExpiresAt,
	}
	// Binding validation belongs to chatworkapi after it creates the binding.
	if identity.AccountID != "" {
		session.AccountID = identity.AccountID
	}
	if session.Method.Validate() != nil || len(session.GrantedCapabilities) == 0 {
		return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork OAuth session is invalid", false)
	}
	seen := make(map[string]struct{}, len(session.GrantedCapabilities))
	for _, capability := range session.GrantedCapabilities {
		if capability == "" {
			return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork OAuth session is invalid", false)
		}
		if _, exists := seen[capability]; exists {
			return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork OAuth session is invalid", false)
		}
		seen[capability] = struct{}{}
	}
	return nil
}

func (identity OAuthIdentity) equal(other OAuthIdentity) bool {
	return identity.Method == other.Method && identity.SubjectID == other.SubjectID && identity.AccountID == other.AccountID &&
		slices.Equal(identity.GrantedCapabilities, other.GrantedCapabilities)
}

// OAuthCredentialSource owns all token-bearing values. Authorize must resolve
// or refresh the stored credential using request.Context, set one Bearer
// Authorization header, and return only secret-free identity metadata.
type OAuthCredentialSource interface {
	Identity(context.Context) (OAuthIdentity, error)
	Authorize(*http.Request) (OAuthIdentity, error)
}

// Client owns the Chatwork API token, its ephemeral authentication bindings,
// and the bounded provider transport. Token-bearing values never leave this
// package.
type Client struct {
	mu       sync.RWMutex
	records  map[authn.BindingID]credentialRecord
	source   requestCredential
	baseURL  string
	http     httpDoer
	newID    func() (string, error)
	readFile func(string) ([]byte, error)
}

// NewFromEnvironment constructs the production adapter. The token is read
// once and the destination cannot be overridden by argv or environment.
func NewFromEnvironment() (*Client, error) {
	token := os.Getenv(TokenEnvironment)
	if token == "" {
		return nil, fault.New(fault.KindAuthentication, "chatwork_token_missing", "Chatwork API token is not configured", false)
	}
	if err := validateToken(token); err != nil {
		return nil, fault.New(fault.KindAuthentication, "chatwork_token_invalid", "Chatwork API token is invalid", false)
	}
	return newClient(ProductionBaseURL, token, productionHTTPClient(), randomBindingID, boundedReadFile), nil
}

// NewWithOAuth constructs the production Chatwork API adapter around an
// infrastructure OAuth source. The fixed provider destination is unchanged.
func NewWithOAuth(source OAuthCredentialSource) (*Client, error) {
	if isNilOAuthSource(source) {
		return nil, fault.New(fault.KindAuthentication, "missing_authenticator", "Chatwork OAuth authentication is not configured", false)
	}
	credential := requestCredential{method: authn.MethodOAuth2, header: credentialHeaderBearer, oauth: source}
	return newClientWithCredential(ProductionBaseURL, credential, productionHTTPClient(), randomBindingID, boundedReadFile), nil
}

// Authenticate creates a fresh process-local binding for the one configured
// token. The returned session contains metadata only.
func (c *Client) Authenticate(ctx context.Context, requirement authn.Requirement) (authn.Session, error) {
	if ctx == nil {
		return authn.Session{}, fault.New(fault.KindContract, "missing_authentication_context", "authentication context is not configured", false)
	}
	if err := ctx.Err(); err != nil {
		return authn.Session{}, fault.Wrap(fault.KindCanceled, "authentication_canceled", "authentication was canceled", false, err)
	}
	if err := requirement.Validate(); err != nil {
		return authn.Session{}, fault.New(fault.KindContract, "invalid_authentication_requirement", "authentication requirement is invalid", false)
	}
	if c == nil || c.records == nil || c.newID == nil {
		return authn.Session{}, fault.New(fault.KindAuthentication, "missing_authenticator", "Chatwork authentication is not configured", false)
	}
	if !allowsMethod(requirement.Methods, c.source.method) || requirement.Authority != chatwork.AuthenticationAuthority || requirement.Audience != chatwork.AuthenticationAudience {
		return authn.Session{}, fault.New(fault.KindAuthentication, "authentication_context_mismatch", "authentication does not match the Chatwork API context", false)
	}
	for _, capability := range requirement.RequiredCapabilities {
		if capability != chatwork.AuthenticationCapability {
			return authn.Session{}, fault.New(fault.KindPermission, "insufficient_authentication_capability", "authentication does not grant a required Chatwork capability", false)
		}
	}

	credential := c.source
	if credential.method.Validate() != nil {
		return authn.Session{}, fault.New(fault.KindAuthentication, "chatwork_token_missing", "Chatwork API token is not configured", false)
	}
	identity := OAuthIdentity{}
	if credential.oauth != nil {
		resolved, err := credential.oauth.Identity(ctx)
		if err != nil {
			return authn.Session{}, publicOAuthFault(err)
		}
		if err := resolved.validate(); err != nil || resolved.Method != credential.method {
			return authn.Session{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork OAuth session is invalid", false)
		}
		identity = resolved
		credential.bound = resolved
	} else if credential.secret == "" {
		return authn.Session{}, fault.New(fault.KindAuthentication, "chatwork_token_missing", "Chatwork API token is not configured", false)
	}

	value, err := c.newID()
	if err != nil {
		return authn.Session{}, fault.New(fault.KindAuthentication, "authentication_failed", "Chatwork authentication binding could not be created", false)
	}
	binding, err := authn.NewBindingID(value)
	if err != nil {
		return authn.Session{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork authentication binding is invalid", false)
	}
	session := authn.Session{
		Method:              credential.method,
		Authority:           chatwork.AuthenticationAuthority,
		Audience:            chatwork.AuthenticationAudience,
		SubjectID:           "configured-chatwork-account",
		BindingID:           binding,
		GrantedCapabilities: append([]string(nil), requirement.RequiredCapabilities...),
	}
	if credential.oauth != nil {
		session.SubjectID = identity.SubjectID
		session.AccountID = identity.AccountID
		session.GrantedCapabilities = append([]string(nil), identity.GrantedCapabilities...)
		session.ExpiresAt = identity.ExpiresAt
	}
	if err := session.Validate(); err != nil {
		return authn.Session{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork authentication session is invalid", false)
	}
	c.mu.Lock()
	c.records[binding] = credentialRecord{credential: credential, session: session.Clone()}
	c.mu.Unlock()
	return session.Clone(), nil
}

func newClient(baseURL, token string, httpClient httpDoer, newID func() (string, error), readFile func(string) ([]byte, error)) *Client {
	return newClientWithCredential(baseURL, requestCredential{method: authn.MethodPAT, header: credentialHeaderChatworkToken, secret: token}, httpClient, newID, readFile)
}

func newClientWithCredential(baseURL string, credential requestCredential, httpClient httpDoer, newID func() (string, error), readFile func(string) ([]byte, error)) *Client {
	client := &Client{
		records:  make(map[authn.BindingID]credentialRecord),
		source:   credential,
		baseURL:  baseURL,
		http:     httpClient,
		newID:    newID,
		readFile: readFile,
	}
	return client
}

func (c *Client) resolve(binding authn.BindingID) (credentialRecord, error) {
	if err := binding.Validate(); err != nil {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "invalid_authentication_binding", "Chatwork authentication binding is invalid", false)
	}
	if c == nil {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "invalid_authentication_binding", "Chatwork authentication binding is unavailable", false)
	}
	c.mu.RLock()
	record, ok := c.records[binding]
	c.mu.RUnlock()
	if !ok || record.session.BindingID != binding || (record.credential.oauth == nil && record.credential.secret == "") {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "invalid_authentication_binding", "Chatwork authentication binding is unavailable", false)
	}
	if err := record.session.Validate(); err != nil {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork authentication session is invalid", false)
	}
	if record.session.Authority != chatwork.AuthenticationAuthority || record.session.Audience != chatwork.AuthenticationAudience {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "authentication_context_mismatch", "authentication does not match the Chatwork API context", false)
	}
	if record.credential.method != record.session.Method {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "authentication_context_mismatch", "authentication does not match the Chatwork API context", false)
	}
	return record, nil
}

func isNilOAuthSource(source OAuthCredentialSource) bool {
	if source == nil {
		return true
	}
	value := reflect.ValueOf(source)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func publicOAuthFault(err error) error {
	if err == nil {
		return nil
	}
	if public, ok := fault.PublicCopy(err); ok {
		return fault.New(public.Kind, public.Code, "Chatwork OAuth authentication failed", public.Retryable, public.NextActions...)
	}
	return fault.New(fault.KindAuthentication, "authentication_failed", "Chatwork OAuth authentication failed", false)
}

func allowsMethod(methods []authn.Method, wanted authn.Method) bool {
	for _, method := range methods {
		if method == wanted {
			return true
		}
	}
	return false
}

func randomBindingID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "cwk-binding-" + hex.EncodeToString(raw[:]), nil
}

func validateToken(token string) error {
	if len(token) < 8 || len(token) > 4096 {
		return fault.New(fault.KindAuthentication, "chatwork_token_invalid", "Chatwork API token is invalid", false)
	}
	for _, r := range token {
		if r < 0x21 || r > 0x7e {
			return fault.New(fault.KindAuthentication, "chatwork_token_invalid", "Chatwork API token is invalid", false)
		}
	}
	return nil
}
