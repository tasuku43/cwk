package chatworkcmd

import (
	"context"
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestAssembleMessageWindowCombinesSenderAndPeriodBeforeIndex(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	messages := []chatwork.Message{
		selectionMessageAt(t, "201", room, a, 99),
		selectionMessageAt(t, "202", room, a, 200),
		selectionMessageAt(t, "203", room, b, 250),
		selectionMessageAt(t, "204", room, a, 300),
		selectionMessageAt(t, "205", room, a, 400),
	}
	filter := chatwork.MessageFilter{
		Senders: []chatwork.Reference{a},
		Period:  chatwork.MessagePeriod{Since: 100, Until: 400},
		Context: chatwork.MessageContextNone, StartIndex: 1, Count: 1,
	}

	selected, selection, err := assembleMessageWindow(messages, filter)
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"204"}) {
		t.Fatalf("selected refs = %v", got)
	}
	if selection == nil || selection.CandidateCount != 2 ||
		!reflect.DeepEqual(selection.SourceSequences, []int{4}) ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{4}) {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestAssembleMessageWindowPeriodCanAddDirectOutsideContext(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	parent := selectionMessageAt(t, "201", room, a, 99)
	child := selectionReply(t, "202", room, a, "201")
	child.SendTime = 100
	filter := chatwork.MessageFilter{
		Period:  chatwork.MessagePeriod{Since: 100, Until: 200},
		Context: chatwork.MessageContextReplies,
	}

	selected, selection, err := assembleMessageWindow([]chatwork.Message{parent, child}, filter)
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"201", "202"}) {
		t.Fatalf("selected refs = %v", got)
	}
	if selection == nil || selection.CandidateCount != 1 ||
		!reflect.DeepEqual(selection.SourceSequences, []int{1, 2}) ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{2}) {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestExecuteAppliesPeriodAfterOneFilterFreePortCall(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	port := &fakePort{result: chatwork.Result{
		MessageRoom: room,
		Coverage:    chatwork.Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages: []chatwork.Message{
			selectionMessageAt(t, "201", room, a, 99),
			selectionMessageAt(t, "202", room, a, 100),
			selectionMessageAt(t, "203", room, a, 199),
			selectionMessageAt(t, "204", room, a, 200),
		},
	}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room,
		MessageFilter: chatwork.MessageFilter{
			Period:  chatwork.MessagePeriod{Since: 100, Until: 200},
			Context: chatwork.MessageContextNone,
		},
	}

	result, err := New(port).Execute(context.Background(), testBinding(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if port.calls != 1 || !reflect.DeepEqual(port.request.MessageFilter, chatwork.MessageFilter{}) {
		t.Fatalf("provider calls=%d filter=%+v", port.calls, port.request.MessageFilter)
	}
	if got := messageValues(result.Messages); !reflect.DeepEqual(got, []string{"202", "203"}) {
		t.Fatalf("selected refs = %v", got)
	}
	if result.MessageSelection == nil || result.MessageSelection.SourceCount != 4 || result.MessageSelection.CandidateCount != 2 {
		t.Fatalf("selection = %+v", result.MessageSelection)
	}
}
