package chatworkapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type clientTaskPort struct {
	client       *Client
	requests     []chatwork.Request
	bindings     []authn.BindingID
	contexts     []context.Context
	prepareCalls int
	prepared     []chatwork.PreparedMessageBody
}

func (p *clientTaskPort) record(ctx context.Context, binding authn.BindingID, request chatwork.Request) {
	p.requests = append(p.requests, request)
	p.bindings = append(p.bindings, binding)
	p.contexts = append(p.contexts, ctx)
}

func (p *clientTaskPort) Execute(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.record(ctx, binding, request)
	return p.client.Execute(ctx, binding, request)
}

func (p *clientTaskPort) PrepareMessageBody(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.PreparedMessageBody, error) {
	p.prepareCalls++
	p.record(ctx, binding, request)
	prepared, err := p.client.PrepareMessageBody(ctx, binding, request)
	if err == nil {
		p.prepared = append(p.prepared, prepared)
	}
	return prepared, err
}

func (p *clientTaskPort) ExecutePreparedMessageSend(ctx context.Context, binding authn.BindingID, request chatwork.Request, prepared chatwork.PreparedMessageBody) (chatwork.Result, error) {
	p.record(ctx, binding, request)
	return p.client.ExecutePreparedMessageSend(ctx, binding, request, prepared)
}

func (p *clientTaskPort) DiscardPreparedMessageBody(prepared chatwork.PreparedMessageBody) {
	p.client.DiscardPreparedMessageBody(prepared)
}

func TestReplyReadSuccessThenExpandedBodyFailureMakesNoPost(t *testing.T) {
	getCalls, postCalls := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/rooms/42/messages/101":
			getCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message_id":"101","account":{"account_id":7,"name":"A"},"body":"target","send_time":1,"update_time":0}`))
		case request.Method == http.MethodPost && request.URL.Path == "/rooms/42/messages":
			postCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message_id":"102"}`))
		default:
			t.Errorf("unexpected provider request: %s %s", request.Method, request.URL.Path)
			http.NotFound(w, request)
		}
	}))
	defer server.Close()
	client, binding := authenticatedClient(t, server)
	request := chatwork.Request{
		Task:               chatwork.TaskMessagesSend,
		Room:               testRef(chatwork.ReferenceRoom, "42"),
		Body:               strings.Repeat("x", chatwork.MaxMessageBodyBytes),
		MessageRecipients:  []chatwork.Reference{testRef(chatwork.ReferenceAccount, "7")},
		MessageReplyTarget: &chatwork.MessageReplyTarget{Message: testRef(chatwork.ReferenceMessage, "101")},
	}
	_, err := chatworkcmd.New(&clientTaskPort{client: client}).Execute(context.Background(), binding, request)
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "chatwork_message_body_too_large" || structured.Retryable {
		t.Fatalf("send error after reply read = %#v", err)
	}
	if getCalls != 1 || postCalls != 0 {
		t.Fatalf("provider calls = GET %d POST %d; want one successful preflight GET and zero POST", getCalls, postCalls)
	}
}

