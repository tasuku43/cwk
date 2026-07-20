package main

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/cli/capsule"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

type messageIndexPort struct {
	result  chatwork.Result
	calls   int
	request chatwork.Request
}

func (p *messageIndexPort) Execute(_ context.Context, _ authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.calls++
	p.request = request
	return p.result, nil
}

func TestActiveMessageIndexSelectsNewestByTypedTimeAndPreservesProviderOrder(t *testing.T) {
	fixture := messageIndexFixture()
	if err := fixture.Source.ValidateFor(chatwork.Request{
		Task: fixture.Source.Task, Room: fixture.Source.MessageRoom, ForceRecent: true,
	}); err != nil {
		t.Fatalf("source semantic fixture: %v", err)
	}

	result, port := executeMessageIndex(t, fixture.Source, fixture.NoneRequest)
	if port.calls != 1 || len(port.request.MessageFilter.Senders) != 0 ||
		port.request.MessageFilter.Context != "" || port.request.MessageFilter.StartIndex != 0 ||
		port.request.MessageFilter.Count != 0 {
		t.Fatalf("provider boundary calls=%d request-filter=%+v", port.calls, port.request.MessageFilter)
	}
	selection := result.MessageSelection
	if selection == nil || selection.SourceCount != 6 || selection.CandidateCount != 6 ||
		selection.Filter.StartIndex != 1 || selection.Filter.Count != 2 ||
		selection.ItemsPerPage != 2 || selection.NextStartIndex != 3 ||
		!reflect.DeepEqual(selection.SourceSequences, []int{1, 3}) ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{1, 3}) {
		t.Fatalf("limit selection = %+v", selection)
	}
	if got := messageValuesForIndex(result.Messages); !reflect.DeepEqual(got, []string{"1201", "1203"}) {
		t.Fatalf("newest primary refs in provider order = %v", got)
	}
	if result.Messages[0].Reply == nil || result.Messages[0].Reply.Resolved || result.Messages[0].Reply.Target.Value != "1202" {
		t.Fatalf("omitted parent was not preserved as canonical unresolved relation: %+v", result.Messages[0].Reply)
	}

	output, err := capsule.Render(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"messages room-ref=3601 count=2 window=recent source-limit=100 complete=false",
		"selection source-count=6 candidate-count=6 start-index=1 count=2 items-per-page=2 next-start-index=3 anchors=[#1,#3]",
		`#1 1201 a1 1702000060 reply=?1202`,
		`#3 1203 a1 1702000050 "Second-newest primary message."`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("limit projection lacks %q:\n%s", want, output)
		}
	}
}

func TestActiveMessageIndexAddsReplyContextAfterPrimaryCount(t *testing.T) {
	fixture := messageIndexFixture()
	result, port := executeMessageIndex(t, fixture.Source, fixture.RepliesRequest)
	if port.calls != 1 || len(result.Messages) != 4 || len(result.Messages) <= fixture.RepliesRequest.MessageFilter.Count {
		t.Fatalf("reply context did not remain bounded but additional: calls=%d messages=%d", port.calls, len(result.Messages))
	}
	if got := messageValuesForIndex(result.Messages); !reflect.DeepEqual(got, []string{"1201", "1202", "1203", "1204"}) {
		t.Fatalf("provider-order anchors/context = %v", got)
	}
	if !reflect.DeepEqual(result.MessageSelection.SourceSequences, []int{1, 2, 3, 4}) ||
		!reflect.DeepEqual(result.MessageSelection.AnchorSequences, []int{1, 3}) {
		t.Fatalf("reply-context provenance = %+v", result.MessageSelection)
	}
	for messageRef, parentRef := range map[string]string{"1201": "1202", "1204": "1203"} {
		message := messageByRef(t, result.Messages, messageRef)
		if message.Reply == nil || !message.Reply.Resolved || message.Reply.Target.Value != parentRef {
			t.Errorf("typed reply %s -> %s = %+v", messageRef, parentRef, message.Reply)
		}
	}

	output, err := capsule.Render(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "selection source-count=6 candidate-count=6 start-index=1 count=2 items-per-page=2 next-start-index=3 context=replies anchors=[#1,#3]") {
		t.Fatalf("reply-context limit metadata is not explicit:\n%s", output)
	}
	if !strings.Contains(output, "#1 1201 a1 1702000060 reply=#2") ||
		!strings.Contains(output, "#4 1204 a2 1702000020 reply=#3") {
		t.Fatalf("reply context is not directly traversable:\n%s", output)
	}
}

