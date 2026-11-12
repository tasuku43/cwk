package chatworkcmd

import (
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestResolveMessageQuoteCandidatesRequiresExactRoomAuthorAndWholeTimestamp(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	otherRoom := relationshipReference(t, chatwork.ReferenceRoom, "43")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	otherAuthor := relationshipReference(t, chatwork.ReferenceAccount, "8")
	quote := quoteMessage(t, "200", room, otherAuthor, 200, chatwork.Relation{
		Kind: "quote", Target: author, ExternalID: "100",
	})
	source := []chatwork.Message{
		quoteMessage(t, "101", room, author, 99),
		quoteMessage(t, "102", room, otherAuthor, 100),
		quoteMessage(t, "103", otherRoom, author, 100),
		quoteMessage(t, "104", room, author, 100),
	}

	got, err := ResolveMessageQuoteCandidates([]chatwork.Message{quote}, source, true, chatwork.MessageAccessNone)
	if err != nil {
		t.Fatal(err)
	}
	match := got[0].QuoteMatches[0]
	if match.State != chatwork.MessageQuoteCandidateUniqueWindow || len(match.Candidates) != 1 {
		t.Fatalf("match = %+v", match)
	}
	candidate := match.Candidates[0]
	if candidate.MessageRef.Value != "104" || candidate.Sequence != 4 || candidate.Scope != chatwork.MessageQuoteCandidateSourceWindow || candidate.IdentityProven {
		t.Fatalf("candidate = %+v", candidate)
	}
	if got[0].Quotes[0].Resolved {
		t.Fatal("a quote candidate proved quote identity")
	}
}

func TestResolveMessageQuoteCandidatesKeepsMultipleMatchesInSourceOrder(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	quote := quoteMessage(t, "200", room, relationshipReference(t, chatwork.ReferenceAccount, "8"), 200, chatwork.Relation{
		Kind: "quote", Target: author, ExternalID: "100",
	})
	source := []chatwork.Message{
		quoteMessage(t, "110", room, author, 100),
		quoteMessage(t, "111", room, author, 100),
	}

	got, err := ResolveMessageQuoteCandidates([]chatwork.Message{quote}, source, true, chatwork.MessageAccessNone)
	if err != nil {
		t.Fatal(err)
	}
	match := got[0].QuoteMatches[0]
	if match.State != chatwork.MessageQuoteCandidateAmbiguous || len(match.Candidates) != 2 ||
		match.Candidates[0].MessageRef != source[0].Ref || match.Candidates[1].MessageRef != source[1].Ref {
		t.Fatalf("match = %+v", match)
	}
}

func TestResolveMessageQuoteCandidatesLimitsPartialSourceClaimToWindow(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	quote := quoteMessage(t, "200", room, relationshipReference(t, chatwork.ReferenceAccount, "8"), 200, chatwork.Relation{
		Kind: "quote", Target: author, ExternalID: "100",
	})
	candidate := quoteMessage(t, "100", room, author, 100)

	got, err := ResolveMessageQuoteCandidates([]chatwork.Message{quote}, []chatwork.Message{candidate}, true, chatwork.MessageAccessPartial)
	if err != nil {
		t.Fatal(err)
	}
	match := got[0].QuoteMatches[0]
	if match.State != chatwork.MessageQuoteCandidateUniqueWindow || len(match.Candidates) != 1 ||
		match.Candidates[0].Scope != chatwork.MessageQuoteCandidateSourceWindow || match.Candidates[0].IdentityProven {
		t.Fatalf("partial source match = %+v", match)
	}
}

func TestResolveMessageQuoteCandidatesDistinguishesMissingAbsentAndUnknown(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	message := quoteMessage(t, "200", room, relationshipReference(t, chatwork.ReferenceAccount, "8"), 200,
		chatwork.Relation{Kind: "quote", Target: author, ExternalID: "100"},
		chatwork.Relation{Kind: "quote", Target: author},
	)
	cases := []struct {
		name      string
		source    []chatwork.Message
		available bool
		access    chatwork.MessageAccessLimitation
		want      []chatwork.MessageQuoteCandidateState
	}{
		{name: "available no match", source: []chatwork.Message{}, available: true, want: []chatwork.MessageQuoteCandidateState{chatwork.MessageQuoteCandidateNoMatch, chatwork.MessageQuoteCandidateMissingTime}},
		{name: "missing source", available: false, want: []chatwork.MessageQuoteCandidateState{chatwork.MessageQuoteCandidateUnknown, chatwork.MessageQuoteCandidateMissingTime}},
		{name: "restricted source", source: []chatwork.Message{}, available: true, access: chatwork.MessageAccessAll, want: []chatwork.MessageQuoteCandidateState{chatwork.MessageQuoteCandidateUnknown, chatwork.MessageQuoteCandidateMissingTime}},
		{name: "partial empty source", source: []chatwork.Message{}, available: true, access: chatwork.MessageAccessPartial, want: []chatwork.MessageQuoteCandidateState{chatwork.MessageQuoteCandidateUnknown, chatwork.MessageQuoteCandidateMissingTime}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveMessageQuoteCandidates([]chatwork.Message{message}, test.source, test.available, test.access)
			if err != nil {
				t.Fatal(err)
			}
			if len(got[0].QuoteMatches) != len(test.want) {
				t.Fatalf("quote matches = %+v", got[0].QuoteMatches)
			}
			for index, state := range test.want {
				if got[0].QuoteMatches[index].State != state || len(got[0].QuoteMatches[index].Candidates) != 0 {
					t.Fatalf("match[%d] = %+v, want %s without candidate", index, got[0].QuoteMatches[index], state)
				}
			}
		})
	}
}