func TestReplyBodyFileIsPreparedOnceBeforeGETThenSentWithExactSuffix(t *testing.T) {
	manualBody := "手書き [To:77]\n[rp aid=8 to=42-88] のまま"
	var operations []string
	var authTokens []string
	var sentBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		operations = append(operations, request.Method+" "+request.URL.Path)
		authTokens = append(authTokens, request.Header.Get("x-chatworktoken"))
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/rooms/42/messages/101":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message_id":"101","account":{"account_id":7,"name":"A"},"body":"target","send_time":1,"update_time":0}`))
		case request.Method == http.MethodPost && request.URL.Path == "/rooms/42/messages":
			if err := request.ParseForm(); err != nil {
				t.Errorf("ParseForm(): %v", err)
			}
			sentBody = request.FormValue("body")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message_id":"102"}`))
		default:
			t.Errorf("unexpected provider request: %s %s", request.Method, request.URL.Path)
			http.NotFound(w, request)
		}
	}))
	defer server.Close()
	client, binding := authenticatedClient(t, server)
	fileReads := 0
	client.readBodyFile = func(ctx context.Context, path string) ([]byte, error) {
		fileReads++
		if path != "message.txt" {
			t.Fatalf("body-file path = %q", path)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return []byte(manualBody), nil
	}
	port := &clientTaskPort{client: client}
	ctx := context.WithValue(context.Background(), struct{}{}, "same-context")
	result, err := chatworkcmd.New(port).Execute(ctx, binding, chatwork.Request{
		Task: chatwork.TaskMessagesSend, Room: testRef(chatwork.ReferenceRoom, "42"), BodyFile: "message.txt",
		MessageRecipients:  []chatwork.Reference{testRef(chatwork.ReferenceAccount, "9")},
		MessageReplyTarget: &chatwork.MessageReplyTarget{Message: testRef(chatwork.ReferenceMessage, "101")},
	})
	if err != nil || result.Task != chatwork.TaskMessagesSend {
		t.Fatalf("result = %+v, error = %#v", result, err)
	}
	wantBody := "[To:9] [rp aid=7 to=42-101] " + manualBody
	if fileReads != 1 || port.prepareCalls != 1 || sentBody != wantBody {
		t.Fatalf("file reads=%d prep calls=%d sent body=%q, want once and exact %q", fileReads, port.prepareCalls, sentBody, wantBody)
	}
	assertPreparedMessageBodyCleared(t, port.prepared, "message.txt", manualBody)
	if !reflect.DeepEqual(operations, []string{"GET /rooms/42/messages/101", "POST /rooms/42/messages"}) {
		t.Fatalf("provider order = %v", operations)
	}
	if len(port.requests) != 3 || port.requests[0].Task != chatwork.TaskMessagesSend ||
		port.requests[1].Task != chatwork.TaskMessagesShow || port.requests[2].Task != chatwork.TaskMessagesSend {
		t.Fatalf("port operation order = %+v", port.requests)
	}
	for index := range port.bindings {
		if port.bindings[index] != binding || port.contexts[index] != ctx {
			t.Fatalf("port operation %d changed binding or context", index)
		}
	}
	for index, token := range authTokens {
		if token != syntheticToken {
			t.Fatalf("provider call %d used token %q", index, token)
		}
	}
}

func TestReplyBodyFileValidationFailuresMakeZeroProviderCalls(t *testing.T) {
	privatePath := "private-message-path"
	privateFailure := "private-reader-failure"
	for _, test := range []struct {
		name string
		data []byte
		err  error
		code string
	}{
		{name: "missing", err: errors.New(privateFailure), code: "chatwork_message_body_file_unreadable"},
		{name: "empty", data: []byte{}, code: "chatwork_message_body_invalid"},
		{name: "invalid UTF-8", data: []byte{0xff}, code: "chatwork_message_body_invalid"},
		{name: "oversized", data: []byte(strings.Repeat("x", chatwork.MaxMessageBodyBytes+1)), code: "chatwork_message_body_too_large"},
	} {
		t.Run(test.name, func(t *testing.T) {
			providerCalls, fileReads := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				providerCalls++
				http.NotFound(w, request)
			}))
			defer server.Close()
			client := newClient(server.URL, syntheticToken, server.Client(), func() (string, error) { return "prepared-file-binding", nil }, func(string) ([]byte, error) { return nil, nil })
			client.readBodyFile = func(context.Context, string) ([]byte, error) {
				fileReads++
				return test.data, test.err
			}
			session, err := client.Authenticate(context.Background(), testRequirement())
			if err != nil {
				t.Fatal(err)
			}
			port := &clientTaskPort{client: client}
			_, err = chatworkcmd.New(port).Execute(context.Background(), session.BindingID, chatwork.Request{
				Task: chatwork.TaskMessagesSend, Room: testRef(chatwork.ReferenceRoom, "42"), BodyFile: privatePath,
				MessageReplyTarget: &chatwork.MessageReplyTarget{Message: testRef(chatwork.ReferenceMessage, "101")},
			})
			var structured *fault.Error
			if !errors.As(err, &structured) || structured.Code != test.code {
				t.Fatalf("preflight error = %#v, want %s", err, test.code)
			}
			if providerCalls != 0 || fileReads != 1 {
				t.Fatalf("provider calls=%d file reads=%d, want zero calls and one read", providerCalls, fileReads)
			}
			if len(port.prepared) != 0 {
				t.Fatalf("invalid body source created %d prepared handles", len(port.prepared))
			}
			if strings.Contains(err.Error(), privatePath) || strings.Contains(err.Error(), privateFailure) {
				t.Fatalf("preflight error leaked private file details: %v", err)
			}
		})
	}
}

