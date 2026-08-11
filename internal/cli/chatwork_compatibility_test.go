package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestMessagesListDeprecatedLimitAliasNormalizesToCount(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	for _, args := range [][]string{
		{"--room", "7", "--limit", "10"},
		{"--room", "7", "--limit=10"},
	} {
		parsed, deprecated, err := parseChatworkInvocationArguments(spec, args)
		if err != nil {
			t.Fatalf("parseChatworkInvocationArguments(%v) error = %v", args, err)
		}
		if got := parsed.first("--count"); got != "10" || parsed.first("--limit") != "" {
			t.Fatalf("parsed %v = %+v", args, parsed)
		}
		want := []deprecatedFlagUse{{Alias: "--limit", Canonical: "--count"}}
		if !reflect.DeepEqual(deprecated, want) {
			t.Fatalf("deprecated flags = %+v, want %+v", deprecated, want)
		}
		request, err := buildChatworkRequest(spec, parsed, nil)
		if err != nil {
			t.Fatal(err)
		}
		if request.MessageFilter.StartIndex != 1 || request.MessageFilter.Count != 10 {
			t.Fatalf("message selection = %+v", request.MessageFilter)
		}
	}
}

func TestMessagesListDeprecatedLimitAliasCannotDuplicateCount(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	for _, args := range [][]string{
		{"--room", "7", "--limit", "10", "--count", "20"},
		{"--room", "7", "--count", "10", "--limit", "20"},
	} {
		if _, _, err := parseChatworkInvocationArguments(spec, args); err == nil || !strings.Contains(err.Error(), "--count は1回だけ") {
			t.Fatalf("parseChatworkInvocationArguments(%v) error = %v", args, err)
		}
	}
}

func TestMessagesListDeprecatedLimitAliasRetainsCountValidation(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	for _, value := range []string{"0", "101", "ten"} {
		parsed, _, err := parseChatworkInvocationArguments(spec, []string{"--room", "7", "--limit", value})
		if err != nil {
			t.Fatalf("alias parse %q error = %v", value, err)
		}
		if _, err := buildChatworkRequest(spec, parsed, nil); err == nil || !strings.Contains(err.Error(), "--count") {
			t.Fatalf("alias value %q build error = %v", value, err)
		}
	}
}

func TestRoomTaskLimitRemainsCanonicalDeadlineWithoutDeprecation(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "room-tasks create")
	parsed, deprecated, err := parseChatworkInvocationArguments(spec, []string{
		"--room", "7", "--body", "task", "--assignee", "1", "--limit", "1700000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(deprecated) != 0 {
		t.Fatalf("deprecated flags = %+v", deprecated)
	}
	request, err := buildChatworkRequest(spec, parsed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if request.Limit != 1700000000 || request.MessageFilter.Count != 0 {
		t.Fatalf("room task request = %+v", request)
	}
}

func TestRunChatworkDeprecatedLimitAliasSucceedsWithMigrationNotice(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{
			Task:        request.Task,
			MessageRoom: request.Room,
			Coverage:    chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
			Messages:    []chatwork.Message{},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "7", "--limit", "10",
	})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls/request = auth %d provider %d %+v", authenticator.calls, port.calls, port.request)
	}
	if got := stderr.String(); got != "warning: --limit は --count に改名されました\n" {
		t.Fatalf("stderr = %q", got)
	}
	if !strings.Contains(stdout.String(), "start-index=1 count=10") {
		t.Fatalf("successful alias output lacks normalized selection:\n%s", stdout.String())
	}
}

func TestRoomsListQueryBindsDiscoveryWithoutChangingExactRoomInputs(t *testing.T) {
	rooms := chatworkRuntimeSpec(t, "rooms list")
	parsed, err := parseChatworkArguments(rooms, []string{"--query", "コスト"})
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildChatworkRequest(rooms, parsed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if request.RoomQuery != "コスト" || request.MemberQuery != "" || request.Room.Value != "" {
		t.Fatalf("rooms list query request = %+v", request)
	}

	show := chatworkRuntimeSpec(t, "rooms show")
	if _, err := parseChatworkArguments(show, []string{"--room", "rid442804132"}); err == nil ||
		!strings.Contains(err.Error(), "1〜32桁の正の10進 room_id") ||
		!strings.Contains(err.Error(), "rooms list --query") {
		t.Fatalf("non-canonical act reference error = %v", err)
	}
}

func TestRunChatworkRoomsListQueryReturnsCanonicalCandidatesAfterOneRead(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "rooms list")
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{
			Task:     request.Task,
			Coverage: chatwork.Coverage{Kind: "provider_collection", Complete: true},
			Rooms: []chatwork.Room{
				{Ref: chatworkRuntimeRef(t, chatwork.ReferenceRoom, "41"), Name: "セキュリティ定例", Type: "group", Role: "member"},
				{Ref: chatworkRuntimeRef(t, chatwork.ReferenceRoom, "42"), Name: "コスト管理", Type: "group", Role: "admin"},
			},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--query", "コスト"})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 || port.request.RoomQuery != "" {
		t.Fatalf("calls/provider request = auth %d provider %d %+v", authenticator.calls, port.calls, port.request)
	}
	for _, want := range []string{
		`room-candidates query="コスト" source-count=2 candidate-count=1 complete=true`,
		`42 "コスト管理"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("room candidate output lacks %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "41") || strings.Contains(stdout.String(), "セキュリティ") {
		t.Fatalf("room candidate output leaked non-match:\n%s", stdout.String())
	}
}

func TestChatworkUnknownFlagSuggestsNearestDeclaredCanonicalName(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	_, err := parseChatworkArguments(spec, []string{"--room", "7", "--coutn", "10"})
	if err == nil || !strings.Contains(err.Error(), `もしかして "--count"`) {
		t.Fatalf("unknown flag error = %v", err)
	}
}
