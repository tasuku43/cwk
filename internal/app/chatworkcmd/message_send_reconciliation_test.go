package chatworkcmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type reconciliationPreparedBody struct{}

func (*reconciliationPreparedBody) IsPreparedMessageBody() {}

type reconciliationPort struct {
	steps          []sendPortStep
	requests       []chatwork.Request
	bindings       []authn.BindingID
	contexts       []context.Context
	preparedInput  chatwork.Request
	finalizedInput chatwork.Request
	comparedBodies []string
	prepareCalls   int
	finalizeCalls  int
	matchCalls     int
	discardCalls   int
	prepareError   error
	finalizeError  error
	matchError     error
	expectedBody   string
}

func (p *reconciliationPort) Execute(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	index := len(p.requests)
	p.requests = append(p.requests, request)
	p.bindings = append(p.bindings, binding)
	p.contexts = append(p.contexts, ctx)
	if index >= len(p.steps) {
		return chatwork.Result{}, errors.New("unexpected provider call")
	}
	step := p.steps[index]
	if step.onCall != nil {
		step.onCall(ctx)
	}
	if step.result.Task == "" {
		step.result.Task = request.Task
	}
	return step.result, step.err
}

func (p *reconciliationPort) PrepareMessageSendReconciliation(_ context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.PreparedMessageBody, error) {
	p.prepareCalls++
	p.preparedInput = request
	if err := binding.Validate(); err != nil {
		return nil, err
	}
	if p.prepareError != nil {
		return nil, p.prepareError
	}
	if request.Body != "" {
		p.expectedBody = request.Body
	}
	return &reconciliationPreparedBody{}, nil
}

func (p *reconciliationPort) FinalizeMessageSendReconciliation(_ context.Context, binding authn.BindingID, request chatwork.Request, _ chatwork.PreparedMessageBody) error {
	p.finalizeCalls++
	p.finalizedInput = request
	if err := binding.Validate(); err != nil {
		return err
	}
	return p.finalizeError
}

func (p *reconciliationPort) MatchMessageSendReconciliation(_ context.Context, binding authn.BindingID, request chatwork.Request, _ chatwork.PreparedMessageBody, bodies []string) ([]bool, error) {
	p.matchCalls++
	if err := binding.Validate(); err != nil {
		return nil, err
	}
	if p.matchError != nil {
		return nil, p.matchError
	}
	p.comparedBodies = append([]string(nil), bodies...)
	matches := make([]bool, len(bodies))
	for index, body := range bodies {
		matches[index] = body == p.expectedBody
	}
	return matches, nil
}

func (p *reconciliationPort) DiscardMessageSendReconciliation(chatwork.PreparedMessageBody) {
	p.discardCalls++
}

