// Package chatwork defines provider-independent task values used by the
// Chatwork product. Wire DTOs and HTTP operation names remain in infrastructure.
package chatwork

import (
	"fmt"
	"strings"
	"time"
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
	TaskMembersFind           Task = "members.find"
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

// MessageContext selects the bounded typed reply context added around messages
// that directly match a message filter. The zero value means that no filter is
// active; an active filter must choose one of the declared policies.
type MessageContext string

const (
	MessageContextNone    MessageContext = "none"
	MessageContextReplies MessageContext = "replies"
	MessageDayTimeZone                   = "Asia/Tokyo"

	// MaxMessageSelectionCount matches the largest provider message window. A
	// smaller public count narrows primary messages only; it does not increase
	// or page beyond this fixed source bound.
	MaxMessageSelectionCount = 100
	// MaxMessageRelationFetches bounds explicit exact-message reads used to
	// complete reply context outside one provider message window.
	MaxMessageRelationFetches = 100
	// DefaultMessageRelationFetches closes ordinary reply chains without an
	// extra option while retaining a small deterministic provider-call bound.
	DefaultMessageRelationFetches = 5
)

var messageDayLocation = time.FixedZone(MessageDayTimeZone, 9*60*60)

// MessagePeriod is one concrete half-open Unix-second interval used only to
// select primary messages from a bounded provider window. Day and TimeZone are
// present only when CLI day vocabulary resolved to this interval.
type MessagePeriod struct {
	Since    int64
	Until    int64
	Day      string
	TimeZone string
}

// NewMessagePeriod validates one explicit half-open interval. A zero bound is
// absent so callers can express since-only and until-only selection.
func NewMessagePeriod(since, until int64) (MessagePeriod, error) {
	period := MessagePeriod{Since: since, Until: until}
	if err := validateMessagePeriod(period); err != nil {
		return MessagePeriod{}, err
	}
	return period, nil
}

// NewMessageDayPeriod converts one exact Japanese calendar day to concrete
// half-open Unix bounds without consulting ambient host time-zone state.
func NewMessageDayPeriod(day string) (MessagePeriod, error) {
	return messageDayPeriod(day)
}

// Contains reports membership in the concrete half-open period. The zero
// period contains every timestamp.
func (p MessagePeriod) Contains(sendTime int64) bool {
	if p.Since > 0 && sendTime < p.Since {
		return false
	}
	if p.Until > 0 && sendTime >= p.Until {
		return false
	}
	return true
}

// MessageFilter selects primary messages authored by any exact canonical
// account and inside the concrete period, with either predicate omitted when
// zero. StartIndex is the 1-based newest-first primary rank, Count keeps at most
// that many primary messages, and Context then adds direct reply neighbors.
type MessageFilter struct {
	Senders    []Reference
	Period     MessagePeriod
	Context    MessageContext
	StartIndex int
	Count      int
}

// MessagePeriodReachability states only what one trustworthy latest-window
// lower boundary can prove about a requested period. It does not claim room
// history completeness or discoverability beyond the provider window.
type MessagePeriodReachability string

const (
	MessagePeriodWithinReachableWindow     MessagePeriodReachability = "within-reachable-window"
	MessagePeriodPartiallyOutsideReachable MessagePeriodReachability = "partially-out-of-reachable-window"
	MessagePeriodOutsideReachableWindow    MessagePeriodReachability = "out-of-reachable-window"
	MessagePeriodReachabilityUnknown       MessagePeriodReachability = "unknown"
)

// MessageReachability records the oldest typed message proven reachable by
// the current latest-window list result. OldestMessage is absent when an empty,
// differential, or access-limited source cannot establish that boundary.
type MessageReachability struct {
	OldestMessage      Reference
	OldestSendTime     int64
	PeriodReachability MessagePeriodReachability
}

// MessageRelationResolutionState distinguishes how one explicit same-room
// reply target was handled. Only source and fetched states carry Message.
type MessageRelationResolutionState string

const (
	MessageRelationResolvedFromSource MessageRelationResolutionState = "source"
	MessageRelationResolvedByFetch    MessageRelationResolutionState = "fetched"
	MessageRelationNotFound           MessageRelationResolutionState = "not-found"
	MessageRelationRestricted         MessageRelationResolutionState = "restricted"
	MessageRelationBudgetExhausted    MessageRelationResolutionState = "budget-exhausted"
)

type MessageRelationTarget struct {
	Target  Reference
	State   MessageRelationResolutionState
	Message *Message
}

// MessageRelationResolution is the result of one explicit finite exact-read
// budget. Targets retain deterministic first-reference order.
type MessageRelationResolution struct {
	FetchLimit    int
	FetchAttempts int
	Targets       []MessageRelationTarget
}

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

	Account                   Reference
	AssignedBy                Reference
	Admins                    []Reference
	Members                   []Reference
	ReadonlyMembers           []Reference
	Assignees                 []Reference
	Name                      string
	Description               string
	Icon                      string
	Body                      string
	Status                    string
	RoomAction                string
	Limit                     int64
	LimitType                 string
	ForceRecent               bool
	SelfUnread                bool
	CreateDownloadURL         bool
	InviteCode                string
	InviteRegenerateCode      bool
	InviteEnabled             bool
	InviteNeedsApproval       bool
	InviteApprovalSet         bool
	DescriptionSet            bool
	FilePath                  string
	FileMessage               string
	MemberQuery               string
	MessageFilter             MessageFilter
	MessageRelationFetchLimit int
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
		"file message": r.FileMessage, "member query": r.MemberQuery,
	} {
		if err := validateText(name, value, 65535); err != nil {
			return err
		}
	}
	if r.Task == TaskMembersFind {
		if r.Room.Value == "" {
			return fmt.Errorf("member discovery requires a room reference")
		}
		if r.MemberQuery == "" || len(r.MemberQuery) > 255 {
			return fmt.Errorf("member discovery query must contain 1 to 255 UTF-8 bytes")
		}
	} else if r.MemberQuery != "" {
		return fmt.Errorf("member query is only valid for members.find")
	}
	if r.Task == TaskRoomsCreate {
		if r.Account.Value == "" {
			return fmt.Errorf("room creation requires an authenticated account reference")
		}
		if r.Name == "" || utf8.RuneCountInString(r.Name) > 255 {
			return fmt.Errorf("room creation name must contain 1 to 255 characters")
		}
		if len(r.Admins) == 0 {
			return fmt.Errorf("room creation requires at least one administrator")
		}
		if r.Icon != "" && !validRoomIconPreset(r.Icon) {
			return fmt.Errorf("room creation icon preset is invalid")
		}
		if !r.InviteEnabled && (r.InviteCode != "" || r.InviteApprovalSet) {
			return fmt.Errorf("room creation invite settings require an enabled invite link")
		}
	}
	if r.InviteCode != "" && !validInviteCode(r.InviteCode) {
		return fmt.Errorf("invite link code must contain 1 to 50 ASCII letters, digits, underscores, or hyphens")
	}
	if r.InviteRegenerateCode && r.Task != TaskInviteLinkUpdate {
		return fmt.Errorf("invite link code regeneration is only valid for invite-link update")
	}
	if r.Task == TaskInviteLinkUpdate {
		if r.Invite.Value == "" {
			return fmt.Errorf("invite-link update requires an invite reference")
		}
		if (r.InviteCode == "") == !r.InviteRegenerateCode {
			return fmt.Errorf("invite-link update requires exactly one explicit code or code regeneration")
		}
		if !r.InviteApprovalSet {
			return fmt.Errorf("invite-link update requires an explicit approval setting")
		}
		if !r.DescriptionSet || r.Description == "" {
			return fmt.Errorf("invite-link update requires an explicit nonempty description")
		}
	}
	if r.Task != TaskMessagesList && messageFilterActive(r.MessageFilter) {
		return fmt.Errorf("message filter is only valid for messages.list")
	}
	if r.MessageRelationFetchLimit < 0 || r.MessageRelationFetchLimit > MaxMessageRelationFetches {
		return fmt.Errorf("message relation fetch limit must be between 0 and %d", MaxMessageRelationFetches)
	}
	if r.Task != TaskMessagesList && r.MessageRelationFetchLimit != 0 {
		return fmt.Errorf("message relation fetch limit is only valid for messages.list")
	}
	if err := validateMessageFilter(r.MessageFilter); err != nil {
		return err
	}
	return nil
}

