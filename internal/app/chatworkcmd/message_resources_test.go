package chatworkcmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type messageResourcesStep struct {
	task   chatwork.Task
	result chatwork.Result
	err    error
}

type messageResourcesPort struct {
	steps    []messageResourcesStep
	requests []chatwork.Request
	contexts []context.Context
	bindings []authn.BindingID
	cancel   context.CancelFunc
}

func (p *messageResourcesPort) Execute(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.requests = append(p.requests, request)
	p.contexts = append(p.contexts, ctx)
	p.bindings = append(p.bindings, binding)
	stepIndex := len(p.requests) - 1
	if stepIndex >= len(p.steps) {
		return chatwork.Result{}, fmt.Errorf("unexpected resource source request %s", request.Task)
	}
	step := p.steps[stepIndex]
	if request.Task != step.task {
		return chatwork.Result{}, fmt.Errorf("source request %s, want %s", request.Task, step.task)
	}
	if p.cancel != nil {
		p.cancel()
	}
	return step.result, step.err
}

func TestMessageResourcesReadsSelectedSourcesAndFiltersExactSameRoomReferences(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	otherRoom := relationshipReference(t, chatwork.ReferenceRoom, "43")
	message := relationshipReference(t, chatwork.ReferenceMessage, "100")
	otherMessage := relationshipReference(t, chatwork.ReferenceMessage, "101")
	files := []chatwork.File{
		messageResourcesFile(t, room, message, "501", "hostile file\nname"),
		messageResourcesFile(t, room, message, "501", "duplicate exact row"),
		messageResourcesFile(t, room, otherMessage, "502", "other message"),
		messageResourcesFile(t, otherRoom, message, "503", "same id from other room"),
	}
	tasks := []chatwork.WorkTask{
		messageResourcesTask(t, room, message, "601", "hostile task\x1b[31m"),
		messageResourcesTask(t, otherRoom, message, "602", "same id from other room"),
		messageResourcesTask(t, room, otherMessage, "603", "other message"),
	}
	port := &messageResourcesPort{steps: []messageResourcesStep{
		{task: chatwork.TaskFilesList, result: chatwork.Result{Task: chatwork.TaskFilesList, Coverage: messageResourcesCoverage(), Files: files}},
		{task: chatwork.TaskRoomTasksList, result: chatwork.Result{Task: chatwork.TaskRoomTasksList, Coverage: messageResourcesCoverage(), Tasks: tasks}},
	}}
	binding := testBinding(t)
	ctx := context.WithValue(context.Background(), messageResourcesContextKey{}, "same-context")
	result, err := New(port).Execute(ctx, binding, chatwork.Request{
		Task: chatwork.TaskMessagesResources, Room: room, Message: message,
		MessageResourcesKind: chatwork.MessageResourceAll,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 2 || port.requests[0].Task != chatwork.TaskFilesList || port.requests[1].Task != chatwork.TaskRoomTasksList {
		t.Fatalf("source request order = %+v", port.requests)
	}
	for index, request := range port.requests {
		if request.Room != room || request.Message != (chatwork.Reference{}) || request.Account != (chatwork.Reference{}) || request.AssignedBy != (chatwork.Reference{}) || request.Status != "" {
			t.Errorf("source %d received filters or wrong scope: %+v", index, request)
		}
		if port.contexts[index] != ctx || port.bindings[index] != binding {
			t.Errorf("source %d changed context or authentication binding", index)
		}
	}
	resources := result.MessageResources
	if resources == nil || result.Task != chatwork.TaskMessagesResources || resources.Room != room || resources.Message != message || resources.Kind != chatwork.MessageResourceAll {
		t.Fatalf("result identity = %+v", resources)
	}
	if resources.Files.SourceCount != 4 || resources.Files.Coverage.Limit != 100 || resources.Files.Coverage.Complete ||
		resources.Files.Absence != chatwork.MessageResourceAbsenceUnknown || len(resources.Files.Matches) != 2 || resources.Files.Matches[0].Ref.Value != "501" || resources.Files.Matches[1].Ref.Value != "501" {
		t.Fatalf("file resource matches/source = %+v", resources.Files)
	}
	if resources.Tasks.SourceCount != 3 || resources.Tasks.Coverage.Limit != 100 || resources.Tasks.Coverage.Complete ||
		resources.Tasks.Absence != chatwork.MessageResourceAbsenceUnknown || len(resources.Tasks.Matches) != 1 || resources.Tasks.Matches[0].Ref.Value != "601" {
		t.Fatalf("room-task resource matches/source = %+v", resources.Tasks)
	}
	if result.ValidateFor(chatwork.Request{Task: chatwork.TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: chatwork.MessageResourceAll}) != nil {
		t.Fatal("application result does not validate for its exact request")
	}
}

func TestMessageResourcesReadsOnlySelectedSourceAndPreservesEmptyMatches(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "100")
	for _, test := range []struct {
		kind chatwork.MessageResourceKind
		task chatwork.Task
	}{
		{kind: chatwork.MessageResourceFiles, task: chatwork.TaskFilesList},
		{kind: chatwork.MessageResourceTasks, task: chatwork.TaskRoomTasksList},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			var result chatwork.Result
			if test.task == chatwork.TaskFilesList {
				result = chatwork.Result{Task: test.task, Coverage: messageResourcesCoverage(), Files: []chatwork.File{messageResourcesFile(t, room, relationshipReference(t, chatwork.ReferenceMessage, "200"), "501", "unrelated")}}
			} else {
				result = chatwork.Result{Task: test.task, Coverage: messageResourcesCoverage(), Tasks: []chatwork.WorkTask{messageResourcesTask(t, room, relationshipReference(t, chatwork.ReferenceMessage, "200"), "601", "unrelated")}}
			}
			port := &messageResourcesPort{steps: []messageResourcesStep{{task: test.task, result: result}}}
			got, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
				Task: chatwork.TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: test.kind,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(port.requests) != 1 || port.requests[0].Task != test.task {
				t.Fatalf("selected source requests = %+v", port.requests)
			}
			if got.MessageResources == nil || got.MessageResources.Kind != test.kind {
				t.Fatalf("kind = %+v", got.MessageResources)
			}
			if test.kind == chatwork.MessageResourceFiles {
				if got.MessageResources.Files == nil || got.MessageResources.Files.Matches == nil || len(got.MessageResources.Files.Matches) != 0 || got.MessageResources.Tasks != nil {
					t.Fatalf("empty file matches are not explicit or unselected tasks appeared: %+v", got.MessageResources)
				}
			} else if got.MessageResources.Tasks == nil || got.MessageResources.Tasks.Matches == nil || len(got.MessageResources.Tasks.Matches) != 0 || got.MessageResources.Files != nil {
				t.Fatalf("empty task matches are not explicit or unselected files appeared: %+v", got.MessageResources)
			}
		})
	}
}

