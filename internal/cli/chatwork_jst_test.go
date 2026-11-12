package cli

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestMessagesJSTCatalogAndActiveView(t *testing.T) {
	catalog := DefaultCatalog()
	if err := catalog.Validate(); err != nil {
		t.Fatalf("DefaultCatalog().Validate() error = %v", err)
	}
	for _, path := range []string{"messages list", "messages show"} {
		command, found := catalog.Lookup(path)
		if !found {
			t.Fatalf("catalog is missing %q", path)
		}
		if !strings.Contains(command.Usage(), "[--jst]") {
			t.Errorf("%s usage does not expose --jst: %q", path, command.Usage())
		}
		var jstInput bool
		for _, input := range command.Agent.Inputs {
			if input.Name == "--jst" {
				jstInput = input.Source == InputSourceFlag && !input.Required && len(input.AllowedValues) == 0
			}
		}
		if !jstInput {
			t.Errorf("%s does not declare --jst as an optional valueless flag", path)
		}
		var jstField *OutputField
		for index := range command.Agent.Output.Fields {
			if command.Agent.Output.Fields[index].Name == "send_time_jst" {
				jstField = &command.Agent.Output.Fields[index]
				break
			}
		}
		if jstField == nil || jstField.Type != OutputFieldTypeString ||
			!strings.Contains(jstField.Description, "--jst") || !strings.Contains(jstField.Description, "unavailable") {
			t.Errorf("%s send_time_jst field = %+v", path, jstField)
		}
		help := renderCommandHelp(command, catalog.commands)
		if len(help) > maxExactHelpBytes {
			t.Errorf("%s exact help is %d bytes, above %d", path, len(help), maxExactHelpBytes)
		}
		if !strings.Contains(string(help), "--jst boolean 任意: +09:00日時を併記。") {
			t.Errorf("%s exact help omits the --jst input description:\n%s", path, help)
		}
	}
	active, _, err := catalog.ActiveView([]string{"rooms list", "messages list", "messages show"})
	if err != nil {
		t.Fatalf("active view with message producers: %v", err)
	}
	for _, path := range []string{"messages list", "messages show"} {
		if _, found := active.Lookup(path); !found {
			t.Errorf("active view is missing %q", path)
		}
	}
}

func TestSplitMessagesJSTBeforeTypedRequest(t *testing.T) {
	for _, path := range []string{"messages list", "messages show"} {
		t.Run(path, func(t *testing.T) {
			spec := chatworkRuntimeSpec(t, path)
			args := []string{"--room", "7", "--jst"}
			if path == "messages show" {
				args = append(args, "--message", "10")
			}
			parsed, err := parseChatworkArguments(spec, args)
			if err != nil {
				t.Fatalf("parseChatworkArguments() error = %v", err)
			}
			requestArguments, options, err := splitChatworkPresentationArguments(spec, parsed)
			if err != nil {
				t.Fatalf("splitChatworkPresentationArguments() error = %v", err)
			}
			if !options.IncludeSendTimeJST {
				t.Fatal("--jst was not retained in presentation options")
			}
			if _, found := requestArguments["--jst"]; found {
				t.Fatalf("presentation option leaked into request arguments: %+v", requestArguments)
			}
			request, err := buildChatworkRequest(spec, requestArguments, nil)
			if err != nil {
				t.Fatalf("buildChatworkRequest() error = %v", err)
			}
			if request.Task != spec.chatwork.Task || request.Room.Value != "7" || !reflect.DeepEqual(request.MessageFilter, chatwork.MessageFilter{}) {
				t.Fatalf("--jst changed the typed task request: %+v", request)
			}
		})
	}
}

