package chatworkapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
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

// requestCredential is the sole token-to-request boundary. The PAT remains
// private to infrastructure and is resolved through an opaque BindingID.
type requestCredential struct {
	method authn.Method
	secret string
}

func (credential requestCredential) authorize(request *http.Request) error {
	if request == nil {
		return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork 認証セッションは無効です", false)
	}
	if credential.secret == "" {
		return fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork 認証セッションは無効です", false)
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
	now      func() time.Time
	newID    func() (string, error)
	readFile func(string) ([]byte, error)
}

// NewFromEnvironment constructs the production adapter. The token is read
// once and the destination cannot be overridden by argv or environment.
func NewFromEnvironment() (*Client, error) {
	token := os.Getenv(TokenEnvironment)
	if token == "" {
		return nil, fault.New(fault.KindAuthentication, "chatwork_token_missing", "Chatwork API トークンが設定されていません", false)
	}
	if err := validateToken(token); err != nil {
		return nil, fault.New(fault.KindAuthentication, "chatwork_token_invalid", "Chatwork API トークンは無効です", false)
	}
	return newClient(ProductionBaseURL, token, productionHTTPClient(), randomBindingID, boundedReadFile), nil
}

// Authenticate creates a fresh process-local binding for the one configured
// token. The returned session contains metadata only.
func (c *Client) Authenticate(ctx context.Context, requirement authn.Requirement) (authn.Session, error) {
	if ctx == nil {
		return authn.Session{}, fault.New(fault.KindContract, "missing_authentication_context", "認証コンテキストが設定されていません", false)
	}
	if err := ctx.Err(); err != nil {
		return authn.Session{}, fault.Wrap(fault.KindCanceled, "authentication_canceled", "認証がキャンセルされました", false, err)
	}
	if err := requirement.Validate(); err != nil {
		return authn.Session{}, fault.New(fault.KindContract, "invalid_authentication_requirement", "認証要件は無効です", false)
	}
	if c == nil || c.records == nil || c.newID == nil {
		return authn.Session{}, fault.New(fault.KindAuthentication, "missing_authenticator", "Chatwork 認証が設定されていません", false)
	}
	if !allowsMethod(requirement.Methods, c.source.method) || requirement.Authority != chatwork.AuthenticationAuthority || requirement.Audience != chatwork.AuthenticationAudience {
		return authn.Session{}, fault.New(fault.KindAuthentication, "authentication_context_mismatch", "認証が Chatwork API コンテキストと一致しません", false)
	}
	for _, capability := range requirement.RequiredCapabilities {
		if capability != chatwork.AuthenticationCapability {
			return authn.Session{}, fault.New(fault.KindPermission, "insufficient_authentication_capability", "認証に必要な Chatwork 権限がありません", false)
		}
	}

	credential := c.source
	if credential.method.Validate() != nil {
		return authn.Session{}, fault.New(fault.KindAuthentication, "chatwork_token_missing", "Chatwork API トークンが設定されていません", false)
	}
	if credential.secret == "" {
		return authn.Session{}, fault.New(fault.KindAuthentication, "chatwork_token_missing", "Chatwork API トークンが設定されていません", false)
	}

	value, err := c.newID()
	if err != nil {
		return authn.Session{}, fault.New(fault.KindAuthentication, "authentication_failed", "Chatwork 認証バインドを作成できませんでした", false)
	}
	binding, err := authn.NewBindingID(value)
	if err != nil {
		return authn.Session{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork 認証バインドは無効です", false)
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
		return authn.Session{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork 認証セッションは無効です", false)
	}
	c.mu.Lock()
	c.records[binding] = credentialRecord{credential: credential, session: session.Clone()}
	c.mu.Unlock()
	if requirement.AccountID != "" {
		identity, identityErr := c.Execute(ctx, binding, chatwork.Request{Task: chatwork.TaskAccountShow})
		if identityErr != nil {
			c.removeBinding(binding)
			return authn.Session{}, accountVerificationFailure(identityErr)
		}
		if identity.Account == nil || identity.Account.Ref.Kind != chatwork.ReferenceAccount {
			c.removeBinding(binding)
			return authn.Session{}, fault.New(fault.KindContract, "chatwork_response_invalid", "Chatwork が認証アカウントを確認できる応答を返しませんでした", false)
		}
		if identity.Account.Ref.Value != requirement.AccountID {
			c.removeBinding(binding)
			return authn.Session{}, fault.New(fault.KindAuthentication, "authentication_context_mismatch", "認証が要求された Chatwork アカウントと一致しません", false)
		}
		session.AccountID = identity.Account.Ref.Value
		session.SubjectID = identity.Account.Ref.Value
		if err := session.Validate(); err != nil {
			c.removeBinding(binding)
			return authn.Session{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork 認証セッションは無効です", false)
		}
		c.mu.Lock()
		record, exists := c.records[binding]
		if exists {
			record.session = session.Clone()
			c.records[binding] = record
		}
		c.mu.Unlock()
		if !exists {
			return authn.Session{}, fault.New(fault.KindAuthentication, "invalid_authentication_binding", "Chatwork 認証バインドを利用できません", false)
		}
	}
	return session.Clone(), nil
}

func accountVerificationFailure(err error) error {
	structured, ok := fault.PublicCopy(err)
	if !ok {
		return fault.New(fault.KindAuthentication, "authentication_failed", "Chatwork 認証アカウントを確認できませんでした", false)
	}
	switch structured.Kind {
	case fault.KindRateLimited:
		mapped := fault.New(fault.KindRateLimited, "chatwork_mutation_rate_limited", "Chatwork の認証アカウント確認はレート上限により完了しませんでした。変更は実行していません", false)
		mapped.RetryAfter = structured.RetryAfter
		return mapped
	case fault.KindUnavailable:
		return fault.New(fault.KindUnavailable, "chatwork_account_verification_failed", "Chatwork の認証アカウントを確認できなかったため、変更は実行していません", false)
	default:
		return structured
	}
}

func (c *Client) removeBinding(binding authn.BindingID) {
	if c == nil {
		return
	}
	c.mu.Lock()
	delete(c.records, binding)
	c.mu.Unlock()
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
		now:      time.Now,
		newID:    newID,
		readFile: readFile,
	}
	return client
}

func (c *Client) resolve(binding authn.BindingID) (credentialRecord, error) {
	if err := binding.Validate(); err != nil {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "invalid_authentication_binding", "Chatwork 認証バインドは無効です", false)
	}
	if c == nil {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "invalid_authentication_binding", "Chatwork 認証バインドを利用できません", false)
	}
	c.mu.RLock()
	record, ok := c.records[binding]
	c.mu.RUnlock()
	if !ok || record.session.BindingID != binding || record.credential.secret == "" {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "invalid_authentication_binding", "Chatwork 認証バインドを利用できません", false)
	}
	if err := record.session.Validate(); err != nil {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "invalid_authentication_session", "Chatwork 認証セッションは無効です", false)
	}
	if record.session.Authority != chatwork.AuthenticationAuthority || record.session.Audience != chatwork.AuthenticationAudience {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "authentication_context_mismatch", "認証が Chatwork API コンテキストと一致しません", false)
	}
	if record.credential.method != record.session.Method {
		return credentialRecord{}, fault.New(fault.KindAuthentication, "authentication_context_mismatch", "認証が Chatwork API コンテキストと一致しません", false)
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
		return fault.New(fault.KindAuthentication, "chatwork_token_invalid", "Chatwork API トークンは無効です", false)
	}
	for _, r := range token {
		if r < 0x21 || r > 0x7e {
			return fault.New(fault.KindAuthentication, "chatwork_token_invalid", "Chatwork API トークンは無効です", false)
		}
	}
	return nil
}
