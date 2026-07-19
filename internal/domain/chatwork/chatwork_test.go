package chatwork

import (
	"fmt"
	"strings"
	"testing"
)

func TestReferencePreservesCanonicalDecimalIdentity(t *testing.T) {
	ref, err := NewReference(ReferenceMessage, "1928374655918273645")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Value != "1928374655918273645" {
		t.Fatalf("value = %q", ref.Value)
	}
}

func TestReferenceRejectsAlternateAndUnsafeForms(t *testing.T) {
	for _, value := range []string{"", "0", "01", "+1", "1 ", "１", "1\n2", "123456789012345678901234567890123"} {
		t.Run(value, func(t *testing.T) {
			if _, err := NewReference(ReferenceRoom, value); err == nil {
				t.Fatalf("NewReference(%q) succeeded", value)
			}
		})
	}
	if _, err := NewReference(ReferenceKind("room"), "1"); err == nil {
		t.Fatal("unknown reference kind succeeded")
	}
}

func TestEveryTaskIsValid(t *testing.T) {
	tasks := []Task{
		TaskAccountShow, TaskAccountStatus, TaskPersonalTasksList, TaskContactsList,
		TaskRoomsList, TaskRoomsCreate, TaskRoomsShow, TaskRoomsUpdate, TaskRoomsLeave,
		TaskRoomsDelete, TaskMembersList, TaskMembersReplace, TaskMessagesList,
		TaskMessagesSend, TaskMessagesMarkRead, TaskMessagesMarkUnread, TaskMessagesShow,
		TaskMessagesUpdate, TaskMessagesDelete, TaskRoomTasksList, TaskRoomTasksCreate,
		TaskRoomTasksShow, TaskRoomTasksSetStatus, TaskFilesList, TaskFilesUpload,
		TaskFilesShow, TaskInviteLinkShow, TaskInviteLinkCreate, TaskInviteLinkUpdate,
		TaskInviteLinkDelete, TaskContactRequestsList, TaskContactRequestsAccept,
		TaskContactRequestsReject,
	}
	if len(tasks) != 33 {
		t.Fatalf("task count = %d, want 33", len(tasks))
	}
	seen := make(map[Task]struct{}, len(tasks))
	for _, task := range tasks {
		if !task.Valid() {
			t.Fatalf("task %q is invalid", task)
		}
		if _, exists := seen[task]; exists {
			t.Fatalf("duplicate task %q", task)
		}
		seen[task] = struct{}{}
	}
	if Task("rooms.get").Valid() {
		t.Fatal("unknown provider-shaped task is valid")
	}
}

func TestRequestRejectsMismatchedAndDuplicateReferences(t *testing.T) {
	account, err := NewReference(ReferenceAccount, "12")
	if err != nil {
		t.Fatal(err)
	}
	request := Request{Task: TaskRoomsCreate, Room: account}
	if err := request.Validate(); err == nil {
		t.Fatal("mismatched room reference succeeded")
	}
	request = Request{Task: TaskRoomsCreate, Admins: []Reference{account, account}}
	if err := request.Validate(); err == nil {
		t.Fatal("duplicate account role succeeded")
	}
}

func TestRequestRejectsInvalidText(t *testing.T) {
	request := Request{Task: TaskMessagesSend, Body: "hello\x00world"}
	if err := request.Validate(); err == nil {
		t.Fatal("NUL body succeeded")
	}
}

