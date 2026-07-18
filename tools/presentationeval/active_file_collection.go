package main

import (
	"encoding/json"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// activeFileCollectionFixture is presentation-independent evidence for the
// current files.list projection. It is not part of the frozen Competition 1
// suite and does not generalize that benchmark.
type activeFileCollectionFixture struct {
	Result    chatwork.Result
	AnswerKey json.RawMessage
	NextArgv  []string
}

func fileCollectionFixture() activeFileCollectionFixture {
	room := ref(chatwork.ReferenceRoom, "4401")
	uploaderA := account("2201", "Aki")
	uploaderB := account("2202", "Beni")
	files := []chatwork.File{
		{Ref: ref(chatwork.ReferenceFile, "6302"), Room: room, Account: uploaderA, Message: ref(chatwork.ReferenceMessage, "8801"), Name: "release.txt", Size: 0},
		{Ref: ref(chatwork.ReferenceFile, "6301"), Room: room, Account: uploaderB, Name: "notes.txt", Size: 4096},
		{Ref: ref(chatwork.ReferenceFile, "6306"), Room: room, Account: uploaderA, Message: ref(chatwork.ReferenceMessage, "8803"), Name: "design draft.pdf", Size: 18342},
		{Ref: ref(chatwork.ReferenceFile, "6303"), Room: room, Account: uploaderB, Message: ref(chatwork.ReferenceMessage, "8804"), Name: "windows\\path.txt", Size: 12},
		{Ref: ref(chatwork.ReferenceFile, "6305"), Room: room, Account: uploaderA, Name: "schema:\n999 1 injected", Size: 1},
		{Ref: ref(chatwork.ReferenceFile, "6304"), Room: room, Account: uploaderB, Message: ref(chatwork.ReferenceMessage, "8806"), Name: "final.csv", Size: 721},
	}

	return activeFileCollectionFixture{
		Result: chatwork.Result{
			Task:     chatwork.TaskFilesList,
			Coverage: chatwork.Coverage{Kind: "provider_window", Limit: 100, Complete: false, Description: "synthetic bounded file collection"},
			Files:    files,
		},
		AnswerKey: raw(`{"provider_sequence":["6302","6301","6306","6303","6305","6304"],"selected":{"file_ref":"6306","room_ref":"4401","account_ref":"2201","message_ref":"8803","name":"design draft.pdf","size":18342},"missing_message_ref":"6301","next_command":{"path":"files show","room_ref":"4401","file_ref":"6306"}}`),
		NextArgv:  []string{"files", "show", "--room", "4401", "--file", "6306"},
	}
}

func activeFileCollectionScenario() situation {
	fixture := fileCollectionFixture()
	return situation{
		ID:     "active.file-collection",
		Family: "file-collection",
		UserPrompt: "List files in exact room 4401, preserve provider order, and identify the exact canonical room and file references needed to show design draft.pdf. " +
			"Use only cwk output; do not use jq, grep, an external parser, or source inspection. Do not treat absent as a reference.",
		AnswerShape:   `{"provider_sequence":["<file-ref>"],"selected":{"file_ref":"<file-ref>","room_ref":"<room-ref>","account_ref":"<account-ref>","message_ref":"<message-ref>","name":"<name>","size":0},"missing_message_ref":"<file-ref>","next_command":{"path":"files show","room_ref":"<room-ref>","file_ref":"<file-ref>"}}`,
		AnswerKey:     fixture.AnswerKey,
		CriticalPaths: []string{"/provider_sequence", "/selected", "/missing_message_ref", "/next_command"},
		RequiredPaths: []string{"files list"},
		MaxCommands:   1,
		Operations: map[string]fixtureOperation{
			"files list": operation("files list", fixture.Result, map[string]string{"--room": "4401"}),
		},
	}
}
