package cli

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestRunMessagesListRejectsInvalidMaxBodyBytesBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	for _, value := range []string{"-1", "+1", "65536", "18446744073709551616", "many", ""} {
		t.Run(strconv.Quote(value), func(t *testing.T) {
			port := &chatworkRuntimePort{}
			cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
			code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
				"--room", "7", "--max-body-bytes", value,
			})
			if code != ExitUsage || !strings.Contains(stderr.String(), "code: invalid_arguments") {
				t.Fatalf("value=%q code=%d stderr=%s", value, code, stderr.String())
			}
			if value == "" && !strings.Contains(stderr.String(), "空でない値") {
				t.Fatalf("empty value error = %s", stderr.String())
			}
			if value != "" && !strings.Contains(stderr.String(), "0 から 65535") {
				t.Fatalf("value=%q did not state the allowed range: %s", value, stderr.String())
			}
			if authenticator.calls != 0 || port.calls != 0 || stdout.Len() != 0 {
				t.Fatalf("value=%q auth=%d provider=%d stdout=%q", value, authenticator.calls, port.calls, stdout.String())
			}
		})
	}
}

func TestRunMessagesListMaxBodyBytesKeepsSelectionAndReplyContextAndCombinesWithMaxBytes(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	account := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "9"), Name: "Synthetic"}
	parent := chatwork.Message{
		Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, "10"), Room: room,
		Sender: account, Body: "parent-context", SendTime: 10,
	}
	anchor := chatwork.Message{
		Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, "11"), Room: room,
		Sender: account, Body: "needle\\\n\u202e selection body", SendTime: 20,
		Replies: []chatwork.Relation{{Kind: "reply", Target: parent.Ref, ExternalID: room.Value}},
	}
	anchorBodyBytes := len(anchor.Body)
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task != chatwork.TaskMessagesList || !request.ForceRecent ||
			request.MessageRelationFetchLimit != 0 || !reflect.DeepEqual(request.MessageFilter, chatwork.MessageFilter{}) {
			t.Fatalf("presentation flag or selection changed provider request: %+v", request)
		}
		return chatwork.Result{
			Task: chatwork.TaskMessagesList, MessageRoom: request.Room,
			Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
			Messages: []chatwork.Message{parent, anchor},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	const budget = 4096
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", room.Value, "--contains", "needle", "--resolve-relations", "0",
		"--max-body-bytes", "1", "--max-bytes", strconv.Itoa(budget),
	})
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("auth/provider calls=%d/%d", authenticator.calls, port.calls)
	}
	got := stdout.String()
	if len(got) > budget {
		t.Fatalf("escaped output bytes=%d exceed max-bytes=%d", len(got), budget)
	}
	for _, want := range []string{
		"complete=false",
		"candidate-count=1",
		"contains=[\"needle\"] context=replies anchors=[#2]",
		"#1 10 a1 10",
		"#2 11 a1 20 reply=#1",
		"body-truncated=true original-body-bytes=14",
		"body-truncated=true original-body-bytes=" + strconv.Itoa(anchorBodyBytes),
		"delivery max-bytes=4096 output-complete=true output-items=1 output-total-items=1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("body-limited result lacks %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, `body-truncated=true original-body-bytes=14 "p"`) ||
		!strings.Contains(got, "body-truncated=true original-body-bytes="+strconv.Itoa(anchorBodyBytes)+` "n"`) {
		t.Fatalf("body prefix was not limited before escaping:\n%s", got)
	}
	if !strings.HasSuffix(got, "delivery max-bytes=4096 output-complete=true output-items=1 output-total-items=1\n") {
		t.Fatalf("delivery metadata is not the final line:\n%s", got)
	}
}

func TestSplitChatworkMaxBodyBytesDoesNotEnterApplicationRequest(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	parsed, err := parseChatworkArguments(spec, []string{"--room", "7", "--max-body-bytes", "0"})
	if err != nil {
		t.Fatal(err)
	}
	requestArguments, options, err := splitChatworkPresentationArguments(spec, parsed)
	if err != nil {
		t.Fatal(err)
	}
	if options.MaxBodyBytes != 0 || !options.LimitMessageBodies {
		t.Fatalf("zero body limit was not retained as an explicit presentation choice: %+v", options)
	}
	if _, exists := requestArguments["--max-body-bytes"]; exists {
		t.Fatalf("presentation flag leaked to request argument map: %+v", requestArguments)
	}
	request, err := buildChatworkRequest(spec, requestArguments, nil)
	if err != nil {
		t.Fatal(err)
	}
	if request.Room.Value != "7" || request.Task != chatwork.TaskMessagesList {
		t.Fatalf("request = %+v", request)
	}
}
