package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/cli/capsule"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

type messageResourcesExpectedSequencePort struct {
	fixture  activeMessageResourcesFixture
	requests []chatwork.Request
	contexts []context.Context
	bindings []authn.BindingID
}

func (p *messageResourcesExpectedSequencePort) Execute(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.requests = append(p.requests, request)
	p.contexts = append(p.contexts, ctx)
	p.bindings = append(p.bindings, binding)
	switch request.Task {
	case chatwork.TaskFilesList:
		return p.fixture.FilesSource, nil
	case chatwork.TaskRoomTasksList:
		return p.fixture.TasksSource, nil
	case chatwork.TaskFilesShow:
		return p.fixture.FileShow, nil
	case chatwork.TaskRoomTasksShow:
		return p.fixture.TaskShow, nil
	default:
		return chatwork.Result{}, errUnexpectedMessageResourceExpectedSequenceTask(request.Task)
	}
}

func TestActiveMessageResourcesFixtureHasPresentationIndependentAnswerKey(t *testing.T) {
	fixture := messageResourcesFixture()
	if err := fixture.FilesSource.ValidateFor(chatwork.Request{Task: chatwork.TaskFilesList, Room: fixture.Room}); err != nil {
		t.Fatalf("file source fixture: %v", err)
	}
	if err := fixture.TasksSource.ValidateFor(chatwork.Request{Task: chatwork.TaskRoomTasksList, Room: fixture.Room}); err != nil {
		t.Fatalf("room-task source fixture: %v", err)
	}
	if err := fixture.Resources.ValidateFor(chatwork.Request{
		Task: chatwork.TaskMessagesResources, Room: fixture.Room, Message: fixture.Message, MessageResourcesKind: chatwork.MessageResourceAll,
	}); err != nil {
		t.Fatalf("resource fixture: %v", err)
	}
	if !json.Valid(fixture.AnswerKey) {
		t.Fatalf("answer key is invalid JSON: %s", fixture.AnswerKey)
	}
	var answer struct {
		RoomRef    string `json:"room_ref"`
		MessageRef string `json:"message_ref"`
		Kind       string `json:"kind"`
		Files      struct {
			SourceCount int      `json:"source_count"`
			Limit       int      `json:"limit"`
			Complete    bool     `json:"complete"`
			Absence     string   `json:"absence"`
			Matched     int      `json:"matched"`
			Refs        []string `json:"file_refs"`
		} `json:"files"`
		Tasks struct {
			SourceCount int      `json:"source_count"`
			Limit       int      `json:"limit"`
			Complete    bool     `json:"complete"`
			Absence     string   `json:"absence"`
			Matched     int      `json:"matched"`
			Refs        []string `json:"task_refs"`
		} `json:"tasks"`
		Verified struct {
			FileShow struct {
				RoomRef    string `json:"room_ref"`
				FileRef    string `json:"file_ref"`
				MessageRef string `json:"message_ref"`
				Name       string `json:"name"`
			} `json:"file_show"`
			RoomTaskShow struct {
				RoomRef    string `json:"room_ref"`
				TaskRef    string `json:"task_ref"`
				MessageRef string `json:"message_ref"`
				Body       string `json:"body"`
				Status     string `json:"status"`
			} `json:"room_task_show"`
		} `json:"verified"`
	}
	if err := json.Unmarshal(fixture.AnswerKey, &answer); err != nil {
		t.Fatal(err)
	}
	resources := fixture.Resources.MessageResources
	if answer.RoomRef != fixture.Room.Value || answer.MessageRef != fixture.Message.Value || answer.Kind != "all" ||
		answer.Files.SourceCount != len(fixture.FilesSource.Files) || answer.Files.Limit != 100 || answer.Files.Complete || answer.Files.Absence != "unknown" || answer.Files.Matched != 1 ||
		!reflect.DeepEqual(answer.Files.Refs, []string{resources.Files.Matches[0].Ref.Value}) ||
		answer.Tasks.SourceCount != len(fixture.TasksSource.Tasks) || answer.Tasks.Limit != 100 || answer.Tasks.Complete || answer.Tasks.Absence != "unknown" || answer.Tasks.Matched != 1 ||
		!reflect.DeepEqual(answer.Tasks.Refs, []string{resources.Tasks.Matches[0].Ref.Value}) ||
		answer.Verified.FileShow.RoomRef != fixture.Room.Value || answer.Verified.FileShow.FileRef != fixture.FileShow.Files[0].Ref.Value || answer.Verified.FileShow.MessageRef != fixture.Message.Value || answer.Verified.FileShow.Name != fixture.FileShow.Files[0].Name ||
		answer.Verified.RoomTaskShow.RoomRef != fixture.Room.Value || answer.Verified.RoomTaskShow.TaskRef != fixture.TaskShow.Tasks[0].Ref.Value || answer.Verified.RoomTaskShow.MessageRef != fixture.Message.Value || answer.Verified.RoomTaskShow.Body != fixture.TaskShow.Tasks[0].Body || answer.Verified.RoomTaskShow.Status != fixture.TaskShow.Tasks[0].Status {
		t.Fatalf("message resources answer key = %+v", answer)
	}
	for _, field := range []string{"expected_command_sequence", "provider_calls", "external_processing_calls", "write_calls"} {
		if strings.Contains(string(fixture.AnswerKey), `"`+field+`"`) {
			t.Errorf("semantic answer key must not claim evaluator-observed field %q: %s", field, fixture.AnswerKey)
		}
	}
	for _, reference := range []chatwork.Reference{resources.Files.Matches[0].Ref, resources.Tasks.Matches[0].Ref} {
		if err := chatwork.ValidateReference(reference.Kind, reference.Value); err != nil {
			t.Fatalf("resource reference %q is not canonical: %v", reference.Value, err)
		}
	}
}

