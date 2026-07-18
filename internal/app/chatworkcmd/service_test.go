package chatworkcmd

import (
	"context"
	"errors"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type fakePort struct {
	calls   int
	request chatwork.Request
	result  chatwork.Result
	err     error
	cancel  context.CancelFunc
}

func (p *fakePort) Execute(_ context.Context, _ authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.calls++
	p.request = request
	if p.cancel != nil {
		p.cancel()
	}
	if p.result.Task == "" {
		p.result.Task = request.Task
	}
	return p.result, p.err
}

func TestExecuteSelectsMessagesAfterOneFilterFreePortCall(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	port := &fakePort{result: chatwork.Result{
		MessageRoom: room,
		Coverage:    chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
		Messages: []chatwork.Message{
			{Ref: relationshipReference(t, chatwork.ReferenceMessage, "101"), Room: room, Sender: chatwork.Account{Ref: b}},
			{Ref: relationshipReference(t, chatwork.ReferenceMessage, "102"), Room: room, Sender: chatwork.Account{Ref: a}},
		},
	}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesList,
		Room: room,
		MessageFilter: chatwork.MessageFilter{
			Senders: []chatwork.Reference{a},
			Context: chatwork.MessageContextNone,
		},
	}

	result, err := New(port).Execute(context.Background(), testBinding(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if port.calls != 1 {
		t.Fatalf("port calls = %d, want 1", port.calls)
	}
	if len(port.request.MessageFilter.Senders) != 0 || port.request.MessageFilter.Context != "" {
		t.Fatalf("local filter leaked to port: %+v", port.request.MessageFilter)
	}
	if len(result.Messages) != 1 || result.Messages[0].Ref.Value != "102" {
		t.Fatalf("selected messages = %+v", result.Messages)
	}
	if result.MessageSelection == nil || result.MessageSelection.SourceCount != 2 ||
		len(result.MessageSelection.SourceSequences) != 1 || result.MessageSelection.SourceSequences[0] != 2 ||
		len(result.MessageSelection.AnchorSequences) != 1 || result.MessageSelection.AnchorSequences[0] != 2 {
		t.Fatalf("selection = %+v", result.MessageSelection)
	}
}

func TestExecuteResolvesUnfilteredMessageWindowWithoutSelection(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	parent := relationshipReference(t, chatwork.ReferenceMessage, "101")
	port := &fakePort{result: chatwork.Result{
		MessageRoom: room,
		Coverage:    chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
		Messages: []chatwork.Message{
			{Ref: parent, Room: room, Sender: chatwork.Account{Ref: relationshipReference(t, chatwork.ReferenceAccount, "7")}},
			{
				Ref: relationshipReference(t, chatwork.ReferenceMessage, "102"), Room: room,
				Sender: chatwork.Account{Ref: relationshipReference(t, chatwork.ReferenceAccount, "8")},
				Reply:  &chatwork.Relation{Kind: "reply", Target: parent, ExternalID: room.Value},
			},
		},
	}}

	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{Task: chatwork.TaskMessagesList, Room: room})
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageSelection != nil {
		t.Fatalf("unfiltered selection = %+v, want nil", result.MessageSelection)
	}
	if result.Messages[1].Reply == nil || !result.Messages[1].Reply.Resolved {
		t.Fatalf("unfiltered reply = %+v, want resolved", result.Messages[1].Reply)
	}
}

func TestExecuteKeepsMessagesShowRelationValidationAfterCLIOwnershipMove(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := chatwork.Message{
		Ref: relationshipReference(t, chatwork.ReferenceMessage, "102"), Room: room,
		Sender: chatwork.Account{Ref: relationshipReference(t, chatwork.ReferenceAccount, "8")},
		Reply: &chatwork.Relation{
			Kind: "reply", Target: relationshipReference(t, chatwork.ReferenceMessage, "101"),
			ExternalID: room.Value, Resolved: true,
		},
	}
	port := &fakePort{result: chatwork.Result{Messages: []chatwork.Message{message}}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room,
		Message: relationshipReference(t, chatwork.ReferenceMessage, "102"),
	}

	if _, err := New(port).Execute(context.Background(), testBinding(t), request); err == nil {
		t.Fatal("messages.show accepted a resolved reply whose parent is absent from the result")
	}
	if port.calls != 1 {
		t.Fatalf("port calls = %d, want 1", port.calls)
	}
}

func testBinding(t *testing.T) authn.BindingID {
	t.Helper()
	binding, err := authn.NewBindingID("test-binding")
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func TestExecuteReturnsMatchingResult(t *testing.T) {
	port := &fakePort{result: chatwork.Result{Rooms: []chatwork.Room{}}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{Task: chatwork.TaskRoomsList})
	if err != nil {
		t.Fatal(err)
	}
	if result.Task != chatwork.TaskRoomsList || port.calls != 1 {
		t.Fatalf("result = %+v, calls = %d", result, port.calls)
	}
}

func TestExecuteRejectsInvalidTaskSpecificResult(t *testing.T) {
	port := &fakePort{result: chatwork.Result{
		Created: []chatwork.Reference{{Kind: chatwork.ReferenceMessage, Value: "3"}},
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesSend,
		Room: chatwork.Reference{Kind: chatwork.ReferenceRoom, Value: "2"},
	})
	var got *fault.Error
	if !errors.As(err, &got) || got.Code != "chatwork_result_invalid" || result.Task != "" || port.calls != 1 {
		t.Fatalf("result = %+v, err = %#v, calls = %d", result, err, port.calls)
	}
}

func TestExecuteRejectsResultIdentityThatDoesNotMatchRequest(t *testing.T) {
	room2 := chatwork.Reference{Kind: chatwork.ReferenceRoom, Value: "2"}
	room9 := chatwork.Reference{Kind: chatwork.ReferenceRoom, Value: "9"}
	message3 := chatwork.Reference{Kind: chatwork.ReferenceMessage, Value: "3"}
	message8 := chatwork.Reference{Kind: chatwork.ReferenceMessage, Value: "8"}
	tests := []struct {
		name    string
		request chatwork.Request
		result  chatwork.Result
	}{
		{
			name:    "created parent room",
			request: chatwork.Request{Task: chatwork.TaskMessagesSend, Room: room2},
			result: chatwork.Result{Task: chatwork.TaskMessagesSend, CreatedInRoom: &chatwork.RoomScopedCreation{
				Refs: []chatwork.Reference{message3}, ParentRoom: room9,
			}},
		},
		{
			name:    "acknowledged target",
			request: chatwork.Request{Task: chatwork.TaskRoomsDelete, Room: room2},
			result: chatwork.Result{Task: chatwork.TaskRoomsDelete, Acknowledgement: &chatwork.Acknowledgement{
				Acknowledged: true, Target: room9,
			}},
		},
		{
			name:    "affected target",
			request: chatwork.Request{Task: chatwork.TaskMessagesUpdate, Room: room2, Message: message3},
			result:  chatwork.Result{Task: chatwork.TaskMessagesUpdate, Affected: []chatwork.Reference{message8}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			port := &fakePort{result: test.result}
			result, err := New(port).Execute(context.Background(), testBinding(t), test.request)
			var got *fault.Error
			if !errors.As(err, &got) || got.Code != "chatwork_result_invalid" || result.Task != "" || port.calls != 1 {
				t.Fatalf("result = %+v, err = %#v, calls = %d", result, err, port.calls)
			}
		})
	}
}

func TestExecuteRejectsBeforePort(t *testing.T) {
	tests := map[string]struct {
		ctx     context.Context
		binding authn.BindingID
		request chatwork.Request
	}{
		"nil context":     {ctx: nil, binding: testBinding(t), request: chatwork.Request{Task: chatwork.TaskRoomsList}},
		"missing binding": {ctx: context.Background(), request: chatwork.Request{Task: chatwork.TaskRoomsList}},
		"invalid task":    {ctx: context.Background(), binding: testBinding(t), request: chatwork.Request{}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			port := &fakePort{}
			if _, err := New(port).Execute(test.ctx, test.binding, test.request); err == nil {
				t.Fatal("Execute() succeeded")
			}
			if port.calls != 0 {
				t.Fatalf("port calls = %d", port.calls)
			}
		})
	}
}

func TestExecutePreservesStructuredFaultAndSanitizesRawError(t *testing.T) {
	structured := fault.Wrap(fault.KindRateLimited, "chatwork_rate_limited", "Chatwork rate limit was reached", true, errors.New("secret body"))
	for name, test := range map[string]struct {
		err      error
		wantCode string
	}{
		"structured": {err: structured, wantCode: "chatwork_rate_limited"},
		"raw":        {err: errors.New("secret body"), wantCode: "unclassified_chatwork_error"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := New(&fakePort{err: test.err}).Execute(context.Background(), testBinding(t), chatwork.Request{Task: chatwork.TaskRoomsList})
			var got *fault.Error
			if !errors.As(err, &got) || got.Code != test.wantCode || errors.Unwrap(got) != nil {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestExecuteSuppressesResultWhenPortIgnoresCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	port := &fakePort{cancel: cancel}
	result, err := New(port).Execute(ctx, testBinding(t), chatwork.Request{Task: chatwork.TaskRoomsList})
	if err == nil || result.Task != "" || port.calls != 1 {
		t.Fatalf("result = %+v, err = %v, calls = %d", result, err, port.calls)
	}
}
