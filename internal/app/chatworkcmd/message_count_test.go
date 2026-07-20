package chatworkcmd

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

func TestAssembleMessageWindowCountRanksTypedSendTimeThenPreservesProviderOrder(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	messages := []chatwork.Message{
		selectionMessageAt(t, "201", room, a, 90),
		selectionMessageAt(t, "202", room, b, 900),
		selectionMessageAt(t, "203", room, a, 200),
		selectionMessageAt(t, "204", room, a, 300),
		selectionMessageAt(t, "205", room, b, 1000),
		selectionMessageAt(t, "206", room, a, 300),
	}

	selected, selection, err := assembleMessageWindow(messages, chatwork.MessageFilter{
		Senders: []chatwork.Reference{a}, Context: chatwork.MessageContextNone, StartIndex: 1, Count: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	// #6 wins the equal-time tie over #4 because it occurs later in provider
	// order. The selected records themselves remain in provider order.
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"204", "206"}) {
		t.Fatalf("selected refs = %v, want provider-order newest typed messages", got)
	}
	if selection == nil || selection.SourceCount != 6 || selection.CandidateCount != 4 ||
		!reflect.DeepEqual(selection.SourceSequences, []int{4, 6}) ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{4, 6}) {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestAssembleMessageWindowCountWithoutSenderRanksTypedSendTime(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	messages := []chatwork.Message{
		selectionMessageAt(t, "201", room, a, 500),
		selectionMessageAt(t, "202", room, a, 100),
		selectionMessageAt(t, "203", room, a, 400),
	}

	selected, selection, err := assembleMessageWindow(messages, chatwork.MessageFilter{
		Context: chatwork.MessageContextNone, StartIndex: 1, Count: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"201"}) {
		t.Fatalf("selected refs = %v, want typed latest message rather than provider tail", got)
	}
	if selection == nil || selection.SourceCount != 3 || selection.CandidateCount != 3 ||
		!reflect.DeepEqual(selection.SourceSequences, []int{1}) ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{1}) {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestAssembleMessageWindowCountRanksOneCombinedSenderORCandidateSet(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	c := relationshipReference(t, chatwork.ReferenceAccount, "9")
	messages := []chatwork.Message{
		selectionMessageAt(t, "201", room, a, 100),
		selectionMessageAt(t, "202", room, b, 400),
		selectionMessageAt(t, "203", room, a, 300),
		selectionMessageAt(t, "204", room, c, 1000),
	}

	selected, selection, err := assembleMessageWindow(messages, chatwork.MessageFilter{
		Senders: []chatwork.Reference{a, b}, Context: chatwork.MessageContextNone, StartIndex: 1, Count: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"202", "203"}) {
		t.Fatalf("selected refs = %v, want two newest across one combined sender OR set", got)
	}
	if selection == nil || selection.CandidateCount != 3 ||
		!reflect.DeepEqual(selection.SourceSequences, []int{2, 3}) ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{2, 3}) {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestAssembleMessageWindowAppliesReplyContextAfterCount(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	parent := selectionMessageAt(t, "201", room, a, 100)
	child := selectionReply(t, "202", room, a, "201")
	child.SendTime = 300

	selected, selection, err := assembleMessageWindow([]chatwork.Message{parent, child}, chatwork.MessageFilter{
		Senders: []chatwork.Reference{a}, Context: chatwork.MessageContextReplies, StartIndex: 1, Count: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"201", "202"}) {
		t.Fatalf("selected refs = %v, want same-sender parent context plus anchor", got)
	}
	if selection == nil || selection.CandidateCount != 2 ||
		!reflect.DeepEqual(selection.SourceSequences, []int{1, 2}) ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{2}) {
		t.Fatalf("selection = %+v", selection)
	}
	if len(selected) <= selection.Filter.Count {
		t.Fatalf("displayed count = %d, want reply context allowed beyond primary count %d", len(selected), selection.Filter.Count)
	}
	if len(selected[1].Replies) != 1 || !selected[1].Replies[0].Resolved || selected[1].Replies[0].Target.Value != "201" {
		t.Fatalf("child reply = %+v, want resolved context parent", selected[1].Replies[0])
	}
}

func TestAssembleMessageWindowAppliesReplyContextToNoSenderCountAnchors(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	parent := selectionMessageAt(t, "201", room, a, 100)
	child := selectionReply(t, "202", room, b, "201")
	child.SendTime = 300

	selected, selection, err := assembleMessageWindow([]chatwork.Message{parent, child}, chatwork.MessageFilter{
		Context: chatwork.MessageContextReplies, StartIndex: 1, Count: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"201", "202"}) {
		t.Fatalf("selected refs = %v, want parent context plus no-sender limit anchor", got)
	}
	if selection == nil || selection.CandidateCount != 2 ||
		!reflect.DeepEqual(selection.SourceSequences, []int{1, 2}) ||
		!reflect.DeepEqual(selection.AnchorSequences, []int{2}) {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestAssembleMessageWindowCountRebasesOmittedParentAsUnresolved(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	b := relationshipReference(t, chatwork.ReferenceAccount, "8")
	parent := selectionMessageAt(t, "201", room, b, 100)
	child := selectionReply(t, "202", room, a, "201")
	child.SendTime = 300

	selected, selection, err := assembleMessageWindow([]chatwork.Message{parent, child}, chatwork.MessageFilter{
		Senders: []chatwork.Reference{a}, Context: chatwork.MessageContextNone, StartIndex: 1, Count: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := messageValues(selected); !reflect.DeepEqual(got, []string{"202"}) {
		t.Fatalf("selected refs = %v", got)
	}
	if selection == nil || selection.CandidateCount != 1 || !reflect.DeepEqual(selection.AnchorSequences, []int{2}) {
		t.Fatalf("selection = %+v", selection)
	}
	if len(selected[0].Replies) != 1 || selected[0].Replies[0].Resolved || selected[0].Replies[0].Target.Value != "201" {
		t.Fatalf("selected reply = %+v, want canonical omitted parent unresolved", selected[0].Replies[0])
	}
}

func TestAssembleMessageWindowCountBoundaries(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	hundred := make([]chatwork.Message, 100)
	for index := range hundred {
		hundred[index] = selectionMessageAt(t, strconv.Itoa(1000+index), room, a, int64(index))
	}

	tests := []struct {
		name       string
		messages   []chatwork.Message
		limit      int
		wantCount  int
		wantSource []int
	}{
		{name: "empty", messages: []chatwork.Message{}, limit: 1, wantCount: 0, wantSource: []int{}},
		{name: "one", messages: hundred[:2], limit: 1, wantCount: 1, wantSource: []int{2}},
		{name: "fewer than limit", messages: hundred[:2], limit: 100, wantCount: 2, wantSource: []int{1, 2}},
		{name: "exact provider bound", messages: hundred, limit: 100, wantCount: 100, wantSource: sequenceRange(1, 100)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selected, selection, err := assembleMessageWindow(test.messages, chatwork.MessageFilter{
				Context: chatwork.MessageContextNone, StartIndex: 1, Count: test.limit,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(selected) != test.wantCount || selection == nil ||
				selection.SourceCount != len(test.messages) || selection.CandidateCount != len(test.messages) ||
				!reflect.DeepEqual(selection.SourceSequences, test.wantSource) ||
				!reflect.DeepEqual(selection.AnchorSequences, test.wantSource) {
				t.Fatalf("selected = %d, selection = %+v", len(selected), selection)
			}
			if selection.SourceSequences == nil || selection.AnchorSequences == nil {
				t.Fatal("limit selection provenance must be explicit even when empty")
			}
		})
	}
}

func TestExecuteAppliesCountAfterOneFilterFreePortCall(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	port := &fakePort{result: chatwork.Result{
		MessageRoom: room,
		Coverage:    chatwork.Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages: []chatwork.Message{
			selectionMessageAt(t, "201", room, a, 100),
			selectionMessageAt(t, "202", room, a, 200),
		},
	}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextNone, StartIndex: 1, Count: 1},
	}

	result, err := New(port).Execute(context.Background(), testBinding(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if port.calls != 1 {
		t.Fatalf("port calls = %d, want 1", port.calls)
	}
	if !reflect.DeepEqual(port.request.MessageFilter, chatwork.MessageFilter{}) {
		t.Fatalf("local index selection leaked to port: %+v", port.request.MessageFilter)
	}
	if got := messageValues(result.Messages); !reflect.DeepEqual(got, []string{"202"}) {
		t.Fatalf("selected refs = %v", got)
	}
	if result.MessageSelection == nil || result.MessageSelection.CandidateCount != 2 {
		t.Fatalf("selection = %+v", result.MessageSelection)
	}
}

func TestExecuteRejectsProviderWindowAboveCoverageBeforeLocalCount(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	a := relationshipReference(t, chatwork.ReferenceAccount, "7")
	messages := make([]chatwork.Message, 101)
	for index := range messages {
		messages[index] = selectionMessageAt(t, strconv.Itoa(1000+index), room, a, int64(index))
	}
	port := &fakePort{result: chatwork.Result{
		MessageRoom: room,
		Coverage:    chatwork.Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages:    messages,
	}}
	request := chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room,
		MessageFilter: chatwork.MessageFilter{Context: chatwork.MessageContextNone, StartIndex: 1, Count: 1},
	}

	result, err := New(port).Execute(context.Background(), testBinding(t), request)
	var got *fault.Error
	if !errors.As(err, &got) || got.Code != "chatwork_result_invalid" || result.Task != "" {
		t.Fatalf("result = %+v, err = %#v", result, err)
	}
	if port.calls != 1 {
		t.Fatalf("port calls = %d, want exactly one rejected provider result", port.calls)
	}
}

func selectionMessageAt(t *testing.T, id string, room, sender chatwork.Reference, sendTime int64) chatwork.Message {
	t.Helper()
	message := selectionMessage(t, id, room, sender)
	message.SendTime = sendTime
	return message
}

func sequenceRange(first, last int) []int {
	result := make([]int, 0, last-first+1)
	for value := first; value <= last; value++ {
		result = append(result, value)
	}
	return result
}
