// Package chatwork defines provider-independent task values used by the
// Chatwork product. Wire DTOs and HTTP operation names remain in infrastructure.
package chatwork

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	AuthenticationAuthority  = "chatwork"
	AuthenticationAudience   = "chatwork-api-v2"
	AuthenticationCapability = "chatwork.api"
)

// ReferenceKind declares which exact provider identity a value represents.
type ReferenceKind string

const (
	ReferenceAccount ReferenceKind = "chatwork-account"
	ReferenceRoom    ReferenceKind = "chatwork-room"
	ReferenceMessage ReferenceKind = "chatwork-message"
	ReferenceTask    ReferenceKind = "chatwork-task"
	ReferenceFile    ReferenceKind = "chatwork-file"
	ReferenceInvite  ReferenceKind = "chatwork-invite-link"
	ReferenceRequest ReferenceKind = "chatwork-contact-request"
)

// Reference preserves the exact validated provider value. Display aliases are
// deliberately not represented by this type.
type Reference struct {
	Kind  ReferenceKind
	Value string
}

func NewReference(kind ReferenceKind, value string) (Reference, error) {
	if err := ValidateReference(kind, value); err != nil {
		return Reference{}, err
	}
	return Reference{Kind: kind, Value: value}, nil
}

func ValidateReference(kind ReferenceKind, value string) error {
	switch kind {
	case ReferenceAccount, ReferenceRoom, ReferenceMessage, ReferenceTask, ReferenceFile, ReferenceInvite, ReferenceRequest:
	default:
		return fmt.Errorf("reference kind is missing or invalid")
	}
	if len(value) == 0 || len(value) > 32 {
		return fmt.Errorf("%s reference must contain 1 to 32 decimal digits", kind)
	}
	if value[0] == '0' {
		return fmt.Errorf("%s reference must use its canonical positive decimal form", kind)
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return fmt.Errorf("%s reference must use its canonical positive decimal form", kind)
		}
	}
	return nil
}

// Task names user outcomes rather than HTTP methods or provider routes.
type Task string

const (
	TaskAccountShow           Task = "account.show"
	TaskAccountStatus         Task = "account.status"
	TaskPersonalTasksList     Task = "personal-tasks.list"
	TaskContactsList          Task = "contacts.list"
	TaskRoomsList             Task = "rooms.list"
	TaskRoomsCreate           Task = "rooms.create"
	TaskRoomsShow             Task = "rooms.show"
	TaskRoomsUpdate           Task = "rooms.update"
	TaskRoomsLeave            Task = "rooms.leave"
	TaskRoomsDelete           Task = "rooms.delete"
	TaskMembersList           Task = "members.list"
	TaskMembersReplace        Task = "members.replace"
	TaskMessagesList          Task = "messages.list"
	TaskMessagesSend          Task = "messages.send"
	TaskMessagesMarkRead      Task = "messages.mark-read"
	TaskMessagesMarkUnread    Task = "messages.mark-unread"
	TaskMessagesShow          Task = "messages.show"
	TaskMessagesUpdate        Task = "messages.update"
	TaskMessagesDelete        Task = "messages.delete"
	TaskRoomTasksList         Task = "room-tasks.list"
	TaskRoomTasksCreate       Task = "room-tasks.create"
	TaskRoomTasksShow         Task = "room-tasks.show"
	TaskRoomTasksSetStatus    Task = "room-tasks.set-status"
	TaskFilesList             Task = "files.list"
	TaskFilesUpload           Task = "files.upload"
	TaskFilesShow             Task = "files.show"
	TaskInviteLinkShow        Task = "invite-link.show"
	TaskInviteLinkCreate      Task = "invite-link.create"
	TaskInviteLinkUpdate      Task = "invite-link.update"
	TaskInviteLinkDelete      Task = "invite-link.delete"
	TaskContactRequestsList   Task = "contact-requests.list"
	TaskContactRequestsAccept Task = "contact-requests.accept"
	TaskContactRequestsReject Task = "contact-requests.reject"
)

// Request is the typed union consumed by the application task boundary.
// Fields unused by the selected Task must remain zero; Validate enforces this
// incrementally as task implementations are added.
type Request struct {
	Task Task

	Room    Reference
	Message Reference
	TaskRef Reference
	File    Reference
	Invite  Reference
	Request Reference

	Account             Reference
	AssignedBy          Reference
	Admins              []Reference
	Members             []Reference
	ReadonlyMembers     []Reference
	Assignees           []Reference
	Name                string
	Description         string
	Icon                string
	Body                string
	Status              string
	RoomAction          string
	Limit               int64
	LimitType           string
	ForceRecent         bool
	SelfUnread          bool
	CreateDownloadURL   bool
	InviteCode          string
	InviteEnabled       bool
	InviteNeedsApproval bool
	InviteApprovalSet   bool
	FilePath            string
	FileMessage         string
}

