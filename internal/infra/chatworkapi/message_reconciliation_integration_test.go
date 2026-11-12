package chatworkapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestMessageSendReconciliationReadsExactReplyThenOneLatestWindowAndNeverPosts(t *testing.T) {
	var operations []string
	var events []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		operations = append(operations, request.Method+" "+request.URL.Path)
		events = append(events, "provider:"+request.Method+" "+request.URL.Path)
		if request.Header.Get("x-chatworktoken") != syntheticToken || request.Header.Get("Authorization") != "" {
			t.Errorf("provider auth headers leaked or changed: x-chatworktoken=%q Authorization=%q", request.Header.Get("x-chatworktoken"), request.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/me":
			_, _ = w.Write([]byte(`{"account_id":9,"name":"Self"}`))
		case request.Method == http.MethodGet && request.URL.Path == "/rooms/42/messages/101":
			_, _ = w.Write([]byte(`{"message_id":"101","account":{"account_id":7,"name":"Reply Author"},"body":"target","send_time":1,"update_time":0}`))
		case request.Method == http.MethodGet && request.URL.Path == "/rooms/42/messages":
			if request.URL.Query().Get("force") != "1" || len(request.URL.Query()) != 1 {
				t.Errorf("latest list query = %v, want only force=1", request.URL.Query())
			}
			_, _ = w.Write([]byte(`[
{"message_id":"201","account":{"account_id":9,"name":"Other author"},"body":"[To:9] [rp aid=7 to=42-101] 日本語\r\nbody","send_time":10,"update_time":0},
{"message_id":"202","account":{"account_id":8,"name":"Different"},"body":"[To:9] [rp aid=7 to=42-101] 日本語\nbody","send_time":11,"update_time":0}
]`))
		default:
			t.Errorf("unexpected provider request: %s %s", request.Method, request.URL.Path)
			http.Error(w, "unexpected", http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	client, binding := authenticatedClient(t, server)
	fileReads := 0
	client.readBodyFile = func(ctx context.Context, path string) ([]byte, error) {
		fileReads++
		events = append(events, "file-read")
		if path != "private-message.txt" {
			t.Fatalf("body-file path = %q", path)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return []byte("日本語\r\nbody"), nil
	}
	ctx := context.WithValue(context.Background(), struct{}{}, "reconciliation-context")
	request := chatwork.Request{
		Task: chatwork.TaskMessagesReconcileSend, Room: testRef(chatwork.ReferenceRoom, "42"),
		BodyFile:           "private-message.txt",
		MessageRecipients:  []chatwork.Reference{testRef(chatwork.ReferenceAccount, "9")},
		MessageReplyTarget: &chatwork.MessageReplyTarget{Message: testRef(chatwork.ReferenceMessage, "101")},
	}
	result, err := chatworkcmd.New(client).Execute(ctx, binding, request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(operations, []string{
		"GET /rooms/42/messages/101", "GET /me", "GET /rooms/42/messages",
	}) {
		t.Fatalf("provider operations = %v; want one exact GET and one latest GET", operations)
	}
	if !reflect.DeepEqual(events, []string{
		"file-read", "provider:GET /rooms/42/messages/101", "provider:GET /me", "provider:GET /rooms/42/messages",
	}) {
		t.Fatalf("operation order = %v; body file must validate before either read", events)
	}
	if fileReads != 1 || result.Task != chatwork.TaskMessagesReconcileSend || result.MessageSendReconciliation == nil {
		t.Fatalf("body reads=%d result=%+v", fileReads, result)
	}
	if result.MessageSendReconciliation.Outcome != "unknown" || result.MessageSendReconciliation.CandidateCount != 1 ||
		result.MessageSendReconciliation.Candidates[0].Message.Value != "201" ||
		result.MessageSendReconciliation.Candidates[0].Sender.Value != "9" ||
		result.MessageSendReconciliation.Candidates[0].SendTime != 10 {
		t.Fatalf("reconciliation result = %+v", result.MessageSendReconciliation)
	}
	if result.Messages != nil {
		t.Fatalf("result retained raw provider bodies: %+v", result.Messages)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"日本語", "private-message.txt", "private-message"} {
		if bytes.Contains(encoded, []byte(private)) {
			t.Errorf("result JSON leaked %q: %s", private, encoded)
		}
	}
}