func TestReplyReadFailureDiscardsPreparedBodyWithoutLeakingIt(t *testing.T) {
	getCalls, postCalls := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.Method + " " + request.URL.Path {
		case "GET /rooms/42/messages/101":
			getCalls++
			http.NotFound(w, request)
		case "POST /rooms/42/messages":
			postCalls++
			http.Error(w, "unexpected post", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected provider request: %s %s", request.Method, request.URL.Path)
			http.NotFound(w, request)
		}
	}))
	defer server.Close()
	client, binding := authenticatedClient(t, server)
	const privateBody = "private-body-canary"
	client.readBodyFile = func(context.Context, string) ([]byte, error) { return []byte(privateBody), nil }
	port := &clientTaskPort{client: client}
	_, err := chatworkcmd.New(port).Execute(context.Background(), binding, chatwork.Request{
		Task: chatwork.TaskMessagesSend, Room: testRef(chatwork.ReferenceRoom, "42"), BodyFile: "private-file-path",
		MessageReplyTarget: &chatwork.MessageReplyTarget{Message: testRef(chatwork.ReferenceMessage, "101")},
	})
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "chatwork_not_found" {
		t.Fatalf("reply read error = %#v", err)
	}
	if getCalls != 1 || postCalls != 0 || len(port.prepared) != 1 {
		t.Fatalf("calls GET=%d POST=%d prepared=%d", getCalls, postCalls, len(port.prepared))
	}
	assertPreparedMessageBodyCleared(t, port.prepared, "private-file-path", privateBody)
}

func TestPreparedMessageBodyRejectsForeignBindingAndChangedSourceBeforeDispatch(t *testing.T) {
	doer := &recordingErrorDoer{err: errors.New("downstream must not be called")}
	client, bindings := preparedBodyTestClient(t, doer, "binding-canary", "client-token-canary")
	otherDoer := &recordingErrorDoer{err: errors.New("foreign client must not be called")}
	otherClient, _ := preparedBodyTestClient(t, otherDoer, "other-binding", "other-client-token")
	request := preparedBodyTestRequest()
	prepared, err := client.PrepareMessageBody(context.Background(), bindings[0], request)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		client  *Client
		binding authn.BindingID
		mutate  func(chatwork.Request) chatwork.Request
	}{
		{name: "different client", client: otherClient, binding: bindings[0]},
		{name: "different binding", client: client, binding: bindings[1]},
		{name: "different room", client: client, binding: bindings[0], mutate: func(input chatwork.Request) chatwork.Request {
			input.Room = testRef(chatwork.ReferenceRoom, "43")
			return input
		}},
		{name: "different ordered recipients", client: client, binding: bindings[0], mutate: func(input chatwork.Request) chatwork.Request {
			input.MessageRecipients = []chatwork.Reference{testRef(chatwork.ReferenceAccount, "8"), testRef(chatwork.ReferenceAccount, "7")}
			return input
		}},
		{name: "different reply target", client: client, binding: bindings[0], mutate: func(input chatwork.Request) chatwork.Request {
			input.MessageReplyTarget.Message = testRef(chatwork.ReferenceMessage, "102")
			return input
		}},
		{name: "different body-file path", client: client, binding: bindings[0], mutate: func(input chatwork.Request) chatwork.Request {
			input.BodyFile = "other-private-path"
			return input
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attempt := clonePreparedMessageBodySource(request)
			if test.mutate != nil {
				attempt = test.mutate(attempt)
			}
			_, err := test.client.ExecutePreparedMessageSend(context.Background(), test.binding, attempt, prepared)
			var structured *fault.Error
			if !errors.As(err, &structured) || structured.Code != "invalid_chatwork_request" {
				t.Fatalf("mismatched prepared handle error = %#v", err)
			}
			if len(doer.requests) != 0 || len(otherDoer.requests) != 0 {
				t.Fatalf("mismatch dispatched provider calls: local=%v foreign=%v", doer.requests, otherDoer.requests)
			}
			handle := prepared.(*preparedMessageBody)
			handle.mu.Lock()
			used, remainingBytes, remainingPath := handle.used, len(handle.bytes), handle.source.BodyFile
			handle.mu.Unlock()
			if used || remainingBytes == 0 || remainingPath != request.BodyFile {
				t.Fatalf("mismatch consumed or changed the live handle: used=%t bytes=%d path=%q", used, remainingBytes, remainingPath)
			}
			if test.name == "different client" {
				otherClient.DiscardPreparedMessageBody(prepared)
				handle.mu.Lock()
				stillOwned := !handle.used && len(handle.bytes) > 0 && handle.source.BodyFile == request.BodyFile
				handle.mu.Unlock()
				if !stillOwned {
					t.Fatal("foreign client discarded another client's prepared handle")
				}
			}
		})
	}
	client.DiscardPreparedMessageBody(prepared)
	assertPreparedMessageBodyCleared(t, []chatwork.PreparedMessageBody{prepared}, request.BodyFile, "private-body-canary")
}