func TestRoomsCreateRequiresAuthenticatedAccountScope(t *testing.T) {
	account := Reference{Kind: ReferenceAccount, Value: "1"}
	request := Request{Task: TaskRoomsCreate, Name: "project", Admins: []Reference{account}}
	if err := request.Validate(); err == nil {
		t.Fatal("room creation without authenticated account reference succeeded")
	}
	request.Account = account
	if err := request.Validate(); err != nil {
		t.Fatalf("room creation with authenticated account reference failed: %v", err)
	}

	for name, mutate := range map[string]func(*Request){
		"missing name":          func(value *Request) { value.Name = "" },
		"oversized name":        func(value *Request) { value.Name = strings.Repeat("名", 256) },
		"missing administrator": func(value *Request) { value.Admins = nil },
		"invalid icon":          func(value *Request) { value.Icon = "arbitrary" },
		"disabled invite input": func(value *Request) { value.InviteCode = "public-room" },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := request
			mutate(&invalid)
			if err := invalid.Validate(); err == nil {
				t.Fatal("invalid room creation input succeeded")
			}
		})
	}
	validIcon := request
	validIcon.Icon = "meeting"
	if err := validIcon.Validate(); err != nil {
		t.Fatalf("official room icon failed: %v", err)
	}
}

func TestInviteLinkUpdateRequiresCompleteExplicitReplacement(t *testing.T) {
	valid := Request{
		Task:              TaskInviteLinkUpdate,
		Invite:            Reference{Kind: ReferenceInvite, Value: "7"},
		InviteCode:        "valid-code_1",
		InviteApprovalSet: true,
		Description:       "replacement",
		DescriptionSet:    true,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("complete invite-link replacement failed: %v", err)
	}
	regenerate := valid
	regenerate.InviteCode = ""
	regenerate.InviteRegenerateCode = true
	if err := regenerate.Validate(); err != nil {
		t.Fatalf("explicit invite-link code regeneration failed: %v", err)
	}

	invalid := map[string]Request{
		"missing invite":          func() Request { value := valid; value.Invite = Reference{}; return value }(),
		"missing code intent":     func() Request { value := valid; value.InviteCode = ""; return value }(),
		"code and regeneration":   func() Request { value := valid; value.InviteRegenerateCode = true; return value }(),
		"missing approval":        func() Request { value := valid; value.InviteApprovalSet = false; return value }(),
		"missing description":     func() Request { value := valid; value.DescriptionSet = false; return value }(),
		"empty description":       func() Request { value := valid; value.Description = ""; return value }(),
		"invalid code characters": func() Request { value := valid; value.InviteCode = "invalid!"; return value }(),
		"oversized code":          func() Request { value := valid; value.InviteCode = string(make([]byte, 51)); return value }(),
	}
	for name, request := range invalid {
		t.Run(name, func(t *testing.T) {
			if err := request.Validate(); err == nil {
				t.Fatal("invalid invite-link replacement succeeded")
			}
		})
	}
}

func TestInviteLinkCreateValidatesOfficialCodeAlphabetAndLength(t *testing.T) {
	valid := Request{Task: TaskInviteLinkCreate, InviteCode: "Az09_-"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid invite-link code failed: %v", err)
	}
	for _, code := range []string{"invalid!", "コード", string(make([]byte, 51))} {
		request := valid
		request.InviteCode = code
		if err := request.Validate(); err == nil {
			t.Errorf("invalid invite-link code %q succeeded", code)
		}
	}
}

func TestInviteLinkMutationResultMustMatchRequestedReference(t *testing.T) {
	requested := Reference{Kind: ReferenceInvite, Value: "7"}
	other := Reference{Kind: ReferenceInvite, Value: "8"}
	request := Request{
		Task: TaskInviteLinkUpdate, Invite: requested, InviteCode: "replacement",
		InviteApprovalSet: true, Description: "description", DescriptionSet: true,
	}
	result := Result{
		Task: TaskInviteLinkUpdate, Coverage: Coverage{Kind: "confirmed", Complete: true},
		InviteLink: &InviteLink{Ref: requested},
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("matching invite-link result failed: %v", err)
	}
	result.InviteLink.Ref = other
	if err := result.ValidateFor(request); err == nil {
		t.Fatal("invite-link result for another target succeeded")
	}
}

