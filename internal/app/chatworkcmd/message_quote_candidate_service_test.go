package chatworkcmd

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestMessagesListMatchesQuotesBeforeLocalFilterAndPrunesHiddenSequence(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	selectedAuthor := relationshipReference(t, chatwork.ReferenceAccount, "8")
	parent := relationMessage(t, room, author, "100", 100)
	quoting := relationMessage(t, room, selectedAuthor, "200", 200)
	quoting.Quotes = []chatwork.Relation{{Kind: "quote", Target: author, ExternalID: "100"}}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task != chatwork.TaskMessagesList || len(request.MessageFilter.Senders) != 0 || len(request.MessageFilter.Contains) != 0 {
			t.Fatalf("local quote-selection inputs leaked to provider: %+v", request)
		}
		return relationListResult(request, []chatwork.Message{parent, quoting}), nil
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room, ForceRecent: true,
		MessageFilter:             chatwork.MessageFilter{Senders: []chatwork.Reference{selectedAuthor}, Context: chatwork.MessageContextReplies},
		MessageRelationFetchLimit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 1 || len(result.Messages) != 1 || result.Messages[0].Ref != quoting.Ref {
		t.Fatalf("calls/messages = %d / %+v", len(port.requests), result.Messages)
	}
	if result.MessageRelationResolution == nil || result.MessageRelationResolution.FetchAttempts != 0 {
		t.Fatalf("quote candidate triggered an exact-message read: %+v", result.MessageRelationResolution)
	}
	match := result.Messages[0].QuoteMatches[0]
	if match.State != chatwork.MessageQuoteCandidateUniqueWindow || len(match.Candidates) != 1 ||
		match.Candidates[0].MessageRef != parent.Ref || match.Candidates[0].Sequence != 0 ||
		match.Candidates[0].IdentityProven || result.Messages[0].Quotes[0].Resolved {
		t.Fatalf("filtered quote match = %+v relation=%+v", match, result.Messages[0].Quotes[0])
	}
}

func TestMessageQuoteCandidateSequenceIsPrunedInSupplementalReplyContext(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	quoteAuthor := relationshipReference(t, chatwork.ReferenceAccount, "7")
	selectedAuthor := relationshipReference(t, chatwork.ReferenceAccount, "8")
	outside := relationMessage(t, room, quoteAuthor, "100", 100)
	parent := relationMessage(t, room, selectedAuthor, "200", 200)
	parent.Quotes = []chatwork.Relation{{Kind: "quote", Target: quoteAuthor, ExternalID: "100"}}
	child := relationMessage(t, room, selectedAuthor, "300", 300)
	child.Replies = []chatwork.Relation{{Kind: "reply", Target: parent.Ref, ExternalID: room.Value}}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if len(request.MessageFilter.Senders) != 0 || len(request.MessageFilter.Contains) != 0 {
			t.Fatalf("local selection leaked to provider: %+v", request)
		}
		return relationListResult(request, []chatwork.Message{outside, parent, child}), nil
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room, ForceRecent: true,
		MessageFilter:             chatwork.MessageFilter{Senders: []chatwork.Reference{selectedAuthor}, Context: chatwork.MessageContextNone, StartIndex: 1, Count: 1},
		MessageRelationFetchLimit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 1 || result.Messages[0].Ref != child.Ref || result.MessageRelationResolution == nil ||
		len(result.MessageRelationResolution.Targets) != 1 || result.MessageRelationResolution.Targets[0].Message == nil {
		t.Fatalf("selected result/relation context = %+v / %+v", result.Messages, result.MessageRelationResolution)
	}
	match := result.MessageRelationResolution.Targets[0].Message.QuoteMatches[0]
	if match.State != chatwork.MessageQuoteCandidateUniqueWindow || len(match.Candidates) != 1 ||
		match.Candidates[0].MessageRef != outside.Ref || match.Candidates[0].Sequence != 0 {
		t.Fatalf("supplemental quote candidate retained an unrendered sequence: %+v", match)
	}
	delivery, err := PlanMessageDelivery(result, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(delivery.RelationTargets) != 1 || delivery.RelationTargets[0].Message == nil ||
		delivery.RelationTargets[0].Message.QuoteMatches[0].Candidates[0].Sequence != 0 {
		t.Fatalf("byte-delivery relation target retained an unrendered quote sequence: %+v", delivery.RelationTargets)
	}
}

func TestMessagesShowExactOnlyLeavesQuoteCandidateUnknownWithoutRecentRead(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchor := relationMessage(t, room, relationshipReference(t, chatwork.ReferenceAccount, "8"), "200", 200)
	anchor.Quotes = []chatwork.Relation{{Kind: "quote", Target: author, ExternalID: "100"}}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task != chatwork.TaskMessagesShow || request.Message != anchor.Ref {
			t.Fatalf("unexpected exact-only provider request: %+v", request)
		}
		return relationShowResult(anchor), nil
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchor.Ref,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 1 || result.MessageShowContext != nil || result.MessageShowAround != nil {
		t.Fatalf("exact-only provider calls/context = %d / %+v", len(port.requests), result)
	}
	match := result.Messages[0].QuoteMatches[0]
	if match.State != chatwork.MessageQuoteCandidateUnknown || len(match.Candidates) != 0 {
		t.Fatalf("quote without a recent source = %+v", match)
	}
}

func TestMessagesShowRepliesUseOnlyExistingRecentSourceForQuoteCandidates(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	anchor := relationMessage(t, room, relationshipReference(t, chatwork.ReferenceAccount, "9"), "200", 200)
	child := relationMessage(t, room, relationshipReference(t, chatwork.ReferenceAccount, "8"), "300", 300)
	child.Replies = []chatwork.Relation{{Kind: "reply", Target: anchor.Ref, ExternalID: room.Value}}
	child.Quotes = []chatwork.Relation{{Kind: "quote", Target: author, ExternalID: "100"}}
	parent := relationMessage(t, room, author, "100", 100)
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		switch request.Task {
		case chatwork.TaskMessagesShow:
			if request.Message != anchor.Ref || request.MessageFilter.Context != "" || request.MessageAround != nil {
				t.Fatalf("unexpected anchor request: %+v", request)
			}
			return relationShowResult(anchor), nil
		case chatwork.TaskMessagesList:
			if !request.ForceRecent || request.MessageFilter.Context != "" || request.MessageRelationFetchLimit != 0 {
				t.Fatalf("quote matching changed the fixed recent read: %+v", request)
			}
			return relationListResult(request, []chatwork.Message{parent, anchor, child}), nil
		default:
			t.Fatalf("unexpected task %s", request.Task)
			return chatwork.Result{}, nil
		}
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchor.Ref,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextReplies},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 2 || len(result.MessageShowContext.Children) != 1 || result.MessageShowContext.Children[0].Ref != child.Ref {
		t.Fatalf("provider calls/reply context = %d / %+v", len(port.requests), result.MessageShowContext)
	}
	match := result.MessageShowContext.Children[0].QuoteMatches[0]
	if match.State != chatwork.MessageQuoteCandidateUniqueWindow || len(match.Candidates) != 1 ||
		match.Candidates[0].MessageRef != parent.Ref || match.Candidates[0].Sequence != 0 {
		t.Fatalf("replies-context quote match = %+v", match)
	}
}

