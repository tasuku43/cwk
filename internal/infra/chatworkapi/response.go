package chatworkapi

import (
	"bytes"
	"encoding/json"
	"unicode/utf8"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type wireID string

func (id *wireID) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	var value string
	if data[0] == '"' {
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
	} else {
		var number json.Number
		if err := json.Unmarshal(data, &number); err != nil {
			return err
		}
		value = number.String()
	}
	*id = wireID(value)
	return nil
}

type accountDTO struct {
	AccountID      wireID `json:"account_id"`
	RoomID         wireID `json:"room_id"`
	Name           string `json:"name"`
	ChatworkID     string `json:"chatwork_id"`
	OrganizationID wireID `json:"organization_id"`
	Organization   string `json:"organization_name"`
	Department     string `json:"department"`
	Title          string `json:"title"`
	URL            string `json:"url"`
	Introduction   string `json:"introduction"`
	Mail           string `json:"mail"`
	Telephone      string `json:"tel_organization"`
	Extension      string `json:"tel_extension"`
	Mobile         string `json:"tel_mobile"`
	Skype          string `json:"skype"`
	Facebook       string `json:"facebook"`
	Twitter        string `json:"twitter"`
	AvatarURL      string `json:"avatar_image_url"`
	LoginMail      string `json:"login_mail"`
	Role           string `json:"role"`
}

