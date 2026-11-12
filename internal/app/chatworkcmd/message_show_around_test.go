package chatworkcmd

import (
	"context"
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestMessagesShowAroundAndRepliesShareOneRecentReadAndDeduplicateVisibleMessages(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchorRef := relationshipReference(t, chatwork.ReferenceMessage, "100")
	anchor := relationMessage(t, room, account, anchorRef.Value, 300)
	anchor.Body = "exact anchor"
	parent := relationMessage(t, room, account, "99", 200)
	parent.Body = "parent before"
	parent.Replies = []chatwork.Relation{{Kind: "reply", Target: relationshipReference(t, chatwork.ReferenceMessage, "98"), ExternalID: room.Value}}
	anchor.Replies = []chatwork.Relation{{Kind: "reply", Target: parent.Ref, ExternalID: room.Value}}
	anchorInRecent := anchor
	anchorInRecent.Body = "stale source copy"
	child := relationMessage(t, room, account, "101", 400)
	child.Body = "typed direct child"
	child.Replies = []chatwork.Relation{{Kind: "reply", Target: anchorRef, ExternalID: room.Value}}
	after := relationMessage(t, room, account, "102", 500)
	after.Body = "[rp rid=42 mid=100] adjacency is not an edge"

	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.MessageAround != nil || !reflect.DeepEqual(request.MessageFilter, chatwork.MessageFilter{}) || request.MessageRelationFetchLimit != 0 {
			t.Fatalf("local message context crossed the provider boundary: %+v", request)
		}
		if request.Task == chatwork.TaskMessagesShow {
			if request.Message != anchorRef || request.MessageAround != nil {
				t.Fatalf("exact anchor request = %+v", request)
			}
			return relationShowResult(anchor), nil
		}
		if request.Task != chatwork.TaskMessagesList || !request.ForceRecent || request.Room != room {
			t.Fatalf("recent request = %+v", request)
		}
		return relationListResult(request, []chatwork.Message{
			relationMessage(t, room, account, "98", 100), parent, anchorInRecent, child, after,
		}), nil
	}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchorRef,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextReplies},
		MessageAround: &chatwork.MessageAroundRequest{Count: 1},
	}
	result, err := New(port).Execute(context.Background(), testBinding(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 2 || port.requests[0].Task != chatwork.TaskMessagesShow || port.requests[0].Message != anchorRef ||
		port.requests[1].Task != chatwork.TaskMessagesList {
		t.Fatalf("exact/list request sequence = %+v", port.requests)
	}
	if result.Messages[0].Body != "exact anchor" {
		t.Fatalf("recent source replaced exact anchor: %+v", result.Messages[0])
	}
	around := result.MessageShowAround
	if around == nil || around.AnchorState != chatwork.MessageAroundAnchorInWindow || around.AnchorSourceSequence != 3 ||
		around.Requested != 1 || around.ActualBefore != 1 || around.ActualAfter != 1 ||
		len(around.Before) != 1 || around.Before[0].Ref.Value != "99" || around.BeforeSequences[0] != 2 ||
		len(around.After) != 1 || around.After[0].Ref.Value != "101" || around.AfterSequences[0] != 4 {
		t.Fatalf("around result = %+v", around)
	}
	if result.MessageShowContext == nil || len(result.MessageShowContext.Children) != 1 || result.MessageShowContext.Children[0].Ref != child.Ref {
		t.Fatalf("typed direct child context = %+v", result.MessageShowContext)
	}
	if !result.Messages[0].Replies[0].Resolved || !around.After[0].Replies[0].Resolved {
		t.Fatalf("explicit visible reply edges were not retained: anchor=%+v around=%+v", result.Messages[0].Replies, around.After[0].Replies)
	}
	if len(parent.Replies) == 0 || parent.Replies[0].Resolved {
		t.Fatalf("parent edge to an unselected earlier message was inferred: %+v", parent.Replies)
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("around result does not satisfy request: %v", err)
	}
}

func TestMessagesShowAroundOnlyUsesOneRecentReadWithExplicitContextNone(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchorRef := relationshipReference(t, chatwork.ReferenceMessage, "100")
	anchor := relationMessage(t, room, account, anchorRef.Value, 200)
	before := relationMessage(t, room, account, "99", 100)
	child := relationMessage(t, room, account, "101", 300)
	child.Replies = []chatwork.Relation{{Kind: "reply", Target: anchorRef, ExternalID: room.Value}}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task == chatwork.TaskMessagesShow {
			return relationShowResult(anchor), nil
		}
		if request.Task != chatwork.TaskMessagesList || request.MessageAround != nil || request.MessageFilter.Context != "" {
			t.Fatalf("around request leaked to provider: %+v", request)
		}
		return relationListResult(request, []chatwork.Message{before, anchor, child}), nil
	}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchorRef,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextNone},
		MessageAround: &chatwork.MessageAroundRequest{Count: 1},
	}
	result, err := New(port).Execute(context.Background(), testBinding(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 2 || result.MessageShowContext != nil || result.MessageShowAround == nil ||
		result.MessageShowAround.ActualBefore != 1 || result.MessageShowAround.ActualAfter != 1 ||
		result.MessageShowAround.Before[0].Ref != before.Ref || result.MessageShowAround.After[0].Ref != child.Ref {
		t.Fatalf("explicit none/around result = calls %+v, context %+v, around %+v", port.requests, result.MessageShowContext, result.MessageShowAround)
	}
	if !result.MessageShowAround.After[0].Replies[0].Resolved {
		t.Fatalf("explicit edge in around source was not retained: %+v", result.MessageShowAround.After[0].Replies)
	}
}

func TestMessagesShowAroundAbsentAnchorUsesTypedOutsideOrUnknownWithoutNeighbors(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchorRef := relationshipReference(t, chatwork.ReferenceMessage, "100")
	anchor := relationMessage(t, room, account, anchorRef.Value, 200)
	unrelated := relationMessage(t, room, account, "101", 300)
	for _, test := range []struct {
		name   string
		access chatwork.MessageAccessLimitation
		state  chatwork.MessageAroundAnchorState
	}{
		{name: "unrestricted absence", access: chatwork.MessageAccessNone, state: chatwork.MessageAroundAnchorOutside},
		{name: "restricted absence", access: chatwork.MessageAccessPartial, state: chatwork.MessageAroundAnchorUnknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
				if request.Task == chatwork.TaskMessagesShow {
					return relationShowResult(anchor), nil
				}
				result := relationListResult(request, []chatwork.Message{unrelated})
				result.MessageAccess = test.access
				return result, nil
			}}
			request := chatwork.Request{
				Task: chatwork.TaskMessagesShow, Room: room, Message: anchorRef,
				MessageAround: &chatwork.MessageAroundRequest{Count: 2},
			}
			result, err := New(port).Execute(context.Background(), testBinding(t), request)
			if err != nil {
				t.Fatal(err)
			}
			around := result.MessageShowAround
			if around == nil || around.AnchorState != test.state || around.AnchorSourceSequence != 0 ||
				around.ActualBefore != 0 || around.ActualAfter != 0 || len(around.Before) != 0 || len(around.After) != 0 ||
				around.Before == nil || around.After == nil {
				t.Fatalf("absent-anchor around result = %+v", around)
			}
		})
	}
}

func TestMessagesShowAroundRejectsOutOfRangeBeforeProviderCalls(t *testing.T) {
	port := &relationResolutionPort{}
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	anchor := relationshipReference(t, chatwork.ReferenceMessage, "100")
	_, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchor,
		MessageAround: &chatwork.MessageAroundRequest{Count: chatwork.MaxMessageAroundCount + 1},
	})
	if err == nil || len(port.requests) != 0 {
		t.Fatalf("out-of-range around crossed provider boundary: calls=%d error=%v", len(port.requests), err)
	}
}
