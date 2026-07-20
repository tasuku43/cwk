package main

import (
	"encoding/json"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// activeMessageIndexFixture proves that index selection is based on typed send
// time while presentation retains provider order. The source order is
// intentionally not chronological.
type activeMessageIndexFixture struct {
	Source         chatwork.Result
	NoneRequest    chatwork.Request
	RepliesRequest chatwork.Request
	NextRequest    chatwork.Request
	AnswerKey      json.RawMessage
	NextArgv       []string
}

type activeMessageIndexScenario struct {
	ID                       string
	UserPrompt               string
	CommandArgv              [][]string
	ProviderCallBudget       int
	ExternalProcessingBudget int
	AnswerShape              string
	AnswerKey                json.RawMessage
}

func messageIndexFixture() activeMessageIndexFixture {
	room := ref(chatwork.ReferenceRoom, "3601")
	a := account("2601", "Aki")
	b := account("2602", "Beni")

	messages := []chatwork.Message{
		withReply(message("1201", room, a, "Newest message replies to an older parent.", 1702000060), "1202", false),
		message("1202", room, b, "Direct parent selected only as context.", 1702000010),
		message("1203", room, a, "Second-newest primary message.", 1702000050),
		withReply(message("1204", room, b, "Direct child selected only as context.", 1702000020), "1203", false),
		message("1205", room, a, "Third-newest message outside the limit.", 1702000040),
		message("1206", room, b, "Another older message.", 1702000030),
	}

	request := chatwork.Request{
		Task:        chatwork.TaskMessagesList,
		Room:        room,
		ForceRecent: true,
		MessageFilter: chatwork.MessageFilter{
			Context:    chatwork.MessageContextNone,
			StartIndex: 1,
			Count:      2,
		},
	}
	repliesRequest := request
	repliesRequest.MessageFilter.Context = chatwork.MessageContextReplies
	nextRequest := request
	nextRequest.MessageFilter.StartIndex = 3
	nextRequest.MessageFilter.Count = 3

	return activeMessageIndexFixture{
		Source: chatwork.Result{
			Task:        chatwork.TaskMessagesList,
			MessageRoom: room,
			Coverage: chatwork.Coverage{
				Kind: "latest_window", Limit: 100, Complete: false,
				Description: "synthetic non-chronological source for newest-message selection",
			},
			Messages: messages,
		},
		NoneRequest:    request,
		RepliesRequest: repliesRequest,
		NextRequest:    nextRequest,
		AnswerKey:      raw(`{"source_count":6,"candidate_count":6,"start_index":1,"requested_count":2,"items_per_page":2,"next_start_index":3,"context":"replies","displayed_sequence":[1,2,3,4],"anchor_sequence":[1,3],"context_sequence":[2,4],"primary_message_refs":["1201","1203"],"continuation":{"start_index":3,"requested_count":3,"items_per_page":3,"anchor_sequence":[4,5,6],"primary_message_refs":["1204","1205","1206"]},"next_command":{"path":"messages show","room_ref":"3601","message_ref":"1201"}}`),
		NextArgv:       []string{"messages", "show", "--room", "3601", "--message", "1201"},
	}
}

func messageIndexScenario() activeMessageIndexScenario {
	fixture := messageIndexFixture()
	return activeMessageIndexScenario{
		ID: "active.message-index",
		UserPrompt: "In exact room 3601, first return the newest two primary messages from the recent provider window with direct typed reply context, then return primary ranks 3 through 5 without repeating ranks 1 through 2. " +
			"Report source count, candidate count, requested and actual page size, next start index, selected anchors, added context, and the exact references needed to show the newest primary message. " +
			"Use exactly two cwk commands; do not use jq, grep, an external parser, source inspection, provider-order assumptions, or extra Chatwork calls.",
		CommandArgv: [][]string{
			{"messages", "list", "--room", "3601", "--count", "2", "--context", "replies"},
			{"messages", "list", "--room", "3601", "--start-index", "3", "--count", "3"},
		},
		ProviderCallBudget:       2,
		ExternalProcessingBudget: 0,
		AnswerShape:              `{"source_count":0,"candidate_count":0,"start_index":0,"requested_count":0,"items_per_page":0,"next_start_index":0,"context":"replies","displayed_sequence":[0],"anchor_sequence":[0],"context_sequence":[0],"primary_message_refs":["<message-ref>"],"continuation":{"start_index":0,"requested_count":0,"items_per_page":0,"anchor_sequence":[0],"primary_message_refs":["<message-ref>"]},"resolved_replies":{"<message-ref>":"<parent-message-ref>"},"next_command":{"path":"messages show","room_ref":"<room-ref>","message_ref":"<message-ref>"}}`,
		AnswerKey:                fixture.AnswerKey,
	}
}
