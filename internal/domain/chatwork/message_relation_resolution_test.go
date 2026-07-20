package chatwork

import "testing"

func TestDeriveMessageReachabilityUsesOnlyTrustworthyRecentBoundary(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	account := Reference{Kind: ReferenceAccount, Value: "7"}
	source := []Message{
		{Ref: Reference{Kind: ReferenceMessage, Value: "101"}, Room: room, Sender: Account{Ref: account}, SendTime: 300},
		{Ref: Reference{Kind: ReferenceMessage, Value: "102"}, Room: room, Sender: Account{Ref: account}, SendTime: 100},
		{Ref: Reference{Kind: ReferenceMessage, Value: "103"}, Room: room, Sender: Account{Ref: account}, SendTime: 200},
	}
	latest := Coverage{Kind: "latest_window", Limit: 100, Complete: false}

	for name, test := range map[string]struct {
		period MessagePeriod
		want   MessagePeriodReachability
	}{
		"within":  {MessagePeriod{Since: 100, Until: 400}, MessagePeriodWithinReachableWindow},
		"partial": {MessagePeriod{Since: 50, Until: 400}, MessagePeriodPartiallyOutsideReachable},
		"before":  {MessagePeriod{Since: 10, Until: 100}, MessagePeriodOutsideReachableWindow},
		"until":   {MessagePeriod{Until: 200}, MessagePeriodPartiallyOutsideReachable},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := DeriveMessageReachability(latest, MessageAccessNone, source, test.period)
			if err != nil {
				t.Fatal(err)
			}
			if got.OldestMessage.Value != "102" || got.OldestSendTime != 100 || got.PeriodReachability != test.want {
				t.Fatalf("reachability = %+v", got)
			}
		})
	}

	for name, test := range map[string]struct {
		coverage Coverage
		access   MessageAccessLimitation
		source   []Message
	}{
		"changes": {Coverage{Kind: "differential_window", Limit: 100}, MessageAccessNone, source},
		"partial": {latest, MessageAccessPartial, source},
		"empty":   {latest, MessageAccessNone, []Message{}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := DeriveMessageReachability(test.coverage, test.access, test.source, MessagePeriod{Until: 50})
			if err != nil {
				t.Fatal(err)
			}
			if got.OldestMessage != (Reference{}) || got.OldestSendTime != 0 || got.PeriodReachability != MessagePeriodReachabilityUnknown {
				t.Fatalf("unproved reachability = %+v", got)
			}
		})
	}
}

func TestMessageRelationResolutionValidatesBudgetEvidenceAndReplyState(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	account := Reference{Kind: ReferenceAccount, Value: "7"}
	target := Reference{Kind: ReferenceMessage, Value: "101"}
	child := Message{
		Ref: Reference{Kind: ReferenceMessage, Value: "102"}, Room: room,
		Sender: Account{Ref: account}, SendTime: 200,
		Reply: &Relation{Kind: "reply", Target: target, ExternalID: room.Value, Resolved: true},
	}
	contextMessage := Message{Ref: target, Room: room, Sender: Account{Ref: account}, SendTime: 100}
	request := Request{Task: TaskMessagesList, Room: room, MessageRelationFetchLimit: 1}
	result := Result{
		Task: TaskMessagesList, MessageRoom: room,
		Coverage:            Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages:            []Message{child},
		MessageReachability: &MessageReachability{OldestMessage: child.Ref, OldestSendTime: child.SendTime},
		MessageRelationResolution: &MessageRelationResolution{
			FetchLimit: 1, FetchAttempts: 1,
			Targets: []MessageRelationTarget{{Target: target, State: MessageRelationResolvedByFetch, Message: &contextMessage}},
		},
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("valid relation resolution failed: %v", err)
	}

	for name, mutate := range map[string]func(*Result){
		"wrong target": func(value *Result) {
			value.MessageRelationResolution.Targets[0].Target = Reference{Kind: ReferenceMessage, Value: "999"}
		},
		"wrong attempt count": func(value *Result) {
			value.MessageRelationResolution.FetchAttempts = 0
		},
		"resolved without context": func(value *Result) {
			value.MessageRelationResolution.Targets[0].Message = nil
		},
		"reply state mismatch": func(value *Result) {
			value.Messages[0].Reply.Resolved = false
		},
	} {
		t.Run(name, func(t *testing.T) {
			copyResult := result
			copyResult.Messages = cloneDomainMessages(result.Messages)
			resolution := *result.MessageRelationResolution
			resolution.Targets = append([]MessageRelationTarget(nil), result.MessageRelationResolution.Targets...)
			copyResult.MessageRelationResolution = &resolution
			mutate(&copyResult)
			if err := copyResult.ValidateFor(request); err == nil {
				t.Fatal("invalid relation resolution passed")
			}
		})
	}
}