func TestActiveMessageIndexContinuesWithoutRepeatingEarlierRanks(t *testing.T) {
	fixture := messageIndexFixture()
	result, port := executeMessageIndex(t, fixture.Source, fixture.NextRequest)
	if port.calls != 1 || result.MessageSelection == nil ||
		result.MessageSelection.Filter.StartIndex != 3 || result.MessageSelection.Filter.Count != 3 ||
		result.MessageSelection.ItemsPerPage != 3 || result.MessageSelection.NextStartIndex != 6 ||
		!reflect.DeepEqual(result.MessageSelection.AnchorSequences, []int{4, 5, 6}) ||
		!reflect.DeepEqual(messageValuesForIndex(result.Messages), []string{"1204", "1205", "1206"}) {
		t.Fatalf("continuation selection = %+v messages=%v", result.MessageSelection, messageValuesForIndex(result.Messages))
	}
}

func TestActiveMessageIndexScenarioUsesTwoCommandsWithoutPostProcessing(t *testing.T) {
	scenario := messageIndexScenario()
	wantArgv := [][]string{
		{"messages", "list", "--room", "3601", "--count", "2", "--context", "replies"},
		{"messages", "list", "--room", "3601", "--start-index", "3", "--count", "3"},
	}
	if scenario.ID != "active.message-index" || !reflect.DeepEqual(scenario.CommandArgv, wantArgv) ||
		scenario.ProviderCallBudget != 2 || scenario.ExternalProcessingBudget != 0 || !json.Valid(scenario.AnswerKey) {
		t.Fatalf("message-index readiness scenario is not closed: %+v", scenario)
	}
	for _, closedWorkaround := range []string{"jq", "grep", "external parser", "source inspection", "provider-order assumptions", "extra Chatwork calls"} {
		if !strings.Contains(scenario.UserPrompt, closedWorkaround) {
			t.Errorf("scenario does not close %q workaround: %q", closedWorkaround, scenario.UserPrompt)
		}
	}

	var answer struct {
		SourceCount        int      `json:"source_count"`
		CandidateCount     int      `json:"candidate_count"`
		StartIndex         int      `json:"start_index"`
		RequestedCount     int      `json:"requested_count"`
		ItemsPerPage       int      `json:"items_per_page"`
		NextStartIndex     int      `json:"next_start_index"`
		AnchorSequence     []int    `json:"anchor_sequence"`
		ContextSequence    []int    `json:"context_sequence"`
		PrimaryMessageRefs []string `json:"primary_message_refs"`
		Continuation       struct {
			StartIndex         int      `json:"start_index"`
			RequestedCount     int      `json:"requested_count"`
			ItemsPerPage       int      `json:"items_per_page"`
			AnchorSequence     []int    `json:"anchor_sequence"`
			PrimaryMessageRefs []string `json:"primary_message_refs"`
		} `json:"continuation"`
		NextCommand struct {
			Path       string `json:"path"`
			RoomRef    string `json:"room_ref"`
			MessageRef string `json:"message_ref"`
		} `json:"next_command"`
	}
	if err := json.Unmarshal(scenario.AnswerKey, &answer); err != nil {
		t.Fatal(err)
	}
	if answer.SourceCount != 6 || answer.CandidateCount != 6 || answer.StartIndex != 1 ||
		answer.RequestedCount != 2 || answer.ItemsPerPage != 2 || answer.NextStartIndex != 3 ||
		!reflect.DeepEqual(answer.AnchorSequence, []int{1, 3}) ||
		!reflect.DeepEqual(answer.ContextSequence, []int{2, 4}) ||
		!reflect.DeepEqual(answer.PrimaryMessageRefs, []string{"1201", "1203"}) ||
		answer.Continuation.StartIndex != 3 || answer.Continuation.RequestedCount != 3 ||
		answer.Continuation.ItemsPerPage != 3 ||
		!reflect.DeepEqual(answer.Continuation.AnchorSequence, []int{4, 5, 6}) ||
		!reflect.DeepEqual(answer.Continuation.PrimaryMessageRefs, []string{"1204", "1205", "1206"}) ||
		answer.NextCommand.Path != "messages show" || answer.NextCommand.RoomRef != "3601" || answer.NextCommand.MessageRef != "1201" ||
		!reflect.DeepEqual(fixtureNextArgv(t), []string{"messages", "show", "--room", "3601", "--message", "1201"}) {
		t.Fatalf("message-index answer key = %+v", answer)
	}
}

func executeMessageIndex(t *testing.T, source chatwork.Result, request chatwork.Request) (chatwork.Result, *messageIndexPort) {
	t.Helper()
	binding, err := authn.NewBindingID("active-message-index-binding")
	if err != nil {
		t.Fatal(err)
	}
	port := &messageIndexPort{result: source}
	result, err := chatworkcmd.New(port).Execute(context.Background(), binding, request)
	if err != nil {
		t.Fatal(err)
	}
	return result, port
}

func messageValuesForIndex(messages []chatwork.Message) []string {
	values := make([]string, len(messages))
	for index, message := range messages {
		values[index] = message.Ref.Value
	}
	return values
}

func fixtureNextArgv(t *testing.T) []string {
	t.Helper()
	return messageIndexFixture().NextArgv
}
