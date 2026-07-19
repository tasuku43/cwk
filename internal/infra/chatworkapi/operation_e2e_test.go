package chatworkapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestEveryTaskUsesTheAuthenticatedTransportPath(t *testing.T) {
	corpus := readResponseFixtureCorpus(t)
	for _, fixture := range corpus.Success {
		fixture := fixture
		t.Run(string(fixture.Task), func(t *testing.T) {
			var calls atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				calls.Add(1)
				if got := request.Header.Get("x-chatworktoken"); got != syntheticToken {
					t.Errorf("x-chatworktoken = %q", got)
				}
				if got := request.Header.Get("Authorization"); got != "" {
					t.Errorf("unexpected authorization header = %q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				if request.Method == http.MethodGet && request.URL.Path == "/me" && fixture.Task == chatwork.TaskRoomsCreate {
					_, _ = w.Write([]byte(`{"account_id":1,"room_id":2,"name":"Synthetic"}`))
					return
				}
				w.WriteHeader(fixture.Status)
				if len(fixture.Body) != 0 {
					_, _ = w.Write(fixture.Body)
				}
			}))
			defer server.Close()

			client := newClient(server.URL, syntheticToken, server.Client(), func() (string, error) { return "operation-e2e-binding", nil }, func(string) ([]byte, error) { return []byte("file"), nil })
			requirement := testRequirement()
			expectedCalls := int64(1)
			if fixture.Task == chatwork.TaskRoomsCreate {
				requirement.AccountID = "1"
				expectedCalls = 2
			}
			session, err := client.Authenticate(context.Background(), requirement)
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.Execute(context.Background(), session.BindingID, completeRequest(fixture.Task))
			if err != nil {
				t.Fatal(err)
			}
			if result.Task != fixture.Task {
				t.Fatalf("result task = %q, want %q", result.Task, fixture.Task)
			}
			if got := calls.Load(); got != expectedCalls {
				t.Fatalf("provider calls = %d, want %d", got, expectedCalls)
			}
		})
	}
}