func TestResultRequiresTaskSpecificSemanticVariant(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "2"}
	message := Reference{Kind: ReferenceMessage, Value: "3"}
	valid := Result{
		Task: TaskMessagesSend,
		CreatedInRoom: &RoomScopedCreation{
			Refs:       []Reference{message},
			ParentRoom: room,
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid room-scoped creation failed: %v", err)
	}

	wrongVariant := valid
	wrongVariant.CreatedInRoom = nil
	wrongVariant.Created = []Reference{message}
	if err := wrongVariant.Validate(); err == nil {
		t.Fatal("generic created reference passed for room-scoped creation")
	}

	missingParent := valid
	missingParent.CreatedInRoom = &RoomScopedCreation{Refs: []Reference{message}}
	if err := missingParent.Validate(); err == nil {
		t.Fatal("room-scoped creation without parent passed")
	}

	duplicate := Result{Task: TaskRoomTasksCreate, CreatedInRoom: &RoomScopedCreation{
		Refs: []Reference{
			{Kind: ReferenceTask, Value: "4"},
			{Kind: ReferenceTask, Value: "4"},
		},
		ParentRoom: room,
	}}
	if err := duplicate.Validate(); err == nil {
		t.Fatal("duplicate created references passed")
	}
}

func TestResultDistinguishesExplicitZeroStateAndAcknowledgement(t *testing.T) {
	if err := (Result{Task: TaskMessagesMarkRead, ReadState: &ReadState{}}).Validate(); err != nil {
		t.Fatalf("explicit zero read state failed: %v", err)
	}
	if err := (Result{Task: TaskMessagesMarkRead}).Validate(); err == nil {
		t.Fatal("absent read state passed")
	}
	if err := (Result{
		Task: TaskRoomsDelete,
		Acknowledgement: &Acknowledgement{
			Acknowledged: true,
			Target:       Reference{Kind: ReferenceRoom, Value: "2"},
		},
	}).Validate(); err != nil {
		t.Fatalf("explicit acknowledgement failed: %v", err)
	}
	if err := (Result{
		Task:            TaskRoomsDelete,
		Acknowledgement: &Acknowledgement{Target: Reference{Kind: ReferenceRoom, Value: "2"}},
	}).Validate(); err == nil {
		t.Fatal("false acknowledgement passed")
	}
}

