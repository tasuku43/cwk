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

type memberNameSenderPort struct {
	fixture  activeMemberNameSenderFixture
	requests []chatwork.Request
}

func (p *memberNameSenderPort) Execute(_ context.Context, _ authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.requests = append(p.requests, request)
	switch request.Task {
	case chatwork.TaskMembersList:
		return p.fixture.Members, nil
	case chatwork.TaskMessagesList:
		return p.fixture.Messages, nil
	default:
		return chatwork.Result{}, nil
	}
}

func TestActiveMemberNameSenderCompletesTwoCommandCanonicalFlow(t *testing.T) {
	fixture := memberNameSenderFixture()
	binding, err := authn.NewBindingID("active-member-name-sender-binding")
	if err != nil {
		t.Fatal(err)
	}
	port := &memberNameSenderPort{fixture: fixture}
	service := chatworkcmd.New(port)

	candidates, err := service.Execute(context.Background(), binding, fixture.FindRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates.Accounts) != 1 || candidates.Accounts[0].Ref.Value != "2501" ||
		candidates.MemberSelection == nil || candidates.MemberSelection.SourceCount != 3 {
		t.Fatalf("candidate result = %+v", candidates)
	}
	candidateOutput, err := capsule.Render(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(candidateOutput, `candidate-count=1`) || !strings.Contains(candidateOutput, `2501 "篠原 花子"`) ||
		strings.Contains(candidateOutput, "2502") || strings.Contains(candidateOutput, "2503") {
		t.Fatalf("candidate projection is not a bounded candidate set:\n%s", candidateOutput)
	}

	messages, err := service.Execute(context.Background(), binding, fixture.SenderRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages.Messages) != 2 || messages.Messages[0].Ref.Value != "1102" || messages.Messages[1].Ref.Value != "1104" {
		t.Fatalf("sender-selected messages = %+v", messages.Messages)
	}
	if len(port.requests) != 2 || port.requests[0].Task != chatwork.TaskMembersList || port.requests[0].MemberQuery != "" ||
		port.requests[1].Task != chatwork.TaskMessagesList || len(port.requests[1].MessageFilter.Senders) != 0 {
		t.Fatalf("provider requests = %+v", port.requests)
	}
}

func TestActiveMemberNameSenderScenarioClosesPredumpAndNameBindingWorkarounds(t *testing.T) {
	scenario := memberNameSenderScenario()
	want := [][]string{
		{"members", "find", "--room", "3501", "--query", "篠原 花子"},
		{"messages", "list", "--room", "3501", "--sender", "2501"},
	}
	if scenario.ID != "active.member-name-sender" || !reflect.DeepEqual(scenario.CommandArgv, want) ||
		scenario.ProviderCallBudget != 2 || scenario.ExternalProcessingBudget != 0 ||
		scenario.FullMessagePredumpBudget != 0 || !json.Valid(scenario.AnswerKey) {
		t.Fatalf("member-name sender scenario is not closed: %+v", scenario)
	}
	for _, closed := range []string{"Do not dump all messages first", "auto-select an ambiguous display name", "jq", "grep", "raw Chatwork notation"} {
		if !strings.Contains(scenario.UserPrompt, closed) {
			t.Errorf("scenario does not close %q: %s", closed, scenario.UserPrompt)
		}
	}
}
