package main

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/cli/capsule"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

type messagePeriodPort struct {
	result  chatwork.Result
	calls   int
	request chatwork.Request
}

func (p *messagePeriodPort) Execute(_ context.Context, _ authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.calls++
	p.request = request
	return p.result, nil
}

func TestActiveMessagePeriodSelectsFortyOfOneHundredWithoutProviderFilter(t *testing.T) {
	fixture := messagePeriodFixture()
	if err := fixture.Source.ValidateFor(chatwork.Request{Task: fixture.Source.Task, Room: fixture.Source.MessageRoom, ForceRecent: true}); err != nil {
		t.Fatalf("source semantic fixture: %v", err)
	}
	result, port := executeMessagePeriod(t, fixture.Source, fixture.Request)
	if port.calls != 1 || !reflect.DeepEqual(port.request.MessageFilter, chatwork.MessageFilter{}) {
		t.Fatalf("provider boundary calls=%d filter=%+v", port.calls, port.request.MessageFilter)
	}
	selection := result.MessageSelection
	if selection == nil || selection.SourceCount != 100 || selection.CandidateCount != 40 ||
		selection.Filter.Period.Day != "2026-07-17" || selection.Filter.Period.TimeZone != chatwork.MessageDayTimeZone ||
		len(selection.AnchorSequences) != 40 || selection.AnchorSequences[0] != 31 || selection.AnchorSequences[39] != 70 ||
		!reflect.DeepEqual(selection.SourceSequences, selection.AnchorSequences) || len(result.Messages) != 40 {
		t.Fatalf("period selection = %+v messages=%d", selection, len(result.Messages))
	}
	for _, message := range result.Messages {
		if !selection.Filter.Period.Contains(message.SendTime) {
			t.Fatalf("out-of-period anchor %s at %d", message.Ref.Value, message.SendTime)
		}
	}
}

func TestActiveMessagePeriodContextCanCrossOnlyOneTypedReplyEdge(t *testing.T) {
	fixture := messagePeriodFixture()
	result, _ := executeMessagePeriod(t, fixture.Source, fixture.ContextRequest)
	if len(result.Messages) != 41 || result.MessageSelection == nil ||
		result.MessageSelection.SourceSequences[0] != 30 || result.MessageSelection.AnchorSequences[0] != 31 {
		t.Fatalf("period reply context = %+v messages=%d", result.MessageSelection, len(result.Messages))
	}
	if len(result.Messages[1].Replies) != 1 || !result.Messages[1].Replies[0].Resolved || result.Messages[1].Replies[0].Target != result.Messages[0].Ref {
		t.Fatalf("cross-boundary typed reply = %+v", result.Messages[1].Replies[0])
	}
	for _, sequence := range result.MessageSelection.AnchorSequences {
		if sequence == 30 {
			t.Fatal("out-of-period context became a primary anchor")
		}
	}
}

func TestActiveMessagePeriodProjectionAndScenarioAreClosed(t *testing.T) {
	fixture := messagePeriodFixture()
	result, _ := executeMessagePeriod(t, fixture.Source, fixture.Request)
	output, err := capsule.Render(result)
	if err != nil {
		t.Fatal(err)
	}
	period := fixture.Request.MessageFilter.Period
	for _, want := range []string{
		"messages room-ref=3701 count=40 window=recent source-limit=100 complete=false",
		"selection source-count=100 since=", " until=", " on=2026-07-17 time-zone=Asia/Tokyo candidate-count=40",
		"anchors=[#31,#32", "#70]", "Decision: archive export", "Owner: canonical account 2702", "Deadline: the rollout review",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("period projection lacks %q:\n%s", want, output)
		}
	}
	if !strings.Contains(output, "since="+formatPeriodUnix(period.Since)) || !strings.Contains(output, "until="+formatPeriodUnix(period.Until)) ||
		strings.Contains(output, "forged since=0") {
		t.Fatalf("effective bounds or omitted hostile record are wrong:\n%s", output)
	}

	scenario := messagePeriodScenario()
	wantArgv := []string{"messages", "list", "--room", "3701", "--on", "2026-07-17"}
	if scenario.ID != "active.message-period" || !reflect.DeepEqual(scenario.CommandArgv, wantArgv) ||
		scenario.ProviderCallBudget != 1 || scenario.ExternalProcessingBudget != 0 || !json.Valid(scenario.AnswerKey) {
		t.Fatalf("period scenario is not closed: %+v", scenario)
	}
	for _, workaround := range []string{"jq", "grep", "external parser", "source inspection", "timestamp calculations", "extra Chatwork calls", "do not claim history"} {
		if !strings.Contains(scenario.UserPrompt, workaround) {
			t.Errorf("scenario does not close %q", workaround)
		}
	}
}

func TestActiveMessagePeriodMeasurementGoldensFreezeSameHundredMessageSource(t *testing.T) {
	fixture := messagePeriodFixture()
	unfiltered := chatwork.Request{Task: fixture.Source.Task, Room: fixture.Source.MessageRoom, ForceRecent: true}
	cases := []struct {
		path    string
		request chatwork.Request
	}{
		{"testdata/active-message-period.unfiltered.txt", unfiltered},
		{"testdata/active-message-period.filtered.txt", fixture.Request},
	}
	outputs := make([]string, len(cases))
	for index, test := range cases {
		result, _ := executeMessagePeriod(t, fixture.Source, test.request)
		output, err := capsule.Render(result)
		if err != nil {
			t.Fatal(err)
		}
		outputs[index] = output
		assertActiveMeasurementGolden(t, test.path, output)
	}
	if len(outputs[1])*2 >= len(outputs[0]) {
		t.Fatalf("one-day period selection did not reduce fixture bytes by more than half: full=%d filtered=%d", len(outputs[0]), len(outputs[1]))
	}
}

func executeMessagePeriod(t *testing.T, source chatwork.Result, request chatwork.Request) (chatwork.Result, *messagePeriodPort) {
	t.Helper()
	binding, err := authn.NewBindingID("active-message-period-binding")
	if err != nil {
		t.Fatal(err)
	}
	port := &messagePeriodPort{result: source}
	result, err := chatworkcmd.New(port).Execute(context.Background(), binding, request)
	if err != nil {
		t.Fatal(err)
	}
	return result, port
}

func formatPeriodUnix(value int64) string {
	return strconv.FormatInt(value, 10)
}
