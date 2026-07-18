package chatworkapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

func TestTransportPolicyIsPinnedAndSingleAttempt(t *testing.T) {
	if RequestTimeout != 20*time.Second || UploadTimeout != 60*time.Second || MaxAttempts != 1 ||
		MaxSuccessResponseBytes != 8*1024*1024 || MaxErrorResponseBytes != 64*1024 || MaxUploadBytes != 5*1024*1024 {
		t.Fatal("Chatwork transport limits drifted from the accepted contract")
	}
	client := productionHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok || !transport.DisableKeepAlives || client.CheckRedirect == nil {
		t.Fatal("production transport must disable implicit replay through connection reuse and redirects")
	}
}

func TestCredentialHeaderStrategyKeepsPATAndBearerInsideInfrastructure(t *testing.T) {
	for name, test := range map[string]struct {
		credential requestCredential
		header     string
		want       string
	}{
		"pat":         {requestCredential{method: authn.MethodPAT, header: credentialHeaderChatworkToken, secret: syntheticToken}, "x-chatworktoken", syntheticToken},
		"bearer seam": {requestCredential{method: authn.MethodOAuth2, header: credentialHeaderBearer, secret: "synthetic-access-token"}, "Authorization", "Bearer synthetic-access-token"},
	} {
		t.Run(name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, "https://example.test", nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := test.credential.authorize(request); err != nil {
				t.Fatal(err)
			}
			if got := request.Header.Get(test.header); got != test.want {
				t.Fatalf("header = %q", got)
			}
		})
	}
}

const syntheticToken = "synthetic-chatwork-token"

func testRequirement() authn.Requirement {
	return authn.Requirement{
		Methods: []authn.Method{authn.MethodPAT}, Authority: chatwork.AuthenticationAuthority,
		Audience: chatwork.AuthenticationAudience, RequiredCapabilities: []string{chatwork.AuthenticationCapability},
	}
}

func authenticatedClient(t *testing.T, server *httptest.Server) (*Client, authn.BindingID) {
	t.Helper()
	client := newClient(server.URL, syntheticToken, server.Client(), func() (string, error) { return "test-binding", nil }, func(string) ([]byte, error) { return []byte("file"), nil })
	session, err := client.Authenticate(context.Background(), testRequirement())
	if err != nil {
		t.Fatal(err)
	}
	return client, session.BindingID
}

func TestAuthenticateCreatesSecretFreeBindingAndRejectsCrossClientUse(t *testing.T) {
	serverCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		serverCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	first, binding := authenticatedClient(t, server)
	second := newClient(server.URL, "other-synthetic-token", server.Client(), func() (string, error) { return "other-binding", nil }, func(string) ([]byte, error) { return nil, nil })
	if _, err := second.Execute(context.Background(), binding, chatwork.Request{Task: chatwork.TaskRoomsList}); err == nil {
		t.Fatal("cross-client binding succeeded")
	}
	if serverCalls != 0 {
		t.Fatalf("provider calls = %d", serverCalls)
	}
	result, err := first.Execute(context.Background(), binding, chatwork.Request{Task: chatwork.TaskRoomsList})
	if err != nil || result.Task != chatwork.TaskRoomsList || serverCalls != 1 {
		t.Fatalf("result = %+v, err = %v, calls = %d", result, err, serverCalls)
	}
	if strings.Contains(binding.String(), syntheticToken) {
		t.Fatal("binding rendered token")
	}
}

func TestAuthenticateRejectsMismatchedRequirement(t *testing.T) {
	client := newClient("http://example.test", syntheticToken, http.DefaultClient, func() (string, error) { return "binding", nil }, boundedReadFile)
	requirement := testRequirement()
	requirement.Audience = "other"
	_, err := client.Authenticate(context.Background(), requirement)
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "authentication_context_mismatch" {
		t.Fatalf("error = %#v", err)
	}
}

func TestNewFromEnvironmentFailsClosedWithoutSafeToken(t *testing.T) {
	for name, token := range map[string]string{"missing": "", "control": "synthetic\ntoken"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(TokenEnvironment, token)
			client, err := NewFromEnvironment()
			if err == nil || client != nil {
				t.Fatalf("client = %#v, err = %v", client, err)
			}
			if token != "" && strings.Contains(err.Error(), token) {
				t.Fatal("authentication fault exposed token")
			}
		})
	}
}