func TestMessageSendReconciliationReturnsEveryExactCandidateAndNeverSends(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	accountA := relationshipReference(t, chatwork.ReferenceAccount, "7")
	accountB := relationshipReference(t, chatwork.ReferenceAccount, "9")
	messageA := relationshipReference(t, chatwork.ReferenceMessage, "101")
	messageB := relationshipReference(t, chatwork.ReferenceMessage, "103")
	ctx := context.WithValue(context.Background(), struct{}{}, "reconcile-context")
	binding := testBinding(t)
	for _, test := range []struct {
		name      string
		messages  []chatwork.Message
		access    chatwork.MessageAccessLimitation
		wantRefs  []chatwork.Reference
		wantState chatwork.MessageSendReconciliationStatus
	}{
		{name: "zero", messages: []chatwork.Message{{Ref: messageA, Room: room, Sender: chatwork.Account{Ref: accountA}, Body: "different", SendTime: 1}}, wantState: chatwork.MessageSendNoCandidatesWindow},
		{name: "restricted latest window has no visible candidates", messages: []chatwork.Message{}, access: chatwork.MessageAccessAll, wantState: chatwork.MessageSendNoCandidatesWindow},
		{name: "one", messages: []chatwork.Message{{Ref: messageA, Room: room, Sender: chatwork.Account{Ref: accountA, Name: "Alice"}, Body: "hello", SendTime: 11}}, wantRefs: []chatwork.Reference{messageA}, wantState: chatwork.MessageSendCandidatesFound},
		{name: "many including another author", messages: []chatwork.Message{
			{Ref: messageA, Room: room, Sender: chatwork.Account{Ref: accountA, Name: "Alice"}, Body: "hello", SendTime: 11},
			{Ref: messageB, Room: room, Sender: chatwork.Account{Ref: accountB, Name: "Other"}, Body: "hello", SendTime: 12},
		}, wantRefs: []chatwork.Reference{messageA}, wantState: chatwork.MessageSendCandidatesFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			port := &reconciliationPort{
				steps: []sendPortStep{{result: chatwork.Result{Task: chatwork.TaskAccountShow, Account: &chatwork.Account{Ref: accountA}}}, {result: chatwork.Result{
					Task: chatwork.TaskMessagesList, MessageRoom: room,
					Coverage:      chatwork.Coverage{Kind: "latest_window", Limit: 100, Complete: false},
					MessageAccess: test.access,
					Messages:      test.messages,
				}}},
			}
			request := chatwork.Request{Task: chatwork.TaskMessagesReconcileSend, Room: room, Body: "hello"}
			result, err := New(port).Execute(ctx, binding, request)
			if err != nil {
				t.Fatal(err)
			}
			if result.Task != chatwork.TaskMessagesReconcileSend || result.MessageSendReconciliation == nil {
				t.Fatalf("result = %+v", result)
			}
			reconciliation := result.MessageSendReconciliation
			if reconciliation.Account != accountA || reconciliation.Outcome != "unknown" || reconciliation.Status != test.wantState ||
				reconciliation.SourceCount != len(test.messages) || reconciliation.CandidateCount != len(test.wantRefs) {
				t.Fatalf("reconciliation = %+v", reconciliation)
			}
			gotRefs := make([]chatwork.Reference, len(reconciliation.Candidates))
			for index, candidate := range reconciliation.Candidates {
				gotRefs[index] = candidate.Message
				if candidate.SenderName == "" || candidate.SendTime == 0 {
					t.Fatalf("candidate omitted author/time: %+v", candidate)
				}
			}
			if len(gotRefs) != len(test.wantRefs) {
				t.Fatalf("candidate refs = %v, want %v", gotRefs, test.wantRefs)
			}
			for index := range gotRefs {
				if gotRefs[index] != test.wantRefs[index] {
					t.Fatalf("candidate refs = %v, want %v", gotRefs, test.wantRefs)
				}
			}
			if len(port.requests) != 2 || port.requests[0].Task != chatwork.TaskAccountShow || port.requests[1].Task != chatwork.TaskMessagesList || !port.requests[1].ForceRecent {
				t.Fatalf("provider requests = %+v; want one latest list read", port.requests)
			}
			if port.prepareCalls != 1 || port.finalizeCalls != 1 || port.matchCalls != 1 || port.discardCalls != 1 {
				t.Fatalf("body handle calls prepare=%d finalize=%d match=%d discard=%d", port.prepareCalls, port.finalizeCalls, port.matchCalls, port.discardCalls)
			}
			if port.contexts[0] != ctx || port.bindings[0] != binding {
				t.Fatalf("provider request changed context/binding")
			}
			if result.MessageAccess != test.access {
				t.Fatalf("access limitation = %v, want %v", result.MessageAccess, test.access)
			}
			if len(port.comparedBodies) != len(test.messages) {
				t.Fatalf("compared bodies = %d, want each visible message", len(port.comparedBodies))
			}
			if result.Messages != nil || result.MessageSendReconciliation.Candidates == nil {
				t.Fatalf("result retained provider message bodies or omitted explicit candidates: %+v", result)
			}
		})
	}
}

