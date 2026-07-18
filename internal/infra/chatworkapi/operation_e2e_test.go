package chatworkapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestEveryTaskMakesOneAuthenticatedCallThroughTheTransport(t *testing.T) {
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
				w.WriteHeader(fixture.Status)
				if len(fixture.Body) != 0 {
					_, _ = w.Write(fixture.Body)
				}
			}))
			defer server.Close()

			client, binding := authenticatedClient(t, server)
			result, err := client.Execute(context.Background(), binding, completeRequest(fixture.Task))
			if err != nil {
				t.Fatal(err)
			}
			if result.Task != fixture.Task {
				t.Fatalf("result task = %q, want %q", result.Task, fixture.Task)
			}
			if got := calls.Load(); got != 1 {
				t.Fatalf("provider calls = %d, want 1", got)
			}
		})
	}
}