func validInviteCode(value string) bool {
	if len(value) < 1 || len(value) > 50 {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character >= 'A' && character <= 'Z') ||
			(character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

var roomIconPresets = []string{
	"meeting", "group", "check", "document", "event", "project", "business", "study",
	"security", "star", "idea", "heart", "magcup", "beer", "music", "sports", "travel",
}

// RoomIconPresetValues returns the fixed official room-create vocabulary.
// Callers receive a copy so catalog construction cannot mutate domain policy.
func RoomIconPresetValues() []string {
	return append([]string(nil), roomIconPresets...)
}

func validRoomIconPreset(value string) bool {
	for _, preset := range roomIconPresets {
		if value == preset {
			return true
		}
	}
	return false
}

func validateMessagePeriod(period MessagePeriod) error {
	if period.Since < 0 || period.Until < 0 {
		return fmt.Errorf("message period bounds must be positive Unix times when present")
	}
	if period.Since > 0 && period.Until > 0 && period.Since >= period.Until {
		return fmt.Errorf("message period must be a non-empty half-open interval")
	}
	if period.Day == "" && period.TimeZone == "" {
		return nil
	}
	if period.Day == "" || period.TimeZone != MessageDayTimeZone {
		return fmt.Errorf("message day period requires an exact day and the fixed time zone")
	}
	expected, err := messageDayPeriod(period.Day)
	if err != nil {
		return err
	}
	if period != expected {
		return fmt.Errorf("message day period bounds do not match its calendar day")
	}
	return nil
}

func messageDayPeriod(day string) (MessagePeriod, error) {
	start, err := time.ParseInLocation("2006-01-02", day, messageDayLocation)
	if err != nil || start.Format("2006-01-02") != day {
		return MessagePeriod{}, fmt.Errorf("message day must use a valid YYYY-MM-DD date")
	}
	period := MessagePeriod{
		Since:    start.Unix(),
		Until:    start.AddDate(0, 0, 1).Unix(),
		Day:      day,
		TimeZone: MessageDayTimeZone,
	}
	if period.Since <= 0 || period.Until <= period.Since {
		return MessagePeriod{}, fmt.Errorf("message day must resolve to a positive non-empty interval")
	}
	return period, nil
}

func validateMessageFilter(filter MessageFilter) error {
	if err := validateMessagePeriod(filter.Period); err != nil {
		return fmt.Errorf("message selection period is invalid: %w", err)
	}
	if filter.StartIndex < 0 || filter.StartIndex > MaxMessageSelectionCount {
		return fmt.Errorf("message selection start index must be between 1 and %d when present", MaxMessageSelectionCount)
	}
	if filter.Count < 0 || filter.Count > MaxMessageSelectionCount {
		return fmt.Errorf("message selection count must be between 1 and %d when present", MaxMessageSelectionCount)
	}
	if filter.Count > 0 && filter.StartIndex == 0 {
		return fmt.Errorf("message selection count requires an explicit or defaulted start index")
	}
	if len(filter.Senders) == 0 && filter.Period == (MessagePeriod{}) && filter.StartIndex == 0 && filter.Count == 0 {
		if filter.Context != "" {
			return fmt.Errorf("message context requires a sender, period, or message index selection")
		}
		return nil
	}
	if len(filter.Senders) > 100 {
		return fmt.Errorf("message filter supports at most 100 sender references")
	}
	if filter.Context != MessageContextNone && filter.Context != MessageContextReplies {
		return fmt.Errorf("message filter context is missing or invalid")
	}
	seen := make(map[Reference]struct{}, len(filter.Senders))
	for _, sender := range filter.Senders {
		if sender.Kind != ReferenceAccount {
			return fmt.Errorf("message filter senders must be account references")
		}
		if err := ValidateReference(sender.Kind, sender.Value); err != nil {
			return fmt.Errorf("message filter sender reference is invalid: %w", err)
		}
		if _, exists := seen[sender]; exists {
			return fmt.Errorf("message filter sender references must be unique")
		}
		seen[sender] = struct{}{}
	}
	return nil
}

func messageFilterActive(filter MessageFilter) bool {
	return len(filter.Senders) > 0 || filter.Period != (MessagePeriod{}) || filter.Context != "" || filter.StartIndex > 0 || filter.Count > 0
}

func (t Task) Valid() bool {
	switch t {
	case TaskAccountShow, TaskAccountStatus, TaskPersonalTasksList, TaskContactsList,
		TaskRoomsList, TaskRoomsCreate, TaskRoomsShow, TaskRoomsUpdate, TaskRoomsLeave,
		TaskRoomsDelete, TaskMembersFind, TaskMembersList, TaskMembersReplace, TaskMessagesList,
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
	Ref           Reference
	Room          Reference
	Sender        Account
	Body          string
	SendTime      int64
	UpdateTime    int64
	RelationState MessageRelationState
	Recipients    []Reference
	Reply         *Relation
	Quotes        []Relation
}

// MessageRelationState records whether the complete reviewed relation set was
// derived from provider notation. The zero value is complete so existing
// provider-independent fixtures cannot accidentally claim uncertainty. An
// unknown state carries no partial relation facts: retaining a few facts after
// one malformed tag would overstate what the notation proved.
type MessageRelationState uint8

const (
	MessageRelationsComplete MessageRelationState = iota
	MessageRelationsUnknown
)

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

// MessageAccessLimitation is the provider-authored completeness signal for one
// messages.list invocation. It is independent from Coverage: a recent or
// differential window is bounded even when no messages inside that window are
// access-restricted.
type MessageAccessLimitation uint8

const (
	MessageAccessNone MessageAccessLimitation = iota
	MessageAccessPartial
	MessageAccessAll
)

// MessageSelection records how an application-owned filter projected one
// provider message window. Source sequences retain their original one-based
// positions even when filtering creates gaps. Anchor sequences identify the
// direct filter matches; other displayed sequences are bounded reply context.
type MessageSelection struct {
	Filter          MessageFilter
	SourceCount     int
	CandidateCount  int
	ItemsPerPage    int
	NextStartIndex  int
	SourceSequences []int
	AnchorSequences []int
}

// MemberSelection records how the application projected one complete room
// member collection by display-name query. A result remains a candidate set;
// it never declares that one matching external name is the selected identity.
type MemberSelection struct {
	Query       string
	SourceCount int
}

// Result is a typed semantic union. Only fields relevant to Task are populated.
type Result struct {
	Task     Task
	Coverage Coverage
	// MessageRoom is the exact room scope of a messages.list window. It remains
	// present when Messages is an explicitly empty collection so presentation
	// never has to reconstruct the requested room from an item.
	MessageRoom               Reference
	MessageAccess             MessageAccessLimitation
	Account                   *Account
	Status                    *Status
	Rooms                     []Room
	Accounts                  []Account
	Messages                  []Message
	Tasks                     []WorkTask
	Files                     []File
	InviteLink                *InviteLink
	Requests                  []ContactRequest
	Created                   []Reference
	Affected                  []Reference
	CreatedInRoom             *RoomScopedCreation
	ReadState                 *ReadState
	Acknowledgement           *Acknowledgement
	MembershipCounts          *MembershipCounts
	MemberSelection           *MemberSelection
	MessageSelection          *MessageSelection
	MessageReachability       *MessageReachability
	MessageRelationResolution *MessageRelationResolution
}

// Validate proves that the semantic union uses the one result variant owned by
// Task. It deliberately validates provider-independent result shape here so a
// renderer cannot guess whether a zero value was absent or explicitly
// returned.
func (r Result) Validate() error {
	if !r.Task.Valid() {
		return fmt.Errorf("Chatwork result task is missing or invalid")
	}
	if r.Task != TaskMessagesList && r.MessageRoom != (Reference{}) {
		return fmt.Errorf("Chatwork result message room is only valid for messages.list")
	}
	if r.Task != TaskMessagesList && r.MessageAccess != MessageAccessNone {
		return fmt.Errorf("Chatwork result message access limitation is only valid for messages.list")
	}
	if r.Task != TaskMessagesList && r.MessageSelection != nil {
		return fmt.Errorf("Chatwork result message selection is only valid for messages.list")
	}
	if r.Task != TaskMessagesList && r.MessageReachability != nil {
		return fmt.Errorf("Chatwork result message reachability is only valid for messages.list")
	}
	if r.Task != TaskMessagesList && r.MessageRelationResolution != nil {
		return fmt.Errorf("Chatwork result message relation resolution is only valid for messages.list")
	}
	if r.Task != TaskMembersFind && r.MemberSelection != nil {
		return fmt.Errorf("Chatwork result member selection is only valid for members.find")
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
	case TaskContactsList, TaskMembersFind, TaskMembersList:
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
	case TaskMembersFind:
		if r.MemberSelection == nil {
			return fmt.Errorf("members.find result is missing selection metadata")
		}
		if err := validateText("member selection query", r.MemberSelection.Query, 255); err != nil || r.MemberSelection.Query == "" {
			return fmt.Errorf("members.find result query is invalid")
		}
		if r.MemberSelection.SourceCount < len(r.Accounts) {
			return fmt.Errorf("members.find source count is smaller than its candidate count")
		}
		if !r.Coverage.Complete {
			return fmt.Errorf("members.find requires a complete room member source")
		}
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
	case TaskMembersFind:
		if r.MemberSelection.Query != request.MemberQuery {
			return fmt.Errorf("members.find result query does not match the request")
		}
		for _, account := range r.Accounts {
			if !strings.Contains(account.Name, request.MemberQuery) {
				return fmt.Errorf("members.find result contains a non-matching account")
			}
		}
	case TaskMessagesList:
		if r.MessageRoom != request.Room {
			return fmt.Errorf("Chatwork result message room does not match the request")
		}
		if messageFilterActive(request.MessageFilter) {
			if r.MessageSelection == nil {
				return fmt.Errorf("Chatwork filtered message result is missing selection metadata")
			}
			if !equalMessageFilters(r.MessageSelection.Filter, request.MessageFilter) {
				return fmt.Errorf("Chatwork result message filter does not match the request")
			}
		} else if r.MessageSelection != nil {
			return fmt.Errorf("Chatwork unfiltered message result must not contain selection metadata")
		}
		if request.MessageFilter.Period != (MessagePeriod{}) {
			if r.MessageReachability == nil || r.MessageReachability.PeriodReachability == "" {
				return fmt.Errorf("Chatwork period-selected message result is missing reachability metadata")
			}
		} else if r.MessageReachability != nil && r.MessageReachability.PeriodReachability != "" {
			return fmt.Errorf("Chatwork message result without a period must not declare period reachability")
		}
		if request.MessageRelationFetchLimit > 0 {
			if r.MessageRelationResolution == nil || r.MessageRelationResolution.FetchLimit != request.MessageRelationFetchLimit {
				return fmt.Errorf("Chatwork message relation resolution does not match the request budget")
			}
		} else if r.MessageRelationResolution != nil {
			return fmt.Errorf("Chatwork message result without a relation budget must not declare relation resolution")
		}
	case TaskMessagesShow:
		if len(r.Messages) != 1 || r.Messages[0].Room != request.Room || r.Messages[0].Ref != request.Message {
			return fmt.Errorf("Chatwork exact message result does not match the requested room and message")
		}
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
	case TaskInviteLinkUpdate, TaskInviteLinkDelete:
		if r.InviteLink.Ref != request.Invite {
			return fmt.Errorf("Chatwork result invite link does not match the request")
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
	validateRefs := func(refs []Reference, kind ReferenceKind, allowMany bool) error {
		if len(refs) == 0 || (!allowMany && len(refs) != 1) {
			return fmt.Errorf("Chatwork result must contain the declared %s reference cardinality", kind)
		}
		seen := make(map[Reference]struct{}, len(refs))
		for _, ref := range refs {
			if err := validateResultReference("mutation identity", ref, kind, false); err != nil {
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
	case TaskAccountShow, TaskContactRequestsAccept:
		return validateResultAccount("account", *r.Account, false)
	case TaskContactsList, TaskMembersFind, TaskMembersList:
		for index, account := range r.Accounts {
			if err := validateResultAccount(fmt.Sprintf("account[%d]", index), account, false); err != nil {
				return err
			}
		}
	case TaskRoomsList, TaskRoomsShow:
		for index, room := range r.Rooms {
			if err := validateResultRoom(fmt.Sprintf("room[%d]", index), room); err != nil {
				return err
			}
		}
	case TaskMessagesList, TaskMessagesShow:
		if r.Task == TaskMessagesList {
			if r.MessageAccess != MessageAccessNone && r.MessageAccess != MessageAccessPartial && r.MessageAccess != MessageAccessAll {
				return fmt.Errorf("Chatwork message access limitation is invalid")
			}
			if r.MessageAccess == MessageAccessPartial && len(r.Messages) == 0 &&
				(r.MessageSelection == nil || r.MessageSelection.SourceCount == 0) {
				return fmt.Errorf("Chatwork partially restricted message result must retain at least one visible message")
			}
			if r.MessageAccess == MessageAccessAll && len(r.Messages) != 0 {
				return fmt.Errorf("Chatwork fully restricted message result must not contain visible messages")
			}
			if err := validateResultReference("message window room", r.MessageRoom, ReferenceRoom, false); err != nil {
				return err
			}
			if r.Coverage.Limit <= 0 {
				return fmt.Errorf("Chatwork message window requires a positive source limit")
			}
			if r.Coverage.Limit > MaxMessageSelectionCount {
				return fmt.Errorf("Chatwork message window exceeds the provider source-limit contract")
			}
			if len(r.Messages) > r.Coverage.Limit {
				return fmt.Errorf("Chatwork message window exceeds its declared source limit")
			}
		}
		for index, message := range r.Messages {
			if err := validateResultMessage(fmt.Sprintf("message[%d]", index), message); err != nil {
				return err
			}
			if r.Task == TaskMessagesList && message.Room != r.MessageRoom {
				return fmt.Errorf("Chatwork result message[%d] room does not match the message window", index)
			}
		}
		if r.Task == TaskMessagesList && r.MessageSelection != nil {
			if err := validateMessageSelection(*r.MessageSelection, r.Messages, r.Coverage); err != nil {
				return err
			}
		}
		if r.Task == TaskMessagesList && r.MessageReachability != nil {
			if err := validateMessageReachability(*r.MessageReachability, r.MessageSelection); err != nil {
				return err
			}
		}
		if r.Task == TaskMessagesList && r.MessageRelationResolution != nil {
			if err := validateMessageRelationResolution(*r.MessageRelationResolution, r.Messages, r.MessageRoom); err != nil {
				return err
			}
		}
	case TaskPersonalTasksList, TaskRoomTasksList, TaskRoomTasksShow:
		accountOptional := r.Task == TaskPersonalTasksList
		for index, task := range r.Tasks {
			if err := validateResultWorkTask(fmt.Sprintf("task[%d]", index), task, accountOptional); err != nil {
				return err
			}
		}
	case TaskFilesList, TaskFilesShow:
		for index, file := range r.Files {
			if err := validateResultFile(fmt.Sprintf("file[%d]", index), file); err != nil {
				return err
			}
		}
	case TaskInviteLinkShow, TaskInviteLinkCreate, TaskInviteLinkUpdate, TaskInviteLinkDelete:
		return validateResultReference("invite link", r.InviteLink.Ref, ReferenceInvite, false)
	case TaskContactRequestsList:
		for index, request := range r.Requests {
			if err := validateResultContactRequest(fmt.Sprintf("contact request[%d]", index), request); err != nil {
				return err
			}
		}
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
		return validateResultReference("parent room", r.CreatedInRoom.ParentRoom, ReferenceRoom, false)
	case TaskRoomTasksCreate:
		if err := validateRefs(r.CreatedInRoom.Refs, ReferenceTask, true); err != nil {
			return err
		}
		return validateResultReference("parent room", r.CreatedInRoom.ParentRoom, ReferenceRoom, false)
	case TaskFilesUpload:
		if err := validateRefs(r.CreatedInRoom.Refs, ReferenceFile, false); err != nil {
			return err
		}
		return validateResultReference("parent room", r.CreatedInRoom.ParentRoom, ReferenceRoom, false)
	case TaskMessagesMarkRead, TaskMessagesMarkUnread:
		if r.ReadState.Unread < 0 || r.ReadState.Mentions < 0 {
			return fmt.Errorf("Chatwork read-state counts must not be negative")
		}
	case TaskRoomsLeave, TaskRoomsDelete:
		if !r.Acknowledgement.Acknowledged {
			return fmt.Errorf("Chatwork room acknowledgement must be explicit")
		}
		return validateResultReference("acknowledged room", r.Acknowledgement.Target, ReferenceRoom, false)
	case TaskContactRequestsReject:
		if !r.Acknowledgement.Acknowledged {
			return fmt.Errorf("Chatwork contact-request acknowledgement must be explicit")
		}
		return validateResultReference("acknowledged contact request", r.Acknowledgement.Target, ReferenceRequest, false)
	case TaskMembersReplace:
		if r.MembershipCounts.Administrators < 0 || r.MembershipCounts.Members < 0 || r.MembershipCounts.Readonly < 0 {
			return fmt.Errorf("Chatwork membership counts must not be negative")
		}
	}
	return nil
}

func validateMessageSelection(selection MessageSelection, messages []Message, coverage Coverage) error {
	if !messageFilterActive(selection.Filter) {
		return fmt.Errorf("Chatwork message selection requires an active filter")
	}
	if err := validateMessageFilter(selection.Filter); err != nil {
		return fmt.Errorf("Chatwork message selection filter is invalid: %w", err)
	}
	if coverage.Limit <= 0 {
		return fmt.Errorf("Chatwork message selection requires a positive source limit")
	}
	if selection.SourceCount < len(messages) || selection.SourceCount > coverage.Limit {
		return fmt.Errorf("Chatwork message selection source count is outside its declared bound")
	}
	if selection.CandidateCount < 0 || selection.CandidateCount > selection.SourceCount {
		return fmt.Errorf("Chatwork message selection candidate count is outside its source window")
	}
	if len(selection.Filter.Senders) == 0 && selection.Filter.Period == (MessagePeriod{}) && selection.CandidateCount != selection.SourceCount {
		return fmt.Errorf("Chatwork message selection without primary predicates must consider every source message")
	}
	if selection.SourceSequences == nil || selection.AnchorSequences == nil {
		return fmt.Errorf("Chatwork message selection provenance must be explicit")
	}
	if len(selection.SourceSequences) != len(messages) {
		return fmt.Errorf("Chatwork message selection sequence count does not match displayed messages")
	}
	previous := 0
	sequenceSet := make(map[int]struct{}, len(selection.SourceSequences))
	for _, sequence := range selection.SourceSequences {
		if sequence <= previous || sequence > selection.SourceCount {
			return fmt.Errorf("Chatwork message source sequences must be strictly increasing within the source window")
		}
		previous = sequence
		sequenceSet[sequence] = struct{}{}
	}
	previous = 0
	anchorSet := make(map[int]struct{}, len(selection.AnchorSequences))
	for _, sequence := range selection.AnchorSequences {
		if sequence <= previous {
			return fmt.Errorf("Chatwork message anchor sequences must be strictly increasing")
		}
		if _, exists := sequenceSet[sequence]; !exists {
			return fmt.Errorf("Chatwork message anchor sequence is not displayed")
		}
		previous = sequence
		anchorSet[sequence] = struct{}{}
	}
	startOffset := 0
	if selection.Filter.StartIndex > 0 {
		startOffset = selection.Filter.StartIndex - 1
	}
	wantAnchors := selection.CandidateCount - startOffset
	if wantAnchors < 0 {
		wantAnchors = 0
	}
	if selection.Filter.Count > 0 && wantAnchors > selection.Filter.Count {
		wantAnchors = selection.Filter.Count
	}
	if len(selection.AnchorSequences) != wantAnchors {
		return fmt.Errorf("Chatwork message selection anchor count does not match its start index and requested count")
	}
	indexed := selection.Filter.StartIndex > 0 || selection.Filter.Count > 0
	if indexed && selection.ItemsPerPage != wantAnchors {
		return fmt.Errorf("Chatwork message selection items-per-page does not match its primary anchors")
	}
	if !indexed && selection.ItemsPerPage != 0 {
		return fmt.Errorf("Chatwork sender-only message selection must not declare page metadata")
	}
	wantNextStartIndex := 0
	if selection.Filter.Count > 0 && startOffset+wantAnchors < selection.CandidateCount {
		wantNextStartIndex = selection.Filter.StartIndex + wantAnchors
	}
	if selection.NextStartIndex != wantNextStartIndex {
		return fmt.Errorf("Chatwork message selection next start index does not match its remaining candidates")
	}
	if selection.Filter.Context == MessageContextNone && !equalSequences(selection.AnchorSequences, selection.SourceSequences) {
		return fmt.Errorf("Chatwork message selection without context must mark every displayed message as an anchor")
	}

	senderSet := make(map[Reference]struct{}, len(selection.Filter.Senders))
	for _, sender := range selection.Filter.Senders {
		senderSet[sender] = struct{}{}
	}
	anchorRefs := make(map[Reference]struct{}, len(anchorSet))
	anchorMessages := make([]Message, 0, len(anchorSet))
	for index, message := range messages {
		_, senderMatches := senderSet[message.Sender.Ref]
		_, markedAnchor := anchorSet[selection.SourceSequences[index]]
		if markedAnchor && len(senderSet) > 0 && !senderMatches {
			return fmt.Errorf("Chatwork message selection anchor does not match its sender filter")
		}
		if markedAnchor && !selection.Filter.Period.Contains(message.SendTime) {
			return fmt.Errorf("Chatwork message selection anchor does not match its period filter")
		}
		if markedAnchor {
			anchorRefs[message.Ref] = struct{}{}
			anchorMessages = append(anchorMessages, message)
		}
	}
	if selection.Filter.Context == MessageContextReplies {
		for index, message := range messages {
			if _, anchor := anchorSet[selection.SourceSequences[index]]; anchor {
				continue
			}
			direct := message.Reply != nil && message.Reply.Resolved
			if direct {
				_, direct = anchorRefs[message.Reply.Target]
			}
			if !direct {
				for _, anchor := range anchorMessages {
					if anchor.Reply != nil && anchor.Reply.Resolved && anchor.Reply.Target == message.Ref {
						direct = true
						break
					}
				}
			}
			if !direct {
				return fmt.Errorf("Chatwork message selection context is not a direct resolved reply neighbor of an anchor")
			}
		}
	}
	return nil
}

// DeriveMessageReachability computes only the lower-bound facts proved by a
// non-limited latest window. Differential, empty, access-limited, or invalid-
// time sources deliberately yield unknown period reachability.
func DeriveMessageReachability(coverage Coverage, access MessageAccessLimitation, source []Message, period MessagePeriod) (MessageReachability, error) {
	if err := validateMessagePeriod(period); err != nil {
		return MessageReachability{}, err
	}
	result := MessageReachability{}
	latest := coverage.Kind == "latest_window" || coverage.Kind == "recent-window"
	if latest && access == MessageAccessNone && len(source) > 0 {
		oldest := source[0]
		for _, message := range source[1:] {
			if message.SendTime < oldest.SendTime {
				oldest = message
			}
		}
		if oldest.SendTime > 0 && oldest.Ref.Kind == ReferenceMessage && ValidateReference(ReferenceMessage, oldest.Ref.Value) == nil {
			result.OldestMessage = oldest.Ref
			result.OldestSendTime = oldest.SendTime
		}
	}
	if period == (MessagePeriod{}) {
		return result, nil
	}
	if result.OldestMessage == (Reference{}) {
		result.PeriodReachability = MessagePeriodReachabilityUnknown
		return result, nil
	}
	result.PeriodReachability = classifyMessagePeriodReachability(period, result.OldestSendTime)
	return result, nil
}

func classifyMessagePeriodReachability(period MessagePeriod, oldest int64) MessagePeriodReachability {
	if period.Until > 0 && period.Until <= oldest {
		return MessagePeriodOutsideReachableWindow
	}
	if period.Since == 0 || period.Since < oldest {
		return MessagePeriodPartiallyOutsideReachable
	}
	return MessagePeriodWithinReachableWindow
}

func validateMessageReachability(reachability MessageReachability, selection *MessageSelection) error {
	oldestPresent := reachability.OldestMessage != (Reference{})
	if oldestPresent {
		if err := validateResultReference("oldest reachable message", reachability.OldestMessage, ReferenceMessage, false); err != nil {
			return err
		}
		if reachability.OldestSendTime <= 0 {
			return fmt.Errorf("Chatwork oldest reachable message requires a positive send time")
		}
	} else if reachability.OldestSendTime != 0 {
		return fmt.Errorf("Chatwork oldest reachable send time requires a message reference")
	}
	period := MessagePeriod{}
	if selection != nil {
		period = selection.Filter.Period
	}
	if period == (MessagePeriod{}) {
		if reachability.PeriodReachability != "" {
			return fmt.Errorf("Chatwork message reachability without a period must not classify the period")
		}
		return nil
	}
	if reachability.PeriodReachability == MessagePeriodReachabilityUnknown {
		if oldestPresent {
			return fmt.Errorf("Chatwork unknown period reachability must not claim an oldest reachable boundary")
		}
		return nil
	}
	if !oldestPresent {
		return fmt.Errorf("Chatwork classified period reachability requires an oldest reachable boundary")
	}
	if reachability.PeriodReachability != classifyMessagePeriodReachability(period, reachability.OldestSendTime) {
		return fmt.Errorf("Chatwork period reachability does not match its oldest reachable boundary")
	}
	return nil
}

func validateMessageRelationResolution(resolution MessageRelationResolution, messages []Message, room Reference) error {
	if resolution.FetchLimit < 1 || resolution.FetchLimit > MaxMessageRelationFetches {
		return fmt.Errorf("Chatwork message relation fetch limit is outside its declared bound")
	}
	if resolution.FetchAttempts < 0 || resolution.FetchAttempts > resolution.FetchLimit {
		return fmt.Errorf("Chatwork message relation fetch attempts exceed their declared limit")
	}
	if resolution.Targets == nil {
		return fmt.Errorf("Chatwork message relation resolution targets must be explicit")
	}
	displayed := make(map[Reference]struct{}, len(messages))
	available := make(map[Reference]struct{}, len(messages)+len(resolution.Targets))
	for _, message := range messages {
		displayed[message.Ref] = struct{}{}
		available[message.Ref] = struct{}{}
	}
	wanted := make([]Reference, 0)
	wantedSet := make(map[Reference]struct{}, len(messages)+len(resolution.Targets))
	for ref := range displayed {
		wantedSet[ref] = struct{}{}
	}
	appendWanted := func(message Message) {
		if message.Reply == nil || message.Reply.Kind != "reply" || message.Reply.ExternalID != room.Value {
			return
		}
		if _, seen := wantedSet[message.Reply.Target]; !seen {
			wantedSet[message.Reply.Target] = struct{}{}
			wanted = append(wanted, message.Reply.Target)
		}
	}
	for _, message := range messages {
		appendWanted(message)
	}
	attempts := 0
	seen := make(map[Reference]struct{}, len(resolution.Targets))
	for index, target := range resolution.Targets {
		if index >= len(wanted) {
			return fmt.Errorf("Chatwork message relation resolution contains a target not reached from displayed reply chains")
		}
		if target.Target != wanted[index] {
			return fmt.Errorf("Chatwork message relation targets do not preserve first-reference order")
		}
		if err := validateResultReference("message relation target", target.Target, ReferenceMessage, false); err != nil {
			return err
		}
		if _, duplicate := seen[target.Target]; duplicate {
			return fmt.Errorf("Chatwork message relation resolution contains a duplicate target")
		}
		seen[target.Target] = struct{}{}
		resolved := target.State == MessageRelationResolvedFromSource || target.State == MessageRelationResolvedByFetch
		switch target.State {
		case MessageRelationResolvedFromSource:
		case MessageRelationResolvedByFetch, MessageRelationNotFound, MessageRelationRestricted:
			attempts++
		case MessageRelationBudgetExhausted:
			if resolution.FetchAttempts != resolution.FetchLimit {
				return fmt.Errorf("Chatwork relation budget exhaustion requires every fetch slot to be consumed")
			}
		default:
			return fmt.Errorf("Chatwork message relation resolution state is invalid")
		}
		if resolved {
			if target.Message == nil {
				return fmt.Errorf("Chatwork resolved message relation target requires context")
			}
			if target.Message.Ref != target.Target || target.Message.Room != room {
				return fmt.Errorf("Chatwork resolved message relation context does not match its target and room")
			}
			if _, present := displayed[target.Message.Ref]; present {
				return fmt.Errorf("Chatwork supplemental relation context duplicates a displayed source message")
			}
			if err := validateResultMessage("message relation context", *target.Message); err != nil {
				return err
			}
			available[target.Message.Ref] = struct{}{}
			appendWanted(*target.Message)
		} else if target.Message != nil {
			return fmt.Errorf("Chatwork unresolved message relation target must not carry context")
		}
	}
	if len(wanted) != len(resolution.Targets) {
		return fmt.Errorf("Chatwork message relation resolution does not cover each unique reachable reply target")
	}
	if attempts != resolution.FetchAttempts {
		return fmt.Errorf("Chatwork message relation fetch attempt count does not match target outcomes")
	}
	checkReplyState := func(message Message) error {
		if message.Reply == nil || message.Reply.Kind != "reply" || message.Reply.ExternalID != room.Value {
			return nil
		}
		_, resolved := available[message.Reply.Target]
		if message.Reply.Resolved != resolved {
			return fmt.Errorf("Chatwork reply state does not match relation resolution evidence")
		}
		return nil
	}
	for _, message := range messages {
		if err := checkReplyState(message); err != nil {
			return err
		}
	}
	for _, target := range resolution.Targets {
		if target.Message != nil {
			if err := checkReplyState(*target.Message); err != nil {
				return err
			}
		}
	}
	return nil
}

func equalSequences(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalMessageFilters(left, right MessageFilter) bool {
	if left.Period != right.Period || left.Context != right.Context || left.StartIndex != right.StartIndex || left.Count != right.Count || len(left.Senders) != len(right.Senders) {
		return false
	}
	for index := range left.Senders {
		if left.Senders[index] != right.Senders[index] {
			return false
		}
	}
	return true
}

func validateResultReference(field string, ref Reference, kind ReferenceKind, optional bool) error {
	if optional && ref == (Reference{}) {
		return nil
	}
	if ref.Kind != kind {
		return fmt.Errorf("Chatwork result %s reference kind is %s, want %s", field, ref.Kind, kind)
	}
	if err := ValidateReference(ref.Kind, ref.Value); err != nil {
		return fmt.Errorf("Chatwork result %s reference is invalid: %w", field, err)
	}
	return nil
}

func validateResultRoom(field string, room Room) error {
	return validateResultReference(field, room.Ref, ReferenceRoom, false)
}

func validateResultAccount(field string, account Account, optional bool) error {
	if err := validateResultReference(field, account.Ref, ReferenceAccount, optional); err != nil {
		return err
	}
	return validateResultReference(field+" room", account.Room, ReferenceRoom, true)
}

func validateResultMessage(field string, message Message) error {
	if err := validateResultReference(field, message.Ref, ReferenceMessage, false); err != nil {
		return err
	}
	if err := validateResultReference(field+" room", message.Room, ReferenceRoom, false); err != nil {
		return err
	}
	if err := validateResultAccount(field+" sender", message.Sender, false); err != nil {
		return err
	}
	if message.RelationState != MessageRelationsComplete && message.RelationState != MessageRelationsUnknown {
		return fmt.Errorf("Chatwork result %s relation state is invalid", field)
	}
	if message.RelationState == MessageRelationsUnknown &&
		(len(message.Recipients) != 0 || message.Reply != nil || len(message.Quotes) != 0) {
		return fmt.Errorf("Chatwork result %s has partial facts for an unknown relation set", field)
	}
	for index, recipient := range message.Recipients {
		if err := validateResultReference(fmt.Sprintf("%s recipient[%d]", field, index), recipient, ReferenceAccount, false); err != nil {
			return err
		}
	}
	if message.Reply != nil {
		if message.Reply.Kind != "reply" {
			return fmt.Errorf("Chatwork result %s reply relation kind is %q, want %q", field, message.Reply.Kind, "reply")
		}
		if err := validateResultReference(field+" reply target", message.Reply.Target, ReferenceMessage, !message.Reply.Resolved); err != nil {
			return err
		}
	}
	for index, quote := range message.Quotes {
		if quote.Kind != "quote" {
			return fmt.Errorf("Chatwork result %s quote[%d] relation kind is %q, want %q", field, index, quote.Kind, "quote")
		}
		if err := validateResultReference(fmt.Sprintf("%s quote[%d] target", field, index), quote.Target, ReferenceAccount, !quote.Resolved); err != nil {
			return err
		}
	}
	return nil
}

func validateResultWorkTask(field string, task WorkTask, accountOptional bool) error {
	if err := validateResultReference(field, task.Ref, ReferenceTask, false); err != nil {
		return err
	}
	if err := validateResultRoom(field+" room", task.Room); err != nil {
		return err
	}
	if err := validateResultAccount(field+" account", task.Account, accountOptional); err != nil {
		return err
	}
	if err := validateResultAccount(field+" assigned by", task.AssignedBy, false); err != nil {
		return err
	}
	return validateResultReference(field+" message", task.Message, ReferenceMessage, false)
}

func validateResultFile(field string, file File) error {
	if err := validateResultReference(field, file.Ref, ReferenceFile, false); err != nil {
		return err
	}
	if err := validateResultReference(field+" room", file.Room, ReferenceRoom, false); err != nil {
		return err
	}
	if err := validateResultAccount(field+" account", file.Account, false); err != nil {
		return err
	}
	return validateResultReference(field+" message", file.Message, ReferenceMessage, true)
}

func validateResultContactRequest(field string, request ContactRequest) error {
	if err := validateResultReference(field, request.Ref, ReferenceRequest, false); err != nil {
		return err
	}
	return validateResultAccount(field+" account", request.Account, false)
}
