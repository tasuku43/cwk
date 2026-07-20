package main

import (
	"encoding/json"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// The active readiness fixtures intentionally keep only the semantic data
// needed by current product tests. The frozen Competition 1 runner and scorer
// are historical evidence, not part of the ongoing repository gate.
type fixtureOperation struct {
	Path         string
	Result       chatwork.Result
	RequiredArgs map[string]string
}

type situation struct {
	ID             string
	Family         string
	UserPrompt     string
	AnswerShape    string
	AnswerKey      json.RawMessage
	CriticalPaths  []string
	RequiredPaths  []string
	ForbiddenPaths []string
	MaxCommands    int
	Operations     map[string]fixtureOperation
}

func operation(path string, result chatwork.Result, args map[string]string) fixtureOperation {
	return fixtureOperation{Path: path, Result: result, RequiredArgs: args}
}

func message(id string, room chatwork.Reference, sender chatwork.Account, body string, sent int64) chatwork.Message {
	return chatwork.Message{
		Ref: ref(chatwork.ReferenceMessage, id), Room: room, Sender: sender,
		Body: body, SendTime: sent, Recipients: []chatwork.Reference{}, Quotes: []chatwork.Relation{},
	}
}

func withTo(value chatwork.Message, target chatwork.Reference) chatwork.Message {
	value.Recipients = []chatwork.Reference{target}
	return value
}

func withReply(value chatwork.Message, target string, resolved bool) chatwork.Message {
	value.Replies = []chatwork.Relation{{
		Kind: "reply", Target: ref(chatwork.ReferenceMessage, target),
		Resolved: resolved, ExternalID: value.Room.Value,
	}}
	return value
}

func account(id, name string) chatwork.Account {
	return chatwork.Account{Ref: ref(chatwork.ReferenceAccount, id), Name: name}
}

func ref(kind chatwork.ReferenceKind, value string) chatwork.Reference {
	result, err := chatwork.NewReference(kind, value)
	if err != nil {
		panic(err)
	}
	return result
}

func raw(value string) json.RawMessage { return json.RawMessage(value) }
