package chatworkapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type oauthSourceStub struct {
	identity           OAuthIdentity
	authorizedIdentity OAuthIdentity
	authorizeErr       error
	authorizeCalls     int
}

func (source *oauthSourceStub) Identity(context.Context) (OAuthIdentity, error) {
	return source.identity, nil
}

func (source *oauthSourceStub) Authorize(request *http.Request) (OAuthIdentity, error) {
	source.authorizeCalls++
	request.Header.Set("Authorization", "Bearer synthetic-access")
	if source.authorizeErr != nil {
		return OAuthIdentity{}, source.authorizeErr
	}
	return source.authorizedIdentity, nil
}

func oauthIdentity(subject string) OAuthIdentity {
	return OAuthIdentity{
		Method: authn.MethodOAuth2, SubjectID: subject, AccountID: subject,
		GrantedCapabilities: []string{chatwork.AuthenticationCapability}, ExpiresAt: time.Now().Add(time.Hour),
	}
}

func TestOAuthCredentialRevalidationRejectsIdentityChangeBeforeProviderCall(t *testing.T) {
	source := &oauthSourceStub{identity: oauthIdentity("account-a"), authorizedIdentity: oauthIdentity("account-b")}
	client := newClientWithCredential("https://api.chatwork.com/v2", requestCredential{method: authn.MethodOAuth2, header: credentialHeaderBearer, oauth: source}, nil, func() (string, error) { return "oauth-test-binding", nil }, boundedReadFile)
	requirement := authn.Requirement{
		Methods: []authn.Method{authn.MethodOAuth2}, Authority: chatwork.AuthenticationAuthority,
		Audience: chatwork.AuthenticationAudience, RequiredCapabilities: []string{chatwork.AuthenticationCapability},
	}
	session, err := client.Authenticate(context.Background(), requirement)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Execute(context.Background(), session.BindingID, chatwork.Request{Task: chatwork.TaskRoomsList})
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "authentication_context_mismatch" {
		t.Fatalf("error = %#v", err)
	}
	if source.authorizeCalls != 1 {
		t.Fatalf("authorize calls = %d", source.authorizeCalls)
	}
}

func TestOAuthCredentialRecordContainsNoStaticTokenCopy(t *testing.T) {
	identity := oauthIdentity("account-a")
	source := &oauthSourceStub{identity: identity, authorizedIdentity: identity}
	client := newClientWithCredential("https://api.chatwork.com/v2", requestCredential{method: authn.MethodOAuth2, header: credentialHeaderBearer, oauth: source}, nil, func() (string, error) { return "oauth-record-binding", nil }, boundedReadFile)
	requirement := authn.Requirement{Methods: []authn.Method{authn.MethodOAuth2}, Authority: chatwork.AuthenticationAuthority, Audience: chatwork.AuthenticationAudience, RequiredCapabilities: []string{chatwork.AuthenticationCapability}}
	session, err := client.Authenticate(context.Background(), requirement)
	if err != nil {
		t.Fatal(err)
	}
	record := client.records[session.BindingID]
	if record.credential.secret != "" || record.credential.oauth == nil || record.credential.bound.SubjectID != identity.SubjectID {
		t.Fatal("OAuth binding retained a static token or lost secret-free identity")
	}
}

func TestOAuthSourceFaultMessageAndHeaderAreRedacted(t *testing.T) {
	identity := oauthIdentity("42")
	source := &oauthSourceStub{
		identity: identity, authorizedIdentity: identity,
		authorizeErr: fault.New(fault.KindAuthentication, "oauth_refresh_failed", "Bearer secret-canary", false),
	}
	credential := requestCredential{method: authn.MethodOAuth2, oauth: source, bound: identity}
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.chatwork.com/v2/me", nil)
	err := credential.authorize(request)
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "oauth_refresh_failed" {
		t.Fatalf("error = %#v", err)
	}
	if request.Header.Get("Authorization") != "" || structured.Message == "Bearer secret-canary" {
		t.Fatal("OAuth source fault leaked its header or private message")
	}
}
