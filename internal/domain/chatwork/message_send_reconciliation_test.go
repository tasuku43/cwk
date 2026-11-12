package chatwork

import "testing"

func TestMessageSendReconciliationResultRequiresUnknownOutcomeAndExactCandidateSet(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	message := Reference{Kind: ReferenceMessage, Value: "101"}
	account := Reference{Kind: ReferenceAccount, Value: "9"}
	request := Request{Task: TaskMessagesReconcileSend, Room: room, Body: "body"}
	coverage := Coverage{Kind: "latest_window", Limit: 100, Complete: false}
	result := Result{
		Task: TaskMessagesReconcileSend, Coverage: coverage, MessageRoom: room,
		MessageSendReconciliation: &MessageSendReconciliation{
			Room: room, Account: Reference{Kind: ReferenceAccount, Value: "9"}, Outcome: "unknown", Status: MessageSendCandidatesFound,
			SourceCount: 3, CandidateCount: 1,
			Candidates: []MessageSendCandidate{{Message: message, Sender: account, SenderName: "Alice", SendTime: 1720000000}},
		},
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("valid reconciliation result failed: %v", err)
	}

	mutations := []struct {
		name   string
		change func(*Result)
	}{
		{"missing account", func(value *Result) { value.MessageSendReconciliation.Account = Reference{} }},
		{"different sender", func(value *Result) { value.MessageSendReconciliation.Candidates[0].Sender.Value = "10" }},
		{"known outcome", func(value *Result) { value.MessageSendReconciliation.Outcome = "sent" }},
		{"candidate count mismatch", func(value *Result) { value.MessageSendReconciliation.CandidateCount = 2 }},
		{"status mismatch", func(value *Result) { value.MessageSendReconciliation.Status = MessageSendNoCandidatesWindow }},
		{"duplicate candidate", func(value *Result) {
			value.MessageSendReconciliation.Candidates = append(value.MessageSendReconciliation.Candidates, value.MessageSendReconciliation.Candidates[0])
			value.MessageSendReconciliation.CandidateCount = 2
		}},
		{"wrong source", func(value *Result) { value.Coverage.Kind = "differential_window" }},
		{"complete history claim", func(value *Result) { value.Coverage.Complete = true }},
		{"wrong room", func(value *Result) {
			value.MessageSendReconciliation.Room = Reference{Kind: ReferenceRoom, Value: "43"}
		}},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			invalid := result
			reconciliation := *result.MessageSendReconciliation
			reconciliation.Candidates = append([]MessageSendCandidate(nil), result.MessageSendReconciliation.Candidates...)
			invalid.MessageSendReconciliation = &reconciliation
			test.change(&invalid)
			if err := invalid.ValidateFor(request); err == nil {
				t.Fatal("invalid reconciliation result passed validation")
			}
		})
	}
}

func TestMessageSendReconciliationNoMatchIsStillUnknownAndExplicit(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	request := Request{Task: TaskMessagesReconcileSend, Room: room, BodyFile: "message.txt"}
	result := Result{
		Task: TaskMessagesReconcileSend, Coverage: Coverage{Kind: "latest_window", Limit: 100}, MessageRoom: room,
		MessageAccess: MessageAccessAll,
		MessageSendReconciliation: &MessageSendReconciliation{
			Room: room, Account: Reference{Kind: ReferenceAccount, Value: "9"}, Outcome: "unknown", Status: MessageSendNoCandidatesWindow,
			SourceCount: 0, CandidateCount: 0, Candidates: []MessageSendCandidate{},
		},
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("valid no-match result failed: %v", err)
	}
	result.MessageSendReconciliation.Candidates = nil
	if err := result.Validate(); err == nil {
		t.Fatal("nil candidate list was accepted as an explicit empty result")
	}
}
