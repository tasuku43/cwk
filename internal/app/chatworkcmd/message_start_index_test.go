package chatworkcmd

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestAssembleMessageWindowStartIndexElevenCountTwentyMeansRanksElevenThroughThirty(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	sender := relationshipReference(t, chatwork.ReferenceAccount, "7")
	messages := make([]chatwork.Message, 35)
	for index := range messages {
		messages[index] = selectionMessageAt(t, fmt.Sprintf("%d", 100+index), room, sender, int64(35-index))
	}

	first, firstSelection, err := assembleMessageWindow(messages, chatwork.MessageFilter{
		Context: chatwork.MessageContextNone, StartIndex: 1, Count: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	next, nextSelection, err := assembleMessageWindow(messages, chatwork.MessageFilter{
		Context: chatwork.MessageContextNone, StartIndex: 11, Count: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 10 || first[0].Ref.Value != "100" || first[9].Ref.Value != "109" ||
		firstSelection.NextStartIndex != 11 {
		t.Fatalf("first slice messages=%v selection=%+v", messageValues(first), firstSelection)
	}
	if len(next) != 20 || next[0].Ref.Value != "110" || next[19].Ref.Value != "129" ||
		nextSelection.ItemsPerPage != 20 || nextSelection.NextStartIndex != 31 {
		t.Fatalf("ranks 11-30 messages=%v selection=%+v", messageValues(next), nextSelection)
	}
}

func TestAssembleMessageWindowStartIndexContinuesAfterFirstCountedSlice(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	sender := relationshipReference(t, chatwork.ReferenceAccount, "7")
	messages := []chatwork.Message{
		selectionMessageAt(t, "201", room, sender, 60),
		selectionMessageAt(t, "202", room, sender, 10),
		selectionMessageAt(t, "203", room, sender, 50),
		selectionMessageAt(t, "204", room, sender, 20),
		selectionMessageAt(t, "205", room, sender, 40),
		selectionMessageAt(t, "206", room, sender, 30),
	}

	first, firstSelection, err := assembleMessageWindow(messages, chatwork.MessageFilter{
		Context:    chatwork.MessageContextNone,
		StartIndex: 1,
		Count:      2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(first); !reflect.DeepEqual(got, []string{"201", "203"}) {
		t.Fatalf("first refs = %v, want newest ranks 1-2 in provider order", got)
	}
	if firstSelection == nil || firstSelection.ItemsPerPage != 2 || firstSelection.NextStartIndex != 3 {
		t.Fatalf("first selection = %+v, want next start index 3", firstSelection)
	}

	next, nextSelection, err := assembleMessageWindow(messages, chatwork.MessageFilter{
		Context:    chatwork.MessageContextNone,
		StartIndex: 3,
		Count:      3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(next); !reflect.DeepEqual(got, []string{"204", "205", "206"}) {
		t.Fatalf("next refs = %v, want ranks 3-5 in provider order", got)
	}
	if nextSelection == nil || nextSelection.CandidateCount != 6 || nextSelection.Filter.StartIndex != 3 ||
		nextSelection.ItemsPerPage != 3 || nextSelection.NextStartIndex != 6 ||
		!reflect.DeepEqual(nextSelection.SourceSequences, []int{4, 5, 6}) ||
		!reflect.DeepEqual(nextSelection.AnchorSequences, []int{4, 5, 6}) {
		t.Fatalf("next selection = %+v", nextSelection)
	}
}

func TestAssembleMessageWindowStartIndexWithoutCountReturnsAllRemainingCandidates(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	sender := relationshipReference(t, chatwork.ReferenceAccount, "7")
	messages := []chatwork.Message{
		selectionMessageAt(t, "201", room, sender, 40),
		selectionMessageAt(t, "202", room, sender, 10),
		selectionMessageAt(t, "203", room, sender, 30),
		selectionMessageAt(t, "204", room, sender, 20),
	}

	selected, selection, err := assembleMessageWindow(messages, chatwork.MessageFilter{
		Context:    chatwork.MessageContextNone,
		StartIndex: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"202", "204"}) {
		t.Fatalf("start-index-only refs = %v, want every candidate after newest two", got)
	}
	if selection == nil || selection.ItemsPerPage != 2 || selection.NextStartIndex != 0 ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{2, 4}) {
		t.Fatalf("start-index-only selection = %+v", selection)
	}
}

func TestAssembleMessageWindowStartIndexBeyondCandidatesIsExplicitEmptySelection(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	sender := relationshipReference(t, chatwork.ReferenceAccount, "7")
	selected, selection, err := assembleMessageWindow([]chatwork.Message{
		selectionMessageAt(t, "201", room, sender, 1),
	}, chatwork.MessageFilter{Context: chatwork.MessageContextNone, StartIndex: 100, Count: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 0 || selection == nil || selection.CandidateCount != 1 ||
		selection.ItemsPerPage != 0 || selection.NextStartIndex != 0 || selection.SourceSequences == nil || selection.AnchorSequences == nil {
		t.Fatalf("empty continuation selected=%v selection=%+v", selected, selection)
	}
}