func TestMessagesShowPrunesAnchorQuoteSequencesAgainstSourceSequencedVisibleMessages(t *testing.T) {
	for _, test := range []struct {
		name             string
		around           bool
		candidateVisible bool
		wantSequence     int
	}{
		{name: "replies-only hidden source candidate", wantSequence: 0},
		{name: "replies-only visible child candidate", candidateVisible: true, wantSequence: 1},
		{name: "around outside anchor has no visible candidate sequence", around: true, wantSequence: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			room := relationshipReference(t, chatwork.ReferenceRoom, "42")
			author := relationshipReference(t, chatwork.ReferenceAccount, "7")
			anchor := relationMessage(t, room, relationshipReference(t, chatwork.ReferenceAccount, "9"), "500", 500)
			anchor.Quotes = []chatwork.Relation{{Kind: "quote", Target: author, ExternalID: "100"}}
			var candidate chatwork.Message
			var recentMessages []chatwork.Message
			if test.candidateVisible {
				candidate = relationMessage(t, room, author, "100", 100)
				candidate.Replies = []chatwork.Relation{{Kind: "reply", Target: anchor.Ref, ExternalID: room.Value}}
				recentMessages = []chatwork.Message{candidate}
			} else {
				candidate = relationMessage(t, room, author, "100", 100)
				if test.around {
					recentMessages = []chatwork.Message{candidate}
				} else {
					child := relationMessage(t, room, relationshipReference(t, chatwork.ReferenceAccount, "8"), "600", 600)
					child.Replies = []chatwork.Relation{{Kind: "reply", Target: anchor.Ref, ExternalID: room.Value}}
					recentMessages = []chatwork.Message{candidate, child}
				}
			}
			port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
				switch request.Task {
				case chatwork.TaskMessagesShow:
					return relationShowResult(anchor), nil
				case chatwork.TaskMessagesList:
					if !request.ForceRecent {
						t.Fatalf("quote candidate used a non-recent source: %+v", request)
					}
					return relationListResult(request, recentMessages), nil
				default:
					t.Fatalf("unexpected provider task %s", request.Task)
					return chatwork.Result{}, nil
				}
			}}
			request := chatwork.Request{Task: chatwork.TaskMessagesShow, Room: room, Message: anchor.Ref}
			if test.around {
				request.MessageAround = &chatwork.MessageAroundRequest{Count: 1}
			} else {
				request.MessageFilter.Context = chatwork.MessageContextReplies
			}
			result, err := New(port).Execute(context.Background(), testBinding(t), request)
			if err != nil {
				t.Fatal(err)
			}
			if len(port.requests) != 2 {
				t.Fatalf("provider calls = %d, want exact plus one existing source read", len(port.requests))
			}
			match := result.Messages[0].QuoteMatches[0]
			if match.State != chatwork.MessageQuoteCandidateUniqueWindow || len(match.Candidates) != 1 ||
				match.Candidates[0].MessageRef != candidate.Ref || match.Candidates[0].Sequence != test.wantSequence {
				t.Fatalf("anchor quote candidate = %+v, want sequence %d", match, test.wantSequence)
			}
			if test.around && (result.MessageShowAround == nil || result.MessageShowAround.AnchorState != chatwork.MessageAroundAnchorOutside) {
				t.Fatalf("around anchor state = %+v, want outside-window", result.MessageShowAround)
			}
		})
	}
}

