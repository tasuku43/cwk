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
	child.Replies = []chatwork.Relation{{Kind: "reply", Target: parent.Ref, ExternalID: room.Value}}
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
	if len(result.Messages) != 1 || result.Messages[0].Ref != child.Ref || len(result.Messages[0].Replies) != 1 || !result.Messages[0].Replies[0].Resolved {
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
	first.Replies = []chatwork.Relation{{Kind: "reply", Target: target, ExternalID: room.Value}}
	second.Replies = []chatwork.Relation{{Kind: "reply", Target: target, ExternalID: room.Value}}
	parent := relationMessage(t, room, account, target.Value, 100)
	parent.Replies = []chatwork.Relation{{Kind: "reply", Target: grandparent, ExternalID: room.Value}}
	root := relationMessage(t, room, account, grandparent.Value, 50)
	root.Replies = []chatwork.Relation{{Kind: "reply", Target: target, ExternalID: room.Value}}
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
		if len(message.Replies) != 1 || !message.Replies[0].Resolved {
			t.Fatalf("reply not resolved: %+v", message)
		}
	}
	resolution := result.MessageRelationResolution
	if resolution.FetchAttempts != 2 || len(resolution.Targets) != 2 ||
		resolution.Targets[0].State != chatwork.MessageRelationResolvedByFetch ||
		resolution.Targets[1].State != chatwork.MessageRelationResolvedByFetch {
		t.Fatalf("resolution = %+v", resolution)
	}
	if len(resolution.Targets[0].Message.Replies) != 1 || !resolution.Targets[0].Message.Replies[0].Resolved ||
		len(resolution.Targets[1].Message.Replies) != 1 || !resolution.Targets[1].Message.Replies[0].Resolved {
		t.Fatalf("chain or cycle state is wrong: %+v", resolution.Targets)
	}
}

func TestMessageRelationResolutionFetchesEveryMultiReplyBranchInOrder(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	firstTarget := relationshipReference(t, chatwork.ReferenceMessage, "901")
	secondTarget := relationshipReference(t, chatwork.ReferenceMessage, "902")
	child := relationMessage(t, room, account, "101", 300)
	child.Replies = []chatwork.Relation{
		{Kind: "reply", Target: firstTarget, ExternalID: room.Value},
		{Kind: "reply", Target: secondTarget, ExternalID: room.Value},
	}
	first := relationMessage(t, room, account, firstTarget.Value, 100)
	second := relationMessage(t, room, account, secondTarget.Value, 200)
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task == chatwork.TaskMessagesList {
			return relationListResult(request, []chatwork.Message{child}), nil
		}
		switch request.Message {
		case firstTarget:
			return relationShowResult(first), nil
		case secondTarget:
			return relationShowResult(second), nil
		default:
			t.Fatalf("unexpected target %v", request.Message)
			return chatwork.Result{}, nil
		}
	}}

	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room, ForceRecent: true, MessageRelationFetchLimit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 3 || port.requests[1].Message != firstTarget || port.requests[2].Message != secondTarget {
		t.Fatalf("provider calls = %+v, want both branches in provider order", port.requests)
	}
	if len(result.Messages[0].Replies) != 2 || !result.Messages[0].Replies[0].Resolved || !result.Messages[0].Replies[1].Resolved {
		t.Fatalf("replies = %+v", result.Messages[0].Replies)
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
		messages[index].Replies = []chatwork.Relation{{Kind: "reply", Target: target, ExternalID: room.Value}}
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
	child.Replies = []chatwork.Relation{{Kind: "reply", Target: target, ExternalID: room.Value}}
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

func TestMessagesShowRepliesResolveTypedSameRoomChainAndStopCycle(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchorRef := relationshipReference(t, chatwork.ReferenceMessage, "103")
	parentRef := relationshipReference(t, chatwork.ReferenceMessage, "102")
	rootRef := relationshipReference(t, chatwork.ReferenceMessage, "101")
	anchor := relationMessage(t, room, account, anchorRef.Value, 300)
	anchor.Replies = []chatwork.Relation{{Kind: "reply", Target: parentRef, ExternalID: room.Value}}
	parent := relationMessage(t, room, account, parentRef.Value, 200)
	parent.Replies = []chatwork.Relation{{Kind: "reply", Target: rootRef, ExternalID: room.Value}}
	root := relationMessage(t, room, account, rootRef.Value, 100)
	root.Replies = []chatwork.Relation{{Kind: "reply", Target: anchorRef, ExternalID: room.Value}}
	child := relationMessage(t, room, account, "104", 400)
	child.Replies = []chatwork.Relation{{Kind: "reply", Target: anchorRef, ExternalID: room.Value}}

	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task == chatwork.TaskMessagesList {
			if !request.ForceRecent || request.Message != (chatwork.Reference{}) || len(request.MessageFilter.Senders) != 0 ||
				request.MessageFilter.Period != (chatwork.MessagePeriod{}) || request.MessageFilter.Context != "" ||
				request.MessageFilter.StartIndex != 0 || request.MessageFilter.Count != 0 || request.MessageRelationFetchLimit != 0 {
				t.Fatalf("local show context leaked to provider list request: %+v", request)
			}
			return relationListResult(request, []chatwork.Message{child}), nil
		}
		switch request.Message {
		case anchorRef:
			return relationShowResult(anchor), nil
		case parentRef:
			return relationShowResult(parent), nil
		case rootRef:
			return relationShowResult(root), nil
		default:
			t.Fatalf("unexpected exact target %v", request.Message)
			return chatwork.Result{}, nil
		}
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchorRef,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextReplies}, MessageRelationFetchLimit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 4 || port.requests[0].Message != anchorRef || port.requests[1].Task != chatwork.TaskMessagesList || port.requests[2].Message != parentRef || port.requests[3].Message != rootRef {
		t.Fatalf("provider requests = %+v", port.requests)
	}
	if len(result.Messages) != 1 || result.Messages[0].Ref != anchorRef || !result.Messages[0].Replies[0].Resolved {
		t.Fatalf("anchor changed or remained unresolved: %+v", result.Messages)
	}
	resolution := result.MessageRelationResolution
	if resolution == nil || resolution.FetchAttempts != 2 || len(resolution.Targets) != 2 ||
		resolution.Targets[0].Target != parentRef || resolution.Targets[1].Target != rootRef ||
		resolution.Targets[0].Message == nil || resolution.Targets[1].Message == nil {
		t.Fatalf("resolution = %+v", resolution)
	}
	if !resolution.Targets[0].Message.Replies[0].Resolved || !resolution.Targets[1].Message.Replies[0].Resolved {
		t.Fatalf("cycle relation state = %+v", resolution)
	}
	if result.MessageShowContext == nil || len(result.MessageShowContext.Children) != 1 || result.MessageShowContext.Children[0].Ref != child.Ref ||
		!result.MessageShowContext.Children[0].Replies[0].Resolved {
		t.Fatalf("direct reply children = %+v", result.MessageShowContext)
	}
}

