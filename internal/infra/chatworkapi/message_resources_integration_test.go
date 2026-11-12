package chatworkapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/cli/capsule"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

func TestMessageResourcesUsesSelectedExactUnfilteredRoomEndpoints(t *testing.T) {
	for _, test := range []struct {
		kind          chatwork.MessageResourceKind
		wantRoutes    []string
		wantFiles     bool
		wantRoomTasks bool
	}{
		{kind: chatwork.MessageResourceFiles, wantRoutes: []string{"/rooms/42/files"}, wantFiles: true},
		{kind: chatwork.MessageResourceTasks, wantRoutes: []string{"/rooms/42/tasks"}, wantRoomTasks: true},
		{kind: chatwork.MessageResourceAll, wantRoutes: []string{"/rooms/42/files", "/rooms/42/tasks"}, wantFiles: true, wantRoomTasks: true},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			var operations []string
			var writes int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				operations = append(operations, request.Method+" "+request.URL.Path)
				if request.Method != http.MethodGet {
					writes++
					t.Errorf("message resources wrote to provider: %s %s", request.Method, request.URL.Path)
				}
				if request.URL.RawQuery != "" || len(request.URL.Query()) != 0 {
					t.Errorf("source request has a filter: %s?%s", request.URL.Path, request.URL.RawQuery)
				}
				if request.RequestURI != request.URL.Path || request.ContentLength != 0 {
					t.Errorf("source request is not an exact body-free GET: URI=%q length=%d", request.RequestURI, request.ContentLength)
				}
				if request.Header.Get("x-chatworktoken") != syntheticToken || request.Header.Get("Authorization") != "" {
					t.Errorf("provider authentication header mismatch: x-chatworktoken=%q Authorization=%q", request.Header.Get("x-chatworktoken"), request.Header.Get("Authorization"))
				}
				w.Header().Set("Content-Type", "application/json")
				switch request.URL.Path {
				case "/rooms/42/files":
					_, _ = w.Write([]byte(`[
{"file_id":501,"account":{"account_id":7,"name":"Aki"},"message_id":"100","filename":"linked.txt","filesize":12,"upload_time":1},
{"file_id":502,"account":{"account_id":8,"name":"Beni"},"message_id":"101","filename":"different-message.txt","filesize":34,"upload_time":2}
]`))
				case "/rooms/42/tasks":
					_, _ = w.Write([]byte(`[
{"task_id":601,"account":{"account_id":9,"name":"Chika"},"assigned_by_account":{"account_id":10,"name":"Dora"},"message_id":"100","body":"linked task","status":"open","limit_time":1,"limit_type":"time"},
{"task_id":602,"account":{"account_id":11,"name":"Ema"},"assigned_by_account":{"account_id":12,"name":"Fumi"},"message_id":"101","body":"different-message task","status":"open","limit_time":1,"limit_type":"time"}
]`))
				default:
					t.Errorf("unexpected provider request: %s %s", request.Method, request.URL.Path)
					http.NotFound(w, request)
				}
			}))
			defer server.Close()

			ctx := context.WithValue(context.Background(), messageResourcesHTTPContextKey{}, "same-call-context")
			client := newClient(server.URL, syntheticToken, &messageResourcesContextDoer{base: server.Client(), key: messageResourcesHTTPContextKey{}, value: "same-call-context"},
				func() (string, error) { return "message-resources-integration-binding", nil }, boundedReadFile)
			session, err := client.Authenticate(ctx, testRequirement())
			if err != nil {
				t.Fatal(err)
			}
			port := &messageResourcesIntegrationPort{client: client}
			result, err := chatworkcmd.New(port).Execute(ctx, session.BindingID, chatwork.Request{
				Task: chatwork.TaskMessagesResources, Room: testRef(chatwork.ReferenceRoom, "42"),
				Message: testRef(chatwork.ReferenceMessage, "100"), MessageResourcesKind: test.kind,
			})
			if err != nil {
				t.Fatal(err)
			}
			wantOperations := make([]string, len(test.wantRoutes))
			for index, route := range test.wantRoutes {
				wantOperations[index] = http.MethodGet + " " + route
			}
			if !reflect.DeepEqual(operations, wantOperations) || writes != 0 {
				t.Fatalf("provider operations/writes = %v/%d, want %v/0", operations, writes, wantOperations)
			}
			if len(port.requests) != len(wantOperations) || len(port.contexts) != len(wantOperations) || len(port.bindings) != len(wantOperations) {
				t.Fatalf("task adapter calls = requests:%d contexts:%d bindings:%d, want %d", len(port.requests), len(port.contexts), len(port.bindings), len(wantOperations))
			}
			for index := range port.requests {
				if port.contexts[index] != ctx || port.bindings[index] != session.BindingID || port.requests[index].Room != testRef(chatwork.ReferenceRoom, "42") {
					t.Errorf("source %d changed context, binding, or room: request=%+v context=%p binding=%s", index, port.requests[index], port.contexts[index], port.bindings[index])
				}
			}
			assertResourceReadPolicy(t, test.kind)
			resources := result.MessageResources
			if resources == nil || resources.Kind != test.kind {
				t.Fatalf("selected kind result = %+v", resources)
			}
			if test.wantFiles {
				if resources.Files == nil || resources.Files.SourceCount != 2 || resources.Files.Coverage.Limit != 100 || resources.Files.Coverage.Complete || resources.Files.Absence != chatwork.MessageResourceAbsenceUnknown ||
					len(resources.Files.Matches) != 1 || resources.Files.Matches[0].Ref.Value != "501" || resources.Files.Matches[0].Message.Value != "100" {
					t.Fatalf("file DTO message_id did not survive the exact join: %+v", resources.Files)
				}
			} else if resources.Files != nil {
				t.Fatalf("unselected files source appeared: %+v", resources.Files)
			}
			if test.wantRoomTasks {
				if resources.Tasks == nil || resources.Tasks.SourceCount != 2 || resources.Tasks.Coverage.Limit != 100 || resources.Tasks.Coverage.Complete || resources.Tasks.Absence != chatwork.MessageResourceAbsenceUnknown ||
					len(resources.Tasks.Matches) != 1 || resources.Tasks.Matches[0].Ref.Value != "601" || resources.Tasks.Matches[0].Message.Value != "100" {
					t.Fatalf("room-task DTO message_id did not survive the exact join: %+v", resources.Tasks)
				}
			} else if resources.Tasks != nil {
				t.Fatalf("unselected room-tasks source appeared: %+v", resources.Tasks)
			}
			if err := result.ValidateFor(chatwork.Request{Task: chatwork.TaskMessagesResources, Room: testRef(chatwork.ReferenceRoom, "42"), Message: testRef(chatwork.ReferenceMessage, "100"), MessageResourcesKind: test.kind}); err != nil {
				t.Fatalf("integrated result is not bound to the selected invocation: %v", err)
			}
		})
	}
}

