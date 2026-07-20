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

type messageClosurePort struct {
	source   chatwork.Result
	exact    map[string]chatwork.Result
	requests []chatwork.Request
}

func (p *messageClosurePort) Execute(_ context.Context, _ authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.requests = append(p.requests, request)
	if request.Task == chatwork.TaskMessagesList {
		return p.source, nil
	}
	return p.exact[request.Message.Value], nil
}

func TestActiveMessageRelationClosureAnswersT1WithBoundedInternalFetches(t *testing.T) {
	fixture := messageRelationClosureFixture()
	result, port := executeMessageClosure(t, fixture.Source, fixture.Exact, fixture.Request)
	if len(port.requests) != 3 || port.requests[0].Task != chatwork.TaskMessagesList ||
		port.requests[1].Message.Value != "9001" || port.requests[2].Message.Value != "9002" {
		t.Fatalf("provider requests = %+v", port.requests)
	}
	for _, request := range port.requests {
		if request.MessageRelationFetchLimit != 0 || !reflect.DeepEqual(request.MessageFilter, chatwork.MessageFilter{}) {
			t.Fatalf("application policy leaked to provider request: %+v", request)
		}
	}
	resolution := result.MessageRelationResolution
	if resolution == nil || resolution.FetchLimit != 5 || resolution.FetchAttempts != 2 || len(resolution.Targets) != 2 {
		t.Fatalf("resolution = %+v", resolution)
	}
	output, err := capsule.Render(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"relation-resolution fetch-limit=5 fetch-attempts=2 targets=2",
		"reply=message-ref:9001", "relations=[reply{state=resolved,target-ref=9002}]",
		"relation-context provenance=fetched message-ref=9001", "Aurora decision: use the staged archive migration",
		"relation-context provenance=fetched message-ref=9002", "canonical account 2802", "deadline: 2026-07-21",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("relation closure output lacks %q:\n%s", want, output)
		}
	}
	scenario := messageRelationClosureScenario()
	if scenario.ID != "active.message-relation-closure" || scenario.ProviderCallBudget != 3 ||
		scenario.ExternalProcessingBudget != 0 || !json.Valid(scenario.AnswerKey) {
		t.Fatalf("relation closure scenario = %+v", scenario)
	}
}

func TestActiveMessageReachabilityStopsT2AfterOneListCall(t *testing.T) {
	fixture := messageReachabilityFixture()
	result, port := executeMessageClosure(t, fixture.Source, nil, fixture.Request)
	if len(port.requests) != 1 || port.requests[0].Task != chatwork.TaskMessagesList {
		t.Fatalf("provider requests = %+v", port.requests)
	}
	if result.MessageSelection == nil || result.MessageSelection.SourceCount != 100 || result.MessageSelection.CandidateCount != 0 || len(result.Messages) != 0 {
		t.Fatalf("unreachable selection = %+v messages=%d", result.MessageSelection, len(result.Messages))
	}
	if result.MessageReachability == nil || result.MessageReachability.PeriodReachability != chatwork.MessagePeriodOutsideReachableWindow ||
		result.MessageReachability.OldestMessage.Value != "4001" {
		t.Fatalf("reachability = %+v", result.MessageReachability)
	}
	output, err := capsule.Render(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"count=0 window=recent source-limit=100",
		"oldest-reachable-message-ref=4001",
		"period-reachability=out-of-reachable-window",
		"selection source-count=100",
		"candidate-count=0",
		"relation-resolution fetch-limit=5 fetch-attempts=0 targets=0",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("reachability output lacks %q:\n%s", want, output)
		}
	}
	scenario := messageReachabilityScenario()
	if scenario.ID != "active.message-period-reachability" || scenario.ProviderCallBudget != 1 ||
		scenario.ExternalProcessingBudget != 0 || !json.Valid(scenario.AnswerKey) {
		t.Fatalf("reachability scenario = %+v", scenario)
	}
	for _, forbiddenStep := range []string{"probe adjacent dates", "messages show", "jq/grep", "inspect source", "unreachable period as an empty day"} {
		if !strings.Contains(scenario.UserPrompt, forbiddenStep) {
			t.Errorf("T2 scenario does not close %q", forbiddenStep)
		}
	}
}

func executeMessageClosure(t *testing.T, source chatwork.Result, exact map[string]chatwork.Result, request chatwork.Request) (chatwork.Result, *messageClosurePort) {
	t.Helper()
	binding, err := authn.NewBindingID("active-message-relation-closure-binding")
	if err != nil {
		t.Fatal(err)
	}
	port := &messageClosurePort{source: source, exact: exact}
	result, err := chatworkcmd.New(port).Execute(context.Background(), binding, request)
	if err != nil {
		t.Fatal(err)
	}
	return result, port
}