func TestMessagesShowRepliesRetainNotFoundAndRespectFetchBudget(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchorRef := relationshipReference(t, chatwork.ReferenceMessage, "103")
	first := relationshipReference(t, chatwork.ReferenceMessage, "101")
	second := relationshipReference(t, chatwork.ReferenceMessage, "102")
	anchor := relationMessage(t, room, account, anchorRef.Value, 300)
	anchor.Replies = []chatwork.Relation{
		{Kind: "reply", Target: first, ExternalID: room.Value},
		{Kind: "reply", Target: second, ExternalID: room.Value},
	}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task == chatwork.TaskMessagesList {
			return relationListResult(request, []chatwork.Message{}), nil
		}
		if request.Message == anchorRef {
			return relationShowResult(anchor), nil
		}
		if request.Message == first {
			return chatwork.Result{}, fault.New(fault.KindNotFound, "chatwork_not_found", "missing", false)
		}
		t.Fatalf("budget-exhausted target was fetched: %v", request.Message)
		return chatwork.Result{}, nil
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchorRef,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextReplies}, MessageRelationFetchLimit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	resolution := result.MessageRelationResolution
	if len(port.requests) != 3 || port.requests[1].Task != chatwork.TaskMessagesList || resolution == nil || resolution.FetchAttempts != 1 || len(resolution.Targets) != 2 ||
		resolution.Targets[0].State != chatwork.MessageRelationNotFound || resolution.Targets[1].State != chatwork.MessageRelationBudgetExhausted {
		t.Fatalf("requests/resolution = %d / %+v", len(port.requests), resolution)
	}
}