func TestPreparedMessageBodyCannotBeReusedAfterDispatch(t *testing.T) {
	doer := &recordingErrorDoer{err: errors.New("synthetic lost response")}
	client, bindings := preparedBodyTestClient(t, doer, "reuse-binding", "reuse-token")
	request := preparedBodyTestRequest()
	prepared, err := client.PrepareMessageBody(context.Background(), bindings[0], request)
	if err != nil {
		t.Fatal(err)
	}
	resolved := clonePreparedMessageBodySource(request)
	resolved.MessageReplyTarget.Sender = testRef(chatwork.ReferenceAccount, "9")
	_, firstErr := client.ExecutePreparedMessageSend(context.Background(), bindings[0], resolved, prepared)
	var firstFault *fault.Error
	if !errors.As(firstErr, &firstFault) || firstFault.Code != "chatwork_mutation_outcome_unknown" || firstFault.Retryable {
		t.Fatalf("first dispatched send error = %#v", firstErr)
	}
	_, secondErr := client.ExecutePreparedMessageSend(context.Background(), bindings[0], resolved, prepared)
	var secondFault *fault.Error
	if !errors.As(secondErr, &secondFault) || secondFault.Code != "invalid_chatwork_request" {
		t.Fatalf("consumed handle reuse error = %#v", secondErr)
	}
	if !reflect.DeepEqual(doer.requests, []string{"POST /rooms/42/messages"}) {
		t.Fatalf("provider calls after first use and rejected reuse = %v", doer.requests)
	}
	assertPreparedMessageBodyCleared(t, []chatwork.PreparedMessageBody{prepared}, request.BodyFile, "private-body-canary")
}

func TestCanceledReplyReadDiscardsPreparedBody(t *testing.T) {
	doer := &recordingErrorDoer{err: errors.New("provider must not be called")}
	client, bindings := preparedBodyTestClient(t, doer, "cancel-binding", "cancel-token")
	base := &clientTaskPort{client: client}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	port := &cancelBeforeReplyReadPort{clientTaskPort: base, cancel: cancel}
	_, err := chatworkcmd.New(port).Execute(ctx, bindings[0], preparedBodyTestRequest())
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "chatwork_send_preflight_not_completed" {
		t.Fatalf("canceled reply preflight error = %#v", err)
	}
	if len(doer.requests) != 0 || len(base.prepared) != 1 {
		t.Fatalf("provider calls=%v prepared handles=%d", doer.requests, len(base.prepared))
	}
	assertPreparedMessageBodyCleared(t, base.prepared, "private-file-path", "private-body-canary")
}

func TestLivePreparedMessageBodyFormattingAndLogsHidePrivateState(t *testing.T) {
	const (
		privateBody    = "private-body-canary"
		privatePath    = "private-file-path-canary"
		privateBinding = "private-binding-canary"
		privateClient  = "private-client-canary"
		privateToken   = "private-token-canary"
	)
	doer := &recordingErrorDoer{err: errors.New("provider must not be called")}
	client, bindings := preparedBodyTestClient(t, doer, privateBinding, privateToken)
	request := preparedBodyTestRequest()
	request.BodyFile = privatePath
	client.baseURL = "https://" + privateClient + ".example"
	prepared, err := client.PrepareMessageBody(context.Background(), bindings[0], request)
	if err != nil {
		t.Fatal(err)
	}

	var buffer bytes.Buffer
	logger := log.New(&buffer, "", 0)
	for _, format := range []string{"%v", "%+v", "%#v"} {
		rendered := fmt.Sprintf(format, prepared)
		logger.Printf("prepared=%s", rendered)
		logger.Printf("prepared-direct="+format, prepared)
	}
	for _, secret := range []string{privateBody, privatePath, privateBinding, privateClient, privateToken} {
		if strings.Contains(buffer.String(), secret) {
			t.Fatalf("prepared handle logging leaked %q: %s", secret, buffer.String())
		}
	}
	if !strings.Contains(buffer.String(), "prepared Chatwork message body") {
		t.Fatalf("prepared body redaction marker missing from logs: %s", buffer.String())
	}
	client.DiscardPreparedMessageBody(prepared)
}

type cancelBeforeReplyReadPort struct {
	*clientTaskPort
	cancel context.CancelFunc
}

