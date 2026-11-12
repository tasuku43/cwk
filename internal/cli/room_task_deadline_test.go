package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestParseRoomTaskDeadlinePreservesUnixSecondsAndNormalizesRFC3339(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int64
	}{
		{name: "unix seconds", value: "1700000000", want: 1700000000},
		{name: "unix seconds with explicit plus", value: "+1700000000", want: 1700000000},
		{name: "unix seconds with legacy leading zero", value: "01700000000", want: 1700000000},
		{name: "whole second offset", value: "2026-11-08T12:00:00+09:00", want: time.Date(2026, 11, 8, 3, 0, 0, 0, time.UTC).Unix()},
		{name: "UTC spelling of same instant", value: "2026-11-08T03:00:00Z", want: time.Date(2026, 11, 8, 3, 0, 0, 0, time.UTC).Unix()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseRoomTaskDeadline(test.value)
			if err != nil {
				t.Fatalf("parseRoomTaskDeadline(%q) error = %v", test.value, err)
			}
			if got != test.want {
				t.Fatalf("parseRoomTaskDeadline(%q) = %d, want %d", test.value, got, test.want)
			}
		})
	}

	offset, _ := parseRoomTaskDeadline("2026-11-08T12:00:00+09:00")
	utc, _ := parseRoomTaskDeadline("2026-11-08T03:00:00Z")
	if offset != utc {
		t.Fatalf("equivalent instants differ: offset=%d UTC=%d", offset, utc)
	}
}

func TestParseRoomTaskDeadlineRejectsInvalidOrAmbiguousValues(t *testing.T) {
	for _, value := range []string{
		"0",
		"-1",
		"9223372036854775808",
		"1970-01-01T00:00:00Z",
		"1969-12-31T23:59:59Z",
		"2026-11-08",
		"2026-11-08T12:00:00",
		"2026-11-08T12:00:00.1+09:00",
		"2026-11-08T12:00:00.000Z",
		"2026-11-08T12:00:00-00:00",
		"2026-02-30T12:00:00Z",
		"2026-11-08T12:00:00+24:00",
		"2026-11-08T12:00:00+09:60",
		"2026-11-08T12:00:00+0900",
		"2026-11-08t12:00:00z",
	} {
		t.Run(value, func(t *testing.T) {
			if got, err := parseRoomTaskDeadline(value); err == nil {
				t.Fatalf("parseRoomTaskDeadline(%q) = %d, want error", value, got)
			}
		})
	}
}

func TestRoomTaskDeadlineCatalogDeclaresBothFormsInBoundedJapaneseHelp(t *testing.T) {
	command := chatworkRuntimeSpec(t, "room-tasks create")
	var limitInput *CommandInput
	for index := range command.Agent.Inputs {
		if command.Agent.Inputs[index].Name == "--limit" {
			limitInput = &command.Agent.Inputs[index]
			break
		}
	}
	if limitInput == nil {
		t.Fatal("room-tasks create does not declare --limit")
	}
	if !strings.Contains(limitInput.Description, "Unix秒") || !strings.Contains(limitInput.Description, "RFC3339") ||
		!strings.Contains(limitInput.Description, "明示タイムゾーン") {
		t.Fatalf("--limit description = %q", limitInput.Description)
	}
	output := renderCommandHelp(command, chatworkCommandSpecs())
	if len(output) > maxExactHelpBytes {
		t.Fatalf("room-tasks create help is %d bytes, exceeds %d", len(output), maxExactHelpBytes)
	}
	for _, want := range []string{"<unix-seconds|RFC3339>", "秒精度RFC3339（明示タイムゾーン付き）"} {
		if !strings.Contains(string(output), want) {
			t.Errorf("room-tasks create help lacks %q:\n%s", want, output)
		}
	}
}

func TestBuildRoomTaskDeadlineUsesOneUnixSecondsRequestAndPreservesLimitType(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "room-tasks create")
	var gotLimits []int64
	for _, test := range []struct {
		value     string
		limitType string
	}{
		{value: "2026-11-08T12:00:00+09:00", limitType: "date"},
		{value: "2026-11-08T03:00:00Z", limitType: "time"},
	} {
		parsed, _, err := parseChatworkInvocationArguments(spec, []string{
			"--room", "7", "--body", "task", "--assignee", "1", "--limit", test.value, "--limit-type", test.limitType,
		})
		if err != nil {
			t.Fatalf("parseChatworkInvocationArguments(%q) error = %v", test.value, err)
		}
		request, err := buildChatworkRequest(spec, parsed, nil)
		if err != nil {
			t.Fatalf("buildChatworkRequest(%q) error = %v", test.value, err)
		}
		if request.LimitType != test.limitType {
			t.Fatalf("limit_type = %q, want %s", request.LimitType, test.limitType)
		}
		gotLimits = append(gotLimits, request.Limit)
	}
	if gotLimits[0] != gotLimits[1] {
		t.Fatalf("equivalent offset request limits = %v", gotLimits)
	}

	parsed, _, err := parseChatworkInvocationArguments(spec, []string{
		"--room", "7", "--body", "task", "--assignee", "1", "--limit", "1700000000", "--limit-type", "time",
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildChatworkRequest(spec, parsed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if request.Limit != 1700000000 || request.LimitType != "time" {
		t.Fatalf("legacy request limit/type = %d/%q", request.Limit, request.LimitType)
	}
}

func TestRunChatworkRejectsInvalidRoomTaskDeadlinesBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "room-tasks create")
	for _, limit := range []string{
		"0",
		"-1",
		"9223372036854775808",
		"2026-11-08T12:00:00",
		"2026-11-08T12:00:00.5+09:00",
		"2026-02-30T12:00:00Z",
		"2026-11-08T12:00:00-00:00",
		"2026-11-08T12:00:00+24:00",
		"2026-11-08T12:00:00+09:60",
		"2026-11-08T12:00:00+0900",
		"1970-01-01T00:00:00Z",
		"1969-12-31T23:59:59Z",
	} {
		t.Run(limit, func(t *testing.T) {
			port := &chatworkRuntimePort{}
			cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
			code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
				"--room", "7", "--body", "task", "--assignee", "1", "--limit", limit,
			})
			if code != ExitUsage {
				t.Fatalf("runChatwork() code = %d, want invalid arguments; stderr=%s", code, stderr.String())
			}
			if authenticator.calls != 0 || port.calls != 0 || stdout.Len() != 0 {
				t.Fatalf("auth=%d provider=%d stdout=%q", authenticator.calls, port.calls, stdout.String())
			}
		})
	}
}

func TestRunChatworkRoomTaskRFC3339DeadlineReachesExistingMutationRequest(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "room-tasks create")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	task := chatworkRuntimeRef(t, chatwork.ReferenceTask, "8")
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{Task: request.Task, CreatedInRoom: &chatwork.RoomScopedCreation{
			Refs: []chatwork.Reference{task}, ParentRoom: room,
		}}, nil
	}}
	cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "7", "--body", "task", "--assignee", "1", "--limit", "2026-11-08T12:00:00+09:00", "--limit-type", "date",
	})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr=%s", code, stderr.String())
	}
	want := time.Date(2026, 11, 8, 3, 0, 0, 0, time.UTC).Unix()
	if authenticator.calls != 1 || port.calls != 1 || port.request.Limit != want || port.request.LimitType != "date" {
		t.Fatalf("auth=%d provider=%d request limit/type=%d/%q want=%d/date", authenticator.calls, port.calls, port.request.Limit, port.request.LimitType, want)
	}
	if err := port.request.Validate(); err != nil {
		t.Fatalf("domain request validation error = %v", err)
	}
}