func TestActiveMessageResourcesExpectedSequenceReusesRefsAndVerifiesBothShowResults(t *testing.T) {
	fixture := messageResourcesFixture()
	scenario := activeMessageResourcesScenario()
	if scenario.ID != "active.message-resources" || scenario.ProviderCallBudget != 4 || scenario.ExternalProcessingBudget != 0 || scenario.WriteCallBudget != 0 ||
		len(scenario.ExpectedCommandArgv) != 3 || len(scenario.Operations) != 3 || !json.Valid(scenario.AnswerKey) {
		t.Fatalf("resource expected sequence is not fixture-complete: %+v", scenario)
	}
	for _, forbidden := range []string{"jq", "grep", "external parser", "source inspection", "caller-side join", "Do not infer resource events", "claim the message exists"} {
		if !strings.Contains(scenario.UserPrompt, forbidden) {
			t.Errorf("scenario prompt does not close %q: %q", forbidden, scenario.UserPrompt)
		}
	}
	for _, leakedCommand := range []string{"messages resources", "files show", "room-tasks show", "Use exactly"} {
		if strings.Contains(scenario.UserPrompt, leakedCommand) {
			t.Errorf("scenario prompt reveals command path or fixed sequence %q: %q", leakedCommand, scenario.UserPrompt)
		}
	}
	for path, operation := range scenario.Operations {
		if operation.Path != path {
			t.Errorf("fixture operation path for %q = %q", path, operation.Path)
		}
	}
	wantSequence := [][]string{
		{"messages", "resources", "--room", fixture.Room.Value, "--message", fixture.Message.Value, "--kind", "all"},
		{"files", "show", "--room", fixture.Room.Value, "--file", fixture.Resources.MessageResources.Files.Matches[0].Ref.Value},
		{"room-tasks", "show", "--room", fixture.Room.Value, "--task", fixture.Resources.MessageResources.Tasks.Matches[0].Ref.Value},
	}
	if !reflect.DeepEqual(scenario.ExpectedCommandArgv, wantSequence) {
		t.Fatalf("fixture expected command arguments = %v, want exact resource refs: %v", scenario.ExpectedCommandArgv, wantSequence)
	}
	wantOperationArgs := map[string]map[string]string{
		"messages resources": {"--room": fixture.Room.Value, "--message": fixture.Message.Value, "--kind": "all"},
		"files show":         {"--room": fixture.Room.Value, "--file": fixture.Resources.MessageResources.Files.Matches[0].Ref.Value},
		"room-tasks show":    {"--room": fixture.Room.Value, "--task": fixture.Resources.MessageResources.Tasks.Matches[0].Ref.Value},
	}
	for path, wantArgs := range wantOperationArgs {
		operation := scenario.Operations[path]
		if !reflect.DeepEqual(operation.RequiredArgs, wantArgs) {
			t.Errorf("fixture operation %q args = %v, want %v", path, operation.RequiredArgs, wantArgs)
		}
	}

	ctx := context.WithValue(context.Background(), messageResourcesExpectedSequenceContextKey{}, "same readiness context")
	binding, err := authn.NewBindingID("active-message-resource-binding")
	if err != nil {
		t.Fatal(err)
	}
	port := &messageResourcesExpectedSequencePort{fixture: fixture}
	service := chatworkcmd.New(port)
	resourcesResult, err := service.Execute(ctx, binding, chatwork.Request{
		Task: chatwork.TaskMessagesResources, Room: fixture.Room, Message: fixture.Message, MessageResourcesKind: chatwork.MessageResourceAll,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resourcesResult.MessageResources, fixture.Resources.MessageResources) {
		t.Fatalf("resource task result = %+v, want fixture links %+v", resourcesResult.MessageResources, fixture.Resources.MessageResources)
	}
	resourcesOutput, err := capsule.Render(resourcesResult)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"file-ref=7711", "task-ref=8811", "source-count=3 limit=100 complete=false absence=unknown matched=1"} {
		if !strings.Contains(resourcesOutput, required) {
			t.Errorf("resource projection lacks %q:\n%s", required, resourcesOutput)
		}
	}

	fileArgs := scenario.ExpectedCommandArgv[1]
	fileRequest := chatwork.Request{
		Task: chatwork.TaskFilesShow,
		Room: ref(chatwork.ReferenceRoom, expectedSequenceFlag(fileArgs, "--room")),
		File: ref(chatwork.ReferenceFile, expectedSequenceFlag(fileArgs, "--file")),
	}
	if fileRequest.File != resourcesResult.MessageResources.Files.Matches[0].Ref {
		t.Fatalf("files show rewrote file reference: got %+v, resources returned %+v", fileRequest.File, resourcesResult.MessageResources.Files.Matches[0].Ref)
	}
	fileShown, err := service.Execute(ctx, binding, fileRequest)
	if err != nil {
		t.Fatalf("exact files show expected-sequence call: %v", err)
	}
	if err := fileShown.ValidateFor(fileRequest); err != nil || fileShown.Files[0].Name != "release-notes.txt" || fileShown.Files[0].Message != fixture.Message {
		t.Fatalf("file show did not verify the linked resource: result=%+v err=%v", fileShown.Files, err)
	}
	fileOutput, err := capsule.Render(fileShown)
	if err != nil || !strings.Contains(fileOutput, "file file-ref=7711 room-ref=4201") || !strings.Contains(fileOutput, `name=untrusted:"release-notes.txt"`) {
		t.Fatalf("file show projection did not confirm the canonical target: output=%s err=%v", fileOutput, err)
	}

	taskArgs := scenario.ExpectedCommandArgv[2]
	taskRequest := chatwork.Request{
		Task:    chatwork.TaskRoomTasksShow,
		Room:    ref(chatwork.ReferenceRoom, expectedSequenceFlag(taskArgs, "--room")),
		TaskRef: ref(chatwork.ReferenceTask, expectedSequenceFlag(taskArgs, "--task")),
	}
	if taskRequest.TaskRef != resourcesResult.MessageResources.Tasks.Matches[0].Ref {
		t.Fatalf("room-tasks show rewrote task reference: got %+v, resources returned %+v", taskRequest.TaskRef, resourcesResult.MessageResources.Tasks.Matches[0].Ref)
	}
	taskShown, err := service.Execute(ctx, binding, taskRequest)
	if err != nil {
		t.Fatalf("exact room-tasks show expected-sequence call: %v", err)
	}
	if err := taskShown.ValidateFor(taskRequest); err != nil || taskShown.Tasks[0].Body != "Review the linked release notes." || taskShown.Tasks[0].Message != fixture.Message {
		t.Fatalf("room-task show did not verify the linked resource: result=%+v err=%v", taskShown.Tasks, err)
	}
	taskOutput, err := capsule.Render(taskShown)
	if err != nil || !strings.Contains(taskOutput, "room-task task-ref=8811") || !strings.Contains(taskOutput, "Review the linked release notes.") {
		t.Fatalf("room-task show projection did not confirm the canonical target: output=%s err=%v", taskOutput, err)
	}

	wantTasks := []chatwork.Task{chatwork.TaskFilesList, chatwork.TaskRoomTasksList, chatwork.TaskFilesShow, chatwork.TaskRoomTasksShow}
	if len(port.requests) != len(wantTasks) {
		t.Fatalf("internal provider call count = %d, want %d: %+v", len(port.requests), len(wantTasks), port.requests)
	}
	for index, request := range port.requests {
		if request.Task != wantTasks[index] || port.contexts[index] != ctx || port.bindings[index] != binding {
			t.Errorf("expected-sequence provider request %d lost order/context/binding: request=%+v context=%p binding=%s", index, request, port.contexts[index], port.bindings[index])
		}
		if request.Task != chatwork.TaskFilesList && request.Task != chatwork.TaskRoomTasksList &&
			request.Task != chatwork.TaskFilesShow && request.Task != chatwork.TaskRoomTasksShow {
			t.Errorf("expected sequence contains a non-read task: %s", request.Task)
		}
	}
}

type messageResourcesExpectedSequenceContextKey struct{}

func expectedSequenceFlag(args []string, name string) string {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == name {
			return args[index+1]
		}
	}
	return ""
}

func errUnexpectedMessageResourceExpectedSequenceTask(task chatwork.Task) error {
	return fmt.Errorf("unexpected resource expected-sequence task: %s", task)
}
