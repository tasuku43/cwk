//go:build timeout_observation

package chatworkapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

// This opt-in observation is intentionally excluded from ordinary test runs.
// It performs only two concurrent requests to a synthetic localhost server.
func TestObserveProductionTimeoutEdges(t *testing.T) {
	const delay = 20*time.Second + 700*time.Millisecond
	var headerCalls, bodyCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/rooms/2/files" ||
			request.Header.Get("x-chatworktoken") != syntheticToken {
			t.Errorf("unexpected synthetic request: %s %s", request.Method, request.URL.Path)
		}
		if err := request.ParseMultipartForm(MaxUploadBytes); err != nil {
			t.Errorf("parse synthetic upload: %v", err)
			return
		}
		files := request.MultipartForm.File["file"]
		if len(files) != 1 {
			t.Errorf("upload file parts = %d, want one", len(files))
			return
		}
		const response = `{"file_id":"9"}`
		switch files[0].Filename {
		case "delayed-header.bin":
			headerCalls.Add(1)
			time.Sleep(delay)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		case "delayed-body.bin":
			bodyCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(delay)
			_, _ = w.Write([]byte(response))
		default:
			t.Errorf("unexpected synthetic filename %q", files[0].Filename)
			http.Error(w, "unexpected synthetic request", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	makeUploadClient := func() (*Client, authn.BindingID) {
		client := newClient(server.URL, syntheticToken, productionHTTPClient(), func() (string, error) {
			return "timeout-observation-binding", nil
		}, func(string) ([]byte, error) { return []byte("synthetic upload bytes"), nil })
		session, err := client.Authenticate(context.Background(), testRequirement())
		if err != nil {
			t.Fatal(err)
		}
		return client, session.BindingID
	}
	headerClient, headerBinding := makeUploadClient()
	bodyClient, bodyBinding := makeUploadClient()
	policy, err := CallPolicy(chatwork.TaskFilesUpload)
	if err != nil {
		t.Fatal(err)
	}
	transport := productionHTTPClient().Transport.(*http.Transport)
	fmt.Printf("selected upload policy=%s attempts=%d client_timeout=%s shared_response_header_timeout=%s delay=%s\n",
		policy.Timeout, policy.MaxAttempts, productionHTTPClient().Timeout, transport.ResponseHeaderTimeout, delay)

	type observation struct {
		name    string
		elapsed time.Duration
		result  string
		err     error
	}
	observations := make(chan observation, 2)
	for _, item := range []struct {
		name     string
		client   *Client
		binding  authn.BindingID
		filename string
	}{
		{name: "delayed-header", client: headerClient, binding: headerBinding, filename: "delayed-header.bin"},
		{name: "delayed-body", client: bodyClient, binding: bodyBinding, filename: "delayed-body.bin"},
	} {
		go func(item struct {
			name     string
			client   *Client
			binding  authn.BindingID
			filename string
		}) {
			ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
			defer cancel()
			request := completeRequest(chatwork.TaskFilesUpload)
			request.FilePath = item.filename
			started := time.Now()
			result, err := item.client.Execute(ctx, item.binding, request)
			elapsed := time.Since(started)
			state := "success"
			if err != nil {
				state = "error"
			}
			if result.Task == chatwork.TaskFilesUpload && err == nil && result.CreatedInRoom != nil {
				state = "confirmed-success"
			}
			observations <- observation{name: item.name, elapsed: elapsed, result: state, err: err}
		}(item)
	}

	seen := make(map[string]observation, 2)
	for range 2 {
		observation := <-observations
		seen[observation.name] = observation
		var structured *fault.Error
		if observation.name == "delayed-header" {
			if !errors.As(observation.err, &structured) || structured.Code != "chatwork_mutation_outcome_unknown" || structured.Retryable {
				t.Fatalf("delayed-header result = %s, err=%#v; want non-retryable unknown mutation outcome", observation.result, observation.err)
			}
			if errors.Is(observation.err, context.DeadlineExceeded) {
				t.Fatalf("delayed-header private deadline escaped classifier: %#v", observation.err)
			}
		} else if observation.result != "confirmed-success" || observation.err != nil {
			t.Fatalf("delayed-body result = %s, err=%#v; want confirmed upload success", observation.result, observation.err)
		}
	}
	if headerCalls.Load() != 1 || bodyCalls.Load() != 1 {
		t.Fatalf("synthetic request counts: header=%d body=%d; want one each", headerCalls.Load(), bodyCalls.Load())
	}
	fmt.Printf("delayed-header elapsed=%s outcome=%s retryable=false requests=%d\n",
		seen["delayed-header"].elapsed.Round(10*time.Millisecond), seen["delayed-header"].result, headerCalls.Load())
	fmt.Printf("delayed-body elapsed=%s outcome=%s requests=%d\n",
		seen["delayed-body"].elapsed.Round(10*time.Millisecond), seen["delayed-body"].result, bodyCalls.Load())
}
