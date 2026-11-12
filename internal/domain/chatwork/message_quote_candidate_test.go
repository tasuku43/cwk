package chatwork

import "testing"

func TestResultValidatesUnresolvedQuoteCandidateEvidence(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	message := Reference{Kind: ReferenceMessage, Value: "200"}
	author := Reference{Kind: ReferenceAccount, Value: "7"}
	candidate := Reference{Kind: ReferenceMessage, Value: "100"}
	valid := Result{
		Task: TaskMessagesList, MessageRoom: room,
		Coverage: Coverage{Kind: "latest_window", Limit: 100},
		Messages: []Message{{
			Ref: message, Room: room, Sender: Account{Ref: Reference{Kind: ReferenceAccount, Value: "8"}},
			Quotes: []Relation{{Kind: "quote", Target: author, ExternalID: "100"}},
			QuoteMatches: []MessageQuoteMatch{{
				State:      MessageQuoteCandidateUniqueWindow,
				Candidates: []MessageQuoteCandidate{{MessageRef: candidate, Room: room, Author: author, SendTime: 100, Scope: MessageQuoteCandidateSourceWindow}},
			}},
		}},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid quote candidate failed: %v", err)
	}
	if valid.Messages[0].Quotes[0].Resolved {
		t.Fatal("candidate validation promoted the quote to resolved")
	}

	tests := []struct {
		name   string
		mutate func(*Result)
	}{
		{"quote resolved", func(result *Result) { result.Messages[0].Quotes[0].Resolved = true }},
		{"unique with multiple candidates", func(result *Result) {
			result.Messages[0].QuoteMatches[0].Candidates = append(result.Messages[0].QuoteMatches[0].Candidates,
				MessageQuoteCandidate{MessageRef: Reference{Kind: ReferenceMessage, Value: "101"}, Room: room, Author: author, SendTime: 100, Scope: MessageQuoteCandidateSourceWindow})
		}},
		{"wrong scope", func(result *Result) { result.Messages[0].QuoteMatches[0].Candidates[0].Scope = "room-history" }},
		{"wrong author evidence", func(result *Result) {
			result.Messages[0].QuoteMatches[0].Candidates[0].Author = Reference{Kind: ReferenceAccount, Value: "8"}
		}},
		{"wrong room evidence", func(result *Result) {
			result.Messages[0].QuoteMatches[0].Candidates[0].Room = Reference{Kind: ReferenceRoom, Value: "43"}
		}},
		{"wrong timestamp evidence", func(result *Result) { result.Messages[0].QuoteMatches[0].Candidates[0].SendTime = 101 }},
		{"negative sequence", func(result *Result) { result.Messages[0].QuoteMatches[0].Candidates[0].Sequence = -1 }},
		{"identity claimed", func(result *Result) { result.Messages[0].QuoteMatches[0].Candidates[0].IdentityProven = true }},
		{"no-match has candidate", func(result *Result) { result.Messages[0].QuoteMatches[0].State = MessageQuoteCandidateNoMatch }},
		{"missing timestamp mismatch", func(result *Result) { result.Messages[0].QuoteMatches[0].State = MessageQuoteCandidateMissingTime }},
		{"missing timestamp unknown", func(result *Result) {
			result.Messages[0].Quotes[0].ExternalID = ""
			result.Messages[0].QuoteMatches[0] = MessageQuoteMatch{State: MessageQuoteCandidateUnknown, Candidates: []MessageQuoteCandidate{}}
		}},
		{"wrong quote count", func(result *Result) { result.Messages[0].QuoteMatches = []MessageQuoteMatch{} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := valid
			result.Messages = append([]Message(nil), valid.Messages...)
			result.Messages[0].Quotes = append([]Relation(nil), valid.Messages[0].Quotes...)
			result.Messages[0].QuoteMatches = append([]MessageQuoteMatch(nil), valid.Messages[0].QuoteMatches...)
			result.Messages[0].QuoteMatches[0].Candidates = append([]MessageQuoteCandidate(nil), valid.Messages[0].QuoteMatches[0].Candidates...)
			test.mutate(&result)
			if err := result.Validate(); err == nil {
				t.Fatal("invalid quote candidate evidence passed validation")
			}
		})
	}
}

func TestResultBindsVisibleQuoteCandidateSequenceToCanonicalSource(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	author := Reference{Kind: ReferenceAccount, Value: "7"}
	candidateRef := Reference{Kind: ReferenceMessage, Value: "100"}
	owner := Message{
		Ref: Reference{Kind: ReferenceMessage, Value: "200"}, Room: room,
		Sender: Account{Ref: Reference{Kind: ReferenceAccount, Value: "8"}}, SendTime: 200,
		Quotes: []Relation{{Kind: "quote", Target: author, ExternalID: "100"}},
		QuoteMatches: []MessageQuoteMatch{{State: MessageQuoteCandidateUniqueWindow, Candidates: []MessageQuoteCandidate{{
			MessageRef: candidateRef, Room: room, Author: author, SendTime: 100, Sequence: 1,
			Scope: MessageQuoteCandidateSourceWindow,
		}}}},
	}
	result := Result{
		Task: TaskMessagesList, MessageRoom: room, Coverage: Coverage{Kind: "latest_window", Limit: 100},
		Messages: []Message{
			{Ref: candidateRef, Room: room, Sender: Account{Ref: author}, SendTime: 100},
			owner,
		},
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("visible source quote candidate failed validation: %v", err)
	}
	result.Messages[1].QuoteMatches[0].Candidates[0].Sequence = 2
	if err := result.Validate(); err == nil {
		t.Fatal("quote candidate with a mismatched source sequence passed validation")
	}
}

func TestResultRejectsQuoteCandidatesForUnknownRelationSet(t *testing.T) {
	result := Result{
		Task: TaskMessagesList, MessageRoom: Reference{Kind: ReferenceRoom, Value: "42"},
		Coverage: Coverage{Kind: "latest_window", Limit: 100},
		Messages: []Message{{
			Ref: Reference{Kind: ReferenceMessage, Value: "200"}, Room: Reference{Kind: ReferenceRoom, Value: "42"},
			Sender: Account{Ref: Reference{Kind: ReferenceAccount, Value: "8"}}, RelationState: MessageRelationsUnknown,
			QuoteMatches: []MessageQuoteMatch{},
		}},
	}
	if err := result.Validate(); err == nil {
		t.Fatal("unknown relation set accepted quote candidate state")
	}
}
