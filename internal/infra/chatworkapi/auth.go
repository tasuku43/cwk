package chatworkapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"sync"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

const TokenEnvironment = "CWK_API_TOKEN" // #nosec G101 -- environment variable name only; no credential value is embedded.

type credentialRecord struct {
	credential requestCredential
	session    authn.Session
}

// requestCredential is the sole token-to-request boundary. The PAT remains
// private to infrastructure and is resolved through an opaque BindingID.
type requestCredential struct {
	method authn.Method
	secret string
}

func (credential requestCredential) authorize(request *http.Request) error {
	if request == nil {
		return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork authentication session is invalid", false)
	}
	if credential.secret == "" {
		return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork authentication session is invalid", false)
	}
	request.Header.Del("Authorization")
	request.Header.Set("x-chatworktoken", credential.secret)
	return nil
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
	if credential.secret == "" {
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
	if err := session.Validate(); err != nil {
		return authn.Session{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork authentication session is invalid", false)
	}
	c.mu.Lock()
	c.records[binding] = credentialRecord{credential: credential, session: session.Clone()}
	c.mu.Unlock()
	return session.Clone(), nil
}

func newClient(baseURL, token string, httpClient httpDoer, newID func() (string, error), readFile func(string) ([]byte, error)) *Client {
	return newClientWithCredential(baseURL, requestCredential{method: authn.MethodPAT, secret: token}, httpClient, newID, readFile)
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
	if !ok || record.session.BindingID != binding || record.credential.secret == "" {
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