func TestResultRejectsContextualReferenceKindLaundering(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "1"}
	account := Reference{Kind: ReferenceAccount, Value: "2"}
	message := Reference{Kind: ReferenceMessage, Value: "3"}
	task := Reference{Kind: ReferenceTask, Value: "4"}
	file := Reference{Kind: ReferenceFile, Value: "5"}
	request := Reference{Kind: ReferenceRequest, Value: "7"}

	validTask := WorkTask{
		Ref: task, Room: Room{Ref: room}, Account: Account{Ref: account},
		AssignedBy: Account{Ref: account}, Message: message,
	}
	validFile := File{Ref: file, Room: room, Account: Account{Ref: account}, Message: message}

	tests := []struct {
		name   string
		result Result
	}{
		{"room identity", Result{Task: TaskRoomsList, Rooms: []Room{{Ref: account}}}},
		{"account identity", Result{Task: TaskContactsList, Accounts: []Account{{Ref: room}}}},
		{"account optional room", Result{Task: TaskContactsList, Accounts: []Account{{Ref: account, Room: account}}}},
		{"message identity", Result{Task: TaskMessagesList, MessageRoom: room, Messages: []Message{{Ref: room, Room: room, Sender: Account{Ref: account}}}}},
		{"message room", Result{Task: TaskMessagesList, MessageRoom: room, Messages: []Message{{Ref: message, Room: account, Sender: Account{Ref: account}}}}},
		{"message sender", Result{Task: TaskMessagesList, MessageRoom: room, Messages: []Message{{Ref: message, Room: room, Sender: Account{Ref: room}}}}},
		{"message sender optional room", Result{Task: TaskMessagesList, MessageRoom: room, Messages: []Message{{Ref: message, Room: room, Sender: Account{Ref: account, Room: account}}}}},
		{"message recipient", Result{Task: TaskMessagesList, MessageRoom: room, Messages: []Message{{Ref: message, Room: room, Sender: Account{Ref: account}, Recipients: []Reference{message}}}}},
		{"task identity", Result{Task: TaskRoomTasksList, Tasks: []WorkTask{{Ref: file, Room: validTask.Room, Account: validTask.Account, AssignedBy: validTask.AssignedBy, Message: message}}}},
		{"task room", Result{Task: TaskRoomTasksList, Tasks: []WorkTask{{Ref: task, Room: Room{Ref: account}, Account: validTask.Account, AssignedBy: validTask.AssignedBy, Message: message}}}},
		{"task account", Result{Task: TaskRoomTasksList, Tasks: []WorkTask{{Ref: task, Room: validTask.Room, Account: Account{Ref: room}, AssignedBy: validTask.AssignedBy, Message: message}}}},
		{"task assigned by", Result{Task: TaskRoomTasksList, Tasks: []WorkTask{{Ref: task, Room: validTask.Room, Account: validTask.Account, AssignedBy: Account{Ref: room}, Message: message}}}},
		{"task message", Result{Task: TaskRoomTasksList, Tasks: []WorkTask{{Ref: task, Room: validTask.Room, Account: validTask.Account, AssignedBy: validTask.AssignedBy, Message: task}}}},
		{"file identity", Result{Task: TaskFilesList, Files: []File{{Ref: message, Room: room, Account: validFile.Account, Message: message}}}},
		{"file room", Result{Task: TaskFilesList, Files: []File{{Ref: file, Room: account, Account: validFile.Account, Message: message}}}},
		{"file account", Result{Task: TaskFilesList, Files: []File{{Ref: file, Room: room, Account: Account{Ref: room}, Message: message}}}},
		{"file message", Result{Task: TaskFilesList, Files: []File{{Ref: file, Room: room, Account: validFile.Account, Message: file}}}},
		{"invite identity", Result{Task: TaskInviteLinkShow, InviteLink: &InviteLink{Ref: room}}},
		{"contact request identity", Result{Task: TaskContactRequestsList, Requests: []ContactRequest{{Ref: account, Account: Account{Ref: account}}}}},
		{"contact request account", Result{Task: TaskContactRequestsList, Requests: []ContactRequest{{Ref: request, Account: Account{Ref: request}}}}},
		{"contact request account room", Result{Task: TaskContactRequestsList, Requests: []ContactRequest{{Ref: request, Account: Account{Ref: account, Room: account}}}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.result.Validate(); err == nil {
				t.Fatal("wrong contextual reference kind passed")
			}
		})
	}
}

func TestResultRejectsMessageRelationKindAndTargetMismatches(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "1"}
	account := Reference{Kind: ReferenceAccount, Value: "2"}
	message := Reference{Kind: ReferenceMessage, Value: "3"}
	valid := func() Result {
		return Result{Task: TaskMessagesList, MessageRoom: room, Messages: []Message{{
			Ref: message, Room: room, Sender: Account{Ref: account},
			Reply:  &Relation{Kind: "reply", Target: message},
			Quotes: []Relation{{Kind: "quote", Target: account}},
		}}}
	}

	tests := []struct {
		name   string
		mutate func(*Result)
	}{
		{"reply relation kind", func(result *Result) { result.Messages[0].Reply.Kind = "quote" }},
		{"reply target kind", func(result *Result) { result.Messages[0].Reply.Target = account }},
		{"resolved reply without target", func(result *Result) {
			result.Messages[0].Reply.Resolved = true
			result.Messages[0].Reply.Target = Reference{}
		}},
		{"quote relation kind", func(result *Result) { result.Messages[0].Quotes[0].Kind = "reply" }},
		{"quote target kind", func(result *Result) { result.Messages[0].Quotes[0].Target = message }},
		{"resolved quote without target", func(result *Result) {
			result.Messages[0].Quotes[0].Resolved = true
			result.Messages[0].Quotes[0].Target = Reference{}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := valid()
			test.mutate(&result)
			if err := result.Validate(); err == nil {
				t.Fatal("mismatched relation passed")
			}
		})
	}
}

