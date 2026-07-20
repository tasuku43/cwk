package chatwork

import "testing"

func TestMessageFilterStartIndexAndCountValidation(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	tests := []struct {
		name    string
		filter  MessageFilter
		wantErr bool
	}{
		{name: "absent", filter: MessageFilter{}},
		{name: "first index", filter: MessageFilter{Context: MessageContextNone, StartIndex: 1}},
		{name: "provider bound index", filter: MessageFilter{Context: MessageContextNone, StartIndex: 100}},
		{name: "index and count", filter: MessageFilter{Context: MessageContextNone, StartIndex: 11, Count: 20}},
		{name: "reply context around indexed slice", filter: MessageFilter{Context: MessageContextReplies, StartIndex: 11, Count: 20}},
		{name: "negative index", filter: MessageFilter{Context: MessageContextNone, StartIndex: -1}, wantErr: true},
		{name: "index above provider bound", filter: MessageFilter{Context: MessageContextNone, StartIndex: 101}, wantErr: true},
		{name: "count without defaulted index", filter: MessageFilter{Context: MessageContextNone, Count: 20}, wantErr: true},
		{name: "negative count", filter: MessageFilter{Context: MessageContextNone, StartIndex: 1, Count: -1}, wantErr: true},
		{name: "count above provider bound", filter: MessageFilter{Context: MessageContextNone, StartIndex: 1, Count: 101}, wantErr: true},
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
		MessageFilter: MessageFilter{Context: MessageContextNone, StartIndex: 1},
	}).Validate(); err == nil {
		t.Fatal("message start index on another task passed")
	}
}

func TestMessageIndexedSelectionBindsExactRequestAndNextStartIndex(t *testing.T) {
	result := validLimitedMessageResult()
	result.MessageSelection.Filter.StartIndex = 2
	result.MessageSelection.Filter.Count = 2
	result.MessageSelection.CandidateCount = 4
	result.MessageSelection.ItemsPerPage = 2
	result.MessageSelection.NextStartIndex = 4
	result.Messages = result.Messages[:2]
	result.MessageSelection.SourceSequences = []int{2, 4}
	result.MessageSelection.AnchorSequences = []int{2, 4}
	if err := result.Validate(); err != nil {
		t.Fatalf("valid indexed selection failed: %v", err)
	}

	request := Request{Task: TaskMessagesList, Room: result.MessageRoom, MessageFilter: result.MessageSelection.Filter}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("exact indexed request binding failed: %v", err)
	}
	different := request
	different.MessageFilter.StartIndex = 3
	if err := result.ValidateFor(different); err == nil {
		t.Fatal("selection bound to a different start index")
	}

	invalidNext := result
	invalidNext.MessageSelection = cloneMessageSelection(result.MessageSelection)
	invalidNext.MessageSelection.NextStartIndex = 3
	if err := invalidNext.Validate(); err == nil {
		t.Fatal("invalid next start index passed")
	}
}

func cloneMessageSelection(selection *MessageSelection) *MessageSelection {
	copy := *selection
	copy.Filter.Senders = append([]Reference(nil), selection.Filter.Senders...)
	copy.SourceSequences = append([]int(nil), selection.SourceSequences...)
	copy.AnchorSequences = append([]int(nil), selection.AnchorSequences...)
	return &copy
}
