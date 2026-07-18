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

// RoomScopedCreation preserves both newly created object identities and the
// exact room scope supplied by the caller. The parent is part of the task
// result rather than presentation reconstruction.
type RoomScopedCreation struct {
	Refs       []Reference
	ParentRoom Reference
}

// ReadState represents the provider-confirmed room counters after a read-state
// mutation. A pointer on Result distinguishes an explicit 0/0 state from a
// task that did not return read-state facts.
type ReadState struct {
	Unread   int64
	Mentions int64
}

// Acknowledgement records a provider-confirmed empty-body mutation and the
// exact target supplied by the caller.
type Acknowledgement struct {
	Acknowledged bool
	Target       Reference
}

// MembershipCounts is the typed members.replace outcome. Its fields use
// provider-independent task vocabulary instead of wire response keys.
type MembershipCounts struct {
	Administrators int64
	Members        int64
	Readonly       int64
}

type Coverage struct {
	Kind        string
	Limit       int
	Complete    bool
	Description string
}

// Result is a typed semantic union. Only fields relevant to Task are populated.
type Result struct {
	Task             Task
	Coverage         Coverage
	Account          *Account
	Status           *Status
	Rooms            []Room
	Accounts         []Account
	Messages         []Message
	Tasks            []WorkTask
	Files            []File
	InviteLink       *InviteLink
	Requests         []ContactRequest
	Created          []Reference
	Affected         []Reference
	CreatedInRoom    *RoomScopedCreation
	ReadState        *ReadState
	Acknowledgement  *Acknowledgement
	MembershipCounts *MembershipCounts
}

// Validate proves that the semantic union uses the one result variant owned by
// Task. It deliberately validates provider-independent result shape here so a
// renderer cannot guess whether a zero value was absent or explicitly
// returned.
func (r Result) Validate() error {
	if !r.Task.Valid() {
		return fmt.Errorf("Chatwork result task is missing or invalid")
	}
	if r.Coverage.Limit < 0 {
		return fmt.Errorf("Chatwork result coverage limit must not be negative")
	}

	variant := r.resultVariant()
	want := ""
	switch r.Task {
	case TaskAccountShow, TaskContactRequestsAccept:
		want = "account"
	case TaskAccountStatus:
		want = "status"
	case TaskPersonalTasksList, TaskRoomTasksList, TaskRoomTasksShow:
		want = "tasks"
	case TaskContactsList, TaskMembersList:
		want = "accounts"
	case TaskRoomsList, TaskRoomsShow:
		want = "rooms"
	case TaskRoomsCreate:
		want = "created"
	case TaskRoomsUpdate, TaskMessagesUpdate, TaskMessagesDelete, TaskRoomTasksSetStatus:
		want = "affected"
	case TaskRoomsLeave, TaskRoomsDelete, TaskContactRequestsReject:
		want = "acknowledgement"
	case TaskMembersReplace:
		want = "membership-counts"
	case TaskMessagesList, TaskMessagesShow:
		want = "messages"
	case TaskMessagesSend, TaskRoomTasksCreate, TaskFilesUpload:
		want = "room-scoped-creation"
	case TaskMessagesMarkRead, TaskMessagesMarkUnread:
		want = "read-state"
	case TaskFilesList, TaskFilesShow:
		want = "files"
	case TaskInviteLinkShow, TaskInviteLinkCreate, TaskInviteLinkUpdate, TaskInviteLinkDelete:
		want = "invite-link"
	case TaskContactRequestsList:
		want = "contact-requests"
	}
	if variant != want {
		return fmt.Errorf("Chatwork result variant is %q, want %q for %s", variant, want, r.Task)
	}

	switch r.Task {
	case TaskRoomsShow:
		if len(r.Rooms) != 1 {
			return fmt.Errorf("rooms.show result must contain exactly one room")
		}
	case TaskMessagesShow:
		if len(r.Messages) != 1 {
			return fmt.Errorf("messages.show result must contain exactly one message")
		}
	case TaskRoomTasksShow:
		if len(r.Tasks) != 1 {
			return fmt.Errorf("room-tasks.show result must contain exactly one task")
		}
	case TaskFilesShow:
		if len(r.Files) != 1 {
			return fmt.Errorf("files.show result must contain exactly one file")
		}
	}

	return r.validateVariantFacts()
}