type roomDTO struct {
	RoomID         wireID `json:"room_id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Role           string `json:"role"`
	Sticky         bool   `json:"sticky"`
	Unread         int64  `json:"unread_num"`
	Mentions       int64  `json:"mention_num"`
	MyTasks        int64  `json:"mytask_num"`
	Messages       int64  `json:"message_num"`
	Files          int64  `json:"file_num"`
	Tasks          int64  `json:"task_num"`
	IconURL        string `json:"icon_path"`
	LastUpdateTime int64  `json:"last_update_time"`
	Description    string `json:"description"`
}

type messageDTO struct {
	MessageID  wireID     `json:"message_id"`
	Account    accountDTO `json:"account"`
	Body       string     `json:"body"`
	SendTime   int64      `json:"send_time"`
	UpdateTime int64      `json:"update_time"`
}

type taskDTO struct {
	TaskID            wireID     `json:"task_id"`
	Room              roomDTO    `json:"room"`
	Account           accountDTO `json:"account"`
	AssignedByAccount accountDTO `json:"assigned_by_account"`
	MessageID         wireID     `json:"message_id"`
	Body              string     `json:"body"`
	LimitTime         int64      `json:"limit_time"`
	Status            string     `json:"status"`
	LimitType         string     `json:"limit_type"`
}

type fileDTO struct {
	FileID      wireID     `json:"file_id"`
	Account     accountDTO `json:"account"`
	MessageID   wireID     `json:"message_id"`
	Filename    string     `json:"filename"`
	Filesize    int64      `json:"filesize"`
	UploadTime  int64      `json:"upload_time"`
	DownloadURL string     `json:"download_url"`
}

type inviteLinkDTO struct {
	Public         bool   `json:"public"`
	URL            string `json:"url"`
	NeedAcceptance bool   `json:"need_acceptance"`
	Description    string `json:"description"`
}

type contactRequestDTO struct {
	RequestID      wireID `json:"request_id"`
	AccountID      wireID `json:"account_id"`
	RoomID         wireID `json:"room_id"`
	Message        string `json:"message"`
	Name           string `json:"name"`
	ChatworkID     string `json:"chatwork_id"`
	OrganizationID wireID `json:"organization_id"`
	Organization   string `json:"organization_name"`
	Department     string `json:"department"`
	AvatarURL      string `json:"avatar_image_url"`
}

type statusDTO struct {
	UnreadRooms  int64 `json:"unread_room_num"`
	MentionRooms int64 `json:"mention_room_num"`
	TaskRooms    int64 `json:"mytask_room_num"`
	Unread       int64 `json:"unread_num"`
	Mentions     int64 `json:"mention_num"`
	Tasks        int64 `json:"mytask_num"`
}

type identityDTO struct {
	RoomID    wireID   `json:"room_id"`
	MessageID wireID   `json:"message_id"`
	TaskID    wireID   `json:"task_id"`
	TaskIDs   []wireID `json:"task_ids"`
	FileID    wireID   `json:"file_id"`
	Unread    int64    `json:"unread_num"`
	Mentions  int64    `json:"mention_num"`
	Admin     []wireID `json:"admin"`
	Member    []wireID `json:"member"`
	Readonly  []wireID `json:"readonly"`
}

func mapResponse(input chatwork.Request, body []byte) (chatwork.Result, error) {
	if !utf8.Valid(body) {
		return chatwork.Result{}, malformedResponse()
	}
	result := chatwork.Result{Task: input.Task, Coverage: coverageFor(input)}
	switch input.Task {
	case chatwork.TaskAccountShow:
		var wire accountDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		account, err := mapAccount(wire)
		if err != nil {
			return chatwork.Result{}, err
		}
		result.Account = &account
	case chatwork.TaskAccountStatus:
		var wire statusDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		result.Status = &chatwork.Status{UnreadRooms: wire.UnreadRooms, MentionRooms: wire.MentionRooms, TaskRooms: wire.TaskRooms, Unread: wire.Unread, Mentions: wire.Mentions, Tasks: wire.Tasks}
	case chatwork.TaskContactsList, chatwork.TaskMembersList:
		var wire []accountDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		result.Accounts = make([]chatwork.Account, 0, len(wire))
		for _, item := range wire {
			account, err := mapAccount(item)
			if err != nil {
				return chatwork.Result{}, err
			}
			result.Accounts = append(result.Accounts, account)
		}
	case chatwork.TaskRoomsList:
		var wire []roomDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		rooms, err := mapRooms(wire)
		if err != nil {
			return chatwork.Result{}, err
		}
		result.Rooms = rooms
	case chatwork.TaskRoomsShow:
		var wire roomDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		room, err := mapRoom(wire)
		if err != nil {
			return chatwork.Result{}, err
		}
		result.Rooms = []chatwork.Room{room}
	case chatwork.TaskMessagesList:
		var wire []messageDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		messages, err := mapMessages(input.Room, wire)
		if err != nil {
			return chatwork.Result{}, err
		}
		result.MessageRoom = input.Room
		result.Messages = messages
	case chatwork.TaskMessagesShow:
		var wire messageDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		messages, err := mapMessages(input.Room, []messageDTO{wire})
		if err != nil {
			return chatwork.Result{}, err
		}
		result.Messages = messages
	case chatwork.TaskPersonalTasksList, chatwork.TaskRoomTasksList:
		var wire []taskDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		tasks, err := mapTasks(input.Room, wire)
		if err != nil {
			return chatwork.Result{}, err
		}
		result.Tasks = tasks
	case chatwork.TaskRoomTasksShow:
		var wire taskDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		tasks, err := mapTasks(input.Room, []taskDTO{wire})
		if err != nil {
			return chatwork.Result{}, err
		}
		result.Tasks = tasks
	case chatwork.TaskFilesList:
		var wire []fileDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		files, err := mapFiles(input.Room, wire)
		if err != nil {
			return chatwork.Result{}, err
		}
		result.Files = files
	case chatwork.TaskFilesShow:
		var wire fileDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		files, err := mapFiles(input.Room, []fileDTO{wire})
		if err != nil {
			return chatwork.Result{}, err
		}
		result.Files = files
	case chatwork.TaskInviteLinkShow, chatwork.TaskInviteLinkCreate, chatwork.TaskInviteLinkUpdate, chatwork.TaskInviteLinkDelete:
		var wire inviteLinkDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		invite, err := inviteReference(input)
		if err != nil {
			return chatwork.Result{}, err
		}
		result.InviteLink = &chatwork.InviteLink{Ref: invite, Public: wire.Public, URL: wire.URL, NeedsApproval: wire.NeedAcceptance, Description: wire.Description}
	case chatwork.TaskContactRequestsList:
		var wire []contactRequestDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		requests, err := mapContactRequests(wire)
		if err != nil {
			return chatwork.Result{}, err
		}
		result.Requests = requests
	case chatwork.TaskContactRequestsAccept:
		var wire accountDTO
		if err := decodeJSON(body, &wire); err != nil {
			return chatwork.Result{}, err
		}
		account, err := mapAccount(wire)
		if err != nil {
			return chatwork.Result{}, err
		}
		result.Account = &account
	case chatwork.TaskRoomsCreate, chatwork.TaskRoomsUpdate, chatwork.TaskMembersReplace,
		chatwork.TaskMessagesSend, chatwork.TaskMessagesMarkRead, chatwork.TaskMessagesMarkUnread,
		chatwork.TaskMessagesUpdate, chatwork.TaskMessagesDelete, chatwork.TaskRoomTasksCreate,
		chatwork.TaskRoomTasksSetStatus, chatwork.TaskFilesUpload:
		if err := mapIdentityResponse(input, body, &result); err != nil {
			return chatwork.Result{}, err
		}
	default:
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_response_unmapped", "Chatwork レスポンスタスクに意味マッピングがありません", false)
	}
	return result, nil
}

func mapIdentityResponse(input chatwork.Request, body []byte, result *chatwork.Result) error {
	var wire identityDTO
	if err := decodeJSON(body, &wire); err != nil {
		return err
	}
	switch input.Task {
	case chatwork.TaskRoomsCreate:
		ref, err := reference(chatwork.ReferenceRoom, wire.RoomID)
		if err != nil {
			return err
		}
		result.Created = []chatwork.Reference{ref}
	case chatwork.TaskRoomsUpdate:
		ref, err := reference(chatwork.ReferenceRoom, wire.RoomID)
		if err != nil {
			return err
		}
		result.Affected = []chatwork.Reference{ref}
	case chatwork.TaskMembersReplace:
		result.MembershipCounts = &chatwork.MembershipCounts{
			Administrators: int64(len(wire.Admin)),
			Members:        int64(len(wire.Member)),
			Readonly:       int64(len(wire.Readonly)),
		}
	case chatwork.TaskMessagesSend:
		ref, err := reference(chatwork.ReferenceMessage, wire.MessageID)
		if err != nil {
			return err
		}
		result.CreatedInRoom = &chatwork.RoomScopedCreation{Refs: []chatwork.Reference{ref}, ParentRoom: input.Room}
	case chatwork.TaskMessagesUpdate, chatwork.TaskMessagesDelete:
		ref, err := reference(chatwork.ReferenceMessage, wire.MessageID)
		if err != nil {
			return err
		}
		result.Affected = []chatwork.Reference{ref}
	case chatwork.TaskMessagesMarkRead, chatwork.TaskMessagesMarkUnread:
		result.ReadState = &chatwork.ReadState{Unread: wire.Unread, Mentions: wire.Mentions}
	case chatwork.TaskRoomTasksCreate:
		created := make([]chatwork.Reference, 0, len(wire.TaskIDs))
		for _, id := range wire.TaskIDs {
			ref, err := reference(chatwork.ReferenceTask, id)
			if err != nil {
				return err
			}
			created = append(created, ref)
		}
		if len(created) == 0 {
			return malformedResponse()
		}
		result.CreatedInRoom = &chatwork.RoomScopedCreation{Refs: created, ParentRoom: input.Room}
	case chatwork.TaskRoomTasksSetStatus:
		ref, err := reference(chatwork.ReferenceTask, wire.TaskID)
		if err != nil {
			return err
		}
		result.Affected = []chatwork.Reference{ref}
	case chatwork.TaskFilesUpload:
		ref, err := reference(chatwork.ReferenceFile, wire.FileID)
		if err != nil {
			return err
		}
		result.CreatedInRoom = &chatwork.RoomScopedCreation{Refs: []chatwork.Reference{ref}, ParentRoom: input.Room}
	}
	return nil
}

func emptyResult(input chatwork.Request) chatwork.Result {
	result := chatwork.Result{Task: input.Task, Coverage: coverageFor(input)}
	switch input.Task {
	case chatwork.TaskContactsList, chatwork.TaskMembersList:
		result.Accounts = []chatwork.Account{}
	case chatwork.TaskRoomsList:
		result.Rooms = []chatwork.Room{}
	case chatwork.TaskMessagesList:
		result.MessageRoom = input.Room
		result.Messages = []chatwork.Message{}
	case chatwork.TaskPersonalTasksList, chatwork.TaskRoomTasksList:
		result.Tasks = []chatwork.WorkTask{}
	case chatwork.TaskFilesList:
		result.Files = []chatwork.File{}
	case chatwork.TaskContactRequestsList:
		result.Requests = []chatwork.ContactRequest{}
	case chatwork.TaskRoomsLeave, chatwork.TaskRoomsDelete:
		result.Acknowledgement = &chatwork.Acknowledgement{Acknowledged: true, Target: input.Room}
	case chatwork.TaskContactRequestsReject:
		result.Acknowledgement = &chatwork.Acknowledgement{Acknowledged: true, Target: input.Request}
	}
	return result
}

func coverageFor(input chatwork.Request) chatwork.Coverage {
	switch input.Task {
	case chatwork.TaskMessagesList:
		kind, description := "differential_window", "messages since the previous differential retrieval, bounded by the provider's 100-message window"
		if input.ForceRecent {
			kind, description = "latest_window", "latest messages bounded by the provider's 100-message window"
		}
		return chatwork.Coverage{Kind: kind, Limit: 100, Complete: false, Description: description}
	case chatwork.TaskPersonalTasksList, chatwork.TaskRoomTasksList, chatwork.TaskFilesList, chatwork.TaskContactRequestsList:
		return chatwork.Coverage{Kind: "provider_window", Limit: 100, Complete: false, Description: "result is bounded by the provider's documented 100-item window"}
	case chatwork.TaskContactsList, chatwork.TaskRoomsList, chatwork.TaskMembersList:
		return chatwork.Coverage{Kind: "provider_collection", Complete: true, Description: "provider returned the complete documented collection for this operation"}
	default:
		return chatwork.Coverage{Kind: "single_operation", Complete: true, Description: "one provider operation returned a complete task result"}
	}
}

func mapAccount(wire accountDTO) (chatwork.Account, error) {
	ref, err := reference(chatwork.ReferenceAccount, wire.AccountID)
	if err != nil {
		return chatwork.Account{}, err
	}
	account := chatwork.Account{Ref: ref, Name: wire.Name, ChatworkID: wire.ChatworkID, OrganizationID: string(wire.OrganizationID), OrganizationName: wire.Organization, Department: wire.Department, Title: wire.Title, URL: wire.URL, Introduction: wire.Introduction, Mail: wire.Mail, Telephone: wire.Telephone, Extension: wire.Extension, Mobile: wire.Mobile, Skype: wire.Skype, Facebook: wire.Facebook, Twitter: wire.Twitter, AvatarURL: wire.AvatarURL, LoginMail: wire.LoginMail, Role: wire.Role}
	if wire.RoomID != "" {
		account.Room, err = reference(chatwork.ReferenceRoom, wire.RoomID)
	}
	return account, err
}

func mapRoom(wire roomDTO) (chatwork.Room, error) {
	ref, err := reference(chatwork.ReferenceRoom, wire.RoomID)
	if err != nil {
		return chatwork.Room{}, err
	}
	return chatwork.Room{Ref: ref, Name: wire.Name, Type: wire.Type, Role: wire.Role, Sticky: wire.Sticky, Unread: wire.Unread, Mentions: wire.Mentions, MyTasks: wire.MyTasks, Messages: wire.Messages, Files: wire.Files, Tasks: wire.Tasks, IconURL: wire.IconURL, LastUpdateTime: wire.LastUpdateTime, Description: wire.Description}, nil
}

func mapRooms(wire []roomDTO) ([]chatwork.Room, error) {
	rooms := make([]chatwork.Room, 0, len(wire))
	for _, item := range wire {
		room, err := mapRoom(item)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func mapMessages(room chatwork.Reference, wire []messageDTO) ([]chatwork.Message, error) {
	messages := make([]chatwork.Message, 0, len(wire))
	for _, item := range wire {
		ref, err := reference(chatwork.ReferenceMessage, item.MessageID)
		if err != nil {
			return nil, err
		}
		sender, err := mapAccount(item.Account)
		if err != nil {
			return nil, err
		}
		recipients, reply, quotes, err := parseNotation(item.Body)
		if err != nil {
			return nil, err
		}
		messages = append(messages, chatwork.Message{Ref: ref, Room: room, Sender: sender, Body: item.Body, SendTime: item.SendTime, UpdateTime: item.UpdateTime, Recipients: recipients, Reply: reply, Quotes: quotes})
	}
	return messages, nil
}

func mapTasks(room chatwork.Reference, wire []taskDTO) ([]chatwork.WorkTask, error) {
	tasks := make([]chatwork.WorkTask, 0, len(wire))
	for _, item := range wire {
		ref, err := reference(chatwork.ReferenceTask, item.TaskID)
		if err != nil {
			return nil, err
		}
		message, err := reference(chatwork.ReferenceMessage, item.MessageID)
		if err != nil {
			return nil, err
		}
		taskRoom := chatwork.Room{Ref: room}
		if item.Room.RoomID != "" {
			taskRoom, err = mapRoom(item.Room)
			if err != nil {
				return nil, err
			}
		}
		var account chatwork.Account
		if item.Account.AccountID != "" {
			account, err = mapAccount(item.Account)
			if err != nil {
				return nil, err
			}
		}
		assignedBy, err := mapAccount(item.AssignedByAccount)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, chatwork.WorkTask{Ref: ref, Room: taskRoom, Account: account, AssignedBy: assignedBy, Message: message, Body: item.Body, LimitTime: item.LimitTime, Status: item.Status, LimitType: item.LimitType})
	}
	return tasks, nil
}

func mapFiles(room chatwork.Reference, wire []fileDTO) ([]chatwork.File, error) {
	files := make([]chatwork.File, 0, len(wire))
	for _, item := range wire {
		ref, err := reference(chatwork.ReferenceFile, item.FileID)
		if err != nil {
			return nil, err
		}
		account, err := mapAccount(item.Account)
		if err != nil {
			return nil, err
		}
		message, err := reference(chatwork.ReferenceMessage, item.MessageID)
		if err != nil {
			return nil, err
		}
		files = append(files, chatwork.File{Ref: ref, Room: room, Account: account, Message: message, Name: item.Filename, Size: item.Filesize, UploadTime: item.UploadTime, DownloadURL: item.DownloadURL})
	}
	return files, nil
}

func mapContactRequests(wire []contactRequestDTO) ([]chatwork.ContactRequest, error) {
	requests := make([]chatwork.ContactRequest, 0, len(wire))
	for _, item := range wire {
		ref, err := reference(chatwork.ReferenceRequest, item.RequestID)
		if err != nil {
			return nil, err
		}
		account, err := mapAccount(accountDTO{AccountID: item.AccountID, RoomID: item.RoomID, Name: item.Name, ChatworkID: item.ChatworkID, OrganizationID: item.OrganizationID, Organization: item.Organization, Department: item.Department, AvatarURL: item.AvatarURL})
		if err != nil {
			return nil, err
		}
		requests = append(requests, chatwork.ContactRequest{Ref: ref, Account: account, Message: item.Message})
	}
	return requests, nil
}

func inviteReference(input chatwork.Request) (chatwork.Reference, error) {
	if input.Invite.Value != "" {
		return input.Invite, nil
	}
	return chatwork.NewReference(chatwork.ReferenceInvite, input.Room.Value)
}

func reference(kind chatwork.ReferenceKind, id wireID) (chatwork.Reference, error) {
	ref, err := chatwork.NewReference(kind, string(id))
	if err != nil {
		return chatwork.Reference{}, malformedResponse()
	}
	return ref, nil
}

func malformedResponse() error {
	return fault.New(fault.KindContract, "chatwork_response_malformed", "Chatwork レスポンスがレビュー済みの通信契約と一致しません", false)
}
