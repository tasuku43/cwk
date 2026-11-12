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

type sendPortStep struct {
	result chatwork.Result
	err    error
	onCall func(context.Context)
}

type sendSequencePort struct {
	steps    []sendPortStep
	requests []chatwork.Request
	bindings []authn.BindingID
	contexts []context.Context
}

func (p *sendSequencePort) Execute(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
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

func TestMessageSendReplyPreflightUsesSameRoomBindingAndContextOnce(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "101")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	recipients := []chatwork.Reference{
		relationshipReference(t, chatwork.ReferenceAccount, "9"),
		relationshipReference(t, chatwork.ReferenceAccount, "3"),
	}
	ctx := context.WithValue(context.Background(), struct{}{}, "same-context")
	binding := testBinding(t)
	port := &sendSequencePort{steps: []sendPortStep{
		{result: chatwork.Result{Messages: []chatwork.Message{{
			Ref: message, Room: room, Sender: chatwork.Account{Ref: account}, Body: "original", RelationState: chatwork.MessageRelationsComplete,
		}}}},
		{result: chatwork.Result{CreatedInRoom: &chatwork.RoomScopedCreation{
			Refs: []chatwork.Reference{relationshipReference(t, chatwork.ReferenceMessage, "102")}, ParentRoom: room,
		}}},
	}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesSend, Room: room, Body: "body",
		MessageRecipients: recipients, MessageReplyTarget: &chatwork.MessageReplyTarget{Message: message},
	}
	result, err := New(port).Execute(ctx, binding, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Task != chatwork.TaskMessagesSend || len(port.requests) != 2 {
		t.Fatalf("result = %+v, provider calls = %d", result, len(port.requests))
	}
	if port.requests[0].Task != chatwork.TaskMessagesShow || port.requests[0].Room != room || port.requests[0].Message != message {
		t.Fatalf("reply preflight request = %+v", port.requests[0])
	}
	send := port.requests[1]
	if send.Task != chatwork.TaskMessagesSend || send.Room != room || send.Body != "body" ||
		!equalReferences(send.MessageRecipients, recipients) || send.MessageReplyTarget == nil ||
		send.MessageReplyTarget.Message != message || send.MessageReplyTarget.Sender != account {
		t.Fatalf("send request = %+v", send)
	}
	for index := range port.bindings {
		if port.bindings[index] != binding || port.contexts[index] != ctx {
			t.Fatalf("call %d did not use the same validated binding/context", index)
		}
	}
}

func TestMessageSendWithoutReplyDoesNotReadBeforeSending(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "102")
	recipients := []chatwork.Reference{relationshipReference(t, chatwork.ReferenceAccount, "9")}
	port := &sendSequencePort{steps: []sendPortStep{{result: chatwork.Result{CreatedInRoom: &chatwork.RoomScopedCreation{
		Refs: []chatwork.Reference{message}, ParentRoom: room,
	}}}}}
	_, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesSend, Room: room, Body: "body", MessageRecipients: recipients,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 1 || port.requests[0].Task != chatwork.TaskMessagesSend ||
		!equalReferences(port.requests[0].MessageRecipients, recipients) || port.requests[0].MessageReplyTarget != nil {
		t.Fatalf("provider calls = %+v, want one direct send and no read", port.requests)
	}
}

func TestMessageSendReplyPreflightFailuresNeverSend(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := relationshipReference(t, chatwork.ReferenceMessage, "101")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	otherRoom := relationshipReference(t, chatwork.ReferenceRoom, "43")
	otherMessage := relationshipReference(t, chatwork.ReferenceMessage, "102")
	ctxCanceled, cancel := context.WithCancel(context.Background())
	defer cancel()
	secretCanary := "provider-private-canary"
	tests := map[string]struct {
		ctx      context.Context
		step     sendPortStep
		wantCode string
	}{
		"provider 404":           {step: sendPortStep{err: fault.New(fault.KindPermission, "chatwork_message_restricted", "message unavailable", false)}, wantCode: "chatwork_message_restricted"},
		"provider not found 404": {step: sendPortStep{err: fault.New(fault.KindNotFound, "chatwork_not_found", "message unavailable", false)}, wantCode: "chatwork_not_found"},
		"provider raw error":     {step: sendPortStep{err: errors.New(secretCanary)}, wantCode: "chatwork_reply_preflight_failed"},
		"canceled after preflight": {ctx: ctxCanceled, step: sendPortStep{
			result: chatwork.Result{Messages: []chatwork.Message{{Ref: message, Room: room, Sender: chatwork.Account{Ref: account}, RelationState: chatwork.MessageRelationsComplete}}},
			onCall: func(context.Context) { cancel() },
		}, wantCode: "chatwork_send_preflight_not_completed"},
		"wrong room":               {step: sendPortStep{result: chatwork.Result{Messages: []chatwork.Message{{Ref: message, Room: otherRoom, Sender: chatwork.Account{Ref: account}, RelationState: chatwork.MessageRelationsComplete}}}}, wantCode: "chatwork_reply_preflight_invalid"},
		"wrong message":            {step: sendPortStep{result: chatwork.Result{Messages: []chatwork.Message{{Ref: otherMessage, Room: room, Sender: chatwork.Account{Ref: account}, RelationState: chatwork.MessageRelationsComplete}}}}, wantCode: "chatwork_reply_preflight_invalid"},
		"invalid sender reference": {step: sendPortStep{result: chatwork.Result{Messages: []chatwork.Message{{Ref: message, Room: room, Sender: chatwork.Account{Ref: chatwork.Reference{Kind: chatwork.ReferenceAccount, Value: "07"}}, RelationState: chatwork.MessageRelationsComplete}}}}, wantCode: "chatwork_reply_preflight_invalid"},
		"missing exact message":    {step: sendPortStep{result: chatwork.Result{Messages: []chatwork.Message{}}}, wantCode: "chatwork_reply_preflight_invalid"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := test.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			port := &sendSequencePort{steps: []sendPortStep{test.step}}
			_, err := New(port).Execute(ctx, testBinding(t), chatwork.Request{
				Task: chatwork.TaskMessagesSend, Room: room, Body: "body",
				MessageReplyTarget: &chatwork.MessageReplyTarget{Message: message},
			})
			var got *fault.Error
			if !errors.As(err, &got) || got.Code != test.wantCode {
				t.Fatalf("error = %#v, want code %q", err, test.wantCode)
			}
			if len(port.requests) != 1 || port.requests[0].Task != chatwork.TaskMessagesShow {
				t.Fatalf("preflight failure made provider calls: %+v", port.requests)
			}
			if strings.Contains(err.Error(), secretCanary) {
				t.Fatalf("public error leaked provider detail: %v", err)
			}
		})
	}
}

func equalReferences(left, right []chatwork.Reference) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
