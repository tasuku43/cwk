package chatwork

import "testing"

func TestMessagesShowContextRequestValidation(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	message := Reference{Kind: ReferenceMessage, Value: "101"}
	base := Request{Task: TaskMessagesShow, Room: room, Message: message}

	valid := []Request{
		base,
		func() Request {
			value := base
			value.MessageFilter.Context = MessageContextNone
			return value
		}(),
		func() Request {
			value := base
			value.MessageFilter.Context = MessageContextReplies
			value.MessageRelationFetchLimit = DefaultMessageRelationFetches
			return value
		}(),
	}
	for index, request := range valid {
		if err := request.Validate(); err != nil {
			t.Errorf("valid request[%d] failed: %v", index, err)
		}
	}

	invalid := []Request{
		func() Request {
			value := base
			value.MessageRelationFetchLimit = 1
			return value
		}(),
		func() Request {
			value := base
			value.MessageFilter.Context = MessageContextNone
			value.MessageRelationFetchLimit = 1
			return value
		}(),
		func() Request {
			value := base
			value.MessageFilter.Context = MessageContext("thread")
			return value
		}(),
		func() Request {
			value := base
			value.MessageFilter.Context = MessageContextReplies
			value.MessageFilter.StartIndex = 1
			return value
		}(),
	}
	for index, request := range invalid {
		if err := request.Validate(); err == nil {
			t.Errorf("invalid request[%d] passed: %+v", index, request)
		}
	}
}

func TestMessagesShowResultBindsTypedRelationResolutionToAnchorAndBudget(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	account := Reference{Kind: ReferenceAccount, Value: "7"}
	anchorRef := Reference{Kind: ReferenceMessage, Value: "102"}
	parentRef := Reference{Kind: ReferenceMessage, Value: "101"}
	request := Request{
		Task: TaskMessagesShow, Room: room, Message: anchorRef,
		MessageFilter: MessageFilter{Context: MessageContextReplies}, MessageRelationFetchLimit: 1,
	}
	anchor := Message{
		Ref: anchorRef, Room: room, Sender: Account{Ref: account}, SendTime: 200,
		Replies: []Relation{{Kind: "reply", Target: parentRef, ExternalID: room.Value, Resolved: true}},
	}
	parent := Message{Ref: parentRef, Room: room, Sender: Account{Ref: account}, SendTime: 100}
	result := Result{
		Task: TaskMessagesShow, Coverage: Coverage{Kind: "single_operation", Complete: true},
		Messages: []Message{anchor},
		MessageShowContext: &MessageShowContext{
			Anchor: anchorRef, Children: []Message{}, SourceSequences: []int{}, SourceCount: 0,
			Coverage: Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		},
		MessageRelationResolution: &MessageRelationResolution{
			FetchLimit: 1, FetchAttempts: 1,
			Targets: []MessageRelationTarget{{Target: parentRef, State: MessageRelationResolvedByFetch, Message: &parent}},
		},
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("valid exact-message resolution failed: %v", err)
	}

	wrongBudget := result
	resolution := *result.MessageRelationResolution
	resolution.FetchLimit = 2
	wrongBudget.MessageRelationResolution = &resolution
	if err := wrongBudget.ValidateFor(request); err == nil {
		t.Fatal("wrong exact-message resolution budget passed")
	}

	withoutBudget := request
	withoutBudget.MessageFilter.Context = MessageContextNone
	withoutBudget.MessageRelationFetchLimit = 0
	if err := result.ValidateFor(withoutBudget); err == nil {
		t.Fatal("unexpected exact-message resolution passed without a request budget")
	}
}

func TestMessagesShowContextBindsDirectTypedChildrenAndBoundedCoverage(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	account := Reference{Kind: ReferenceAccount, Value: "7"}
	anchorRef := Reference{Kind: ReferenceMessage, Value: "100"}
	anchor := Message{Ref: anchorRef, Room: room, Sender: Account{Ref: account}, Body: "anchor"}
	child := Message{
		Ref: Reference{Kind: ReferenceMessage, Value: "101"}, Room: room,
		Sender: Account{Ref: account}, Body: "child",
		Replies: []Relation{{Kind: "reply", Target: anchorRef, ExternalID: room.Value, Resolved: true}},
	}
	request := Request{
		Task: TaskMessagesShow, Room: room, Message: anchorRef,
		MessageFilter: MessageFilter{Context: MessageContextReplies},
	}
	result := Result{
		Task: TaskMessagesShow, Coverage: Coverage{Kind: "single_operation", Complete: true},
		Messages: []Message{anchor},
		MessageShowContext: &MessageShowContext{
			Anchor: anchorRef, Children: []Message{child}, SourceSequences: []int{2}, SourceCount: 3,
			Coverage: Coverage{Kind: "latest_window", Limit: 100, Complete: false}, Access: MessageAccessPartial,
		},
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("valid reply context failed: %v", err)
	}

	tests := map[string]func(*Result){
		"wrong anchor":          func(value *Result) { value.MessageShowContext.Anchor = child.Ref },
		"out of range sequence": func(value *Result) { value.MessageShowContext.SourceSequences[0] = 4 },
		"wrong room": func(value *Result) {
			value.MessageShowContext.Children[0].Room = Reference{Kind: ReferenceRoom, Value: "43"}
		},
		"unresolved edge":    func(value *Result) { value.MessageShowContext.Children[0].Replies[0].Resolved = false },
		"wrong typed target": func(value *Result) { value.MessageShowContext.Children[0].Replies[0].Target = child.Ref },
		"unbounded coverage": func(value *Result) { value.MessageShowContext.Coverage.Limit = 101 },
		"resolved hidden child target": func(value *Result) {
			value.MessageShowContext.Children[0].Replies = append(value.MessageShowContext.Children[0].Replies, Relation{
				Kind: "reply", Target: Reference{Kind: ReferenceMessage, Value: "999"}, ExternalID: room.Value, Resolved: true,
			})
		},
		"empty partial source": func(value *Result) {
			value.MessageShowContext.Children = []Message{}
			value.MessageShowContext.SourceSequences = []int{}
			value.MessageShowContext.SourceCount = 0
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			copyResult := result
			copyContext := *result.MessageShowContext
			copyContext.Children = append([]Message(nil), result.MessageShowContext.Children...)
			copyContext.SourceSequences = append([]int(nil), result.MessageShowContext.SourceSequences...)
			copyContext.Children[0].Replies = append([]Relation(nil), result.MessageShowContext.Children[0].Replies...)
			copyResult.MessageShowContext = &copyContext
			mutate(&copyResult)
			if err := copyResult.ValidateFor(request); err == nil {
				t.Fatal("invalid exact-message context passed")
			}
		})
	}
}