func TestMessageSendReconciliationReplyReadPrecedesOneListWithSameBinding(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "101")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	ctx := context.WithValue(context.Background(), struct{}{}, "same-context")
	binding := testBinding(t)
	port := &reconciliationPort{steps: []sendPortStep{
		{result: chatwork.Result{Messages: []chatwork.Message{{Ref: message, Room: room, Sender: chatwork.Account{Ref: account, Name: "Author"}, Body: "target", SendTime: 1}}}},
		{result: chatwork.Result{Task: chatwork.TaskAccountShow, Account: &chatwork.Account{Ref: account}}},
		{result: chatwork.Result{Task: chatwork.TaskMessagesList, MessageRoom: room, Coverage: chatwork.Coverage{Kind: "latest_window", Limit: 100}, Messages: []chatwork.Message{}}},
	}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesReconcileSend, Room: room, Body: "payload",
		MessageRecipients:  []chatwork.Reference{relationshipReference(t, chatwork.ReferenceAccount, "9")},
		MessageReplyTarget: &chatwork.MessageReplyTarget{Message: message},
	}
	_, err := New(port).Execute(ctx, binding, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 3 || port.requests[0].Task != chatwork.TaskMessagesShow || port.requests[1].Task != chatwork.TaskAccountShow || port.requests[2].Task != chatwork.TaskMessagesList {
		t.Fatalf("provider order = %+v", port.requests)
	}
	if port.requests[0].Room != room || port.requests[0].Message != message || !port.requests[2].ForceRecent {
		t.Fatalf("provider request targets = %+v", port.requests)
	}
	if port.finalizedInput.MessageReplyTarget == nil || port.finalizedInput.MessageReplyTarget.Sender != account {
		t.Fatalf("reply author was not bound into the common expanded-body request: %+v", port.finalizedInput)
	}
	if port.bindings[0] != binding || port.bindings[1] != binding || port.contexts[0] != ctx || port.contexts[1] != ctx || port.bindings[2] != binding || port.contexts[2] != ctx {
		t.Fatal("exact reply read and latest list did not share binding/context")
	}
	if port.prepareCalls != 1 || port.finalizeCalls != 1 || port.matchCalls != 1 || port.discardCalls != 1 {
		t.Fatalf("body handle calls = prepare %d finalize %d match %d discard %d", port.prepareCalls, port.finalizeCalls, port.matchCalls, port.discardCalls)
	}
}

