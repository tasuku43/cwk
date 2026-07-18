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