func TestMessageResources403And429AbortWithoutPartialRenderedOutputOrWrites(t *testing.T) {
	for _, test := range []struct {
		name           string
		failRoute      string
		status         int
		wantCode       string
		wantOperations []string
	}{
		{name: "files forbidden", failRoute: "/rooms/42/files", status: http.StatusForbidden, wantCode: "chatwork_permission_denied", wantOperations: []string{"GET /rooms/42/files"}},
		{name: "room tasks rate limited", failRoute: "/rooms/42/tasks", status: http.StatusTooManyRequests, wantCode: "chatwork_rate_limited", wantOperations: []string{"GET /rooms/42/files", "GET /rooms/42/tasks"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var operations []string
			var writes int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				operations = append(operations, request.Method+" "+request.URL.Path)
				if request.Method != http.MethodGet {
					writes++
				}
				if request.URL.RawQuery != "" || len(request.URL.Query()) != 0 {
					t.Errorf("source request contains filters: %s", request.URL.String())
				}
				if request.RequestURI != request.URL.Path || request.ContentLength != 0 {
					t.Errorf("source request is not an exact body-free GET: URI=%q length=%d", request.RequestURI, request.ContentLength)
				}
				w.Header().Set("Content-Type", "application/json")
				if request.URL.Path == test.failRoute {
					w.WriteHeader(test.status)
					_, _ = w.Write([]byte(`{"errors":["private-provider-error-canary"]}`))
					return
				}
				switch request.URL.Path {
				case "/rooms/42/files":
					_, _ = w.Write([]byte(`[{"file_id":501,"account":{"account_id":7},"message_id":"100","filename":"partial.txt","filesize":12,"upload_time":1}]`))
				case "/rooms/42/tasks":
					_, _ = w.Write([]byte(`[{"task_id":601,"account":{"account_id":9},"assigned_by_account":{"account_id":10},"message_id":"100","body":"partial task","status":"open","limit_time":1,"limit_type":"time"}]`))
				default:
					t.Errorf("unexpected provider request: %s %s", request.Method, request.URL.Path)
					http.NotFound(w, request)
				}
			}))
			defer server.Close()

			ctx := context.Background()
			client := newClient(server.URL, syntheticToken, server.Client(), func() (string, error) { return "message-resources-fault-binding", nil }, boundedReadFile)
			session, err := client.Authenticate(ctx, testRequirement())
			if err != nil {
				t.Fatal(err)
			}
			result, taskErr := chatworkcmd.New(client).Execute(ctx, session.BindingID, chatwork.Request{
				Task: chatwork.TaskMessagesResources, Room: testRef(chatwork.ReferenceRoom, "42"),
				Message: testRef(chatwork.ReferenceMessage, "100"), MessageResourcesKind: chatwork.MessageResourceAll,
			})
			structured, ok := fault.PublicCopy(taskErr)
			if !ok || structured.Code != test.wantCode || result.Task != "" || result.MessageResources != nil {
				t.Fatalf("fault/result = %+v / %+v", structured, result)
			}
			if !reflect.DeepEqual(operations, test.wantOperations) || writes != 0 {
				t.Fatalf("fault operations/writes = %v/%d, want %v/0", operations, writes, test.wantOperations)
			}

			// The CLI's error branch renders only successful results. Keep that
			// boundary in this real-transport test so a successful first source
			// cannot become partial output when the second source fails.
			var stdout bytes.Buffer
			if taskErr == nil {
				output, renderErr := capsule.Render(result)
				if renderErr != nil {
					t.Fatal(renderErr)
				}
				stdout.WriteString(output)
			}
			if stdout.Len() != 0 {
				t.Fatalf("failed composite read emitted partial stdout: %s", stdout.String())
			}
		})
	}
}

