package chatwork

import "testing"

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
		{"message identity", Result{Task: TaskMessagesList, Messages: []Message{{Ref: room, Room: room, Sender: Account{Ref: account}}}}},
		{"message room", Result{Task: TaskMessagesList, Messages: []Message{{Ref: message, Room: account, Sender: Account{Ref: account}}}}},
		{"message sender", Result{Task: TaskMessagesList, Messages: []Message{{Ref: message, Room: room, Sender: Account{Ref: room}}}}},
		{"message sender optional room", Result{Task: TaskMessagesList, Messages: []Message{{Ref: message, Room: room, Sender: Account{Ref: account, Room: account}}}}},
		{"message recipient", Result{Task: TaskMessagesList, Messages: []Message{{Ref: message, Room: room, Sender: Account{Ref: account}, Recipients: []Reference{message}}}}},
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
		return Result{Task: TaskMessagesList, Messages: []Message{{
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
		{"quote relation kind", func(result *Result) { result.Messages[0].Quotes[0].Kind = "reply" }},
		{"quote target kind", func(result *Result) { result.Messages[0].Quotes[0].Target = message }},
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

func TestResultAllowsDeclaredOptionalZeroReferences(t *testing.T) {
	room := Reference{Kind: ReferenceRoom, Value: "1"}
	account := Reference{Kind: ReferenceAccount, Value: "2"}
	message := Reference{Kind: ReferenceMessage, Value: "3"}
	task := Reference{Kind: ReferenceTask, Value: "4"}

	results := []Result{
		{Task: TaskAccountShow, Account: &Account{Ref: account}},
		{Task: TaskMessagesList, Messages: []Message{{Ref: message, Room: room, Sender: Account{Ref: account}}}},
		{Task: TaskPersonalTasksList, Tasks: []WorkTask{{
			Ref: task, Room: Room{Ref: room}, AssignedBy: Account{Ref: account}, Message: message,
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