func TestRunMessagesListJSTIsPresentationOnly(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	account := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "9"), Room: room, Name: "Synthetic"}
	message := chatwork.Message{
		Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, "10"), Room: room, Sender: account,
		Body: "message", SendTime: 1700000000,
	}
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task != chatwork.TaskMessagesList || request.Room != room || !reflect.DeepEqual(request.MessageFilter, chatwork.MessageFilter{}) {
			t.Fatalf("JST option changed the provider request: %+v", request)
		}
		return chatwork.Result{
			Task: chatwork.TaskMessagesList, MessageRoom: room,
			Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
			Messages: []chatwork.Message{message},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", room.Value, "--jst"})
	if code != ExitOK {
		t.Fatalf("runChatwork() code=%d stderr=%s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("auth/provider calls=%d/%d, want one each", authenticator.calls, port.calls)
	}
	for _, want := range []string{
		"schema: #sequence message-ref actor sent [send-time-jst]",
		"#1 10 a1 1700000000 send-time-jst=2023-11-15T07:13:20+09:00",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("list output lacks %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunMessagesShowJSTIsPresentationOnly(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages show")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	message := chatwork.Message{
		Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, "10"), Room: room,
		Sender: chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "9"), Room: room, Name: "Synthetic"},
		Body:   "anchor", SendTime: 1700000000,
	}
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task != chatwork.TaskMessagesShow || request.Room != room || request.Message != message.Ref {
			t.Fatalf("JST option changed the provider request: %+v", request)
		}
		return chatwork.Result{Task: chatwork.TaskMessagesShow, Messages: []chatwork.Message{message}}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", room.Value, "--message", message.Ref.Value, "--jst"})
	if code != ExitOK {
		t.Fatalf("runChatwork() code=%d stderr=%s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("auth/provider calls=%d/%d, want one each", authenticator.calls, port.calls)
	}
	if !strings.Contains(stdout.String(), "send-time=1700000000 send-time-jst=2023-11-15T07:13:20+09:00") {
		t.Fatalf("show output lacks the separate JST field or original Unix second:\n%s", stdout.String())
	}
}

func TestRunChatworkRejectsJSTValueBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	port := &chatworkRuntimePort{}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", "7", "--jst=false"})
	if code != ExitUsage || !strings.Contains(stderr.String(), "--jst は値を受け付けません") {
		t.Fatalf("invalid --jst value code=%d stderr=%s", code, stderr.String())
	}
	if authenticator.calls != 0 || port.calls != 0 || stdout.Len() != 0 {
		t.Fatalf("invalid --jst value auth=%d provider=%d stdout=%q", authenticator.calls, port.calls, stdout.String())
	}
}

func TestRunMessagesListJSTCombinesWithBodyAndOutputBudgets(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	account := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "9"), Room: room, Name: "Synthetic"}
	messages := make([]chatwork.Message, 3)
	for index := range messages {
		messages[index] = chatwork.Message{
			Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, fmt.Sprint(index+10)), Room: room,
			Sender: account, Body: strings.Repeat("B", 300), SendTime: 1700000000 + int64(index),
		}
	}
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task != chatwork.TaskMessagesList || request.Room != room || !reflect.DeepEqual(request.MessageFilter, chatwork.MessageFilter{}) {
			t.Fatalf("presentation options changed the provider request: %+v", request)
		}
		return chatwork.Result{
			Task: chatwork.TaskMessagesList, MessageRoom: room,
			Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false}, Messages: messages,
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	const budget = 1050
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", room.Value, "--jst", "--max-body-bytes", "64", "--max-bytes", fmt.Sprint(budget),
	})
	if code != ExitOK {
		t.Fatalf("runChatwork() code=%d stderr=%s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("auth/provider calls=%d/%d, want one each", authenticator.calls, port.calls)
	}
	output := stdout.String()
	if len(output) > budget {
		t.Fatalf("output bytes=%d exceeds budget=%d", len(output), budget)
	}
	if !strings.Contains(output, "[send-time-jst]") || !strings.Contains(output, "body-truncated=true original-body-bytes=300") {
		t.Fatalf("budgeted projection lost JST schema or body-limit metadata:\n%s", output)
	}
	if strings.Count(output, "send-time-jst=") == 0 || !strings.Contains(output, "delivery max-bytes=1050") || !strings.Contains(output, "output-total-items=3") {
		t.Fatalf("budget redraw omitted JST timestamps or delivery facts:\n%s", output)
	}
	if strings.Count(output, "send-time-jst=") >= len(messages) {
		t.Fatalf("budget did not redraw with an atomic prefix of message rows:\n%s", output)
	}
	if strings.Contains(output, "--jst") {
		t.Fatalf("presentation input leaked into output:\n%s", output)
	}
}
