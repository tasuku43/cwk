package chatworkcmd

import (
	"context"
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestAssembleMessageWindowContainsUsesLiteralORAndConjoinsOtherFilters(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	messages := []chatwork.Message{
		selectionMessageAt(t, "301", room, a, 200), // exact term, in period and sender set
		selectionMessageAt(t, "302", room, b, 300), // exact term, wrong sender
		selectionMessageAt(t, "303", room, a, 100), // exact term, before period
		selectionMessageAt(t, "304", room, a, 300), // case mismatch
		selectionMessageAt(t, "305", room, a, 300), // alternate OR term
		selectionMessageAt(t, "306", room, a, 250), // decomposed Unicode, no normalization
	}
	messages[0].Body = "prefix Needle suffix"
	messages[1].Body = "Needle"
	messages[2].Body = "Needle"
	messages[3].Body = "needle"
	messages[4].Body = "dog 🐕"
	messages[5].Body = "cafe\u0301"

	filter := chatwork.MessageFilter{
		Senders:    []chatwork.Reference{a},
		Contains:   []string{"Needle", "🐕", "é", "Needle"},
		Period:     chatwork.MessagePeriod{Since: 200, Until: 400},
		StartIndex: 1,
		Count:      2,
		Context:    chatwork.MessageContextNone,
	}
	selected, selection, err := assembleMessageWindow(messages, filter)
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"301", "305"}) {
		t.Fatalf("selected refs = %v, want provider order after typed-time ranking", got)
	}
	if selection == nil || selection.CandidateCount != 2 || selection.ItemsPerPage != 2 || selection.NextStartIndex != 0 ||
		!reflect.DeepEqual(selection.SourceSequences, []int{1, 5}) || !reflect.DeepEqual(selection.AnchorSequences, []int{1, 5}) {
		t.Fatalf("selection = %+v", selection)
	}
	if !reflect.DeepEqual(selection.Filter.Contains, filter.Contains) {
		t.Fatalf("stored exact terms = %#v, want %#v", selection.Filter.Contains, filter.Contains)
	}
	selection.Filter.Contains[0] = "mutated"
	if filter.Contains[0] != "Needle" {
		t.Fatal("selection aliases input term storage")
	}
}

func TestAssembleMessageWindowContainsPaginationRanksOnlyMatches(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	sender := relationshipReference(t, chatwork.ReferenceAccount, "7")
	messages := []chatwork.Message{
		selectionMessageAt(t, "301", room, sender, 200),
		selectionMessageAt(t, "302", room, sender, 300),
		selectionMessageAt(t, "303", room, sender, 100),
	}
	messages[0].Body = "hit older"
	messages[1].Body = "a hit newer"
	messages[2].Body = "miss"

	filter := chatwork.MessageFilter{
		Contains: []string{"hit"}, StartIndex: 1, Count: 1, Context: chatwork.MessageContextNone,
	}
	first, firstSelection, err := assembleMessageWindow(messages, filter)
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(first); !reflect.DeepEqual(got, []string{"302"}) || firstSelection == nil ||
		firstSelection.CandidateCount != 2 || firstSelection.NextStartIndex != 2 || !reflect.DeepEqual(firstSelection.AnchorSequences, []int{2}) {
		t.Fatalf("first page = %v, selection = %+v", got, firstSelection)
	}

	filter.StartIndex = firstSelection.NextStartIndex
	second, secondSelection, err := assembleMessageWindow(messages, filter)
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(second); !reflect.DeepEqual(got, []string{"301"}) || secondSelection == nil ||
		secondSelection.CandidateCount != 2 || secondSelection.NextStartIndex != 0 || !reflect.DeepEqual(secondSelection.AnchorSequences, []int{1}) {
		t.Fatalf("second page = %v, selection = %+v", got, secondSelection)
	}
}

func TestAssembleMessageWindowContainsAddsOnlyDirectNonmatchingReplyContext(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	parent := selectionMessage(t, "301", room, b)
	anchor := selectionReply(t, "302", room, a, "301")
	child := selectionReply(t, "303", room, b, "302")
	parent.Body, anchor.Body, child.Body = "parent context", "target phrase", "child context"
	source := []chatwork.Message{parent, anchor, child}

	selected, selection, err := assembleMessageWindow(source, chatwork.MessageFilter{
		Contains: []string{"target"}, Context: chatwork.MessageContextReplies,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"301", "302", "303"}) || selection == nil ||
		selection.CandidateCount != 1 || !reflect.DeepEqual(selection.SourceSequences, []int{1, 2, 3}) ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{2}) {
		t.Fatalf("selected=%v selection=%+v", got, selection)
	}
	if got := []string{selected[0].Body, selected[1].Body, selected[2].Body}; !reflect.DeepEqual(got, []string{"parent context", "target phrase", "child context"}) {
		t.Fatalf("context bodies = %q", got)
	}

	none, noneSelection, err := assembleMessageWindow(source, chatwork.MessageFilter{
		Contains: []string{"target"}, Context: chatwork.MessageContextNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(none); !reflect.DeepEqual(got, []string{"302"}) || noneSelection == nil ||
		!reflect.DeepEqual(noneSelection.SourceSequences, []int{2}) || !reflect.DeepEqual(noneSelection.AnchorSequences, []int{2}) {
		t.Fatalf("context=none selected=%v selection=%+v", got, noneSelection)
	}
}

func TestExecuteAppliesContainsAfterOneFilterFreeProviderCall(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	sender := relationshipReference(t, chatwork.ReferenceAccount, "7")
	port := &fakePort{result: chatwork.Result{
		MessageRoom: room,
		Coverage:    chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
		Messages: []chatwork.Message{
			{Ref: relationshipReference(t, chatwork.ReferenceMessage, "301"), Room: room, Sender: chatwork.Account{Ref: sender}, Body: "needle"},
			{Ref: relationshipReference(t, chatwork.ReferenceMessage, "302"), Room: room, Sender: chatwork.Account{Ref: sender}, Body: "not a match"},
		},
	}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room,
		MessageFilter: chatwork.MessageFilter{Contains: []string{"needle"}, Context: chatwork.MessageContextNone},
	}
	result, err := New(port).Execute(context.Background(), testBinding(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if port.calls != 1 || !reflect.DeepEqual(port.request.MessageFilter, chatwork.MessageFilter{}) {
		t.Fatalf("port calls=%d filter=%+v", port.calls, port.request.MessageFilter)
	}
	if len(result.Messages) != 1 || result.Messages[0].Ref.Value != "301" || result.MessageSelection == nil || result.MessageSelection.CandidateCount != 1 {
		t.Fatalf("result = %+v", result)
	}
}