func TestMessagesShowAroundUsesExistingSourceSequenceForQuoteCandidate(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	parent := relationMessage(t, room, author, "100", 100)
	anchor := relationMessage(t, room, relationshipReference(t, chatwork.ReferenceAccount, "8"), "200", 200)
	anchor.Quotes = []chatwork.Relation{{Kind: "quote", Target: author, ExternalID: "100"}}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task == chatwork.TaskMessagesShow {
			return relationShowResult(anchor), nil
		}
		if request.Task == chatwork.TaskMessagesList && request.ForceRecent {
			return relationListResult(request, []chatwork.Message{parent, anchor}), nil
		}
		t.Fatalf("unexpected provider request: %+v", request)
		return chatwork.Result{}, nil
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesShow, Room: room, Message: anchor.Ref,
		MessageAround: &chatwork.MessageAroundRequest{Count: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.requests) != 2 || result.MessageShowAround == nil || len(result.MessageShowAround.Before) != 1 {
		t.Fatalf("provider calls/around = %d / %+v", len(port.requests), result.MessageShowAround)
	}
	match := result.Messages[0].QuoteMatches[0]
	if match.State != chatwork.MessageQuoteCandidateUniqueWindow || len(match.Candidates) != 1 ||
		match.Candidates[0].MessageRef != parent.Ref || match.Candidates[0].Sequence != 1 {
		t.Fatalf("around quote match = %+v", match)
	}
	if !reflect.DeepEqual(result.MessageShowAround.BeforeSequences, []int{1}) {
		t.Fatalf("around sequences = %v", result.MessageShowAround.BeforeSequences)
	}
}

func TestMessageQuoteMatchingRejectsSourceOverflowBeforeFiltering(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "7")
	messages := make([]chatwork.Message, chatwork.MaxMessageSelectionCount+1)
	for index := range messages {
		messages[index] = relationMessage(t, room, account, strconv.Itoa(1000+index), int64(index+1))
	}
	port := &relationResolutionPort{execute: func(request chatwork.Request) (chatwork.Result, error) {
		return relationListResult(request, messages), nil
	}}
	result, err := New(port).Execute(context.Background(), testBinding(t), chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room, ForceRecent: true,
		MessageFilter: chatwork.MessageFilter{Senders: []chatwork.Reference{account}, Context: chatwork.MessageContextNone},
	})
	if err == nil || result.Task != "" || len(port.requests) != 1 {
		t.Fatalf("overflow source returned a selected result: result=%+v err=%v calls=%d", result, err, len(port.requests))
	}
}

func TestMessageQuoteCandidateLosesSequenceWhenByteDeliveryOmitsCandidate(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	quoteAuthor := relationshipReference(t, chatwork.ReferenceAccount, "8")
	candidate := relationMessage(t, room, author, "100", 100)
	quote := relationMessage(t, room, quoteAuthor, "200", 200)
	quote.Quotes = []chatwork.Relation{{Kind: "quote", Target: author, ExternalID: "100"}}
	annotated, err := ResolveMessageQuoteCandidates([]chatwork.Message{candidate, quote}, []chatwork.Message{candidate, quote}, true, chatwork.MessageAccessNone)
	if err != nil {
		t.Fatal(err)
	}
	result := chatwork.Result{
		Task: chatwork.TaskMessagesList, MessageRoom: room,
		Coverage: chatwork.Coverage{Kind: "latest_window", Limit: 100},
		Messages: annotated,
	}
	delivery, err := PlanMessageDelivery(result, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(delivery.Messages) != 1 || delivery.Messages[0].Ref != quote.Ref {
		t.Fatalf("delivery messages = %+v", delivery.Messages)
	}
	match := delivery.Messages[0].QuoteMatches[0]
	if match.State != chatwork.MessageQuoteCandidateUniqueWindow || len(match.Candidates) != 1 ||
		match.Candidates[0].MessageRef != candidate.Ref || match.Candidates[0].Sequence != 0 {
		t.Fatalf("delivered candidate = %+v", match)
	}
}