func TestMessageSendReconciliationFailuresDoNotReturnNoCandidates(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "101")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	for _, test := range []struct {
		name             string
		request          chatwork.Request
		steps            []sendPortStep
		prepareErr       error
		wantCode         string
		wantCalls        int
		wantPrepareCalls int
	}{
		{
			name:     "invalid duplicate recipient before preparation",
			request:  chatwork.Request{Task: chatwork.TaskMessagesReconcileSend, Room: room, Body: "private-body-canary", MessageRecipients: []chatwork.Reference{account, account}},
			wantCode: "invalid_chatwork_task", wantCalls: 0,
		},
		{
			name:     "invalid literal body before preparation",
			request:  chatwork.Request{Task: chatwork.TaskMessagesReconcileSend, Room: room, Body: "bad\x00body"},
			wantCode: "invalid_chatwork_task", wantCalls: 0,
		},
		{
			name:       "body file invalid before provider",
			request:    chatwork.Request{Task: chatwork.TaskMessagesReconcileSend, Room: room, BodyFile: "private-path"},
			prepareErr: fault.New(fault.KindInvalidInput, "chatwork_message_body_file_unreadable", "body file unavailable", false),
			wantCode:   "chatwork_message_body_file_unreadable", wantCalls: 0, wantPrepareCalls: 1,
		},
		{
			name:     "reply read fault does not become empty result",
			request:  chatwork.Request{Task: chatwork.TaskMessagesReconcileSend, Room: room, Body: "body", MessageReplyTarget: &chatwork.MessageReplyTarget{Message: message}},
			steps:    []sendPortStep{{err: fault.New(fault.KindPermission, "chatwork_message_restricted", "private", false)}},
			wantCode: "chatwork_message_restricted", wantCalls: 1, wantPrepareCalls: 1,
		},
		{
			name:     "wrong reply identity does not list",
			request:  chatwork.Request{Task: chatwork.TaskMessagesReconcileSend, Room: room, Body: "body", MessageReplyTarget: &chatwork.MessageReplyTarget{Message: message}},
			steps:    []sendPortStep{{result: chatwork.Result{Messages: []chatwork.Message{{Ref: message, Room: room, Sender: chatwork.Account{Ref: chatwork.Reference{Kind: chatwork.ReferenceAccount, Value: "07"}}}}}}},
			wantCode: "chatwork_result_invalid", wantCalls: 1, wantPrepareCalls: 1,
		},
		{
			name:     "account lookup fault does not list",
			request:  chatwork.Request{Task: chatwork.TaskMessagesReconcileSend, Room: room, Body: "body"},
			steps:    []sendPortStep{{err: fault.New(fault.KindPermission, "account_unavailable", "unavailable", false)}},
			wantCode: "account_unavailable", wantCalls: 1, wantPrepareCalls: 1,
		},
		{
			name:     "invalid account identity does not list",
			request:  chatwork.Request{Task: chatwork.TaskMessagesReconcileSend, Room: room, Body: "body"},
			steps:    []sendPortStep{{result: chatwork.Result{Account: &chatwork.Account{Ref: chatwork.Reference{Kind: chatwork.ReferenceAccount, Value: "07"}}}}},
			wantCode: "chatwork_result_invalid", wantCalls: 1, wantPrepareCalls: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			port := &reconciliationPort{steps: test.steps, prepareError: test.prepareErr}
			_, err := New(port).Execute(context.Background(), testBinding(t), test.request)
			var structured *fault.Error
			if !errors.As(err, &structured) || structured.Code != test.wantCode {
				t.Fatalf("error = %#v, want code %q", err, test.wantCode)
			}
			if len(port.requests) != test.wantCalls || port.prepareCalls != test.wantPrepareCalls {
				t.Fatalf("requests=%d prepare=%d, want provider reads=%d and prepares=%d", len(port.requests), port.prepareCalls, test.wantCalls, test.wantPrepareCalls)
			}
			if test.wantCalls > 0 && port.discardCalls != 1 {
				t.Fatalf("prepared body discard calls = %d, want 1", port.discardCalls)
			}
			if strings.Contains(err.Error(), "private-path") || strings.Contains(err.Error(), "private-body-canary") {
				t.Fatalf("error leaked body/path: %v", err)
			}
		})
	}
}

func TestMessageSendReconciliationDropsSemanticBodyReferences(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	message := relationshipReference(t, chatwork.ReferenceMessage, "101")
	source := []chatwork.Message{{Ref: message, Room: room, Sender: chatwork.Account{Ref: account}, Body: "hello", SendTime: 1,
		BodySegments: &chatwork.MessageBodySegments{Segments: []chatwork.MessageBodySegment{{Kind: chatwork.MessageBodyText, Text: "hello", SourceEnd: 5}}}}}
	port := &reconciliationPort{steps: []sendPortStep{{result: chatwork.Result{Task: chatwork.TaskAccountShow, Account: &chatwork.Account{Ref: account}}}, {result: chatwork.Result{Task: chatwork.TaskMessagesList, MessageRoom: room, Coverage: chatwork.Coverage{Kind: "latest_window", Limit: 100}, Messages: source}}}}
	_, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{Task: chatwork.TaskMessagesReconcileSend, Room: room, Body: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if source[0].Body != "" || source[0].BodySegments != nil {
		t.Fatalf("body references retained: %+v", source[0])
	}
}