func assertResourceReadPolicy(t *testing.T, kind chatwork.MessageResourceKind) {
	t.Helper()
	for _, task := range []chatwork.Task{chatwork.TaskFilesList, chatwork.TaskRoomTasksList} {
		if kind == chatwork.MessageResourceFiles && task != chatwork.TaskFilesList || kind == chatwork.MessageResourceTasks && task != chatwork.TaskRoomTasksList {
			continue
		}
		policy, err := CallPolicy(task)
		if err != nil {
			t.Fatalf("CallPolicy(%s): %v", task, err)
		}
		if err := policy.Validate(operation.EffectRead); err != nil || policy.MaxAttempts != 1 || policy.Timeout != RequestTimeout {
			t.Fatalf("%s policy = %+v err=%v, want one 20-second read attempt", task, policy, err)
		}
	}
}

type messageResourcesIntegrationPort struct {
	client   *Client
	requests []chatwork.Request
	contexts []context.Context
	bindings []authn.BindingID
}

func (p *messageResourcesIntegrationPort) Execute(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.requests = append(p.requests, request)
	p.contexts = append(p.contexts, ctx)
	p.bindings = append(p.bindings, binding)
	return p.client.Execute(ctx, binding, request)
}

type messageResourcesHTTPContextKey struct{}

type messageResourcesContextDoer struct {
	base interface {
		Do(*http.Request) (*http.Response, error)
	}
	key   any
	value any
}

func (d *messageResourcesContextDoer) Do(request *http.Request) (*http.Response, error) {
	if got := request.Context().Value(d.key); got != d.value {
		return nil, fmt.Errorf("HTTP request lost caller context value: got %v", got)
	}
	return d.base.Do(request)
}
