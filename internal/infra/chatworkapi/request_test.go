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
		DescriptionSet: true,
		Limit:          1700000000, LimitType: "time", ForceRecent: true, SelfUnread: true,
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
		{chatwork.TaskInviteLinkUpdate, []string{"code=public-room", "description=description", "need_acceptance=0"}},
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

func TestRoomsCreateDoesNotSendAuthenticatedAccountAsProviderField(t *testing.T) {
	spec, err := (&Client{}).buildRequest(completeRequest(chatwork.TaskRoomsCreate))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(spec.body)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"account", "owner"} {
		if strings.Contains(string(body), field+"=") {
			t.Fatalf("room creation body contains unsupported %s field: %q", field, body)
		}
	}
}

func TestInviteLinkUpdateEncodesCompleteReplacementAndExplicitRegeneration(t *testing.T) {
	explicit := completeRequest(chatwork.TaskInviteLinkUpdate)
	regenerate := explicit
	regenerate.InviteCode = ""
	regenerate.InviteRegenerateCode = true

	for name, test := range map[string]struct {
		request chatwork.Request
		body    string
	}{
		"explicit code":   {request: explicit, body: "code=public-room&description=description&need_acceptance=0"},
		"regenerate code": {request: regenerate, body: "description=description&need_acceptance=0"},
	} {
		t.Run(name, func(t *testing.T) {
			spec, err := (&Client{}).buildRequest(test.request)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(spec.body)
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != test.body {
				t.Fatalf("body = %q, want %q", body, test.body)
			}
		})
	}
}

func TestInviteLinkUpdateBuilderRejectsIncompleteReplacement(t *testing.T) {
	valid := completeRequest(chatwork.TaskInviteLinkUpdate)
	tests := map[string]func(*chatwork.Request){
		"missing code intent":   func(request *chatwork.Request) { request.InviteCode = "" },
		"code and regeneration": func(request *chatwork.Request) { request.InviteRegenerateCode = true },
		"missing approval":      func(request *chatwork.Request) { request.InviteApprovalSet = false },
		"missing description":   func(request *chatwork.Request) { request.DescriptionSet = false },
		"invalid code":          func(request *chatwork.Request) { request.InviteCode = "invalid!" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := valid
			mutate(&request)
			if _, err := (&Client{}).buildRequest(request); err == nil {
				t.Fatal("incomplete invite-link replacement reached request construction")
			}
		})
	}
}

func TestInviteLinkCreateIncludesExplicitDescription(t *testing.T) {
	spec, err := (&Client{}).buildRequest(completeRequest(chatwork.TaskInviteLinkCreate))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(spec.body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "description=description") {
		t.Fatalf("invite-link create body = %q", body)
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
	for name, filter := range map[string]chatwork.MessageFilter{
		"sender and context": {
			Senders: []chatwork.Reference{testRef(chatwork.ReferenceAccount, "1")},
			Context: chatwork.MessageContextReplies,
		},
		"count":       {StartIndex: 1, Count: 10},
		"start index": {StartIndex: 10},
		"period":      {Period: chatwork.MessagePeriod{Since: 100, Until: 200}, Context: chatwork.MessageContextNone},
	} {
		t.Run(name, func(t *testing.T) {
			input := completeRequest(chatwork.TaskMessagesList)
			input.MessageFilter = filter
			if _, err := (&Client{}).buildRequest(input); err == nil {
				t.Fatal("application-owned message selection crossed the Chatwork request boundary")
			}
		})
	}
	input := completeRequest(chatwork.TaskMessagesList)
	input.MessageRelationFetchLimit = 1
	if _, err := (&Client{}).buildRequest(input); err == nil {
		t.Fatal("application-owned message relation fetch budget crossed the Chatwork request boundary")
	}
}

func TestBuildMessageListRequestEmitsOnlyDocumentedForceQuery(t *testing.T) {
	for name, test := range map[string]struct {
		force bool
		path  string
	}{
		"changes": {path: "/rooms/2/messages"},
		"recent":  {force: true, path: "/rooms/2/messages?force=1"},
	} {
		t.Run(name, func(t *testing.T) {
			input := completeRequest(chatwork.TaskMessagesList)
			input.ForceRecent = test.force
			spec, err := (&Client{}).buildRequest(input)
			if err != nil {
				t.Fatal(err)
			}
			if spec.method != http.MethodGet || spec.path != test.path {
				t.Fatalf("request = %s %s, want GET %s", spec.method, spec.path, test.path)
			}
			if strings.Contains(spec.path, "limit=") || strings.Contains(spec.path, "count=") ||
				strings.Contains(spec.path, "since=") || strings.Contains(spec.path, "until=") || strings.Contains(spec.path, "on=") ||
				strings.Contains(spec.path, "start") || strings.Contains(spec.path, "skip") || strings.Contains(spec.path, "offset") {
				t.Fatalf("application selection leaked into provider query %q", spec.path)
			}
		})
	}
}