func TestResultDistinguishesUnknownMessageRelationsFromAbsentRelations(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "1"}
	account := Reference{Kind: ReferenceAccount, Value: "2"}
	message := Reference{Kind: ReferenceMessage, Value: "3"}
	base := Result{Task: TaskMessagesList, MessageRoom: room, Coverage: Coverage{Limit: 100}, Messages: []Message{{
		Ref: message, Room: room, Sender: Account{Ref: account}, RelationState: MessageRelationsUnknown,
	}}}
	if err := base.Validate(); err != nil {
		t.Fatalf("unknown relation set failed: %v", err)
	}

	partial := base
	partial.Messages = append([]Message(nil), base.Messages...)
	partial.Messages[0].Recipients = []Reference{account}
	if err := partial.Validate(); err == nil {
		t.Fatal("unknown relation set with partial relation facts passed")
	}

	invalid := base
	invalid.Messages = append([]Message(nil), base.Messages...)
	invalid.Messages[0].RelationState = MessageRelationState(255)
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid relation state passed")
	}
}

func TestResultValidatesMessageAccessLimitationCardinality(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "1"}
	account := Reference{Kind: ReferenceAccount, Value: "2"}
	message := Message{Ref: Reference{Kind: ReferenceMessage, Value: "3"}, Room: room, Sender: Account{Ref: account}}

	partial := Result{Task: TaskMessagesList, MessageRoom: room, MessageAccess: MessageAccessPartial, Coverage: Coverage{Limit: 100}, Messages: []Message{message}}
	if err := partial.Validate(); err != nil {
		t.Fatalf("partially restricted result failed: %v", err)
	}
	partial.Messages = []Message{}
	if err := partial.Validate(); err == nil {
		t.Fatal("partially restricted source without visible messages passed")
	}
	partial.MessageSelection = &MessageSelection{
		Filter:      MessageFilter{Senders: []Reference{account}, Context: MessageContextNone},
		SourceCount: 1, CandidateCount: 0, SourceSequences: []int{}, AnchorSequences: []int{},
	}
	if err := partial.Validate(); err != nil {
		t.Fatalf("partially restricted source with an explicitly empty local selection failed: %v", err)
	}

	all := Result{Task: TaskMessagesList, MessageRoom: room, MessageAccess: MessageAccessAll, Coverage: Coverage{Limit: 100}, Messages: []Message{}}
	if err := all.Validate(); err != nil {
		t.Fatalf("fully restricted result failed: %v", err)
	}
	all.Messages = []Message{message}
	if err := all.Validate(); err == nil {
		t.Fatal("fully restricted result with a visible message passed")
	}

	wrongTask := Result{Task: TaskRoomsList, MessageAccess: MessageAccessPartial, Rooms: []Room{}}
	if err := wrongTask.Validate(); err == nil {
		t.Fatal("message access limitation on another task passed")
	}
}

func TestResultAllowsDeclaredOptionalZeroReferences(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "1"}
	account := Reference{Kind: ReferenceAccount, Value: "2"}
	message := Reference{Kind: ReferenceMessage, Value: "3"}
	task := Reference{Kind: ReferenceTask, Value: "4"}

	results := []Result{
		{Task: TaskAccountShow, Account: &Account{Ref: account}},
		{Task: TaskMessagesList, MessageRoom: room, Coverage: Coverage{Limit: 100}, Messages: []Message{{Ref: message, Room: room, Sender: Account{Ref: account}}}},
		{Task: TaskMessagesList, MessageRoom: room, Coverage: Coverage{Limit: 100}, Messages: []Message{{
			Ref: message, Room: room, Sender: Account{Ref: account},
			Reply: &Relation{Kind: "reply"}, Quotes: []Relation{{Kind: "quote"}},
		}}},
		{Task: TaskPersonalTasksList, Tasks: []WorkTask{{
			Ref: task, Room: Room{Ref: room}, AssignedBy: Account{Ref: account}, Message: message,
		}}},
		{Task: TaskFilesList, Files: []File{{
			Ref: Reference{Kind: ReferenceFile, Value: "6"}, Room: room, Account: Account{Ref: account},
		}}},
		{Task: TaskContactRequestsList, Requests: []ContactRequest{{
			Ref: Reference{Kind: ReferenceRequest, Value: "5"}, Account: Account{Ref: account},
		}}},
	}
	for _, result := range results {
		if err := result.Validate(); err != nil {
			t.Errorf("%s optional zero reference failed: %v", result.Task, err)
		}
	}
}

