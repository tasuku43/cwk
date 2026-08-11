package chatworkcmd

import (
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestPlanMessageDeliveryUsesNewestAnchorRankAndPreservesProviderOrder(t *testing.T) {
	result := deliveryResult([]chatwork.Message{
		deliveryMessage("1", 100),
		deliveryMessage("2", 300),
		deliveryMessage("3", 200),
	})
	delivery, err := PlanMessageDelivery(result, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := deliveryMessageRefs(delivery.Messages), []string{"2", "3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("delivered refs=%v want=%v", got, want)
	}
	if !reflect.DeepEqual(delivery.SourceSequences, []int{2, 3}) || !reflect.DeepEqual(delivery.AnchorSequences, []int{2, 3}) {
		t.Fatalf("delivery sequences=%v anchors=%v", delivery.SourceSequences, delivery.AnchorSequences)
	}
	if delivery.OutputItems != 2 || delivery.OutputTotalItems != 3 || delivery.OutputComplete || delivery.OutputNextStartIndex != 3 || delivery.OutputNextCount != 0 {
		t.Fatalf("delivery metadata=%+v", delivery)
	}

	next, selection, err := assembleMessageWindow(result.Messages, chatwork.MessageFilter{
		StartIndex: delivery.OutputNextStartIndex,
		Count:      1,
		Context:    chatwork.MessageContextNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := deliveryMessageRefs(next); !reflect.DeepEqual(got, []string{"1"}) || selection.NextStartIndex != 0 {
		t.Fatalf("continued refs=%v selection=%+v", got, selection)
	}
}

func TestPlanMessageDeliveryPublishesRemainingCountWithoutExpandingOriginalSlice(t *testing.T) {
	source := []chatwork.Message{
		deliveryMessage("1", 10), deliveryMessage("2", 20), deliveryMessage("3", 30),
		deliveryMessage("4", 40), deliveryMessage("5", 50),
	}
	result := deliveryResult(source)
	selected, selection, err := assembleMessageWindow(source, chatwork.MessageFilter{
		StartIndex: 1, Count: 3, Context: chatwork.MessageContextNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	result.Messages, result.MessageSelection = selected, selection
	delivery, err := PlanMessageDelivery(result, 1)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.OutputNextStartIndex != 2 || delivery.OutputNextCount != 2 {
		t.Fatalf("continuation = start %d count %d", delivery.OutputNextStartIndex, delivery.OutputNextCount)
	}
	continued, _, err := assembleMessageWindow(source, chatwork.MessageFilter{
		StartIndex: delivery.OutputNextStartIndex,
		Count:      delivery.OutputNextCount,
		Context:    chatwork.MessageContextNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := deliveryMessageRefs(continued), []string{"3", "4"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("continued refs=%v want=%v", got, want)
	}
}

func TestPlanMessageDeliveryKeepsDirectReplyContextAsOneAtomicItem(t *testing.T) {
	room := deliveryRoom()
	parent := deliveryMessage("10", 100)
	child := deliveryMessage("11", 300)
	child.Replies = []chatwork.Relation{{Kind: "reply", ExternalID: room.Value, Target: parent.Ref}}
	other := deliveryMessage("12", 200)
	result := deliveryResult([]chatwork.Message{parent, child, other})
	selected, selection, err := assembleMessageWindow(result.Messages, chatwork.MessageFilter{
		StartIndex: 1,
		Count:      2,
		Context:    chatwork.MessageContextReplies,
	})
	if err != nil {
		t.Fatal(err)
	}
	result.Messages = selected
	result.MessageSelection = selection
	before := *selection

	delivery, err := PlanMessageDelivery(result, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := deliveryMessageRefs(delivery.Messages), []string{"10", "11"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("atomic reply delivery refs=%v want=%v", got, want)
	}
	if !reflect.DeepEqual(delivery.SourceSequences, []int{1, 2}) || !reflect.DeepEqual(delivery.AnchorSequences, []int{2}) ||
		delivery.OutputTotalItems != 2 || delivery.OutputNextStartIndex != 2 || delivery.OutputNextCount != 1 || delivery.OutputComplete {
		t.Fatalf("delivery=%+v", delivery)
	}
	if len(delivery.Messages[1].Replies) != 1 || !delivery.Messages[1].Replies[0].Resolved {
		t.Fatalf("delivered reply context lost resolution: %+v", delivery.Messages[1].Replies)
	}
	if !reflect.DeepEqual(*result.MessageSelection, before) {
		t.Fatalf("planner mutated original selection: before=%+v after=%+v", before, *result.MessageSelection)
	}
}

func TestPlanMessageDeliveryCarriesReachableRelationClosureAtomically(t *testing.T) {
	room := deliveryRoom()
	grandparent := deliveryMessage("98", 10)
	parent := deliveryMessage("99", 20)
	parent.Replies = []chatwork.Relation{{Kind: "reply", ExternalID: room.Value, Target: grandparent.Ref, Resolved: true}}
	child := deliveryMessage("100", 30)
	child.Replies = []chatwork.Relation{{Kind: "reply", ExternalID: room.Value, Target: parent.Ref, Resolved: true}}
	sender := child.Sender.Ref
	result := deliveryResult([]chatwork.Message{child})
	result.MessageSelection = &chatwork.MessageSelection{
		Filter: chatwork.MessageFilter{
			Senders: []chatwork.Reference{sender},
			Context: chatwork.MessageContextNone,
		},
		SourceCount:     3,
		CandidateCount:  1,
		SourceSequences: []int{3},
		AnchorSequences: []int{3},
	}
	result.MessageRelationResolution = &chatwork.MessageRelationResolution{
		FetchLimit: 5,
		Targets: []chatwork.MessageRelationTarget{
			{Target: parent.Ref, State: chatwork.MessageRelationResolvedFromSource, Message: &parent},
			{Target: grandparent.Ref, State: chatwork.MessageRelationResolvedFromSource, Message: &grandparent},
		},
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("fixture: %v", err)
	}

	delivery, err := PlanMessageDelivery(result, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(delivery.RelationTargets) != 2 || delivery.RelationTargets[0].Target != parent.Ref || delivery.RelationTargets[1].Target != grandparent.Ref {
		t.Fatalf("relation closure=%+v", delivery.RelationTargets)
	}
	if !delivery.Messages[0].Replies[0].Resolved || !delivery.RelationTargets[0].Message.Replies[0].Resolved {
		t.Fatalf("relation closure was not kept resolved: messages=%+v targets=%+v", delivery.Messages, delivery.RelationTargets)
	}

	empty, err := PlanMessageDelivery(result, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Messages) != 0 || len(empty.RelationTargets) != 0 || empty.OutputTotalItems != 1 || empty.OutputNextStartIndex != 1 || empty.OutputNextCount != 0 || empty.OutputComplete {
		t.Fatalf("zero-item delivery=%+v", empty)
	}
}

func TestPlanMessageDeliveryRejectsInvalidTaskAndItemCount(t *testing.T) {
	if _, err := MessageDeliveryAnchorCount(chatwork.Result{Task: chatwork.TaskAccountShow}); err == nil {
		t.Fatal("non-message task was accepted")
	}
	result := deliveryResult([]chatwork.Message{deliveryMessage("1", 1)})
	for _, count := range []int{-1, 2} {
		if _, err := PlanMessageDelivery(result, count); err == nil {
			t.Errorf("outputItems=%d was accepted", count)
		}
	}
}

func deliveryResult(messages []chatwork.Message) chatwork.Result {
	return chatwork.Result{
		Task:        chatwork.TaskMessagesList,
		MessageRoom: deliveryRoom(),
		Messages:    messages,
		Coverage: chatwork.Coverage{
			Kind:     "latest_window",
			Limit:    100,
			Complete: false,
		},
		MessageAccess: chatwork.MessageAccessNone,
	}
}

func deliveryMessage(value string, sendTime int64) chatwork.Message {
	room := deliveryRoom()
	ref, _ := chatwork.NewReference(chatwork.ReferenceMessage, value)
	accountRef, _ := chatwork.NewReference(chatwork.ReferenceAccount, "7")
	return chatwork.Message{
		Ref:        ref,
		Room:       room,
		Sender:     chatwork.Account{Ref: accountRef, Name: "Synthetic"},
		Body:       "body-" + value,
		SendTime:   sendTime,
		Replies:    []chatwork.Relation{},
		Recipients: []chatwork.Reference{},
		Quotes:     []chatwork.Relation{},
	}
}

func deliveryRoom() chatwork.Reference {
	room, _ := chatwork.NewReference(chatwork.ReferenceRoom, "42")
	return room
}

func deliveryMessageRefs(messages []chatwork.Message) []string {
	refs := make([]string, len(messages))
	for index, message := range messages {
		refs[index] = message.Ref.Value
	}
	return refs
}