func (r Request) Validate() error {
	if !r.Task.Valid() {
		return fmt.Errorf("Chatwork task is missing or invalid")
	}
	for _, pair := range []struct {
		ref  Reference
		kind ReferenceKind
	}{
		{r.Room, ReferenceRoom}, {r.Message, ReferenceMessage},
		{r.TaskRef, ReferenceTask}, {r.File, ReferenceFile}, {r.Invite, ReferenceInvite},
		{r.Request, ReferenceRequest}, {r.Account, ReferenceAccount},
		{r.AssignedBy, ReferenceAccount},
	} {
		if pair.ref.Value == "" {
			continue
		}
		if pair.ref.Kind != pair.kind {
			return fmt.Errorf("task reference kind mismatch: got %s, want %s", pair.ref.Kind, pair.kind)
		}
		if err := ValidateReference(pair.ref.Kind, pair.ref.Value); err != nil {
			return err
		}
	}
	for _, refs := range [][]Reference{r.Admins, r.Members, r.ReadonlyMembers, r.Assignees} {
		seen := make(map[string]struct{}, len(refs))
		for _, ref := range refs {
			if ref.Kind != ReferenceAccount {
				return fmt.Errorf("member and assignee references must be account references")
			}
			if err := ValidateReference(ref.Kind, ref.Value); err != nil {
				return err
			}
			if _, exists := seen[ref.Value]; exists {
				return fmt.Errorf("account references must be unique within one role")
			}
			seen[ref.Value] = struct{}{}
		}
	}
	for name, value := range map[string]string{
		"name": r.Name, "description": r.Description, "body": r.Body,
		"file message": r.FileMessage,
	} {
		if err := validateText(name, value, 65535); err != nil {
			return err
		}
	}
	return nil
}

func (t Task) Valid() bool {
	switch t {
	case TaskAccountShow, TaskAccountStatus, TaskPersonalTasksList, TaskContactsList,
		TaskRoomsList, TaskRoomsCreate, TaskRoomsShow, TaskRoomsUpdate, TaskRoomsLeave,
		TaskRoomsDelete, TaskMembersList, TaskMembersReplace, TaskMessagesList,
		TaskMessagesSend, TaskMessagesMarkRead, TaskMessagesMarkUnread, TaskMessagesShow,
		TaskMessagesUpdate, TaskMessagesDelete, TaskRoomTasksList, TaskRoomTasksCreate,
		TaskRoomTasksShow, TaskRoomTasksSetStatus, TaskFilesList, TaskFilesUpload,
		TaskFilesShow, TaskInviteLinkShow, TaskInviteLinkCreate, TaskInviteLinkUpdate,
		TaskInviteLinkDelete, TaskContactRequestsList, TaskContactRequestsAccept,
		TaskContactRequestsReject:
		return true
	default:
		return false
	}
}

func validateText(name, value string, limit int) error {
	if !utf8.ValidString(value) || len(value) > limit {
		return fmt.Errorf("%s must be valid UTF-8 within %d bytes", name, limit)
	}
	if strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("%s must not contain NUL", name)
	}
	return nil
}

type Account struct {
	Ref              Reference
	Room             Reference
	Name             string
	ChatworkID       string
	OrganizationID   string
	OrganizationName string
	Department       string
	Title            string
	URL              string
	Introduction     string
	Mail             string
	Telephone        string
	Extension        string
	Mobile           string
	Skype            string
	Facebook         string
	Twitter          string
	AvatarURL        string
	LoginMail        string
	Role             string
}

type Status struct {
	UnreadRooms  int64
	MentionRooms int64
	TaskRooms    int64
	Unread       int64
	Mentions     int64
	Tasks        int64
}

type Room struct {
	Ref            Reference
	Name           string
	Type           string
	Role           string
	Sticky         bool
	Unread         int64
	Mentions       int64
	MyTasks        int64
	Messages       int64
	Files          int64
	Tasks          int64
	IconURL        string
	LastUpdateTime int64
	Description    string
}

type Message struct {
	Ref        Reference
	Room       Reference
	Sender     Account
	Body       string
	SendTime   int64
	UpdateTime int64
	Recipients []Reference
	Reply      *Relation
	Quotes     []Relation
}

type Relation struct {
	Kind       string
	Target     Reference
	Resolved   bool
	ExternalID string
}

type WorkTask struct {
	Ref        Reference
	Room       Room
	Account    Account
	AssignedBy Account
	Message    Reference
	Body       string
	LimitTime  int64
	Status     string
	LimitType  string
}

type File struct {
	Ref         Reference
	Room        Reference
	Account     Account
	Message     Reference
	Name        string
	Size        int64
	UploadTime  int64
	DownloadURL string
}

type InviteLink struct {
	Ref           Reference
	Public        bool
	URL           string
	NeedsApproval bool
	Description   string
}

type ContactRequest struct {
	Ref     Reference
	Account Account
	Message string
}

type Coverage struct {
	Kind        string
	Limit       int
	Complete    bool
	Description string
}

// Result is a typed semantic union. Only fields relevant to Task are populated.
type Result struct {
	Task       Task
	Coverage   Coverage
	Account    *Account
	Status     *Status
	Rooms      []Room
	Accounts   []Account
	Messages   []Message
	Tasks      []WorkTask
	Files      []File
	InviteLink *InviteLink
	Requests   []ContactRequest
	Created    []Reference
	Affected   []Reference
	Unread     int64
	Mentions   int64
	RoleCounts map[string]int64
}
