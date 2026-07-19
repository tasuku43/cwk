package chatworkapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestCredentialAuthorizationKeepsPATInsideInfrastructure(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://example.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "unexpected-synthetic-value")
	credential := requestCredential{method: authn.MethodPAT, secret: syntheticToken}
	if err := credential.authorize(request); err != nil {
		t.Fatal(err)
	}
	if got := request.Header.Get("x-chatworktoken"); got != syntheticToken {
		t.Fatalf("x-chatworktoken = %q", got)
	}
	if got := request.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want empty", got)
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

func TestAuthenticateVerifiesRequiredAccountThroughPrivateBinding(t *testing.T) {
	providerCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerCalls++
		if r.Method != http.MethodGet || r.URL.Path != "/me" {
			t.Fatalf("identity request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("x-chatworktoken") != syntheticToken {
			t.Fatal("identity request did not use the private credential binding")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"account_id":7,"room_id":2,"name":"Synthetic"}`))
	}))
	defer server.Close()

	client := newClient(server.URL, syntheticToken, server.Client(), func() (string, error) { return "verified-binding", nil }, boundedReadFile)
	requirement := testRequirement()
	requirement.AccountID = "7"
	session, err := client.Authenticate(context.Background(), requirement)
	if err != nil {
		t.Fatal(err)
	}
	if providerCalls != 1 {
		t.Fatalf("identity calls = %d, want 1", providerCalls)
	}
	if session.AccountID != "7" || session.SubjectID != "7" {
		t.Fatalf("verified session = %+v", session)
	}
	record, err := client.resolve(session.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	if record.session.AccountID != "7" || record.session.SubjectID != "7" {
		t.Fatalf("stored verified session = %+v", record.session)
	}
}

func TestRoomsCreateRequiresTheVerifiedBindingAccountAtTransportBoundary(t *testing.T) {
	providerCalls := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerCalls = append(providerCalls, r.Method+" "+r.URL.Path)
		if r.Header.Get("x-chatworktoken") != syntheticToken {
			t.Fatal("provider request did not use the private credential binding")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/me":
			_, _ = w.Write([]byte(`{"account_id":7,"room_id":2,"name":"Synthetic"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/rooms":
			_, _ = w.Write([]byte(`{"room_id":9}`))
		default:
			t.Fatalf("unexpected request = %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	generic := newClient(server.URL, syntheticToken, server.Client(), func() (string, error) { return "generic-room-binding", nil }, boundedReadFile)
	genericSession, err := generic.Authenticate(context.Background(), testRequirement())
	if err != nil {
		t.Fatal(err)
	}
	genericRequest := completeRequest(chatwork.TaskRoomsCreate)
	_, err = generic.Execute(context.Background(), genericSession.BindingID, genericRequest)
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "authentication_context_mismatch" {
		t.Fatalf("generic binding error = %#v", err)
	}
	if len(providerCalls) != 0 {
		t.Fatalf("generic binding reached provider: %v", providerCalls)
	}

	verified := newClient(server.URL, syntheticToken, server.Client(), func() (string, error) { return "verified-room-binding", nil }, boundedReadFile)
	requirement := testRequirement()
	requirement.AccountID = "7"
	verifiedSession, err := verified.Authenticate(context.Background(), requirement)
	if err != nil {
		t.Fatal(err)
	}
	mismatch := completeRequest(chatwork.TaskRoomsCreate)
	mismatch.Account = testRef(chatwork.ReferenceAccount, "8")
	_, err = verified.Execute(context.Background(), verifiedSession.BindingID, mismatch)
	if !errors.As(err, &structured) || structured.Code != "authentication_context_mismatch" {
		t.Fatalf("mismatched verified binding error = %#v", err)
	}
	if len(providerCalls) != 1 || providerCalls[0] != "GET /me" {
		t.Fatalf("mismatched binding calls = %v", providerCalls)
	}

	exact := completeRequest(chatwork.TaskRoomsCreate)
	exact.Account = testRef(chatwork.ReferenceAccount, "7")
	result, err := verified.Execute(context.Background(), verifiedSession.BindingID, exact)
	if err != nil {
		t.Fatal(err)
	}
	if len(providerCalls) != 2 || providerCalls[1] != "POST /rooms" ||
		len(result.Created) != 1 || result.Created[0].Value != "9" {
		t.Fatalf("verified room creation calls/result = %v / %+v", providerCalls, result)
	}
}

func TestAuthenticateRejectsAccountMismatchAndRemovesProvisionalBinding(t *testing.T) {
	identityCalls, roomCreates := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/me":
			identityCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"account_id":8,"room_id":2,"name":"Different"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/rooms":
			roomCreates++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"room_id":9}`))
		default:
			t.Fatalf("unexpected request = %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := newClient(server.URL, syntheticToken, server.Client(), func() (string, error) { return "mismatch-binding", nil }, boundedReadFile)
	requirement := testRequirement()
	requirement.AccountID = "7"
	_, err := client.Authenticate(context.Background(), requirement)
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "authentication_context_mismatch" {
		t.Fatalf("error = %#v", err)
	}
	if identityCalls != 1 || roomCreates != 0 {
		t.Fatalf("provider calls = identity %d, room create %d; want 1, 0", identityCalls, roomCreates)
	}
	if len(client.records) != 0 {
		t.Fatalf("provisional bindings retained after mismatch: %d", len(client.records))
	}
}

func TestAuthenticateMapsAccountVerificationFailuresToMutationSafeFaults(t *testing.T) {
	fixedNow := time.Unix(1_700_000_000, 0).UTC()
	tests := []struct {
		name      string
		status    int
		wantKind  fault.Kind
		wantCode  string
		wantRetry time.Duration
	}{
		{name: "rate limited", status: http.StatusTooManyRequests, wantKind: fault.KindRateLimited, wantCode: "chatwork_mutation_rate_limited", wantRetry: 30 * time.Second},
		{name: "provider unavailable", status: http.StatusServiceUnavailable, wantKind: fault.KindUnavailable, wantCode: "chatwork_account_verification_failed"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			identityCalls, roomCreates := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/me":
					identityCalls++
					if test.status == http.StatusTooManyRequests {
						w.Header().Set("x-ratelimit-reset", strconv.FormatInt(fixedNow.Add(test.wantRetry).Unix(), 10))
						w.Header().Set("Date", fixedNow.Format(http.TimeFormat))
					}
					w.WriteHeader(test.status)
					_, _ = w.Write([]byte(`{"errors":["private provider detail"]}`))
				case r.Method == http.MethodPost && r.URL.Path == "/rooms":
					roomCreates++
				default:
					t.Fatalf("unexpected request = %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			client := newClient(server.URL, syntheticToken, server.Client(), func() (string, error) { return "preflight-binding", nil }, boundedReadFile)
			client.now = func() time.Time { return fixedNow }
			requirement := testRequirement()
			requirement.AccountID = "7"
			_, err := client.Authenticate(context.Background(), requirement)
			var structured *fault.Error
			if !errors.As(err, &structured) || structured.Kind != test.wantKind || structured.Code != test.wantCode || structured.Retryable {
				t.Fatalf("error = %#v", err)
			}
			if structured.RetryAfter != test.wantRetry {
				t.Fatalf("retry after = %s, want %s", structured.RetryAfter, test.wantRetry)
			}
			if identityCalls != 1 || roomCreates != 0 || len(client.records) != 0 {
				t.Fatalf("calls/bindings = identity %d, room create %d, bindings %d", identityCalls, roomCreates, len(client.records))
			}
		})
	}
}

func TestAuthenticateMapsAccountVerificationTransportFailureWithoutMutationReplay(t *testing.T) {
	const privateCanary = "private account verification transport detail"
	doer := &recordingErrorDoer{err: errors.New(privateCanary)}
	client := newClient("http://example.test", syntheticToken, doer, func() (string, error) { return "transport-binding", nil }, boundedReadFile)
	requirement := testRequirement()
	requirement.AccountID = "7"

	_, err := client.Authenticate(context.Background(), requirement)
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Kind != fault.KindUnavailable ||
		structured.Code != "chatwork_account_verification_failed" || structured.Retryable {
		t.Fatalf("error = %#v", err)
	}
	if len(doer.requests) != 1 || doer.requests[0] != "GET /me" {
		t.Fatalf("requests = %v", doer.requests)
	}
	if len(client.records) != 0 {
		t.Fatalf("provisional bindings retained after transport failure: %d", len(client.records))
	}
	if strings.Contains(err.Error(), privateCanary) {
		t.Fatal("private transport detail leaked through account verification fault")
	}
}

func TestAuthenticatePreservesAccountVerificationCancellationBeforeMutation(t *testing.T) {
	doer := &recordingErrorDoer{err: context.DeadlineExceeded}
	client := newClient("http://example.test", syntheticToken, doer, func() (string, error) { return "canceled-binding", nil }, boundedReadFile)
	requirement := testRequirement()
	requirement.AccountID = "7"

	_, err := client.Authenticate(context.Background(), requirement)
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Kind != fault.KindCanceled ||
		structured.Code != "operation_canceled" || !structured.Retryable {
		t.Fatalf("error = %#v", err)
	}
	if len(doer.requests) != 1 || doer.requests[0] != "GET /me" {
		t.Fatalf("requests = %v", doer.requests)
	}
	if len(client.records) != 0 {
		t.Fatalf("provisional bindings retained after cancellation: %d", len(client.records))
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

func TestNewFromEnvironmentPinsPATAndProductionDestination(t *testing.T) {
	t.Setenv(TokenEnvironment, syntheticToken)
	client, err := NewFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if client.baseURL != ProductionBaseURL {
		t.Fatalf("base URL = %q, want %q", client.baseURL, ProductionBaseURL)
	}
	if client.source.method != authn.MethodPAT || client.source.secret != syntheticToken {
		t.Fatal("production client did not retain the PAT inside its private credential record")
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

func TestMessageRetrievalDistinguishesAccessLimitationFromEmptyAndNotFound(t *testing.T) {
	messageBody := `[{"message_id":"3","account":{"account_id":1,"name":"A"},"body":"visible","send_time":1,"update_time":0}]`
	tests := []struct {
		name       string
		task       chatwork.Task
		status     int
		headers    http.Header
		body       string
		wantAccess chatwork.MessageAccessLimitation
		wantCode   string
	}{
		{name: "visible list", task: chatwork.TaskMessagesList, status: http.StatusOK, body: messageBody, wantAccess: chatwork.MessageAccessNone},
		{name: "partially restricted list", task: chatwork.TaskMessagesList, status: http.StatusOK, headers: http.Header{messageLimitationHeader: {"true"}, messageLimitationSummaryHeader: {"private provider reason"}}, body: messageBody, wantAccess: chatwork.MessageAccessPartial},
		{name: "partial with missing summary remains restricted", task: chatwork.TaskMessagesList, status: http.StatusOK, headers: http.Header{messageLimitationHeader: {"true"}}, body: messageBody, wantAccess: chatwork.MessageAccessPartial},
		{name: "true zero list", task: chatwork.TaskMessagesList, status: http.StatusNoContent, wantAccess: chatwork.MessageAccessNone},
		{name: "fully restricted list", task: chatwork.TaskMessagesList, status: http.StatusNoContent, headers: http.Header{messageLimitationHeader: {"true"}, messageLimitationSummaryHeader: {"private provider reason"}}, wantAccess: chatwork.MessageAccessAll},
		{name: "restricted single message", task: chatwork.TaskMessagesShow, status: http.StatusNotFound, headers: http.Header{messageLimitationHeader: {"true"}, messageLimitationSummaryHeader: {"private provider reason"}}, body: `{"errors":["private provider reason"]}`, wantCode: "chatwork_message_restricted"},
		{name: "nonexistent single message", task: chatwork.TaskMessagesShow, status: http.StatusNotFound, body: `{"errors":["private provider reason"]}`, wantCode: "chatwork_not_found"},
		{name: "false is not an official value", task: chatwork.TaskMessagesList, status: http.StatusNoContent, headers: http.Header{messageLimitationHeader: {"false"}}, wantCode: "chatwork_message_limitation_invalid"},
		{name: "duplicate true values are ambiguous", task: chatwork.TaskMessagesList, status: http.StatusNoContent, headers: http.Header{messageLimitationHeader: {"true", "true"}}, wantCode: "chatwork_message_limitation_invalid"},
		{name: "summary alone is invalid", task: chatwork.TaskMessagesList, status: http.StatusNoContent, headers: http.Header{messageLimitationSummaryHeader: {"private provider reason"}}, wantCode: "chatwork_message_limitation_invalid"},
		{name: "limitation on undocumented status is invalid", task: chatwork.TaskMessagesShow, status: http.StatusOK, headers: http.Header{messageLimitationHeader: {"true"}}, body: strings.Trim(messageBody, "[]"), wantCode: "chatwork_message_limitation_invalid"},
		{name: "200 empty array is not normal empty", task: chatwork.TaskMessagesList, status: http.StatusOK, body: `[]`, wantCode: "chatwork_response_malformed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				for name, values := range test.headers {
					for _, value := range values {
						w.Header().Add(name, value)
					}
				}
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			client, binding := authenticatedClient(t, server)
			result, err := client.Execute(context.Background(), binding, completeRequest(test.task))
			if test.wantCode != "" {
				var structured *fault.Error
				if !errors.As(err, &structured) || structured.Code != test.wantCode {
					t.Fatalf("error = %#v, want code %q", err, test.wantCode)
				}
				if strings.Contains(err.Error(), "private provider reason") {
					t.Fatal("fault exposed provider limitation summary or body")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.MessageAccess != test.wantAccess {
				t.Fatalf("access = %v, want %v", result.MessageAccess, test.wantAccess)
			}
		})
	}
}

func TestRateLimitRetryAfterUsesOnlyStrictOfficialReset(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	validReset := strconv.FormatInt(now.Add(2*time.Minute).Unix(), 10)
	tests := []struct {
		name   string
		header http.Header
		want   time.Duration
	}{
		{name: "valid", header: http.Header{"X-Ratelimit-Reset": {validReset}}, want: 2 * time.Minute},
		{name: "missing", header: http.Header{}},
		{name: "retry-after is not Chatwork evidence", header: http.Header{"Retry-After": {"10"}}},
		{name: "empty", header: http.Header{"X-Ratelimit-Reset": {""}}},
		{name: "surrounding whitespace", header: http.Header{"X-Ratelimit-Reset": {" " + validReset}}},
		{name: "plus sign", header: http.Header{"X-Ratelimit-Reset": {"+" + validReset}}},
		{name: "negative", header: http.Header{"X-Ratelimit-Reset": {"-1"}}},
		{name: "non decimal", header: http.Header{"X-Ratelimit-Reset": {"1.5"}}},
		{name: "overflow", header: http.Header{"X-Ratelimit-Reset": {"999999999999999999999999"}}},
		{name: "not future", header: http.Header{"X-Ratelimit-Reset": {strconv.FormatInt(now.Unix(), 10)}}},
		{name: "past", header: http.Header{"X-Ratelimit-Reset": {strconv.FormatInt(now.Add(-time.Second).Unix(), 10)}}},
		{name: "beyond documented window", header: http.Header{"X-Ratelimit-Reset": {strconv.FormatInt(now.Add(5*time.Minute+time.Second).Unix(), 10)}}},
		{name: "combined value", header: http.Header{"X-Ratelimit-Reset": {validReset + ", " + validReset}}},
		{name: "duplicate", header: http.Header{"X-Ratelimit-Reset": {validReset, validReset}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := rateLimitRetryAfter(test.header, now); got != test.want {
				t.Fatalf("retry after = %s, want %s", got, test.want)
			}
		})
	}
}

func TestRateLimitResetMustAlsoMatchValidResponseDate(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	header := http.Header{
		"Date":              {now.Add(-10 * time.Minute).Format(http.TimeFormat)},
		"X-Ratelimit-Reset": {strconv.FormatInt(now.Add(time.Minute).Unix(), 10)},
	}
	if got := rateLimitRetryAfter(header, now); got != 0 {
		t.Fatalf("retry after = %s, want unknown for a reset outside the response window", got)
	}

	header.Set("Date", "not-an-http-date")
	if got := rateLimitRetryAfter(header, now); got != time.Minute {
		t.Fatalf("retry after with unusable response Date = %s, want current-clock fallback", got)
	}
}

func TestRateLimitResetUsesValidProviderDateAsTheDurationBaseline(t *testing.T) {
	reset := time.Date(2026, time.July, 19, 12, 2, 0, 0, time.UTC)
	header := http.Header{
		"Date":              {time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC).Format(http.TimeFormat)},
		"X-Ratelimit-Reset": {strconv.FormatInt(reset.Unix(), 10)},
	}
	for name, localNow := range map[string]time.Time{
		"local clock ahead":  time.Date(2026, time.July, 19, 12, 1, 0, 0, time.UTC),
		"local clock behind": time.Date(2026, time.July, 19, 11, 59, 0, 0, time.UTC),
	} {
		t.Run(name, func(t *testing.T) {
			if got := rateLimitRetryAfter(header, localNow); got != 2*time.Minute {
				t.Fatalf("retry after = %s, want provider-clock 2m", got)
			}
		})
	}
}

func TestProviderRateLimitSeparatesReadsMutationsAndRoomPostLimit(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	generalHeader := http.Header{"X-Ratelimit-Reset": {strconv.FormatInt(now.Add(2*time.Minute).Unix(), 10)}}
	tests := []struct {
		name          string
		task          chatwork.Task
		header        http.Header
		body          string
		wantCode      string
		wantRetryable bool
		wantAfter     time.Duration
	}{
		{name: "read general", task: chatwork.TaskRoomsList, header: generalHeader.Clone(), body: `{"errors":["rate limited"]}`, wantCode: "chatwork_rate_limited", wantRetryable: true, wantAfter: 2 * time.Minute},
		{name: "mutation general", task: chatwork.TaskRoomsUpdate, header: generalHeader.Clone(), body: `{"errors":["rate limited"]}`, wantCode: "chatwork_mutation_rate_limited", wantAfter: 2 * time.Minute},
		{name: "message room limit", task: chatwork.TaskMessagesSend, body: `{"errors":["Rate limit for message posting per room exceeded."]}`, wantCode: "chatwork_mutation_rate_limited", wantAfter: 10 * time.Second},
		{name: "task room limit", task: chatwork.TaskRoomTasksCreate, body: ` { "errors" : [ "Rate limit for message posting per room exceeded." ] } `, wantCode: "chatwork_mutation_rate_limited", wantAfter: 10 * time.Second},
		{name: "special endpoint unknown body falls back to general", task: chatwork.TaskMessagesSend, header: generalHeader.Clone(), body: `{"errors":["unknown"]}`, wantCode: "chatwork_mutation_rate_limited", wantAfter: 2 * time.Minute},
		{name: "documented body does not apply to another mutation", task: chatwork.TaskRoomsUpdate, body: `{"errors":["Rate limit for message posting per room exceeded."]}`, wantCode: "chatwork_mutation_rate_limited"},
		{name: "missing timing is unknown", task: chatwork.TaskRoomsList, body: `{"errors":["rate limited"]}`, wantCode: "chatwork_rate_limited", wantRetryable: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     test.header,
				Body:       io.NopCloser(strings.NewReader(test.body)),
			}
			if response.Header == nil {
				response.Header = make(http.Header)
			}
			err := providerFault(test.task, response, now)
			var structured *fault.Error
			if !errors.As(err, &structured) || structured.Code != test.wantCode ||
				structured.Retryable != test.wantRetryable || structured.RetryAfter != test.wantAfter {
				t.Fatalf("error = %+v, want code=%s retryable=%t retry_after=%s", structured, test.wantCode, test.wantRetryable, test.wantAfter)
			}
		})
	}
}

func TestRoomPostRateLimitRequiresExactBoundedJSON(t *testing.T) {
	valid := `{"errors":["Rate limit for message posting per room exceeded."]}`
	for name, body := range map[string]string{
		"extra error":       `{"errors":["Rate limit for message posting per room exceeded.","other"]}`,
		"extra field":       `{"errors":["Rate limit for message posting per room exceeded."],"other":true}`,
		"duplicate key":     `{"errors":["Rate limit for message posting per room exceeded."],"errors":["Rate limit for message posting per room exceeded."]}`,
		"trailing value":    valid + `{}`,
		"malformed":         `{"errors":`,
		"different message": `{"errors":["rate limited"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if documentedRoomPostRateLimit([]byte(body)) {
				t.Fatal("non-exact provider body selected the room-post retry timing")
			}
		})
	}
	if !documentedRoomPostRateLimit([]byte(valid)) {
		t.Fatal("documented provider body was not recognized")
	}
	if documentedRoomPostRateLimit([]byte(strings.Repeat("x", MaxErrorResponseBytes+1))) {
		t.Fatal("oversized provider body selected the room-post retry timing")
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

type recordingErrorDoer struct {
	requests []string
	err      error
}

func (d *recordingErrorDoer) Do(request *http.Request) (*http.Response, error) {
	d.requests = append(d.requests, request.Method+" "+request.URL.Path)
	return nil, d.err
}
