package chatwork

import (
	"strconv"
	"testing"
)

func TestMessageFilterLimitValidation(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	tests := []struct {
		name    string
		filter  MessageFilter
		wantErr bool
	}{
		{name: "absent", filter: MessageFilter{}},
		{name: "one", filter: MessageFilter{Context: MessageContextNone, Limit: 1}},
		{name: "provider bound", filter: MessageFilter{Context: MessageContextNone, Limit: 100}},
		{name: "reply context around limit anchors", filter: MessageFilter{Context: MessageContextReplies, Limit: 10}},
		{name: "negative", filter: MessageFilter{Context: MessageContextNone, Limit: -1}, wantErr: true},
		{name: "above provider bound", filter: MessageFilter{Context: MessageContextNone, Limit: 101}, wantErr: true},
		{name: "context without selector", filter: MessageFilter{Context: MessageContextReplies}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := (Request{Task: TaskMessagesList, Room: room, MessageFilter: test.filter}).Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}

	if err := (Request{
		Task:          TaskRoomsList,
		MessageFilter: MessageFilter{Context: MessageContextNone, Limit: 1},
	}).Validate(); err == nil {
		t.Fatal("message limit on another task passed")
	}
}

func TestMessageLimitSelectionValidatesCandidateCountAndExactRequest(t *testing.T) {
	valid := validLimitedMessageResult()
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid limited selection failed: %v", err)
	}
	request := Request{
		Task: TaskMessagesList, Room: valid.MessageRoom,
		MessageFilter: valid.MessageSelection.Filter,
	}
	if err := valid.ValidateFor(request); err != nil {
		t.Fatalf("exact limited request binding failed: %v", err)
	}

	differentLimit := request
	differentLimit.MessageFilter.Limit = 1
	if err := valid.ValidateFor(differentLimit); err == nil {
		t.Fatal("selection bound to a different local limit")
	}

	tests := map[string]func(*Result){
		"negative candidate count": func(result *Result) {
			result.MessageSelection.CandidateCount = -1
		},
		"candidate count above source": func(result *Result) {
			result.MessageSelection.CandidateCount = 5
		},
		"candidate count below anchors": func(result *Result) {
			result.MessageSelection.CandidateCount = 1
		},
		"too few anchors for candidate and limit": func(result *Result) {
			result.MessageSelection.AnchorSequences = []int{4}
			result.MessageSelection.SourceSequences = []int{4}
			result.Messages = result.Messages[1:]
		},
		"no-sender candidate count differs from source": func(result *Result) {
			result.MessageSelection.Filter.Senders = nil
			result.MessageSelection.CandidateCount = 3
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			result := validLimitedMessageResult()
			mutate(&result)
			if err := result.Validate(); err == nil {
				t.Fatal("invalid limited selection passed")
			}
		})
	}
}

func TestMessageLimitAllowsMatchingSenderToReappearOnlyAsDirectContext(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	sender := Reference{Kind: ReferenceAccount, Value: "7"}
	parent := Message{
		Ref: Reference{Kind: ReferenceMessage, Value: "201"}, Room: room,
		Sender: Account{Ref: sender}, SendTime: 100,
	}
	child := Message{
		Ref: Reference{Kind: ReferenceMessage, Value: "202"}, Room: room,
		Sender: Account{Ref: sender}, SendTime: 200,
		Reply: &Relation{
			Kind: "reply", Target: parent.Ref, ExternalID: room.Value, Resolved: true,
		},
	}
	result := Result{
		Task: TaskMessagesList, MessageRoom: room,
		Coverage: Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages: []Message{parent, child},
		MessageSelection: &MessageSelection{
			Filter:          MessageFilter{Senders: []Reference{sender}, Context: MessageContextReplies, Limit: 1},
			SourceCount:     2,
			CandidateCount:  2,
			SourceSequences: []int{1, 2},
			AnchorSequences: []int{2},
		},
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("same-sender direct context was rejected: %v", err)
	}

	unrelated := result
	unrelated.Messages = append([]Message(nil), result.Messages...)
	unrelated.Messages[1].Reply = nil
	if err := unrelated.Validate(); err == nil {
		t.Fatal("same-sender non-anchor without a direct reply edge passed")
	}
}

func TestMessageSourceWindowCannotExceedCoverageBeforeLocalLimit(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	sender := Reference{Kind: ReferenceAccount, Value: "7"}
	messages := make([]Message, 101)
	for index := range messages {
		messages[index] = Message{
			Ref:      Reference{Kind: ReferenceMessage, Value: strconv.Itoa(1000 + index)},
			Room:     room,
			Sender:   Account{Ref: sender},
			SendTime: int64(index),
		}
	}
	result := Result{
		Task: TaskMessagesList, MessageRoom: room,
		Coverage: Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages: messages,
	}
	if err := result.Validate(); err == nil {
		t.Fatal("101-message provider source passed a declared 100-message coverage bound")
	}

	result.Messages = result.Messages[:100]
	result.Coverage.Limit = 101
	if err := result.Validate(); err == nil {
		t.Fatal("provider source-limit above the fixed 100-message contract passed")
	}
}

func validLimitedMessageResult() Result {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	sender := Reference{Kind: ReferenceAccount, Value: "7"}
	return Result{
		Task: TaskMessagesList, MessageRoom: room,
		Coverage: Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages: []Message{
			{
				Ref: Reference{Kind: ReferenceMessage, Value: "201"}, Room: room,
				Sender: Account{Ref: sender}, SendTime: 200,
			},
			{
				Ref: Reference{Kind: ReferenceMessage, Value: "203"}, Room: room,
				Sender: Account{Ref: sender}, SendTime: 300,
			},
		},
		MessageSelection: &MessageSelection{
			Filter:          MessageFilter{Senders: []Reference{sender}, Context: MessageContextNone, Limit: 2},
			SourceCount:     4,
			CandidateCount:  4,
			SourceSequences: []int{2, 4},
			AnchorSequences: []int{2, 4},
		},
	}
}