func TestMessageResourcesAbortsAfterSecondSourceFaultWithoutReturningPartialResult(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "100")
	permission := fault.New(fault.KindPermission, "chatwork_permission_denied", "permission denied", false)
	port := &messageResourcesPort{steps: []messageResourcesStep{
		{task: chatwork.TaskFilesList, result: chatwork.Result{Task: chatwork.TaskFilesList, Coverage: messageResourcesCoverage(), Files: []chatwork.File{messageResourcesFile(t, room, message, "501", "hidden partial")}}},
		{task: chatwork.TaskRoomTasksList, err: permission},
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: chatwork.MessageResourceAll,
	})
	structured, ok := fault.PublicCopy(err)
	if !ok || structured.Code != "chatwork_permission_denied" || result.Task != "" || result.MessageResources != nil || len(port.requests) != 2 {
		t.Fatalf("result/error/requests = %+v / %v / %+v", result, err, port.requests)
	}
}

func TestMessageResourcesPreservesRateLimitFaultWithoutReturningPartialResult(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "100")
	rateLimit := fault.New(fault.KindRateLimited, "chatwork_rate_limited", "Chatwork rate limit was reached", true)
	port := &messageResourcesPort{steps: []messageResourcesStep{
		{task: chatwork.TaskFilesList, result: chatwork.Result{Task: chatwork.TaskFilesList, Coverage: messageResourcesCoverage(), Files: []chatwork.File{messageResourcesFile(t, room, message, "501", "hidden partial")}}},
		{task: chatwork.TaskRoomTasksList, err: rateLimit},
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: chatwork.MessageResourceAll,
	})
	structured, ok := fault.PublicCopy(err)
	if !ok || structured.Code != "chatwork_rate_limited" || !structured.Retryable || result.Task != "" || result.MessageResources != nil || len(port.requests) != 2 {
		t.Fatalf("result/error/requests = %+v / %v / %+v", result, err, port.requests)
	}
}