func TestMessageListRequiresExactRoomScopeEvenWhenEmpty(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "1"}
	valid := Result{Task: TaskMessagesList, MessageRoom: room, Coverage: Coverage{Limit: 100}, Messages: []Message{}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("empty scoped message window failed: %v", err)
	}

	missing := valid
	missing.MessageRoom = Reference{}
	if err := missing.Validate(); err == nil {
		t.Fatal("empty message window without room scope passed")
	}

	wrongKind := valid
	wrongKind.MessageRoom = Reference{Kind: ReferenceAccount, Value: "1"}
	if err := wrongKind.Validate(); err == nil {
		t.Fatal("message window with non-room scope passed")
	}

	wrongTask := Result{Task: TaskRoomsList, MessageRoom: room, Rooms: []Room{}}
	if err := wrongTask.Validate(); err == nil {
		t.Fatal("message room scope on another task passed")
	}
}

func TestMessageListBindsEveryMessageAndRequestToExactRoom(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "1"}
	otherRoom := Reference{Kind: ReferenceRoom, Value: "2"}
	message := Reference{Kind: ReferenceMessage, Value: "3"}
	account := Reference{Kind: ReferenceAccount, Value: "4"}
	valid := Result{Task: TaskMessagesList, MessageRoom: room, Coverage: Coverage{Limit: 100}, Messages: []Message{{
		Ref: message, Room: room, Sender: Account{Ref: account},
	}}}
	if err := valid.ValidateFor(Request{Task: TaskMessagesList, Room: room}); err != nil {
		t.Fatalf("exact message room binding failed: %v", err)
	}

	mixed := valid
	mixed.Messages = append([]Message(nil), valid.Messages...)
	mixed.Messages[0].Room = otherRoom
	if err := mixed.Validate(); err == nil {
		t.Fatal("message from outside the declared window room passed")
	}

	if err := valid.ValidateFor(Request{Task: TaskMessagesList, Room: otherRoom}); err == nil {
		t.Fatal("message window bound to a different requested room passed")
	}
}

func TestRequestValidatesMessageFilter(t *testing.T) {
	sender := Reference{Kind: ReferenceAccount, Value: "7"}
	valid := Request{Task: TaskMessagesList, MessageFilter: MessageFilter{
		Senders: []Reference{sender}, Context: MessageContextNone,
	}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid sender filter failed: %v", err)
	}
	valid.MessageFilter.Context = MessageContextReplies
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid reply-context filter failed: %v", err)
	}

	hundred := make([]Reference, 100)
	for index := range hundred {
		hundred[index] = Reference{Kind: ReferenceAccount, Value: fmt.Sprint(index + 1)}
	}
	if err := (Request{Task: TaskMessagesList, MessageFilter: MessageFilter{
		Senders: hundred, Context: MessageContextNone,
	}}).Validate(); err != nil {
		t.Fatalf("100-sender boundary failed: %v", err)
	}

	tests := map[string]MessageFilter{
		"context without sender": {Context: MessageContextNone},
		"sender without context": {Senders: []Reference{sender}},
		"unknown context": {
			Senders: []Reference{sender}, Context: MessageContext("thread"),
		},
		"wrong sender kind": {
			Senders: []Reference{{Kind: ReferenceRoom, Value: "7"}}, Context: MessageContextNone,
		},
		"malformed sender": {
			Senders: []Reference{{Kind: ReferenceAccount, Value: "07"}}, Context: MessageContextNone,
		},
		"duplicate sender": {
			Senders: []Reference{sender, sender}, Context: MessageContextNone,
		},
		"too many senders": {
			Senders: append(hundred, Reference{Kind: ReferenceAccount, Value: "101"}), Context: MessageContextNone,
		},
	}
	for name, filter := range tests {
		t.Run(name, func(t *testing.T) {
			if err := (Request{Task: TaskMessagesList, MessageFilter: filter}).Validate(); err == nil {
				t.Fatal("invalid message filter passed")
			}
		})
	}

	if err := (Request{Task: TaskRoomsList, MessageFilter: MessageFilter{
		Senders: []Reference{sender}, Context: MessageContextNone,
	}}).Validate(); err == nil {
		t.Fatal("message filter on another task passed")
	}
	if err := (Request{Task: TaskRoomsList, MessageFilter: MessageFilter{Senders: []Reference{}}}).Validate(); err != nil {
		t.Fatalf("semantic zero filter on another task failed: %v", err)
	}
}

