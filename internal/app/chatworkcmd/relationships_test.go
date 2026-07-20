package chatworkcmd

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

func TestResolveMessageRelationsResolvesOnlyExplicitSameRoomWindowTarget(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	parent := chatwork.Message{
		Ref:  relationshipReference(t, chatwork.ReferenceMessage, "101"),
		Room: room,
	}
	reply := chatwork.Message{
		Ref:        relationshipReference(t, chatwork.ReferenceMessage, "102"),
		Room:       room,
		Recipients: []chatwork.Reference{relationshipReference(t, chatwork.ReferenceAccount, "7")},
		Replies: []chatwork.Relation{{
			Kind:       "reply",
			Target:     parent.Ref,
			ExternalID: room.Value,
		}},
		Quotes: []chatwork.Relation{{
			Kind:       "quote",
			Target:     relationshipReference(t, chatwork.ReferenceAccount, "8"),
			Resolved:   true,
			ExternalID: "1700000000",
		}},
	}
	input := []chatwork.Message{parent, reply}
	wantInput := cloneMessages(input)

	got, err := ResolveMessageRelations(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(got[1].Replies) != 1 || !got[1].Replies[0].Resolved {
		t.Fatalf("reply = %#v", got[1].Replies[0])
	}
	if got[1].Replies[0].Target != parent.Ref || got[1].Room != room ||
		!reflect.DeepEqual(got[1].Quotes, reply.Quotes) {
		t.Fatalf("canonical or quote facts changed: %#v", got[1])
	}
	if !reflect.DeepEqual(input, wantInput) {
		t.Fatalf("input mutated: got %#v, want %#v", input, wantInput)
	}

	got[1].Recipients[0].Value = "99"
	got[1].Replies[0].Target.Value = "99"
	got[1].Quotes[0].ExternalID = "changed"
	if !reflect.DeepEqual(input, wantInput) {
		t.Fatal("output aliases relationship-bearing input storage")
	}
}

func TestResolveMessageRelationsResolvesEveryExplicitReply(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	first := chatwork.Message{Ref: relationshipReference(t, chatwork.ReferenceMessage, "101"), Room: room}
	second := chatwork.Message{Ref: relationshipReference(t, chatwork.ReferenceMessage, "102"), Room: room}
	child := chatwork.Message{
		Ref: relationshipReference(t, chatwork.ReferenceMessage, "103"), Room: room,
		Replies: []chatwork.Relation{
			{Kind: "reply", Target: first.Ref, ExternalID: room.Value},
			{Kind: "reply", Target: second.Ref, ExternalID: room.Value},
		},
	}

	got, err := ResolveMessageRelations([]chatwork.Message{first, second, child})
	if err != nil {
		t.Fatal(err)
	}
	if len(got[2].Replies) != 2 || !got[2].Replies[0].Resolved || !got[2].Replies[1].Resolved ||
		got[2].Replies[0].Target != first.Ref || got[2].Replies[1].Target != second.Ref {
		t.Fatalf("replies = %+v, want both resolved in provider order", got[2].Replies)
	}
}

func TestResolveMessageRelationsKeepsUnprovenExplicitRepliesUnresolved(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	otherRoom := relationshipReference(t, chatwork.ReferenceRoom, "43")
	tests := map[string]chatwork.Relation{
		"target outside window": {
			Kind:       "reply",
			Target:     relationshipReference(t, chatwork.ReferenceMessage, "999"),
			ExternalID: room.Value,
		},
		"parent room mismatch": {
			Kind:       "reply",
			Target:     relationshipReference(t, chatwork.ReferenceMessage, "101"),
			ExternalID: otherRoom.Value,
		},
		"non reply relation": {
			Kind:       "quote",
			Target:     relationshipReference(t, chatwork.ReferenceMessage, "101"),
			ExternalID: room.Value,
		},
	}

	for name, relation := range tests {
		t.Run(name, func(t *testing.T) {
			messages := []chatwork.Message{
				{Ref: relationshipReference(t, chatwork.ReferenceMessage, "101"), Room: room},
				{Ref: relationshipReference(t, chatwork.ReferenceMessage, "102"), Room: room, Replies: []chatwork.Relation{relation}},
			}
			got, err := ResolveMessageRelations(messages)
			if err != nil {
				t.Fatal(err)
			}
			if got[1].Replies[0].Resolved {
				t.Fatalf("reply was fabricated: %#v", got[1].Replies[0])
			}
		})
	}
}

func TestResolveMessageRelationsIgnoresNonRelationAndHostileText(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	messages := []chatwork.Message{
		{
			Ref:      relationshipReference(t, chatwork.ReferenceMessage, "101"),
			Room:     room,
			Sender:   chatwork.Account{Name: "same name"},
			Body:     "[rp aid=7 to=42-101] copied only\n  reply-looking layout\u202e",
			SendTime: 100,
		},
		{
			Ref:        relationshipReference(t, chatwork.ReferenceMessage, "102"),
			Room:       room,
			Sender:     chatwork.Account{Name: "same name"},
			Body:       "[To:7] [qt][qtmeta aid=7 time=100]quoted[/qt]",
			SendTime:   101,
			Recipients: []chatwork.Reference{relationshipReference(t, chatwork.ReferenceAccount, "7")},
			Quotes: []chatwork.Relation{{
				Kind: "quote", Target: relationshipReference(t, chatwork.ReferenceAccount, "7"), ExternalID: "100",
			}},
		},
	}

	got, err := ResolveMessageRelations(messages)
	if err != nil {
		t.Fatal(err)
	}
	for index := range got {
		if len(got[index].Replies) != 0 {
			t.Fatalf("message %d gained a reply: %#v", index, got[index].Replies[0])
		}
	}
	if !reflect.DeepEqual(got[1].Quotes, messages[1].Quotes) {
		t.Fatalf("quote state changed: %#v", got[1].Quotes)
	}
}

func TestResolveMessageRelationsRejectsDuplicateAndInconsistentFactsSafely(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	message := chatwork.Message{Ref: relationshipReference(t, chatwork.ReferenceMessage, "101"), Room: room}
	tests := map[string]struct {
		messages []chatwork.Message
		code     string
	}{
		"duplicate references": {
			messages: []chatwork.Message{message, message},
			code:     "duplicate_chatwork_message_reference",
		},
		"resolved target outside window": {
			messages: []chatwork.Message{{
				Ref:  relationshipReference(t, chatwork.ReferenceMessage, "102"),
				Room: room,
				Replies: []chatwork.Relation{{
					Kind:       "reply",
					Target:     relationshipReference(t, chatwork.ReferenceMessage, "987654321"),
					Resolved:   true,
					ExternalID: room.Value,
				}},
			}},
			code: "inconsistent_chatwork_message_relation",
		},
		"resolved parent room mismatch": {
			messages: []chatwork.Message{message, {
				Ref:  relationshipReference(t, chatwork.ReferenceMessage, "102"),
				Room: room,
				Replies: []chatwork.Relation{{
					Kind:       "reply",
					Target:     message.Ref,
					Resolved:   true,
					ExternalID: "999999999",
				}},
			}},
			code: "inconsistent_chatwork_message_relation",
		},
		"resolved non reply kind": {
			messages: []chatwork.Message{message, {
				Ref:  relationshipReference(t, chatwork.ReferenceMessage, "102"),
				Room: room,
				Replies: []chatwork.Relation{{
					Kind:       "quote",
					Target:     message.Ref,
					Resolved:   true,
					ExternalID: room.Value,
				}},
			}},
			code: "inconsistent_chatwork_message_relation",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ResolveMessageRelations(test.messages)
			if err == nil || got != nil {
				t.Fatalf("got = %#v, err = %v", got, err)
			}
			var structured *fault.Error
			if !errors.As(err, &structured) || structured.Kind != fault.KindContract ||
				structured.Code != test.code || structured.Retryable || errors.Unwrap(structured) != nil ||
				structured.Validate() != nil {
				t.Fatalf("unsafe contract fault = %#v", err)
			}
			if structured.Message == "987654321" || structured.Message == "999999999" {
				t.Fatalf("fault leaked relationship data: %q", structured.Message)
			}
		})
	}
}