func TestMessageRelationResolutionValidatesRecursiveContextOrder(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	account := Reference{Kind: ReferenceAccount, Value: "7"}
	parentRef := Reference{Kind: ReferenceMessage, Value: "101"}
	rootRef := Reference{Kind: ReferenceMessage, Value: "100"}
	child := Message{
		Ref: Reference{Kind: ReferenceMessage, Value: "102"}, Room: room,
		Sender: Account{Ref: account}, SendTime: 200,
		Reply: &Relation{Kind: "reply", Target: parentRef, ExternalID: room.Value, Resolved: true},
	}
	parent := Message{
		Ref: parentRef, Room: room, Sender: Account{Ref: account}, SendTime: 100,
		Reply: &Relation{Kind: "reply", Target: rootRef, ExternalID: room.Value, Resolved: true},
	}
	root := Message{Ref: rootRef, Room: room, Sender: Account{Ref: account}, SendTime: 50}
	request := Request{Task: TaskMessagesList, Room: room, MessageRelationFetchLimit: 2}
	result := Result{
		Task: TaskMessagesList, MessageRoom: room,
		Coverage:            Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages:            []Message{child},
		MessageReachability: &MessageReachability{OldestMessage: child.Ref, OldestSendTime: child.SendTime},
		MessageRelationResolution: &MessageRelationResolution{
			FetchLimit: 2, FetchAttempts: 2,
			Targets: []MessageRelationTarget{
				{Target: parentRef, State: MessageRelationResolvedByFetch, Message: &parent},
				{Target: rootRef, State: MessageRelationResolvedByFetch, Message: &root},
			},
		},
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("valid recursive resolution failed: %v", err)
	}
	result.MessageRelationResolution.Targets[0], result.MessageRelationResolution.Targets[1] =
		result.MessageRelationResolution.Targets[1], result.MessageRelationResolution.Targets[0]
	if err := result.ValidateFor(request); err == nil {
		t.Fatal("out-of-order recursive resolution passed")
	}
}

func TestExactMessageResultBindsRequestedRoomAndMessage(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "42"}
	message := Reference{Kind: ReferenceMessage, Value: "101"}
	account := Reference{Kind: ReferenceAccount, Value: "7"}
	request := Request{Task: TaskMessagesShow, Room: room, Message: message}
	result := Result{
		Task:     TaskMessagesShow,
		Coverage: Coverage{Kind: "single_operation", Complete: true},
		Messages: []Message{{Ref: message, Room: room, Sender: Account{Ref: account}, SendTime: 100}},
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatal(err)
	}
	result.Messages[0].Ref = Reference{Kind: ReferenceMessage, Value: "102"}
	if err := result.ValidateFor(request); err == nil {
		t.Fatal("mismatched exact message passed")
	}
}

func cloneDomainMessages(messages []Message) []Message {
	cloned := append([]Message(nil), messages...)
	for index := range cloned {
		if cloned[index].Reply != nil {
			reply := *cloned[index].Reply
			cloned[index].Reply = &reply
		}
	}
	return cloned
}
