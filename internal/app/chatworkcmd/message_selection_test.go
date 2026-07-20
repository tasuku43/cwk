package chatworkcmd

import (
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestAssembleMessageWindowSelectsSenderORAndPreservesSourceSequences(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	c := relationshipReference(t, chatwork.ReferenceAccount, "9")
	messages := []chatwork.Message{
		selectionMessage(t, "101", room, c),
		selectionMessage(t, "102", room, a),
		selectionMessage(t, "103", room, c),
		selectionMessage(t, "104", room, b),
	}
	wantInput := cloneMessages(messages)

	filter := chatwork.MessageFilter{
		Senders: []chatwork.Reference{a, b}, Context: chatwork.MessageContextNone,
	}
	selected, selection, err := assembleMessageWindow(messages, filter)
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"102", "104"}) {
		t.Fatalf("selected refs = %v", got)
	}
	if selection == nil || selection.SourceCount != 4 ||
		!reflect.DeepEqual(selection.SourceSequences, []int{2, 4}) ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{2, 4}) {
		t.Fatalf("selection = %+v", selection)
	}
	if !reflect.DeepEqual(messages, wantInput) {
		t.Fatal("selection mutated source messages")
	}
	selection.Filter.Senders[0].Value = "99"
	if filter.Senders[0].Value != "7" {
		t.Fatal("selection aliases filter sender storage")
	}
}

func TestAssembleMessageWindowReturnsExplicitEmptySelection(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	messages := []chatwork.Message{selectionMessage(t, "101", room, relationshipReference(t, chatwork.ReferenceAccount, "7"))}

	selected, selection, err := assembleMessageWindow(messages, chatwork.MessageFilter{
		Senders: []chatwork.Reference{relationshipReference(t, chatwork.ReferenceAccount, "9")},
		Context: chatwork.MessageContextNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	if selected == nil || len(selected) != 0 {
		t.Fatalf("selected = %#v, want explicit empty", selected)
	}
	if selection == nil || selection.SourceCount != 1 || selection.SourceSequences == nil || selection.AnchorSequences == nil ||
		len(selection.SourceSequences) != 0 || len(selection.AnchorSequences) != 0 {
		t.Fatalf("selection = %+v, want explicit empty sequences", selection)
	}
}

func TestAssembleMessageWindowAddsOnlyDirectReplyNeighborsAndRebasesRelations(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	c := relationshipReference(t, chatwork.ReferenceAccount, "9")
	messages := []chatwork.Message{
		selectionMessage(t, "101", room, c),
		selectionReply(t, "102", room, b, "101"),
		selectionReply(t, "103", room, a, "102"),
		selectionReply(t, "104", room, b, "103"),
		selectionReply(t, "105", room, c, "104"),
	}

	selected, selection, err := assembleMessageWindow(messages, chatwork.MessageFilter{
		Senders: []chatwork.Reference{a}, Context: chatwork.MessageContextReplies,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"102", "103", "104"}) {
		t.Fatalf("selected refs = %v, want direct parent, anchor, direct child", got)
	}
	if !reflect.DeepEqual(selection.SourceSequences, []int{2, 3, 4}) || !reflect.DeepEqual(selection.AnchorSequences, []int{3}) {
		t.Fatalf("selection = %+v", selection)
	}
	if len(selected[0].Replies) != 1 || selected[0].Replies[0].Resolved || selected[0].Replies[0].Target.Value != "101" {
		t.Fatalf("context parent relation = %+v, want canonical omitted grandparent unresolved", selected[0].Replies[0])
	}
	if len(selected[1].Replies) != 1 || !selected[1].Replies[0].Resolved || selected[1].Replies[0].Target.Value != "102" {
		t.Fatalf("anchor relation = %+v, want resolved direct parent", selected[1].Replies[0])
	}
	if len(selected[2].Replies) != 1 || !selected[2].Replies[0].Resolved || selected[2].Replies[0].Target.Value != "103" {
		t.Fatalf("context child relation = %+v, want resolved anchor", selected[2].Replies[0])
	}
}

func TestAssembleMessageWindowIncludesEveryDirectParentOfMultiReplyAnchor(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	first := selectionMessage(t, "101", room, b)
	second := selectionMessage(t, "102", room, b)
	anchor := selectionMessage(t, "103", room, a)
	anchor.Replies = []chatwork.Relation{
		{Kind: "reply", Target: first.Ref, ExternalID: room.Value},
		{Kind: "reply", Target: second.Ref, ExternalID: room.Value},
	}

	selected, selection, err := assembleMessageWindow([]chatwork.Message{first, second, anchor}, chatwork.MessageFilter{
		Senders: []chatwork.Reference{a}, Context: chatwork.MessageContextReplies,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"101", "102", "103"}) {
		t.Fatalf("selected refs = %v, want both parents and anchor", got)
	}
	if !reflect.DeepEqual(selection.AnchorSequences, []int{3}) || len(selected[2].Replies) != 2 ||
		!selected[2].Replies[0].Resolved || !selected[2].Replies[1].Resolved {
		t.Fatalf("selection=%+v replies=%+v", selection, selected[2].Replies)
	}
}

func TestAssembleMessageWindowDoesNotInferContextFromRawBody(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	anchor := selectionMessage(t, "101", room, a)
	looksRelated := selectionMessage(t, "102", room, b)
	looksRelated.Body = "[rp aid=7 to=42-101] copied raw text only"
	looksRelated.Recipients = []chatwork.Reference{a}
	looksRelated.Quotes = []chatwork.Relation{{Kind: "quote", Target: a}}

	selected, selection, err := assembleMessageWindow([]chatwork.Message{anchor, looksRelated}, chatwork.MessageFilter{
		Senders: []chatwork.Reference{a}, Context: chatwork.MessageContextReplies,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"101"}) {
		t.Fatalf("body, To, or quote fabricated reply context: %v", got)
	}
	if !reflect.DeepEqual(selection.SourceSequences, []int{1}) || !reflect.DeepEqual(selection.AnchorSequences, []int{1}) {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestAssembleMessageWindowKeepsUnfilteredWindowResolvedWithoutMetadata(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	parent := selectionMessage(t, "101", room, relationshipReference(t, chatwork.ReferenceAccount, "7"))
	child := selectionReply(t, "102", room, relationshipReference(t, chatwork.ReferenceAccount, "8"), "101")

	selected, selection, err := assembleMessageWindow([]chatwork.Message{parent, child}, chatwork.MessageFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if selection != nil {
		t.Fatalf("selection = %+v, want nil", selection)
	}
	if len(selected) != 2 || len(selected[1].Replies) != 1 || !selected[1].Replies[0].Resolved {
		t.Fatalf("unfiltered messages = %+v", selected)
	}
}

func selectionMessage(t *testing.T, id string, room chatwork.Reference, sender chatwork.Reference) chatwork.Message {
	t.Helper()
	return chatwork.Message{
		Ref: relationshipReference(t, chatwork.ReferenceMessage, id), Room: room,
		Sender: chatwork.Account{Ref: sender},
	}
}

func selectionReply(t *testing.T, id string, room chatwork.Reference, sender chatwork.Reference, parent string) chatwork.Message {
	t.Helper()
	message := selectionMessage(t, id, room, sender)
	message.Replies = []chatwork.Relation{{
		Kind: "reply", Target: relationshipReference(t, chatwork.ReferenceMessage, parent), ExternalID: room.Value,
	}}
	return message
}

func messageValues(messages []chatwork.Message) []string {
	values := make([]string, len(messages))
	for index, message := range messages {
		values[index] = message.Ref.Value
	}
	return values
}