func (p *cancelBeforeReplyReadPort) Execute(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	if request.Task == chatwork.TaskMessagesShow {
		p.cancel()
	}
	return p.clientTaskPort.Execute(ctx, binding, request)
}

func preparedBodyTestClient(t *testing.T, doer *recordingErrorDoer, idPrefix, token string) (*Client, []authn.BindingID) {
	t.Helper()
	var nextID int
	client := newClient("http://prepared-body.example", token, doer, func() (string, error) {
		nextID++
		return fmt.Sprintf("%s-%d", idPrefix, nextID), nil
	}, nil)
	client.readBodyFile = func(context.Context, string) ([]byte, error) {
		return []byte("private-body-canary"), nil
	}
	bindings := make([]authn.BindingID, 0, 2)
	for range 2 {
		session, err := client.Authenticate(context.Background(), testRequirement())
		if err != nil {
			t.Fatal(err)
		}
		bindings = append(bindings, session.BindingID)
	}
	return client, bindings
}

func preparedBodyTestRequest() chatwork.Request {
	return chatwork.Request{
		Task: chatwork.TaskMessagesSend, Room: testRef(chatwork.ReferenceRoom, "42"), BodyFile: "private-file-path",
		MessageRecipients: []chatwork.Reference{
			testRef(chatwork.ReferenceAccount, "7"), testRef(chatwork.ReferenceAccount, "8"),
		},
		MessageReplyTarget: &chatwork.MessageReplyTarget{Message: testRef(chatwork.ReferenceMessage, "101")},
	}
}

func assertPreparedMessageBodyCleared(t *testing.T, handles []chatwork.PreparedMessageBody, path, body string) {
	t.Helper()
	if len(handles) != 1 {
		t.Fatalf("prepared handles = %d, want one", len(handles))
	}
	handle, ok := handles[0].(*preparedMessageBody)
	if !ok || handle == nil {
		t.Fatalf("prepared body handle has unexpected type %T", handles[0])
	}
	handle.mu.Lock()
	used, remainingBytes := handle.used, len(handle.bytes)
	remainingPath, owner, binding := handle.source.BodyFile, handle.owner, handle.binding
	handle.mu.Unlock()
	if !used || remainingBytes != 0 || remainingPath != "" || owner != nil || binding != (authn.BindingID{}) {
		t.Fatalf("prepared body state: used=%t bytes=%d path=%q owner-set=%t binding-set=%t, want cleared one-use handle", used, remainingBytes, remainingPath, owner != nil, binding != (authn.BindingID{}))
	}
	for _, representation := range []string{
		fmt.Sprintf("%v", handles[0]), fmt.Sprintf("%+v", handles[0]), fmt.Sprintf("%#v", handles[0]),
	} {
		if strings.Contains(representation, path) || strings.Contains(representation, body) {
			t.Fatalf("prepared body formatting leaked private data: %q", representation)
		}
	}
}

func TestLostReplySendResponseRemainsUnknownAfterPOST(t *testing.T) {
	var getCalls, postCalls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.Method + " " + request.URL.Path {
		case "GET /rooms/42/messages/101":
			getCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message_id":"101","account":{"account_id":7,"name":"A"},"body":"target","send_time":1,"update_time":0}`))
		case "POST /rooms/42/messages":
			postCalls.Add(1)
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Error("HTTP server does not support connection hijacking")
				return
			}
			connection, _, err := hijacker.Hijack()
			if err == nil {
				_ = connection.Close() // The server accepted the POST but lost its response.
			}
		default:
			t.Errorf("unexpected provider request: %s %s", request.Method, request.URL.Path)
		}
	}))
	defer server.Close()
	client, binding := authenticatedClient(t, server)
	client.readBodyFile = func(context.Context, string) ([]byte, error) { return []byte("body"), nil }
	_, err := chatworkcmd.New(client).Execute(context.Background(), binding, chatwork.Request{
		Task: chatwork.TaskMessagesSend, Room: testRef(chatwork.ReferenceRoom, "42"), BodyFile: "message.txt",
		MessageReplyTarget: &chatwork.MessageReplyTarget{Message: testRef(chatwork.ReferenceMessage, "101")},
	})
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "chatwork_mutation_outcome_unknown" || structured.Retryable {
		t.Fatalf("lost POST response = %#v", err)
	}
	if get, post := getCalls.Load(), postCalls.Load(); get != 1 || post != 1 {
		t.Fatalf("calls = GET %d POST %d, want one of each", get, post)
	}
}