func TestMessageResourcesCancellationStopsBeforeNextSource(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "100")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	port := &messageResourcesPort{
		steps:  []messageResourcesStep{{task: chatwork.TaskFilesList, result: chatwork.Result{Task: chatwork.TaskFilesList, Coverage: messageResourcesCoverage(), Files: []chatwork.File{}}}},
		cancel: cancel,
	}
	result, err := New(port).Execute(ctx, testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: chatwork.MessageResourceAll,
	})
	structured, ok := fault.PublicCopy(err)
	if !ok || structured.Code != "operation_canceled" || result.Task != "" || len(port.requests) != 1 {
		t.Fatalf("result/error/requests = %+v / %v / %+v", result, err, port.requests)
	}
}

func TestMessageResourcesRejectsOverBoundSourceBeforeFiltering(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "100")
	files := make([]chatwork.File, chatwork.MaxMessageResourceSourceItems+1)
	for index := range files {
		files[index] = messageResourcesFile(t, room, message, fmt.Sprint(index+1), "file")
	}
	port := &messageResourcesPort{steps: []messageResourcesStep{{
		task:   chatwork.TaskFilesList,
		result: chatwork.Result{Task: chatwork.TaskFilesList, Coverage: messageResourcesCoverage(), Files: files},
	}}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: chatwork.MessageResourceFiles,
	})
	structured, ok := fault.PublicCopy(err)
	if !ok || structured.Code != "chatwork_result_invalid" || result.Task != "" {
		t.Fatalf("result/error = %+v / %v", result, err)
	}
}

func TestMessageResourcesRequestRejectsUnusedFiltersAndInvalidReferencesBeforePort(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "100")
	port := &messageResourcesPort{}
	for _, request := range []chatwork.Request{
		{Task: chatwork.TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: "unknown"},
		{Task: chatwork.TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: chatwork.MessageResourceFiles, Account: relationshipReference(t, chatwork.ReferenceAccount, "7")},
		{Task: chatwork.TaskMessagesResources, Room: room, Message: chatwork.Reference{Kind: chatwork.ReferenceMessage, Value: "0100"}, MessageResourcesKind: chatwork.MessageResourceFiles},
	} {
		if _, err := New(port).Execute(context.Background(), testBinding(t), request); err == nil {
			t.Errorf("invalid request passed: %+v", request)
		}
	}
	if len(port.requests) != 0 {
		t.Fatalf("invalid requests reached provider: %+v", port.requests)
	}
}

func messageResourcesCoverage() chatwork.Coverage {
	return chatwork.Coverage{Kind: "provider_window", Limit: chatwork.MaxMessageResourceSourceItems, Complete: false, Description: "bounded provider window"}
}

func messageResourcesFile(t *testing.T, room, message chatwork.Reference, refValue, name string) chatwork.File {
	t.Helper()
	return chatwork.File{
		Ref: relationshipReference(t, chatwork.ReferenceFile, refValue), Room: room,
		Account: chatwork.Account{Ref: relationshipReference(t, chatwork.ReferenceAccount, "7"), Name: "external"},
		Message: message, Name: name,
	}
}

func messageResourcesTask(t *testing.T, room, message chatwork.Reference, refValue, body string) chatwork.WorkTask {
	t.Helper()
	return chatwork.WorkTask{
		Ref:        relationshipReference(t, chatwork.ReferenceTask, refValue),
		Room:       chatwork.Room{Ref: room},
		Account:    chatwork.Account{Ref: relationshipReference(t, chatwork.ReferenceAccount, "8"), Name: "external"},
		AssignedBy: chatwork.Account{Ref: relationshipReference(t, chatwork.ReferenceAccount, "9"), Name: "external"},
		Message:    message, Body: body, Status: "open",
	}
}

type messageResourcesContextKey struct{}