func TestResolveMessageRelationsPreservesProviderOrderAcrossDeepInterleavedReplies(t *testing.T) {
	room := relationshipReference(t, chatwork.ReferenceRoom, "42")
	messages := make([]chatwork.Message, 50)
	for index := range messages {
		messages[index] = chatwork.Message{
			Ref:  relationshipReference(t, chatwork.ReferenceMessage, fmt.Sprint(1000+index)),
			Room: room,
		}
		if index > 0 {
			parent := (index - 1) / 2
			messages[index].Replies = []chatwork.Relation{{
				Kind: "reply", Target: messages[parent].Ref, ExternalID: room.Value,
			}}
		}
	}

	got, err := ResolveMessageRelations(messages)
	if err != nil {
		t.Fatal(err)
	}
	for index := range got {
		if got[index].Ref != messages[index].Ref {
			t.Fatalf("provider order changed at %d: got %s, want %s", index, got[index].Ref.Value, messages[index].Ref.Value)
		}
		if index > 0 && (len(got[index].Replies) != 1 || !got[index].Replies[0].Resolved) {
			t.Fatalf("reply %d was not resolved: %+v", index, got[index].Replies[0])
		}
	}
}

func relationshipReference(t *testing.T, kind chatwork.ReferenceKind, value string) chatwork.Reference {
	t.Helper()
	reference, err := chatwork.NewReference(kind, value)
	if err != nil {
		t.Fatal(err)
	}
	return reference
}
