package chatwork

import (
	"testing"
	"time"
)

func TestMessagePeriodValidationAndTokyoDay(t *testing.T) {
	day, err := NewMessageDayPeriod("2026-07-17")
	if err != nil {
		t.Fatal(err)
	}
	location := time.FixedZone(MessageDayTimeZone, 9*60*60)
	wantStart := time.Date(2026, 7, 17, 0, 0, 0, 0, location).Unix()
	if day.Since != wantStart || day.Until != wantStart+24*60*60 || day.Day != "2026-07-17" || day.TimeZone != MessageDayTimeZone {
		t.Fatalf("day period = %+v", day)
	}
	if !day.Contains(day.Since) || !day.Contains(day.Until-1) || day.Contains(day.Since-1) || day.Contains(day.Until) {
		t.Fatal("day period is not half-open")
	}

	for name, period := range map[string]MessagePeriod{
		"since only": {Since: 100},
		"until only": {Until: 200},
		"both":       {Since: 100, Until: 200},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewMessagePeriod(period.Since, period.Until); err != nil {
				t.Fatal(err)
			}
		})
	}
	for name, period := range map[string]MessagePeriod{
		"negative":         {Since: -1},
		"empty":            {Since: 100, Until: 100},
		"reversed":         {Since: 200, Until: 100},
		"day without zone": {Since: day.Since, Until: day.Until, Day: day.Day},
		"wrong day bounds": {Since: day.Since + 1, Until: day.Until, Day: day.Day, TimeZone: MessageDayTimeZone},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateMessagePeriod(period); err == nil {
				t.Fatal("invalid period passed")
			}
		})
	}
	for _, value := range []string{"7/17", "2026-7-17", "2026-02-30", "today"} {
		if _, err := NewMessageDayPeriod(value); err == nil {
			t.Fatalf("invalid day %q passed", value)
		}
	}
}

func TestMessagePeriodFilterAndSelectionBindExactAnchors(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	sender := Reference{Kind: ReferenceAccount, Value: "7"}
	period := MessagePeriod{Since: 200, Until: 400}
	request := Request{
		Task: TaskMessagesList, Room: room,
		MessageFilter: MessageFilter{Period: period, Context: MessageContextReplies},
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("period request failed: %v", err)
	}

	parent := Message{
		Ref: Reference{Kind: ReferenceMessage, Value: "201"}, Room: room,
		Sender: Account{Ref: sender}, SendTime: 100,
	}
	child := Message{
		Ref: Reference{Kind: ReferenceMessage, Value: "202"}, Room: room,
		Sender: Account{Ref: sender}, SendTime: 200,
		Replies: []Relation{{Kind: "reply", Target: parent.Ref, ExternalID: room.Value, Resolved: true}},
	}
	result := Result{
		Task: TaskMessagesList, MessageRoom: room,
		Coverage: Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages: []Message{parent, child},
		MessageSelection: &MessageSelection{
			Filter:          request.MessageFilter,
			SourceCount:     2,
			CandidateCount:  1,
			SourceSequences: []int{1, 2},
			AnchorSequences: []int{2},
		},
		MessageReachability: &MessageReachability{
			OldestMessage: parent.Ref, OldestSendTime: parent.SendTime,
			PeriodReachability: MessagePeriodWithinReachableWindow,
		},
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("out-of-period direct context was rejected: %v", err)
	}

	invalid := result
	invalid.MessageSelection = cloneMessageSelection(result.MessageSelection)
	invalid.MessageSelection.AnchorSequences = []int{1}
	if err := invalid.Validate(); err == nil {
		t.Fatal("out-of-period primary anchor passed")
	}
}
