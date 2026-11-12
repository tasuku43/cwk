package chatworkapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

func TestReconciliationMatcherUsesExpandedSendBodyByteForByteAndConsumesOnce(t *testing.T) {
	providerCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		providerCalls++
		t.Errorf("unexpected HTTP request during body matching: %s %s", request.Method, request.URL.Path)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer server.Close()
	client, binding := authenticatedClient(t, server)
	reads := 0
	client.readBodyFile = func(_ context.Context, path string) ([]byte, error) {
		reads++
		if path != "private-message.txt" {
			t.Fatalf("body-file path = %q", path)
		}
		return []byte("日本語\r\n [To:77] 手書き\n"), nil
	}
	request := reconciliationTestRequest()
	request.Body = ""
	request.BodyFile = "private-message.txt"
	prepared, err := client.PrepareMessageSendReconciliation(context.Background(), binding, request)
	if err != nil {
		t.Fatal(err)
	}
	resolved := request
	resolved.MessageReplyTarget = &chatwork.MessageReplyTarget{
		Message: request.MessageReplyTarget.Message,
		Sender:  testRef(chatwork.ReferenceAccount, "7"),
	}
	if err := client.FinalizeMessageSendReconciliation(context.Background(), binding, resolved, prepared); err != nil {
		t.Fatal(err)
	}
	want, err := expandMessageBody(resolved, "日本語\r\n [To:77] 手書き\n")
	if err != nil {
		t.Fatal(err)
	}
	close := "[To:9] [To:3] [rp aid=7 to=42-101] "
	if !strings.HasPrefix(want, close) {
		t.Fatalf("expanded body = %q, want common builder prefix %q", want, close)
	}
	matches, err := client.MatchMessageSendReconciliation(context.Background(), binding, resolved, prepared, []string{
		want,
		want + " ",
		close + "日本語\n [To:77] 手書き\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 3 || !matches[0] || matches[1] || matches[2] {
		t.Fatalf("exact body matches = %v", matches)
	}
	if reads != 1 || providerCalls != 0 {
		t.Fatalf("file reads=%d provider calls=%d; want one read and no HTTP", reads, providerCalls)
	}
	handle := prepared.(*preparedMessageSendReconciliation)
	handle.mu.Lock()
	cleared := len(handle.body) == 0 && handle.owner == nil && handle.binding == (authn.BindingID{}) && handle.used
	handle.mu.Unlock()
	if !cleared {
		t.Fatal("matcher did not zero and consume its one-use body handle")
	}
	if _, err := client.MatchMessageSendReconciliation(context.Background(), binding, resolved, prepared, []string{want}); err == nil {
		t.Fatal("second use of the matcher succeeded")
	}
	client.DiscardMessageSendReconciliation(prepared)
	if reads != 1 || providerCalls != 0 {
		t.Fatalf("cleanup reread body or called provider: file reads=%d HTTP=%d", reads, providerCalls)
	}
}

func TestReconciliationMatcherBindsClientBindingAndRequest(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client, binding := authenticatedClient(t, server)
	otherClient, otherBinding := authenticatedClient(t, server)
	client.newID = func() (string, error) { return "second-binding", nil }
	secondSession, err := client.Authenticate(context.Background(), testRequirement())
	if err != nil {
		t.Fatal(err)
	}
	otherBinding = secondSession.BindingID
	request := reconciliationTestRequest()
	prepared, err := client.PrepareMessageSendReconciliation(context.Background(), binding, request)
	if err != nil {
		t.Fatal(err)
	}
	resolved := request
	resolved.MessageReplyTarget = &chatwork.MessageReplyTarget{Message: request.MessageReplyTarget.Message, Sender: testRef(chatwork.ReferenceAccount, "7")}
	if err := client.FinalizeMessageSendReconciliation(context.Background(), binding, resolved, prepared); err != nil {
		t.Fatal(err)
	}
	for _, attempt := range []struct {
		name    string
		client  *Client
		binding authn.BindingID
		request chatwork.Request
	}{
		{name: "different client", client: otherClient, binding: binding, request: resolved},
		{name: "different binding", client: client, binding: otherBinding, request: resolved},
		{name: "different recipient order", client: client, binding: binding, request: reconciliationRequestWithRecipients(resolved, "3", "9")},
	} {
		t.Run(attempt.name, func(t *testing.T) {
			if _, err := attempt.client.MatchMessageSendReconciliation(context.Background(), attempt.binding, attempt.request, prepared, []string{"payload"}); err == nil {
				t.Fatal("mismatched matcher binding/request succeeded")
			}
		})
	}
	want, err := expandMessageBody(resolved, request.Body)
	if err != nil {
		t.Fatal(err)
	}
	matches, err := client.MatchMessageSendReconciliation(context.Background(), binding, resolved, prepared, []string{want})
	if err != nil || len(matches) != 1 || !matches[0] {
		t.Fatalf("correct matcher use = %v, %v", matches, err)
	}
}

func TestReconciliationBodyFileValidationFaultsBeforeAnyProviderRead(t *testing.T) {
	providerCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		providerCalls++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client, binding := authenticatedClient(t, server)
	privateCanary := "private-reader-error-canary"
	client.readBodyFile = func(context.Context, string) ([]byte, error) {
		return nil, errors.New(privateCanary)
	}
	request := reconciliationTestRequest()
	request.Body = ""
	request.BodyFile = "private-path-canary"
	_, err := client.PrepareMessageSendReconciliation(context.Background(), binding, request)
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Code != "chatwork_message_body_file_unreadable" || structured.Retryable {
		t.Fatalf("body-file preparation error = %#v", err)
	}
	if strings.Contains(err.Error(), privateCanary) || strings.Contains(err.Error(), request.BodyFile) || providerCalls != 0 {
		t.Fatalf("private data leaked or provider was called: error=%v calls=%d", err, providerCalls)
	}

	for name, value := range map[string][]byte{
		"empty": []byte{}, "invalid UTF-8": {0xff}, "oversized": []byte(strings.Repeat("x", chatwork.MaxMessageBodyBytes+1)),
	} {
		t.Run(name, func(t *testing.T) {
			client.readBodyFile = func(context.Context, string) ([]byte, error) { return value, nil }
			_, err := client.PrepareMessageSendReconciliation(context.Background(), binding, request)
			if err == nil || providerCalls != 0 {
				t.Fatalf("invalid body succeeded or called provider: err=%v calls=%d", err, providerCalls)
			}
		})
	}
}

func reconciliationTestRequest() chatwork.Request {
	return chatwork.Request{
		Task: chatwork.TaskMessagesReconcileSend, Room: testRef(chatwork.ReferenceRoom, "42"), Body: "payload",
		MessageRecipients:  []chatwork.Reference{testRef(chatwork.ReferenceAccount, "9"), testRef(chatwork.ReferenceAccount, "3")},
		MessageReplyTarget: &chatwork.MessageReplyTarget{Message: testRef(chatwork.ReferenceMessage, "101")},
	}
}

func reconciliationRequestWithRecipients(request chatwork.Request, recipients ...string) chatwork.Request {
	request.MessageRecipients = make([]chatwork.Reference, len(recipients))
	for index, recipient := range recipients {
		request.MessageRecipients[index] = testRef(chatwork.ReferenceAccount, recipient)
	}
	return request
}
