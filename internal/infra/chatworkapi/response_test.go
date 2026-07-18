package chatworkapi

import (
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestMapResponseCoversResourceShapes(t *testing.T) {
	tests := []struct {
		name  string
		in    chatwork.Request
		body  string
		check func(t *testing.T, result chatwork.Result)
	}{
		{"account", completeRequest(chatwork.TaskAccountShow), `{"account_id":1,"room_id":2,"name":"Alice"}`, func(t *testing.T, result chatwork.Result) {
			if result.Account == nil || result.Account.Ref.Value != "1" || result.Account.Room.Value != "2" {
				t.Fatalf("result = %+v", result)
			}
		}},
		{"status", completeRequest(chatwork.TaskAccountStatus), `{"unread_room_num":2,"mention_room_num":1,"mytask_room_num":3,"unread_num":12,"mention_num":1,"mytask_num":8}`, func(t *testing.T, result chatwork.Result) {
			if result.Status == nil || result.Status.Unread != 12 {
				t.Fatalf("result = %+v", result)
			}
		}},
		{"rooms", completeRequest(chatwork.TaskRoomsList), `[{"room_id":2,"name":"Room"}]`, func(t *testing.T, result chatwork.Result) {
			if len(result.Rooms) != 1 || result.Rooms[0].Ref.Value != "2" {
				t.Fatalf("result = %+v", result)
			}
		}},
		{"members", completeRequest(chatwork.TaskMembersList), `[{"account_id":1,"name":"Alice","role":"admin"}]`, func(t *testing.T, result chatwork.Result) {
			if len(result.Accounts) != 1 || result.Accounts[0].Role != "admin" {
				t.Fatalf("result = %+v", result)
			}
		}},
		{"messages", completeRequest(chatwork.TaskMessagesList), `[{"message_id":"3","account":{"account_id":1,"name":"Alice"},"body":"[To:9] [rp aid=8 to=2-7]","send_time":1,"update_time":0}]`, func(t *testing.T, result chatwork.Result) {
			if len(result.Messages) != 1 || result.MessageRoom.Value != "2" || len(result.Messages[0].Recipients) != 2 || result.Messages[0].Reply.Target.Value != "7" || result.Coverage.Complete {
				t.Fatalf("result = %+v", result)
			}
		}},
		{"tasks", completeRequest(chatwork.TaskRoomTasksList), `[{"task_id":4,"account":{"account_id":1},"assigned_by_account":{"account_id":2},"message_id":"3","body":"task","limit_time":1,"status":"open","limit_type":"time"}]`, func(t *testing.T, result chatwork.Result) {
			if len(result.Tasks) != 1 || result.Tasks[0].Ref.Value != "4" {
				t.Fatalf("result = %+v", result)
			}
		}},
		{"files", completeRequest(chatwork.TaskFilesList), `[{"file_id":5,"account":{"account_id":1},"message_id":"3","filename":"a.txt","filesize":4,"upload_time":1}]`, func(t *testing.T, result chatwork.Result) {
			if len(result.Files) != 1 || result.Files[0].Ref.Value != "5" {
				t.Fatalf("result = %+v", result)
			}
		}},
		{"invite", completeRequest(chatwork.TaskInviteLinkShow), `{"public":true,"url":"https://example.test/g/code","need_acceptance":false,"description":"invite"}`, func(t *testing.T, result chatwork.Result) {
			if result.InviteLink == nil || result.InviteLink.Ref.Kind != chatwork.ReferenceInvite || result.InviteLink.Ref.Value != "2" {
				t.Fatalf("result = %+v", result)
			}
		}},
		{"requests", completeRequest(chatwork.TaskContactRequestsList), `[{"request_id":6,"account_id":1,"room_id":2,"message":"hello","name":"Alice"}]`, func(t *testing.T, result chatwork.Result) {
			if len(result.Requests) != 1 || result.Requests[0].Ref.Value != "6" {
				t.Fatalf("result = %+v", result)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := mapResponse(test.in, []byte(test.body))
			if err != nil {
				t.Fatal(err)
			}
			test.check(t, result)
		})
	}
}

func TestMapMessageListPreservesProviderOrderAndExactRoomScope(t *testing.T) {
	request := completeRequest(chatwork.TaskMessagesList)
	result, err := mapResponse(request, []byte(`[
		{"message_id":"30","account":{"account_id":1,"name":"A"},"body":"third time","send_time":300,"update_time":0},
		{"message_id":"10","account":{"account_id":2,"name":"B"},"body":"first time","send_time":100,"update_time":0},
		{"message_id":"20","account":{"account_id":3,"name":"C"},"body":"second time","send_time":200,"update_time":0}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageRoom != request.Room {
		t.Fatalf("message room = %+v, want %+v", result.MessageRoom, request.Room)
	}
	want := []string{"30", "10", "20"}
	for index, message := range result.Messages {
		if message.Ref.Value != want[index] || message.Room != request.Room {
			t.Fatalf("message[%d] = %+v, want ref %s in exact room", index, message, want[index])
		}
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("mapped message window is not request-bound: %v", err)
	}
}

func TestEmptyMessageListPreservesExactRoomScope(t *testing.T) {
	request := completeRequest(chatwork.TaskMessagesList)
	result := emptyResult(request)
	if result.MessageRoom != request.Room || result.Messages == nil || len(result.Messages) != 0 {
		t.Fatalf("empty message result = %+v", result)
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("empty message window is not request-bound: %v", err)
	}
}

func TestMapIdentityResponses(t *testing.T) {
	tests := []struct {
		task       chatwork.Task
		body       string
		kind       chatwork.ReferenceKind
		roomScoped bool
	}{
		{chatwork.TaskRoomsCreate, `{"room_id":2}`, chatwork.ReferenceRoom, false},
		{chatwork.TaskMessagesSend, `{"message_id":"3"}`, chatwork.ReferenceMessage, true},
		{chatwork.TaskRoomTasksCreate, `{"task_ids":[4]}`, chatwork.ReferenceTask, true},
		{chatwork.TaskFilesUpload, `{"file_id":5}`, chatwork.ReferenceFile, true},
	}
	for _, test := range tests {
		request := completeRequest(test.task)
		result, err := mapResponse(request, []byte(test.body))
		if err != nil {
			t.Fatal(err)
		}
		created := result.Created
		if test.roomScoped {
			if result.CreatedInRoom == nil || result.CreatedInRoom.ParentRoom != request.Room {
				t.Fatalf("task %s room-scoped result = %+v", test.task, result)
			}
			created = result.CreatedInRoom.Refs
		}
		if len(created) != 1 || created[0].Kind != test.kind {
			t.Fatalf("task %s result = %+v", test.task, result)
		}
	}
}

func TestMapIdentityResponsesPreserveExplicitZeroReadStateAndMembershipNames(t *testing.T) {
	read, err := mapResponse(completeRequest(chatwork.TaskMessagesMarkRead), []byte(`{"unread_num":0,"mention_num":0}`))
	if err != nil {
		t.Fatal(err)
	}
	if read.ReadState == nil || read.ReadState.Unread != 0 || read.ReadState.Mentions != 0 {
		t.Fatalf("read state = %+v", read.ReadState)
	}

	members, err := mapResponse(completeRequest(chatwork.TaskMembersReplace), []byte(`{"admin":[1,2],"member":[3],"readonly":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if members.MembershipCounts == nil || members.MembershipCounts.Administrators != 2 ||
		members.MembershipCounts.Members != 1 || members.MembershipCounts.Readonly != 0 {
		t.Fatalf("membership counts = %+v", members.MembershipCounts)
	}
}

func TestMapResponseRejectsMissingIdentityInvalidUTF8AndTrailingJSON(t *testing.T) {
	for _, body := range [][]byte{[]byte(`[{"name":"missing id"}]`), {0xff}, []byte(`[] {}`)} {
		if _, err := mapResponse(completeRequest(chatwork.TaskRoomsList), body); err == nil {
			t.Fatalf("mapResponse(%q) succeeded", body)
		}
	}
}