// ValidateFor additionally binds result identities to the exact validated
// request. A structurally valid result for another parent or target is not a
// valid outcome of this invocation.
func (r Result) ValidateFor(request Request) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("Chatwork result request is invalid: %w", err)
	}
	if r.Task != request.Task {
		return fmt.Errorf("Chatwork result task is %s, want %s", r.Task, request.Task)
	}
	if err := r.Validate(); err != nil {
		return err
	}

	switch r.Task {
	case TaskMessagesSend, TaskRoomTasksCreate, TaskFilesUpload:
		if r.CreatedInRoom.ParentRoom != request.Room {
			return fmt.Errorf("Chatwork result parent room does not match the request")
		}
	case TaskRoomsLeave, TaskRoomsDelete:
		if r.Acknowledgement.Target != request.Room {
			return fmt.Errorf("Chatwork result acknowledged room does not match the request")
		}
	case TaskContactRequestsReject:
		if r.Acknowledgement.Target != request.Request {
			return fmt.Errorf("Chatwork result acknowledged contact request does not match the request")
		}
	case TaskRoomsUpdate:
		if len(r.Affected) != 1 || r.Affected[0] != request.Room {
			return fmt.Errorf("Chatwork result affected room does not match the request")
		}
	case TaskMessagesUpdate, TaskMessagesDelete:
		if len(r.Affected) != 1 || r.Affected[0] != request.Message {
			return fmt.Errorf("Chatwork result affected message does not match the request")
		}
	case TaskRoomTasksSetStatus:
		if len(r.Affected) != 1 || r.Affected[0] != request.TaskRef {
			return fmt.Errorf("Chatwork result affected task does not match the request")
		}
	}
	return nil
}

func (r Result) resultVariant() string {
	variants := make([]string, 0, 16)
	add := func(name string, present bool) {
		if present {
			variants = append(variants, name)
		}
	}
	add("account", r.Account != nil)
	add("status", r.Status != nil)
	add("rooms", r.Rooms != nil)
	add("accounts", r.Accounts != nil)
	add("messages", r.Messages != nil)
	add("tasks", r.Tasks != nil)
	add("files", r.Files != nil)
	add("invite-link", r.InviteLink != nil)
	add("contact-requests", r.Requests != nil)
	add("created", r.Created != nil)
	add("affected", r.Affected != nil)
	add("room-scoped-creation", r.CreatedInRoom != nil)
	add("read-state", r.ReadState != nil)
	add("acknowledgement", r.Acknowledgement != nil)
	add("membership-counts", r.MembershipCounts != nil)
	if len(variants) == 1 {
		return variants[0]
	}
	if len(variants) == 0 {
		return "absent"
	}
	return strings.Join(variants, "+")
}

func (r Result) validateVariantFacts() error {
	validateRef := func(ref Reference, kind ReferenceKind) error {
		if ref.Kind != kind {
			return fmt.Errorf("Chatwork result reference kind is %s, want %s", ref.Kind, kind)
		}
		return ValidateReference(ref.Kind, ref.Value)
	}
	validateRefs := func(refs []Reference, kind ReferenceKind, allowMany bool) error {
		if len(refs) == 0 || (!allowMany && len(refs) != 1) {
			return fmt.Errorf("Chatwork result must contain the declared %s reference cardinality", kind)
		}
		seen := make(map[Reference]struct{}, len(refs))
		for _, ref := range refs {
			if err := validateRef(ref, kind); err != nil {
				return err
			}
			if _, exists := seen[ref]; exists {
				return fmt.Errorf("Chatwork result contains a duplicate %s reference", kind)
			}
			seen[ref] = struct{}{}
		}
		return nil
	}

	switch r.Task {
	case TaskRoomsCreate:
		return validateRefs(r.Created, ReferenceRoom, false)
	case TaskRoomsUpdate:
		return validateRefs(r.Affected, ReferenceRoom, false)
	case TaskMessagesUpdate, TaskMessagesDelete:
		return validateRefs(r.Affected, ReferenceMessage, false)
	case TaskRoomTasksSetStatus:
		return validateRefs(r.Affected, ReferenceTask, false)
	case TaskMessagesSend:
		if err := validateRefs(r.CreatedInRoom.Refs, ReferenceMessage, false); err != nil {
			return err
		}
		return validateRef(r.CreatedInRoom.ParentRoom, ReferenceRoom)
	case TaskRoomTasksCreate:
		if err := validateRefs(r.CreatedInRoom.Refs, ReferenceTask, true); err != nil {
			return err
		}
		return validateRef(r.CreatedInRoom.ParentRoom, ReferenceRoom)
	case TaskFilesUpload:
		if err := validateRefs(r.CreatedInRoom.Refs, ReferenceFile, false); err != nil {
			return err
		}
		return validateRef(r.CreatedInRoom.ParentRoom, ReferenceRoom)
	case TaskMessagesMarkRead, TaskMessagesMarkUnread:
		if r.ReadState.Unread < 0 || r.ReadState.Mentions < 0 {
			return fmt.Errorf("Chatwork read-state counts must not be negative")
		}
	case TaskRoomsLeave, TaskRoomsDelete:
		if !r.Acknowledgement.Acknowledged {
			return fmt.Errorf("Chatwork room acknowledgement must be explicit")
		}
		return validateRef(r.Acknowledgement.Target, ReferenceRoom)
	case TaskContactRequestsReject:
		if !r.Acknowledgement.Acknowledged {
			return fmt.Errorf("Chatwork contact-request acknowledgement must be explicit")
		}
		return validateRef(r.Acknowledgement.Target, ReferenceRequest)
	case TaskMembersReplace:
		if r.MembershipCounts.Administrators < 0 || r.MembershipCounts.Members < 0 || r.MembershipCounts.Readonly < 0 {
			return fmt.Errorf("Chatwork membership counts must not be negative")
		}
	}
	return nil
}