func TestResultValidatesFilteredMessageSelection(t *testing.T) {
	valid := validFilteredMessageResult()
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid filtered selection failed: %v", err)
	}

	withoutContext := validFilteredMessageResult()
	withoutContext.MessageSelection.Filter.Context = MessageContextNone
	withoutContext.MessageSelection.Filter.Senders = []Reference{
		{Kind: ReferenceAccount, Value: "7"},
		{Kind: ReferenceAccount, Value: "8"},
	}
	withoutContext.MessageSelection.CandidateCount = 2
	withoutContext.MessageSelection.AnchorSequences = []int{2, 4}
	if err := withoutContext.Validate(); err != nil {
		t.Fatalf("valid context-free selection failed: %v", err)
	}

	empty := validFilteredMessageResult()
	empty.Messages = []Message{}
	empty.MessageSelection.SourceCount = 0
	empty.MessageSelection.CandidateCount = 0
	empty.MessageSelection.SourceSequences = []int{}
	empty.MessageSelection.AnchorSequences = []int{}
	if err := empty.Validate(); err != nil {
		t.Fatalf("valid empty filtered selection failed: %v", err)
	}
	empty.MessageSelection.SourceSequences = nil
	if err := empty.Validate(); err == nil {
		t.Fatal("empty filtered selection accepted nil source provenance")
	}
	empty.MessageSelection.SourceSequences = []int{}
	empty.MessageSelection.AnchorSequences = nil
	if err := empty.Validate(); err == nil {
		t.Fatal("empty filtered selection accepted nil anchor provenance")
	}

	tests := map[string]func(*Result){
		"inactive filter": func(result *Result) {
			result.MessageSelection.Filter = MessageFilter{}
		},
		"non-positive source limit": func(result *Result) {
			result.Coverage.Limit = 0
		},
		"fewer source messages than displayed": func(result *Result) {
			result.MessageSelection.SourceCount = 1
		},
		"source count above limit": func(result *Result) {
			result.MessageSelection.SourceCount = 101
		},
		"unaligned source sequences": func(result *Result) {
			result.MessageSelection.SourceSequences = []int{2}
		},
		"zero source sequence": func(result *Result) {
			result.MessageSelection.SourceSequences = []int{0, 4}
		},
		"source sequence above count": func(result *Result) {
			result.MessageSelection.SourceSequences = []int{2, 5}
		},
		"decreasing source sequences": func(result *Result) {
			result.MessageSelection.SourceSequences = []int{4, 2}
		},
		"duplicate source sequences": func(result *Result) {
			result.MessageSelection.SourceSequences = []int{2, 2}
		},
		"decreasing anchors": func(result *Result) {
			result.MessageSelection.AnchorSequences = []int{4, 2}
		},
		"anchor outside display": func(result *Result) {
			result.MessageSelection.AnchorSequences = []int{3}
		},
		"context-free non-anchor display": func(result *Result) {
			result.MessageSelection.Filter.Context = MessageContextNone
			result.MessageSelection.AnchorSequences = []int{2}
		},
		"anchor sender mismatch": func(result *Result) {
			result.MessageSelection.AnchorSequences = []int{4}
		},
		"unrelated non-anchor context": func(result *Result) {
			result.Messages[1].Reply = nil
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			result := validFilteredMessageResult()
			mutate(&result)
			if err := result.Validate(); err == nil {
				t.Fatal("invalid message selection passed")
			}
		})
	}

	otherTask := Result{
		Task: TaskRoomsList, Rooms: []Room{},
		MessageSelection: valid.MessageSelection,
	}
	if err := otherTask.Validate(); err == nil {
		t.Fatal("message selection on another result task passed")
	}
}