func TestExecuteSendsTokenOnlyToSelectedServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-chatworktoken"); got != syntheticToken {
			t.Fatalf("token header = %q", got)
		}
		if r.Method != http.MethodGet || r.URL.Path != "/rooms" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	client, binding := authenticatedClient(t, server)
	if _, err := client.Execute(context.Background(), binding, chatwork.Request{Task: chatwork.TaskRoomsList}); err != nil {
		t.Fatal(err)
	}
}

func TestProductionTransportDoesNotFollowRedirects(t *testing.T) {
	sinkCalls := 0
	sink := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { sinkCalls++ }))
	defer sink.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, sink.URL, http.StatusFound) }))
	defer source.Close()
	client := newClient(source.URL, syntheticToken, productionHTTPClient(), func() (string, error) { return "redirect-binding", nil }, boundedReadFile)
	session, err := client.Authenticate(context.Background(), testRequirement())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Execute(context.Background(), session.BindingID, chatwork.Request{Task: chatwork.TaskRoomsList})
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "chatwork_unexpected_response" {
		t.Fatalf("error = %#v", err)
	}
	if sinkCalls != 0 {
		t.Fatalf("redirect destination calls = %d", sinkCalls)
	}
}

func TestProviderFaultsAreStableAndSecretFree(t *testing.T) {
	for status, want := range map[int]string{
		http.StatusBadRequest: "chatwork_invalid_request", http.StatusUnauthorized: "chatwork_authentication_failed",
		http.StatusForbidden: "chatwork_permission_denied", http.StatusNotFound: "chatwork_not_found",
		http.StatusTooManyRequests: "chatwork_rate_limited", http.StatusServiceUnavailable: "chatwork_unavailable",
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"errors":["` + syntheticToken + `"]}`))
			}))
			defer server.Close()
			client, binding := authenticatedClient(t, server)
			_, err := client.Execute(context.Background(), binding, chatwork.Request{Task: chatwork.TaskRoomsList})
			var structured *fault.Error
			if !errors.As(err, &structured) || structured.Code != want {
				t.Fatalf("error = %#v", err)
			}
			if strings.Contains(err.Error(), syntheticToken) {
				t.Fatal("fault exposed provider body")
			}
		})
	}
}

func TestUndocumentedEmptySuccessIsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer server.Close()
	client, binding := authenticatedClient(t, server)
	_, err := client.Execute(context.Background(), binding, chatwork.Request{Task: chatwork.TaskRoomsList})
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "chatwork_unexpected_response" {
		t.Fatalf("error = %#v", err)
	}
}

func TestBoundedBodiesAndUpload(t *testing.T) {
	if _, err := readBounded(strings.NewReader(strings.Repeat("x", MaxSuccessResponseBytes+1)), MaxSuccessResponseBytes); err == nil {
		t.Fatal("oversized success response accepted")
	}
	client := newClient("http://example.test", syntheticToken, http.DefaultClient, func() (string, error) { return "binding", nil }, func(string) ([]byte, error) {
		return make([]byte, MaxUploadBytes+1), nil
	})
	if _, err := client.multipartRequest("/rooms/2/files", completeRequest(chatwork.TaskFilesUpload)); err == nil {
		t.Fatal("oversized upload accepted")
	}
}

func TestMutationTransportFailureIsNotRetryable(t *testing.T) {
	client := newClient("http://example.test", syntheticToken, roundTripError{errors.New("private transport details")}, func() (string, error) { return "mutation-binding", nil }, boundedReadFile)
	session, err := client.Authenticate(context.Background(), testRequirement())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Execute(context.Background(), session.BindingID, completeRequest(chatwork.TaskMessagesSend))
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "chatwork_mutation_outcome_unknown" || structured.Retryable {
		t.Fatalf("error = %#v", err)
	}
}

type roundTripError struct{ err error }

func (r roundTripError) Do(*http.Request) (*http.Response, error) { return nil, r.err }
