package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

type activeMessageRelationClosureFixture struct {
	Source    chatwork.Result
	Exact     map[string]chatwork.Result
	Request   chatwork.Request
	AnswerKey json.RawMessage
}

type activeMessageReachabilityFixture struct {
	Source    chatwork.Result
	Request   chatwork.Request
	AnswerKey json.RawMessage
}

type activeMessageClosureScenario struct {
	ID                       string
	UserPrompt               string
	CommandArgv              []string
	ProviderCallBudget       int
	ExternalProcessingBudget int
	AnswerShape              string
	AnswerKey                json.RawMessage
}

func messageRelationClosureFixture() activeMessageRelationClosureFixture {
	room := ref(chatwork.ReferenceRoom, "3801")
	aki := account("2801", "Aki")
	beni := account("2802", "Beni")
	messages := make([]chatwork.Message, 100)
	for index := range messages {
		messages[index] = message(
			fmt.Sprint(3001+index), room, aki,
			fmt.Sprintf("Synthetic bounded source message %03d.", index+1),
			1784200000+int64(index)*60,
		)
	}
	messages[99] = withReply(messages[99], "9001", false)
	messages[99].Body = "Aurora follow-up depends on an explicit parent chain outside the source window."
	firstParent := withReply(message("9001", room, beni, "Aurora decision: use the staged archive migration.", 1784100000), "9002", false)
	secondParent := message("9002", room, beni, "Aurora owner: canonical account 2802; deadline: 2026-07-21.", 1784100060)
	request := chatwork.Request{
		Task: chatwork.TaskMessagesList, Room: room, ForceRecent: true,
		MessageRelationFetchLimit: chatwork.DefaultMessageRelationFetches,
	}
	return activeMessageRelationClosureFixture{
		Source: chatwork.Result{
			Task: chatwork.TaskMessagesList, MessageRoom: room,
			Coverage: chatwork.Coverage{Kind: "latest_window", Limit: 100, Complete: false, Description: "synthetic latest-100 source"},
			Messages: messages,
		},
		Exact: map[string]chatwork.Result{
			"9001": {Task: chatwork.TaskMessagesShow, Coverage: chatwork.Coverage{Kind: "single_operation", Complete: true}, Messages: []chatwork.Message{firstParent}},
			"9002": {Task: chatwork.TaskMessagesShow, Coverage: chatwork.Coverage{Kind: "single_operation", Complete: true}, Messages: []chatwork.Message{secondParent}},
		},
		Request:   request,
		AnswerKey: raw(`{"decision":"use the staged archive migration","owner_account_ref":"2802","deadline":"2026-07-21","source_count":100,"relation_fetch_limit":5,"relation_fetch_attempts":2,"resolved_targets":["9001","9002"],"provider_calls":3,"external_processing_calls":0}`),
	}
}

func messageReachabilityFixture() activeMessageReachabilityFixture {
	room := ref(chatwork.ReferenceRoom, "3802")
	aki := account("2801", "Aki")
	oldest := time.Date(2026, 7, 17, 0, 0, 0, 0, time.FixedZone(chatwork.MessageDayTimeZone, 9*60*60)).Unix()
	messages := make([]chatwork.Message, 100)
	for index := range messages {
		messages[index] = message(fmt.Sprint(4001+index), room, aki, "Synthetic reachable-window message.", oldest+int64(index)*60)
	}
	period, err := chatwork.NewMessageDayPeriod("2026-07-08")
	if err != nil {
		panic(err)
	}
	return activeMessageReachabilityFixture{
		Source: chatwork.Result{
			Task: chatwork.TaskMessagesList, MessageRoom: room,
			Coverage: chatwork.Coverage{Kind: "latest_window", Limit: 100, Complete: false, Description: "synthetic latest-100 source"},
			Messages: messages,
		},
		Request: chatwork.Request{
			Task: chatwork.TaskMessagesList, Room: room, ForceRecent: true,
			MessageFilter:             chatwork.MessageFilter{Period: period, Context: chatwork.MessageContextNone},
			MessageRelationFetchLimit: chatwork.DefaultMessageRelationFetches,
		},
		AnswerKey: raw(fmt.Sprintf(`{"requested_day":"2026-07-08","candidate_count":0,"period_reachability":"out-of-reachable-window","oldest_reachable_message_ref":"4001","oldest_reachable_send_time":%d,"provider_calls":1,"external_processing_calls":0}`, oldest)),
	}
}

func messageRelationClosureScenario() activeMessageClosureScenario {
	fixture := messageRelationClosureFixture()
	return activeMessageClosureScenario{
		ID:                       "active.message-relation-closure",
		UserPrompt:               "In exact room 3801, recover the Aurora decision, canonical owner, and deadline from the bounded message result. Use one cwk command and no messages show command, jq, grep, custom parser, raw Chatwork-notation parsing, source inspection, or guessed relation. State the provider-call count and do not claim arbitrary history access.",
		CommandArgv:              []string{"messages", "list", "--room", "3801"},
		ProviderCallBudget:       3,
		ExternalProcessingBudget: 0,
		AnswerShape:              `{"decision":"<text>","owner_account_ref":"<account-ref>","deadline":"YYYY-MM-DD","source_count":0,"relation_fetch_limit":0,"relation_fetch_attempts":0,"resolved_targets":["<message-ref>"],"provider_calls":0,"external_processing_calls":0}`,
		AnswerKey:                fixture.AnswerKey,
	}
}

func messageReachabilityScenario() activeMessageClosureScenario {
	fixture := messageReachabilityFixture()
	return activeMessageClosureScenario{
		ID:                       "active.message-period-reachability",
		UserPrompt:               "In exact room 3802, determine whether messages list can answer for Tokyo day 2026-07-08. Use one cwk command; do not probe adjacent dates, run messages show, use jq/grep, inspect source, or treat an unreachable period as an empty day.",
		CommandArgv:              []string{"messages", "list", "--room", "3802", "--on", "2026-07-08"},
		ProviderCallBudget:       1,
		ExternalProcessingBudget: 0,
		AnswerShape:              `{"requested_day":"YYYY-MM-DD","candidate_count":0,"period_reachability":"within-reachable-window|partially-out-of-reachable-window|out-of-reachable-window|unknown","oldest_reachable_message_ref":"<message-ref>","oldest_reachable_send_time":0,"provider_calls":0,"external_processing_calls":0}`,
		AnswerKey:                fixture.AnswerKey,
	}
}