func TestMessagesShowRepliesWithZeroRelationBudgetStillFindsOnlyTypedDirectChildren(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchorRef := relationshipReference(t, chatwork.ReferenceMessage, "100")
	anchor := relationMessage(t, room, account, anchorRef.Value, 100)
	typed := relationMessage(t, room, account, "101", 200)
	typed.Replies = []chatwork.Relation{{Kind: "reply", Target: anchorRef, ExternalID: room.Value}}
	typedSecond := relationMessage(t, room, account, "104", 500)
	typedSecond.Replies = []chatwork.Relation{{Kind: "reply", Target: anchorRef, ExternalID: room.Value}}
	wrongRoom := relationMessage(t, room, account, "102", 300)
	wrongRoom.Replies = []chatwork.Relation{{Kind: "reply", Target: anchorRef, ExternalID: "999"}}
	prose := relationMessage(t, room, account, "103", 400)
	prose.Body = "[rp rid=42 mid=100] reply to 100"

	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task == chatwork.TaskMessagesShow {
			return relationShowResult(anchor), nil
		}
		return relationListResult(request, []chatwork.Message{typed, anchor, wrongRoom, prose, typedSecond}), nil
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchorRef,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextReplies}, MessageRelationFetchLimit: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 2 || port.requests[1].Task != chatwork.TaskMessagesList || !port.requests[1].ForceRecent {
		t.Fatalf("provider requests = %+v", port.requests)
	}
	if result.MessageRelationResolution != nil || result.MessageShowContext == nil || len(result.MessageShowContext.Children) != 2 ||
		result.MessageShowContext.Children[0].Ref != typed.Ref || result.MessageShowContext.Children[1].Ref != typedSecond.Ref ||
		result.MessageShowContext.SourceSequences[0] != 1 || result.MessageShowContext.SourceSequences[1] != 5 {
		t.Fatalf("typed direct children = %+v resolution=%+v", result.MessageShowContext, result.MessageRelationResolution)
	}
}

func TestMessagesShowZeroBudgetDoesNotResolveTargetsHiddenByItsVisibleContext(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchorRef := relationshipReference(t, chatwork.ReferenceMessage, "100")
	parentRef := relationshipReference(t, chatwork.ReferenceMessage, "99")
	otherRef := relationshipReference(t, chatwork.ReferenceMessage, "98")
	anchor := relationMessage(t, room, account, anchorRef.Value, 300)
	anchor.Replies = []chatwork.Relation{{Kind: "reply", Target: parentRef, ExternalID: room.Value}}
	child := relationMessage(t, room, account, "101", 400)
	child.Replies = []chatwork.Relation{
		{Kind: "reply", Target: anchorRef, ExternalID: room.Value},
		{Kind: "reply", Target: otherRef, ExternalID: room.Value},
	}
	parent := relationMessage(t, room, account, parentRef.Value, 200)
	other := relationMessage(t, room, account, otherRef.Value, 100)

	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task == chatwork.TaskMessagesShow {
			return relationShowResult(anchor), nil
		}
		return relationListResult(request, []chatwork.Message{other, parent, anchor, child}), nil
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchorRef,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextReplies}, MessageRelationFetchLimit: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageRelationResolution != nil || result.Messages[0].Replies[0].Resolved {
		t.Fatalf("hidden anchor parent was published as resolved: messages=%+v resolution=%+v", result.Messages, result.MessageRelationResolution)
	}
	if result.MessageShowContext == nil || len(result.MessageShowContext.Children) != 1 {
		t.Fatalf("direct child context = %+v", result.MessageShowContext)
	}
	replies := result.MessageShowContext.Children[0].Replies
	if len(replies) != 2 || !replies[0].Resolved || replies[1].Resolved {
		t.Fatalf("child relations were not rebound to visible context: %+v", replies)
	}
}

func TestMessagesShowRepliesPreservesIncompleteRestrictedRecentCoverage(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchor := relationMessage(t, room, account, "100", 100)
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task == chatwork.TaskMessagesShow {
			return relationShowResult(anchor), nil
		}
		result := relationListResult(request, []chatwork.Message{relationMessage(t, room, account, "99", 50)})
		result.MessageAccess = chatwork.MessageAccessPartial
		return result, nil
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchor.Ref,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextReplies},
	})
	if err != nil {
		t.Fatal(err)
	}
	context := result.MessageShowContext
	if context == nil || len(context.Children) != 0 || context.SourceCount != 1 || context.Coverage.Complete || context.Coverage.Limit != 100 || context.Access != chatwork.MessageAccessPartial {
		t.Fatalf("show context coverage = %+v", context)
	}
}

func TestMessagesShowNoneKeepsSingleExactRead(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchorRef := relationshipReference(t, chatwork.ReferenceMessage, "102")
	parentRef := relationshipReference(t, chatwork.ReferenceMessage, "101")
	anchor := relationMessage(t, room, account, anchorRef.Value, 200)
	anchor.Replies = []chatwork.Relation{{Kind: "reply", Target: parentRef, ExternalID: room.Value}}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Message != anchorRef {
			t.Fatalf("unexpected context fetch %v", request.Message)
		}
		return relationShowResult(anchor), nil
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchorRef,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextNone},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 1 || result.MessageRelationResolution != nil || result.Messages[0].Replies[0].Resolved {
		t.Fatalf("provider requests/result = %d / %+v", len(port.requests), result)
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
