package cli

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/cli/capsule"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

func TestSplitChatworkPresentationArgumentsKeepsMaxBytesOutOfRequest(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	parsed, err := parseChatworkArguments(spec, []string{"--room", "7", "--count", "2", "--max-bytes", "4096"})
	if err != nil {
		t.Fatal(err)
	}
	requestArguments, options, err := splitChatworkPresentationArguments(spec, parsed)
	if err != nil {
		t.Fatal(err)
	}
	if options.MaxBytes != 4096 {
		t.Fatalf("MaxBytes=%d", options.MaxBytes)
	}
	if _, leaked := requestArguments["--max-bytes"]; leaked {
		t.Fatalf("request arguments contain --max-bytes: %+v", requestArguments)
	}
	got, err := buildChatworkRequest(spec, requestArguments, nil)
	if err != nil {
		t.Fatal(err)
	}
	baselineArguments, err := parseChatworkArguments(spec, []string{"--room", "7", "--count", "2"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := buildChatworkRequest(spec, baselineArguments, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("presentation option changed request\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestRunChatworkRejectsInvalidMaxBytesBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	for _, value := range []string{"0", "-1", strconv.Itoa(maxChatworkOutputBytes + 1), "many"} {
		t.Run(value, func(t *testing.T) {
			port := &chatworkRuntimePort{}
			cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
			code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", "7", "--max-bytes", value})
			if code != ExitUsage || !strings.Contains(stderr.String(), "code: invalid_arguments") || !strings.Contains(stderr.String(), "1 から 16777216") {
				t.Fatalf("value=%q code=%d stderr=%s", value, code, stderr.String())
			}
			if authenticator.calls != 0 || port.calls != 0 || stdout.Len() != 0 {
				t.Fatalf("value=%q auth=%d provider=%d stdout=%q", value, authenticator.calls, port.calls, stdout.String())
			}
		})
	}
}

func TestRenderChatworkResultWithoutBudgetIsByteIdentical(t *testing.T) {
	result := outputBudgetResult(t, 3, 20)
	want, err := capsule.Render(result)
	if err != nil {
		t.Fatal(err)
	}
	got, err := renderChatworkResult(result, chatworkPresentationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("unbudgeted output changed\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestRunChatworkMaxBytesUsesOneProviderReadAndPreservesBothContinuations(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	providerResult := outputBudgetResult(t, 3, 3000)
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		result := providerResult
		result.MessageRoom = request.Room
		for index := range result.Messages {
			result.Messages[index].Room = request.Room
		}
		return result, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)

	const budget = 4096
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "7", "--count", "2", "--resolve-relations", "0", "--max-bytes", strconv.Itoa(budget),
	})
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls auth=%d provider=%d", authenticator.calls, port.calls)
	}
	if stdout.Len() > budget {
		t.Fatalf("output bytes=%d budget=%d", stdout.Len(), budget)
	}
	got := stdout.String()
	for _, want := range []string{
		" complete=false ",
		"selection source-count=3 candidate-count=3 start-index=1 count=2 items-per-page=2 next-start-index=3",
		"#3 103 ",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output lacks %q:\n%s", want, got)
		}
	}
	wantLast := "delivery max-bytes=4096 output-complete=false output-items=1 output-total-items=2 output-next-start-index=2 output-next-count=1\n"
	if !strings.HasSuffix(got, wantLast) {
		t.Fatalf("delivery metadata is not final:\n%s", got)
	}
	if strings.Contains(got, "#2 102 ") || strings.Contains(got, "#1 101 ") {
		t.Fatalf("budget did not keep the newest anchor prefix:\n%s", got)
	}
	if port.request.Task != chatwork.TaskMessagesList || port.request.Room.Value != "7" {
		t.Fatalf("provider request lost its task scope: %+v", port.request)
	}
}

func TestRunChatworkMaxBytesTooSmallWritesNoStdout(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	result := outputBudgetResult(t, 1, 40)
	minimum := minimumOutputBudget(t, result)
	if minimum <= 1 {
		t.Fatalf("unexpected minimum=%d", minimum)
	}
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		result.MessageRoom = request.Room
		result.Messages[0].Room = request.Room
		return result, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "7", "--resolve-relations", "0", "--max-bytes", strconv.Itoa(minimum - 1),
	})
	if code != ExitUsage || !strings.Contains(stderr.String(), "code: output_budget_too_small") {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("too-small budget wrote %d bytes: %q", stdout.Len(), stdout.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls auth=%d provider=%d", authenticator.calls, port.calls)
	}
}

func TestRenderChatworkMaxBytesExactMinimumIncludesFinalNewline(t *testing.T) {
	result := outputBudgetResult(t, 1, 40)
	minimum := minimumOutputBudget(t, result)
	output, err := renderChatworkResult(result, chatworkPresentationOptions{MaxBytes: minimum})
	if err != nil {
		t.Fatalf("minimum=%d error=%v", minimum, err)
	}
	if len(output) > minimum || !strings.HasSuffix(output, "\n") {
		t.Fatalf("output bytes=%d minimum=%d", len(output), minimum)
	}
	wantLast := "delivery max-bytes=" + strconv.Itoa(minimum) + " output-complete=true output-items=1 output-total-items=1\n"
	if !strings.HasSuffix(output, wantLast) {
		t.Fatalf("delivery metadata is not the final line:\n%s", output)
	}
	if _, err := renderChatworkResult(result, chatworkPresentationOptions{MaxBytes: minimum - 1}); err == nil {
		t.Fatal("one byte below the exact minimum unexpectedly fit")
	}
}

func minimumOutputBudget(t *testing.T, result chatwork.Result) int {
	t.Helper()
	for budget := 1; budget <= 8192; budget++ {
		_, err := renderChatworkResult(result, chatworkPresentationOptions{MaxBytes: budget})
		if err == nil {
			return budget
		}
		public, ok := fault.PublicCopy(err)
		if !ok || public.Code != "output_budget_too_small" {
			t.Fatalf("budget=%d error=%v", budget, err)
		}
	}
	t.Fatal("no fitting output budget found")
	return 0
}

func outputBudgetResult(t *testing.T, count, bodyBytes int) chatwork.Result {
	t.Helper()
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	account := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "9"), Name: "Synthetic"}
	messages := make([]chatwork.Message, count)
	for index := range messages {
		messages[index] = chatwork.Message{
			Ref:      chatworkRuntimeRef(t, chatwork.ReferenceMessage, strconv.Itoa(101+index)),
			Room:     room,
			Sender:   account,
			Body:     strings.Repeat(strconv.Itoa(index+1), bodyBytes),
			SendTime: int64(index + 1),
		}
	}
	result := chatwork.Result{
		Task:        chatwork.TaskMessagesList,
		MessageRoom: room,
		Coverage:    chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
		Messages:    messages,
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return result
}
