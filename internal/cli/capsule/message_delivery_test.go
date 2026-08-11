package capsule

import (
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestRenderWithMessageDeliveryEndsWithCompleteBudgetMetadata(t *testing.T) {
	result := resultForTask(chatwork.TaskMessagesList)
	result.Messages = append(result.Messages, chatwork.Message{
		Ref:      reference(chatwork.ReferenceMessage, "101"),
		Room:     result.MessageRoom,
		Sender:   result.Messages[0].Sender,
		Body:     "newer",
		SendTime: 2,
	})
	delivery, err := chatworkcmd.PlanMessageDelivery(result, 1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RenderWithOptions(result, Options{MaxBytes: 4096, MessageDelivery: &delivery})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, " complete=false ") {
		t.Fatalf("provider completeness changed:\n%s", got)
	}
	wantLast := "delivery max-bytes=4096 output-complete=false output-items=1 output-total-items=2 output-next-start-index=2\n"
	if !strings.HasSuffix(got, wantLast) {
		t.Fatalf("delivery metadata is not the final newline-terminated line:\n%s", got)
	}
	if strings.Count(got, "delivery max-bytes=") != 1 {
		t.Fatalf("delivery line count is not one:\n%s", got)
	}
}

func TestRenderWithMessageDeliveryProjectsRelationTargetCount(t *testing.T) {
	room := reference(chatwork.ReferenceRoom, "42")
	account := chatwork.Account{Ref: reference(chatwork.ReferenceAccount, "7"), Name: "Synthetic"}
	oldParent := chatwork.Message{Ref: reference(chatwork.ReferenceMessage, "10"), Room: room, Sender: account, Body: "old parent", SendTime: 10}
	newParent := chatwork.Message{Ref: reference(chatwork.ReferenceMessage, "20"), Room: room, Sender: account, Body: "new parent", SendTime: 20}
	oldChild := chatwork.Message{
		Ref: reference(chatwork.ReferenceMessage, "11"), Room: room, Sender: account, Body: "old child", SendTime: 30,
		Replies: []chatwork.Relation{{Kind: "reply", ExternalID: room.Value, Target: oldParent.Ref, Resolved: true}},
	}
	newChild := chatwork.Message{
		Ref: reference(chatwork.ReferenceMessage, "21"), Room: room, Sender: account, Body: "new child", SendTime: 40,
		Replies: []chatwork.Relation{{Kind: "reply", ExternalID: room.Value, Target: newParent.Ref, Resolved: true}},
	}
	result := chatwork.Result{
		Task: chatwork.TaskMessagesList, MessageRoom: room,
		Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
		Messages: []chatwork.Message{oldChild, newChild},
		MessageRelationResolution: &chatwork.MessageRelationResolution{
			FetchLimit: 2, FetchAttempts: 2,
			Targets: []chatwork.MessageRelationTarget{
				{Target: oldParent.Ref, State: chatwork.MessageRelationResolvedByFetch, Message: &oldParent},
				{Target: newParent.Ref, State: chatwork.MessageRelationResolvedByFetch, Message: &newParent},
			},
		},
	}
	delivery, err := chatworkcmd.PlanMessageDelivery(result, 1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RenderWithOptions(result, Options{MaxBytes: 4096, MessageDelivery: &delivery})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "relation-resolution fetch-limit=2 fetch-attempts=2 targets=1\n") {
		t.Fatalf("relation target count does not describe the projection:\n%s", got)
	}
	if strings.Contains(got, "message-ref=10") || !strings.Contains(got, "message-ref=20") {
		t.Fatalf("delivery did not keep exactly the newest anchor closure:\n%s", got)
	}
}

func TestRenderWithMessageDeliveryRejectsInconsistentTotal(t *testing.T) {
	result := resultForTask(chatwork.TaskMessagesList)
	delivery, err := chatworkcmd.PlanMessageDelivery(result, 1)
	if err != nil {
		t.Fatal(err)
	}
	delivery.OutputTotalItems = 2
	if _, err := RenderWithOptions(result, Options{MaxBytes: 4096, MessageDelivery: &delivery}); err == nil {
		t.Fatal("inconsistent complete delivery total was accepted")
	}
}

func TestRenderWithMessageDeliveryRequiresRemainingCountOnlyForCountBoundedSlice(t *testing.T) {
	result := resultForTask(chatwork.TaskMessagesList)
	result.Messages = append(result.Messages, chatwork.Message{
		Ref: reference(chatwork.ReferenceMessage, "101"), Room: result.MessageRoom,
		Sender: result.Messages[0].Sender, Body: "newer", SendTime: 2,
	})
	result.MessageSelection = &chatwork.MessageSelection{
		Filter:      chatwork.MessageFilter{StartIndex: 1, Count: 2, Context: chatwork.MessageContextNone},
		SourceCount: 2, CandidateCount: 2, SourceSequences: []int{1, 2},
		AnchorSequences: []int{1, 2}, ItemsPerPage: 2,
	}
	delivery, err := chatworkcmd.PlanMessageDelivery(result, 1)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.OutputNextCount != 1 {
		t.Fatalf("remaining count = %d", delivery.OutputNextCount)
	}
	delivery.OutputNextCount = 0
	if _, err := RenderWithOptions(result, Options{MaxBytes: 4096, MessageDelivery: &delivery}); err == nil {
		t.Fatal("count-bounded incomplete delivery without remaining count was accepted")
	}

	result.MessageSelection = nil
	delivery, err = chatworkcmd.PlanMessageDelivery(result, 1)
	if err != nil {
		t.Fatal(err)
	}
	delivery.OutputNextCount = 1
	if _, err := RenderWithOptions(result, Options{MaxBytes: 4096, MessageDelivery: &delivery}); err == nil {
		t.Fatal("unbounded delivery with remaining count was accepted")
	}
}
