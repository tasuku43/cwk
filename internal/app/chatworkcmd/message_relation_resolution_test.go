package chatworkcmd

import (
	"context"
	"errors"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type relationResolutionPort struct {
	requests []chatwork.Request
	execute  func(chatwork.Request) (chatwork.Result, error)
}

func (p *relationResolutionPort) Execute(_ context.Context, _ authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.requests = append(p.requests, request)
	return p.execute(request)
}

func TestMessageRelationResolutionReusesCountOmittedSourceWithoutFetch(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	parent := relationMessage(t, room, account, "101", 100)
	child := relationMessage(t, room, account, "102", 200)
	child.Reply = &chatwork.Relation{Kind: "reply", Target: parent.Ref, ExternalID: room.Value}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task != chatwork.TaskMessagesList {
			t.Fatalf("unexpected task %s", request.Task)
		}
		return relationListResult(request, []chatwork.Message{parent, child}), nil
	}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room, ForceRecent: true,
		MessageFilter:             chatwork.MessageFilter{StartIndex: 1, Count: 1, Context: chatwork.MessageContextNone},
		MessageRelationFetchLimit: 1,
	}
	result, err := New(port).Execute(context.Background(), testBinding(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 1 {
		t.Fatalf("provider calls = %d, want list only", len(port.requests))
	}
	if len(result.Messages) != 1 || result.Messages[0].Ref != child.Ref || result.Messages[0].Reply == nil || !result.Messages[0].Reply.Resolved {
		t.Fatalf("selected child = %+v", result.Messages)
	}
	resolution := result.MessageRelationResolution
	if resolution == nil || resolution.FetchAttempts != 0 || len(resolution.Targets) != 1 ||
		resolution.Targets[0].State != chatwork.MessageRelationResolvedFromSource || resolution.Targets[0].Message == nil || resolution.Targets[0].Message.Ref != parent.Ref {
		t.Fatalf("resolution = %+v", resolution)
	}
}

func TestMessageRelationResolutionRecursivelyFetchesUniqueChainWithinBudget(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	target := relationshipReference(t, chatwork.ReferenceMessage, "999")
	grandparent := relationshipReference(t, chatwork.ReferenceMessage, "888")
	first := relationMessage(t, room, account, "101", 200)
	second := relationMessage(t, room, account, "102", 300)
	first.Reply = &chatwork.Relation{Kind: "reply", Target: target, ExternalID: room.Value}
	second.Reply = &chatwork.Relation{Kind: "reply", Target: target, ExternalID: room.Value}
	parent := relationMessage(t, room, account, target.Value, 100)
	parent.Reply = &chatwork.Relation{Kind: "reply", Target: grandparent, ExternalID: room.Value}
	root := relationMessage(t, room, account, grandparent.Value, 50)
	root.Reply = &chatwork.Relation{Kind: "reply", Target: target, ExternalID: room.Value}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		switch request.Task {
		case chatwork.TaskMessagesList:
			return relationListResult(request, []chatwork.Message{first, second}), nil
		case chatwork.TaskMessagesShow:
			switch request.Message {
			case target:
				return relationShowResult(parent), nil
			case grandparent:
				return relationShowResult(root), nil
			default:
				t.Fatalf("exact target = %v", request.Message)
				return chatwork.Result{}, nil
			}
		default:
			t.Fatalf("unexpected task %s", request.Task)
			return chatwork.Result{}, nil
		}
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room, ForceRecent: true, MessageRelationFetchLimit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 3 || port.requests[1].Message != target || port.requests[2].Message != grandparent {
		t.Fatalf("provider calls = %+v, want breadth-first chain", port.requests)
	}
	for _, message := range result.Messages {
		if message.Reply == nil || !message.Reply.Resolved {
			t.Fatalf("reply not resolved: %+v", message)
		}
	}
	resolution := result.MessageRelationResolution
	if resolution.FetchAttempts != 2 || len(resolution.Targets) != 2 ||
		resolution.Targets[0].State != chatwork.MessageRelationResolvedByFetch ||
		resolution.Targets[1].State != chatwork.MessageRelationResolvedByFetch {
		t.Fatalf("resolution = %+v", resolution)
	}
	if resolution.Targets[0].Message.Reply == nil || !resolution.Targets[0].Message.Reply.Resolved ||
		resolution.Targets[1].Message.Reply == nil || !resolution.Targets[1].Message.Reply.Resolved {
		t.Fatalf("chain or cycle state is wrong: %+v", resolution.Targets)
	}
}

func TestMessageRelationResolutionReportsUnavailableAndBudgetExhaustedTargets(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	refs := []chatwork.Reference{
		relationshipReference(t, chatwork.ReferenceMessage, "901"),
		relationshipReference(t, chatwork.ReferenceMessage, "902"),
		relationshipReference(t, chatwork.ReferenceMessage, "903"),
	}
	messages := make([]chatwork.Message, len(refs))
	for index, target := range refs {
		messages[index] = relationMessage(t, room, account, string(rune('1'+index))+"01", int64(200+index))
		messages[index].Reply = &chatwork.Relation{Kind: "reply", Target: target, ExternalID: room.Value}
	}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task == chatwork.TaskMessagesList {
			return relationListResult(request, messages), nil
		}
		switch request.Message {
		case refs[0]:
			return chatwork.Result{}, fault.New(fault.KindNotFound, "chatwork_not_found", "missing", false)
		case refs[1]:
			return chatwork.Result{}, fault.New(fault.KindPermission, "chatwork_message_restricted", "restricted", false)
		default:
			t.Fatalf("budget-exhausted target was fetched: %v", request.Message)
			return chatwork.Result{}, nil
		}
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room, ForceRecent: true, MessageRelationFetchLimit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	resolution := result.MessageRelationResolution
	want := []chatwork.MessageRelationResolutionState{
		chatwork.MessageRelationNotFound,
		chatwork.MessageRelationRestricted,
		chatwork.MessageRelationBudgetExhausted,
	}
	if resolution.FetchAttempts != 2 || len(port.requests) != 3 || len(resolution.Targets) != len(want) {
		t.Fatalf("resolution/calls = %+v / %d", resolution, len(port.requests))
	}
	for index, state := range want {
		if resolution.Targets[index].State != state || resolution.Targets[index].Target != refs[index] || resolution.Targets[index].Message != nil {
			t.Fatalf("target[%d] = %+v", index, resolution.Targets[index])
		}
	}
}

func TestMessageRelationResolutionAbortsOnTransientFetchFailure(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	target := relationshipReference(t, chatwork.ReferenceMessage, "999")
	child := relationMessage(t, room, account, "101", 200)
	child.Reply = &chatwork.Relation{Kind: "reply", Target: target, ExternalID: room.Value}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task == chatwork.TaskMessagesList {
			return relationListResult(request, []chatwork.Message{child}), nil
		}
		return chatwork.Result{}, fault.Wrap(fault.KindUnavailable, "chatwork_unavailable", "unavailable", true, errors.New("private"))
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room, ForceRecent: true, MessageRelationFetchLimit: 1,
	})
	if err == nil || result.Task != "" {
		t.Fatalf("transient fetch returned partial success: result=%+v err=%v", result, err)
	}
}

func relationMessage(t *testing.T, room, account chatwork.Reference, id string, sendTime int64) chatwork.Message {
	t.Helper()
	return chatwork.Message{
		Ref: relationshipReference(t, chatwork.ReferenceMessage, id), Room: room,
		Sender: chatwork.Account{Ref: account, Name: "Synthetic"}, Body: "body", SendTime: sendTime,
	}
}

func relationListResult(request chatwork.Request, messages []chatwork.Message) chatwork.Result {
	return chatwork.Result{
		Task: chatwork.TaskMessagesList, MessageRoom: request.Room,
		Coverage: chatwork.Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages: messages,
	}
}

func relationShowResult(message chatwork.Message) chatwork.Result {
	return chatwork.Result{
		Task:     chatwork.TaskMessagesShow,
		Coverage: chatwork.Coverage{Kind: "single_operation", Complete: true},
		Messages: []chatwork.Message{message},
	}
}