func TestResolveMessageQuoteCandidatesDoesNotMutateInputAndRejectsRepeatedResolution(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	message := quoteMessage(t, "200", room, relationshipReference(t, chatwork.ReferenceAccount, "8"), 200, chatwork.Relation{
		Kind: "quote", Target: author, ExternalID: "100",
	})
	source := []chatwork.Message{quoteMessage(t, "100", room, author, 100)}
	input := cloneMessages([]chatwork.Message{message})

	got, err := ResolveMessageQuoteCandidates([]chatwork.Message{message}, source, true, chatwork.MessageAccessNone)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(message, input[0]) {
		t.Fatal("quote resolution mutated its input")
	}
	if _, err := ResolveMessageQuoteCandidates(got, source, true, chatwork.MessageAccessNone); err == nil {
		t.Fatal("repeated quote resolution silently replaced source provenance")
	}
}

func TestRemoveHiddenMessageQuoteCandidateSequencesPreservesCanonicalIdentityAndState(t *testing.T) {
	visible := quoteMessage(t, "200", relationshipReference(t, chatwork.ReferenceRoom, "42"), relationshipReference(t, chatwork.ReferenceAccount, "8"), 200)
	hidden := relationshipReference(t, chatwork.ReferenceMessage, "100")
	author := relationshipReference(t, chatwork.ReferenceAccount, "7")
	visible.Quotes = []chatwork.Relation{{Kind: "quote", Target: author, ExternalID: "100"}}
	visible.QuoteMatches = []chatwork.MessageQuoteMatch{{
		State: chatwork.MessageQuoteCandidateUniqueWindow,
		Candidates: []chatwork.MessageQuoteCandidate{{
			MessageRef: hidden, Room: visible.Room, Author: author, SendTime: 100,
			Sequence: 1, Scope: chatwork.MessageQuoteCandidateSourceWindow,
		}},
	}}
	messages := []chatwork.Message{visible}

	RemoveHiddenMessageQuoteCandidateSequences(messages)
	got := messages[0].QuoteMatches[0]
	if got.State != chatwork.MessageQuoteCandidateUniqueWindow || len(got.Candidates) != 1 ||
		got.Candidates[0].MessageRef != hidden || got.Candidates[0].Sequence != 0 {
		t.Fatalf("match = %+v", got)
	}
}

func TestResolveMessageRelationsDeepCopiesQuoteCandidateProvenance(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	account := relationshipReference(t, chatwork.ReferenceAccount, "8")
	candidate := relationshipReference(t, chatwork.ReferenceMessage, "100")
	message := quoteMessage(t, "200", room, account, 200, chatwork.Relation{
		Kind: "quote", Target: relationshipReference(t, chatwork.ReferenceAccount, "7"), ExternalID: "100",
	})
	message.QuoteMatches = []chatwork.MessageQuoteMatch{{
		State: chatwork.MessageQuoteCandidateUniqueWindow,
		Candidates: []chatwork.MessageQuoteCandidate{{
			MessageRef: candidate, Room: room, Author: relationshipReference(t, chatwork.ReferenceAccount, "7"), SendTime: 100,
			Sequence: 1, Scope: chatwork.MessageQuoteCandidateSourceWindow,
		}},
	}}
	resolved, err := ResolveMessageRelations([]chatwork.Message{message})
	if err != nil {
		t.Fatal(err)
	}
	resolved[0].QuoteMatches[0].Candidates[0].MessageRef.Value = "999"
	if message.QuoteMatches[0].Candidates[0].MessageRef != candidate {
		t.Fatal("cloned quote candidate storage aliases its source")
	}
}

func quoteMessage(t *testing.T, id string, room, sender chatwork.Reference, sendTime int64, quotes ...chatwork.Relation) chatwork.Message {
	t.Helper()
	return chatwork.Message{
		Ref: relationshipReference(t, chatwork.ReferenceMessage, id), Room: room,
		Sender: chatwork.Account{Ref: sender}, SendTime: sendTime,
		RelationState: chatwork.MessageRelationsComplete,
		Recipients:    []chatwork.Reference{}, Replies: []chatwork.Relation{}, Quotes: append([]chatwork.Relation{}, quotes...),
	}
}
