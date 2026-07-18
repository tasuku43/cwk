package chatworkapi

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func testRef(kind chatwork.ReferenceKind, value string) chatwork.Reference {
	return chatwork.Reference{Kind: kind, Value: value}
}

func completeRequest(task chatwork.Task) chatwork.Request {
	account := testRef(chatwork.ReferenceAccount, "1")
	return chatwork.Request{
		Task: task, Room: testRef(chatwork.ReferenceRoom, "2"),
		Message: testRef(chatwork.ReferenceMessage, "3"), TaskRef: testRef(chatwork.ReferenceTask, "4"),
		File: testRef(chatwork.ReferenceFile, "5"), Invite: testRef(chatwork.ReferenceInvite, "2"),
		Request: testRef(chatwork.ReferenceRequest, "6"), Account: account, AssignedBy: account,
		Admins: []chatwork.Reference{account}, Assignees: []chatwork.Reference{account},
		Name: "room", Description: "description", Icon: "group", Body: "body", Status: "open",
		Limit: 1700000000, LimitType: "time", ForceRecent: true, SelfUnread: true,
		CreateDownloadURL: true, InviteCode: "public-room", InviteEnabled: true,
		InviteNeedsApproval: false, InviteApprovalSet: true, FilePath: "fixture.bin", FileMessage: "file message",
	}
}

func TestBuildRequestMapsEveryTaskToReviewedOperation(t *testing.T) {
	tests := map[chatwork.Task]string{
		chatwork.TaskAccountShow:           "GET /me",
		chatwork.TaskAccountStatus:         "GET /my/status",
		chatwork.TaskPersonalTasksList:     "GET /my/tasks?assigned_by_account_id=1&status=open",
		chatwork.TaskContactsList:          "GET /contacts",
		chatwork.TaskRoomsList:             "GET /rooms",
		chatwork.TaskRoomsCreate:           "POST /rooms",
		chatwork.TaskRoomsShow:             "GET /rooms/2",
		chatwork.TaskRoomsUpdate:           "PUT /rooms/2",
		chatwork.TaskRoomsLeave:            "DELETE /rooms/2",
		chatwork.TaskRoomsDelete:           "DELETE /rooms/2",
		chatwork.TaskMembersList:           "GET /rooms/2/members",
		chatwork.TaskMembersReplace:        "PUT /rooms/2/members",
		chatwork.TaskMessagesList:          "GET /rooms/2/messages?force=1",
		chatwork.TaskMessagesSend:          "POST /rooms/2/messages",
		chatwork.TaskMessagesMarkRead:      "PUT /rooms/2/messages/read",
		chatwork.TaskMessagesMarkUnread:    "PUT /rooms/2/messages/unread",
		chatwork.TaskMessagesShow:          "GET /rooms/2/messages/3",
		chatwork.TaskMessagesUpdate:        "PUT /rooms/2/messages/3",
		chatwork.TaskMessagesDelete:        "DELETE /rooms/2/messages/3",
		chatwork.TaskRoomTasksList:         "GET /rooms/2/tasks?account_id=1&assigned_by_account_id=1&status=open",
		chatwork.TaskRoomTasksCreate:       "POST /rooms/2/tasks",
		chatwork.TaskRoomTasksShow:         "GET /rooms/2/tasks/4",
		chatwork.TaskRoomTasksSetStatus:    "PUT /rooms/2/tasks/4/status",
		chatwork.TaskFilesList:             "GET /rooms/2/files?account_id=1",
		chatwork.TaskFilesUpload:           "POST /rooms/2/files",
		chatwork.TaskFilesShow:             "GET /rooms/2/files/5?create_download_url=1",
		chatwork.TaskInviteLinkShow:        "GET /rooms/2/link",
		chatwork.TaskInviteLinkCreate:      "POST /rooms/2/link",
		chatwork.TaskInviteLinkUpdate:      "PUT /rooms/2/link",
		chatwork.TaskInviteLinkDelete:      "DELETE /rooms/2/link",
		chatwork.TaskContactRequestsList:   "GET /incoming_requests",
		chatwork.TaskContactRequestsAccept: "PUT /incoming_requests/6",
		chatwork.TaskContactRequestsReject: "DELETE /incoming_requests/6",
	}
	client := newClient("http://example.test", "synthetic-token", http.DefaultClient, func() (string, error) { return "test-binding", nil }, func(string) ([]byte, error) { return []byte("file"), nil })
	for task, want := range tests {
		t.Run(string(task), func(t *testing.T) {
			spec, err := client.buildRequest(completeRequest(task))
			if err != nil {
				t.Fatal(err)
			}
			if got := spec.method + " " + spec.path; got != want {
				t.Fatalf("request = %q, want %q", got, want)
			}
			policy, err := CallPolicy(task)
			if err != nil || policy.MaxAttempts != MaxAttempts {
				t.Fatalf("policy = %+v, err = %v", policy, err)
			}
		})
	}
	if len(tests) != 33 {
		t.Fatalf("task mappings = %d, want 33 task outcomes over 32 operations", len(tests))
	}
}

func TestBuildRequestEncodesMutationFields(t *testing.T) {
	client := newClient("http://example.test", "synthetic-token", http.DefaultClient, func() (string, error) { return "test-binding", nil }, func(string) ([]byte, error) { return []byte("file"), nil })
	for _, test := range []struct {
		task chatwork.Task
		want []string
	}{
		{chatwork.TaskRoomsCreate, []string{"members_admin_ids=1", "name=room", "link=1", "link_need_acceptance=0"}},
		{chatwork.TaskRoomsLeave, []string{"action_type=leave"}},
		{chatwork.TaskRoomsDelete, []string{"action_type=delete"}},
		{chatwork.TaskMessagesSend, []string{"body=body", "self_unread=1"}},
		{chatwork.TaskRoomTasksCreate, []string{"body=body", "to_ids=1", "limit=1700000000", "limit_type=time"}},
		{chatwork.TaskInviteLinkUpdate, []string{"code=public-room", "need_acceptance=0"}},
	} {
		t.Run(string(test.task), func(t *testing.T) {
			spec, err := client.buildRequest(completeRequest(test.task))
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(spec.body)
			if err != nil {
				t.Fatal(err)
			}
			for _, fragment := range test.want {
				if !strings.Contains(string(data), fragment) {
					t.Fatalf("body %q does not contain %q", data, fragment)
				}
			}
		})
	}
}

func TestInviteMutationUsesInviteReferenceNotRoomFallback(t *testing.T) {
	input := completeRequest(chatwork.TaskInviteLinkDelete)
	input.Room = testRef(chatwork.ReferenceRoom, "999")
	input.Invite = testRef(chatwork.ReferenceInvite, "2")
	spec, err := (&Client{}).buildRequest(input)
	if err != nil {
		t.Fatal(err)
	}
	if spec.path != "/rooms/2/link" {
		t.Fatalf("path = %q", spec.path)
	}
}

func TestBuildMessageListRequestRejectsApplicationSelectionFields(t *testing.T) {
	input := completeRequest(chatwork.TaskMessagesList)
	input.MessageFilter = chatwork.MessageFilter{
		Senders: []chatwork.Reference{testRef(chatwork.ReferenceAccount, "1")},
		Context: chatwork.MessageContextReplies,
	}

	if _, err := (&Client{}).buildRequest(input); err == nil {
		t.Fatal("application-owned message selection crossed the Chatwork request boundary")
	}
}