func TestFilteredMessageSelectionBindsExactlyToRequest(t *testing.T) {
	result := validFilteredMessageResult()
	request := Request{
		Task: TaskMessagesList, Room: result.MessageRoom,
		MessageFilter: result.MessageSelection.Filter,
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("exact filtered request binding failed: %v", err)
	}

	missing := result
	missing.MessageSelection = nil
	if err := missing.ValidateFor(request); err == nil {
		t.Fatal("filtered request accepted a result without selection metadata")
	}

	differentSender := request
	differentSender.MessageFilter.Senders = []Reference{{Kind: ReferenceAccount, Value: "8"}}
	if err := result.ValidateFor(differentSender); err == nil {
		t.Fatal("selection bound to a different sender filter")
	}

	differentContext := request
	differentContext.MessageFilter.Context = MessageContextNone
	if err := result.ValidateFor(differentContext); err == nil {
		t.Fatal("selection bound to a different context filter")
	}

	unfilteredResult := result
	unfilteredResult.MessageSelection = nil
	unfilteredRequest := request
	unfilteredRequest.MessageFilter = MessageFilter{}
	if err := unfilteredResult.ValidateFor(unfilteredRequest); err != nil {
		t.Fatalf("unfiltered result binding failed: %v", err)
	}
	if err := result.ValidateFor(unfilteredRequest); err == nil {
		t.Fatal("unfiltered request accepted selection metadata")
	}

	ordered := validFilteredMessageResult()
	second := Reference{Kind: ReferenceAccount, Value: "8"}
	ordered.MessageSelection.Filter.Senders = append(ordered.MessageSelection.Filter.Senders, second)
	ordered.MessageSelection.CandidateCount = 2
	ordered.MessageSelection.AnchorSequences = []int{2, 4}
	orderedRequest := request
	orderedRequest.MessageFilter = ordered.MessageSelection.Filter
	if err := ordered.ValidateFor(orderedRequest); err != nil {
		t.Fatalf("ordered multi-sender binding failed: %v", err)
	}
	reversed := orderedRequest
	reversed.MessageFilter.Senders = []Reference{second, request.MessageFilter.Senders[0]}
	if err := ordered.ValidateFor(reversed); err == nil {
		t.Fatal("selection sender order was not bound exactly")
	}
}

func validFilteredMessageResult() Result {
	room := Reference{Kind: ReferenceRoom, Value: "1"}
	sender := Reference{Kind: ReferenceAccount, Value: "7"}
	return Result{
		Task: TaskMessagesList, MessageRoom: room,
		Coverage: Coverage{Kind: "latest_window", Limit: 100, Complete: false},
		Messages: []Message{
			{Ref: Reference{Kind: ReferenceMessage, Value: "101"}, Room: room, Sender: Account{Ref: sender}},
			{
				Ref: Reference{Kind: ReferenceMessage, Value: "102"}, Room: room,
				Sender: Account{Ref: Reference{Kind: ReferenceAccount, Value: "8"}},
				Reply: &Relation{
					Kind: "reply", Target: Reference{Kind: ReferenceMessage, Value: "101"}, ExternalID: "1", Resolved: true,
				},
			},
		},
		MessageSelection: &MessageSelection{
			Filter:         MessageFilter{Senders: []Reference{sender}, Context: MessageContextReplies},
			SourceCount:    4,
			CandidateCount: 1, SourceSequences: []int{2, 4}, AnchorSequences: []int{2},
		},
	}
}
