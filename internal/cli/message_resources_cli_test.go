package cli

import (
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

func TestMessagesResourcesCLIUsesExactReferencesAndPrintsBoundedSources(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages resources")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "42")
	message := chatworkRuntimeRef(t, chatwork.ReferenceMessage, "100")
	account := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "7"), Name: "hostile uploader\nname"}
	file := chatwork.File{Ref: chatworkRuntimeRef(t, chatwork.ReferenceFile, "501"), Room: room, Account: account, Message: message, Name: "hostile file\x1b"}
	task := chatwork.WorkTask{
		Ref: chatworkRuntimeRef(t, chatwork.ReferenceTask, "601"), Room: chatwork.Room{Ref: room}, Account: account,
		AssignedBy: chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "8"), Name: "external"},
		Message:    message, Body: "[download] hostile body",
	}
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		switch request.Task {
		case chatwork.TaskFilesList:
			if request.Room != room || request.Message != (chatwork.Reference{}) || request.Account != (chatwork.Reference{}) {
				t.Fatalf("files source request = %+v", request)
			}
			return chatwork.Result{Task: request.Task, Coverage: resourceCoverage(), Files: []chatwork.File{file}}, nil
		case chatwork.TaskRoomTasksList:
			if request.Room != room || request.Message != (chatwork.Reference{}) || request.Status != "" {
				t.Fatalf("room-task source request = %+v", request)
			}
			return chatwork.Result{Task: request.Task, Coverage: resourceCoverage(), Tasks: []chatwork.WorkTask{task}}, nil
		default:
			t.Fatalf("unexpected provider task %s", request.Task)
			return chatwork.Result{}, nil
		}
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", room.Value, "--message", message.Value, "--kind", "all"})
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 2 || len(port.requests) != 2 ||
		port.requests[0].Task != chatwork.TaskFilesList || port.requests[1].Task != chatwork.TaskRoomTasksList {
		t.Fatalf("auth/provider calls = %d/%d requests=%+v", authenticator.calls, port.calls, port.requests)
	}
	want := "message-resources room-ref=42 message-ref=100 kind=all\n" +
		"files source-count=1 limit=100 complete=false absence=unknown matched=1\n  file-ref=501\n" +
		"room-tasks source-count=1 limit=100 complete=false absence=unknown matched=1\n  task-ref=601\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	for _, forbidden := range []string{"hostile", "download", "body", "messages show", "event"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Errorf("resource output leaked %q: %s", forbidden, stdout.String())
		}
	}
	if _, err := chatwork.NewReference(chatwork.ReferenceFile, "501"); err != nil {
		t.Fatalf("file reference is not reusable: %v", err)
	}
	if _, err := chatwork.NewReference(chatwork.ReferenceTask, "601"); err != nil {
		t.Fatalf("task reference is not reusable: %v", err)
	}
}

func TestMessagesResourcesCLISecondSourceFaultProducesNoPartialStdout(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages resources")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "42")
	message := chatworkRuntimeRef(t, chatwork.ReferenceMessage, "100")
	file := chatwork.File{Ref: chatworkRuntimeRef(t, chatwork.ReferenceFile, "501"), Room: room,
		Account: chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "7")}, Message: message}
	calls := 0
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		calls++
		if request.Task == chatwork.TaskFilesList {
			return chatwork.Result{Task: request.Task, Coverage: resourceCoverage(), Files: []chatwork.File{file}}, nil
		}
		return chatwork.Result{}, fault.New(fault.KindPermission, "chatwork_permission_denied", "permission denied", false)
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", room.Value, "--message", message.Value, "--kind", "all"})
	if code != ExitPermission || stdout.Len() != 0 || authenticator.calls != 1 || port.calls != 2 || calls != 2 ||
		!strings.Contains(stderr.String(), "code: chatwork_permission_denied") {
		t.Fatalf("exit/auth/provider/stdout/stderr = %d/%d/%d/%q/%s", code, authenticator.calls, port.calls, stdout.String(), stderr.String())
	}
}

func TestMessagesResourcesCLIRejectsUnsupportedKindBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages resources")
	port := &chatworkRuntimePort{}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", "42", "--message", "100", "--kind", "downloads"})
	if code != ExitUsage || authenticator.calls != 0 || port.calls != 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "--kind は files、tasks、all") {
		t.Fatalf("exit/auth/provider/output = %d/%d/%d/%q/%s", code, authenticator.calls, port.calls, stdout.String(), stderr.String())
	}
}

func resourceCoverage() chatwork.Coverage {
	return chatwork.Coverage{Kind: "provider_window", Limit: 100, Complete: false, Description: "bounded source"}
}
