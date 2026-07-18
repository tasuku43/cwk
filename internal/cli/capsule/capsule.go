// Package capsule renders the candidate-P task-oriented projection.
//
// The projection is deliberately presentation-only. It selects a fixed set of
// fields for each typed task result, preserves provider order, and never
// derives relationships from external text.
package capsule

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// Render returns the deterministic task projection of result.
func Render(result chatwork.Result) (string, error) {
	if err := result.Validate(); err != nil {
		return "", fmt.Errorf("task projection result: %w", err)
	}
	if err := validateReferences(result); err != nil {
		return "", err
	}
	if err := validateExternalText(result); err != nil {
		return "", err
	}

	var output strings.Builder
	switch result.Task {
	case chatwork.TaskAccountShow:
		renderOwnAccount(&output, *result.Account)
	case chatwork.TaskAccountStatus:
		renderStatus(&output, *result.Status)
	case chatwork.TaskPersonalTasksList:
		renderPersonalTasks(&output, result.Tasks, result.Coverage)
	case chatwork.TaskContactsList:
		renderContacts(&output, result.Accounts, result.Coverage)
	case chatwork.TaskRoomsList:
		renderRooms(&output, result.Rooms, result.Coverage)
	case chatwork.TaskRoomsShow:
		renderRoom(&output, result.Rooms[0])
	case chatwork.TaskRoomsCreate:
		line(&output, "created room-ref=%s", ref(result.Created[0]))
	case chatwork.TaskRoomsUpdate:
		line(&output, "updated room-ref=%s", ref(result.Affected[0]))
	case chatwork.TaskRoomsLeave:
		line(&output, "left room-ref=%s", ref(result.Acknowledgement.Target))
	case chatwork.TaskRoomsDelete:
		line(&output, "deleted room-ref=%s", ref(result.Acknowledgement.Target))
	case chatwork.TaskMembersList:
		renderMembers(&output, result.Accounts, result.Coverage)
	case chatwork.TaskMembersReplace:
		renderMembershipCounts(&output, *result.MembershipCounts)
	case chatwork.TaskMessagesList:
		window, err := messageWindow(result.Coverage.Kind)
		if err != nil {
			return "", err
		}
		renderMessages(&output, result.Messages, result.Coverage, window)
	case chatwork.TaskMessagesShow:
		renderMessage(&output, result.Messages[0])
	case chatwork.TaskMessagesSend:
		line(&output, "created message-ref=%s room-ref=%s", ref(result.CreatedInRoom.Refs[0]), ref(result.CreatedInRoom.ParentRoom))
	case chatwork.TaskMessagesMarkRead:
		line(&output, "marked-read unread=%d mentions=%d", result.ReadState.Unread, result.ReadState.Mentions)
	case chatwork.TaskMessagesMarkUnread:
		line(&output, "marked-unread unread=%d mentions=%d", result.ReadState.Unread, result.ReadState.Mentions)
	case chatwork.TaskMessagesUpdate:
		line(&output, "updated message-ref=%s", ref(result.Affected[0]))
	case chatwork.TaskMessagesDelete:
		line(&output, "deleted message-ref=%s", ref(result.Affected[0]))
	case chatwork.TaskRoomTasksList:
		renderRoomTasks(&output, result.Tasks, result.Coverage)
	case chatwork.TaskRoomTasksShow:
		renderRoomTask(&output, result.Tasks[0])
	case chatwork.TaskRoomTasksCreate:
		renderCreatedTasks(&output, *result.CreatedInRoom)
	case chatwork.TaskRoomTasksSetStatus:
		line(&output, "updated task-ref=%s", ref(result.Affected[0]))
	case chatwork.TaskFilesList:
		renderFiles(&output, result.Files, result.Coverage)
	case chatwork.TaskFilesShow:
		renderFile(&output, result.Files[0], true)
	case chatwork.TaskFilesUpload:
		line(&output, "created file-ref=%s room-ref=%s", ref(result.CreatedInRoom.Refs[0]), ref(result.CreatedInRoom.ParentRoom))
	case chatwork.TaskInviteLinkShow:
		renderInviteLink(&output, "invite-link", *result.InviteLink)
	case chatwork.TaskInviteLinkCreate:
		renderInviteLink(&output, "created invite-link", *result.InviteLink)
	case chatwork.TaskInviteLinkUpdate:
		renderInviteLink(&output, "updated invite-link", *result.InviteLink)
	case chatwork.TaskInviteLinkDelete:
		renderInviteLink(&output, "deleted invite-link", *result.InviteLink)
	case chatwork.TaskContactRequestsList:
		renderContactRequests(&output, result.Requests, result.Coverage)
	case chatwork.TaskContactRequestsAccept:
		line(&output, "accepted account-ref=%s room-ref=%s", ref(result.Account.Ref), ref(result.Account.Room))
	case chatwork.TaskContactRequestsReject:
		line(&output, "rejected request-ref=%s", ref(result.Acknowledgement.Target))
	default:
		return "", fmt.Errorf("task projection has no route for %s", result.Task)
	}
	return output.String(), nil
}

func renderCollectionHeader(output *strings.Builder, noun string, count int, coverage chatwork.Coverage) {
	fmt.Fprintf(output, "%s count=%d", noun, count)
	if coverage.Limit > 0 {
		fmt.Fprintf(output, " limit=%d", coverage.Limit)
	}
	fmt.Fprintf(output, " complete=%t\n", coverage.Complete)
}

func renderOwnAccount(output *strings.Builder, account chatwork.Account) {
	fmt.Fprintf(output, "account account-ref=%s name=untrusted:%s", ref(account.Ref), quoted(account.Name))
	renderOrganization(output, account)
	output.WriteByte('\n')
}

func renderStatus(output *strings.Builder, status chatwork.Status) {
	line(output, "status unread=%d mentions=%d tasks=%d", status.Unread, status.Mentions, status.Tasks)
}

func renderContacts(output *strings.Builder, accounts []chatwork.Account, coverage chatwork.Coverage) {
	renderCollectionHeader(output, "contacts", len(accounts), coverage)
	for _, account := range accounts {
		fmt.Fprintf(output, "  account-ref=%s room-ref=%s name=untrusted:%s", ref(account.Ref), ref(account.Room), quoted(account.Name))
		renderOrganization(output, account)
		output.WriteByte('\n')
	}
}

func renderMembers(output *strings.Builder, accounts []chatwork.Account, coverage chatwork.Coverage) {
	renderCollectionHeader(output, "members", len(accounts), coverage)
	for _, account := range accounts {
		line(output, "  account-ref=%s name=untrusted:%s role=%s",
			ref(account.Ref), quoted(account.Name), atom(account.Role))
	}
}

func renderRooms(output *strings.Builder, rooms []chatwork.Room, coverage chatwork.Coverage) {
	renderCollectionHeader(output, "rooms", len(rooms), coverage)
	for _, room := range rooms {
		renderRoomLine(output, "  ", room)
	}
}

func renderRoom(output *strings.Builder, room chatwork.Room) {
	renderRoomLine(output, "room ", room)
}

func renderRoomLine(output *strings.Builder, prefix string, room chatwork.Room) {
	line(output, "%sroom-ref=%s name=untrusted:%s type=%s role=%s unread=%d mentions=%d tasks=%d",
		prefix, ref(room.Ref), quoted(room.Name), atom(room.Type), atom(room.Role), room.Unread, room.Mentions, room.Tasks)
}

func renderMessages(output *strings.Builder, messages []chatwork.Message, coverage chatwork.Coverage, window string) {
	fmt.Fprintf(output, "messages count=%d window=%s", len(messages), window)
	if coverage.Limit > 0 {
		fmt.Fprintf(output, " limit=%d", coverage.Limit)
	}
	fmt.Fprintf(output, " complete=%t unresolved-relations=%d\n", coverage.Complete, countUnresolved(messages))
	for _, message := range messages {
		renderMessageLine(output, "  ", message)
	}
}

func renderMessage(output *strings.Builder, message chatwork.Message) {
	renderMessageLine(output, "message ", message)
}

func renderMessageLine(output *strings.Builder, prefix string, message chatwork.Message) {
	line(output, "%smessage-ref=%s room-ref=%s sender-ref=%s sender-name=untrusted:%s send-time=%d relations=%s body=untrusted:%s",
		prefix, ref(message.Ref), ref(message.Room), ref(message.Sender.Ref), quoted(message.Sender.Name),
		message.SendTime, relations(message), quoted(message.Body))
}

func messageWindow(kind string) (string, error) {
	switch kind {
	case "latest_window", "recent-window":
		return "recent", nil
	case "differential_window", "changes-window":
		return "changes", nil
	default:
		return "", fmt.Errorf("task projection unknown message window %q", kind)
	}
}

func relations(message chatwork.Message) string {
	values := make([]string, 0, len(message.Recipients)+1+len(message.Quotes))
	for _, recipient := range message.Recipients {
		values = append(values, fmt.Sprintf("to{target-ref=%s}", ref(recipient)))
	}
	if message.Reply != nil {
		values = append(values, relation("reply", *message.Reply))
	}
	for _, quote := range message.Quotes {
		values = append(values, relation("quote", quote))
	}
	if len(values) == 0 {
		return "none"
	}
	return "[" + strings.Join(values, ",") + "]"
}

func relation(kind string, value chatwork.Relation) string {
	state := "unresolved"
	if value.Resolved {
		state = "resolved"
	}
	return fmt.Sprintf("%s{state=%s,target-ref=%s}", kind, state, ref(value.Target))
}

func renderPersonalTasks(output *strings.Builder, tasks []chatwork.WorkTask, coverage chatwork.Coverage) {
	renderCollectionHeader(output, "personal-tasks", len(tasks), coverage)
	for _, task := range tasks {
		line(output, "  task-ref=%s room-ref=%s assigned-by-ref=%s message-ref=%s body=untrusted:%s status=%s",
			ref(task.Ref), ref(task.Room.Ref), ref(task.AssignedBy.Ref), ref(task.Message), quoted(task.Body), atom(task.Status))
	}
}

func renderRoomTasks(output *strings.Builder, tasks []chatwork.WorkTask, coverage chatwork.Coverage) {
	renderCollectionHeader(output, "room-tasks", len(tasks), coverage)
	for _, task := range tasks {
		renderRoomTaskLine(output, "  ", task)
	}
}

func renderRoomTask(output *strings.Builder, task chatwork.WorkTask) {
	renderRoomTaskLine(output, "room-task ", task)
}

func renderRoomTaskLine(output *strings.Builder, prefix string, task chatwork.WorkTask) {
	line(output, "%stask-ref=%s room-ref=%s account-ref=%s message-ref=%s body=untrusted:%s status=%s limit-time=%d",
		prefix, ref(task.Ref), ref(task.Room.Ref), ref(task.Account.Ref), ref(task.Message), quoted(task.Body), atom(task.Status), task.LimitTime)
}

func renderCreatedTasks(output *strings.Builder, creation chatwork.RoomScopedCreation) {
	line(output, "created-tasks count=%d room-ref=%s", len(creation.Refs), ref(creation.ParentRoom))
	for _, task := range creation.Refs {
		line(output, "  task-ref=%s", ref(task))
	}
}

func renderFiles(output *strings.Builder, files []chatwork.File, coverage chatwork.Coverage) {
	renderCollectionHeader(output, "files", len(files), coverage)
	for _, file := range files {
		renderFile(output, file, false)
	}
}

func renderFile(output *strings.Builder, file chatwork.File, show bool) {
	prefix := "  "
	if show {
		prefix = "file "
	}
	fmt.Fprintf(output, "%sfile-ref=%s room-ref=%s account-ref=%s message-ref=%s name=untrusted:%s size=%d",
		prefix, ref(file.Ref), ref(file.Room), ref(file.Account.Ref), ref(file.Message), quoted(file.Name), file.Size)
	if show && file.DownloadURL != "" {
		fmt.Fprintf(output, " download-url=untrusted:%s", quoted(file.DownloadURL))
	}
	output.WriteByte('\n')
}

func renderInviteLink(output *strings.Builder, label string, invite chatwork.InviteLink) {
	fmt.Fprintf(output, "%s invite-ref=%s public=%t", label, ref(invite.Ref), invite.Public)
	if invite.Public {
		if invite.URL != "" {
			fmt.Fprintf(output, " url=untrusted:%s", quoted(invite.URL))
		}
		fmt.Fprintf(output, " needs-approval=%t", invite.NeedsApproval)
		if invite.Description != "" {
			fmt.Fprintf(output, " description=untrusted:%s", quoted(invite.Description))
		}
	}
	output.WriteByte('\n')
}

func renderContactRequests(output *strings.Builder, requests []chatwork.ContactRequest, coverage chatwork.Coverage) {
	renderCollectionHeader(output, "contact-requests", len(requests), coverage)
	for _, request := range requests {
		fmt.Fprintf(output, "  request-ref=%s account-ref=%s name=untrusted:%s", ref(request.Ref), ref(request.Account.Ref), quoted(request.Account.Name))
		if request.Message != "" {
			fmt.Fprintf(output, " message=untrusted:%s", quoted(request.Message))
		}
		output.WriteByte('\n')
	}
}

func renderMembershipCounts(output *strings.Builder, counts chatwork.MembershipCounts) {
	line(output, "membership-counts administrators=%d members=%d readonly=%d",
		counts.Administrators, counts.Members, counts.Readonly)
}

func renderOrganization(output *strings.Builder, account chatwork.Account) {
	if account.OrganizationName == "" && account.Department == "" {
		return
	}
	output.WriteString(" organization={")
	separator := ""
	if account.OrganizationName != "" {
		fmt.Fprintf(output, "name=untrusted:%s", quoted(account.OrganizationName))
		separator = ","
	}
	if account.Department != "" {
		fmt.Fprintf(output, "%sdepartment=untrusted:%s", separator, quoted(account.Department))
	}
	output.WriteByte('}')
}

func ref(value chatwork.Reference) string {
	if value.Kind == "" && value.Value == "" {
		return "absent"
	}
	return value.Value
}

func countUnresolved(messages []chatwork.Message) int {
	count := 0
	for _, message := range messages {
		if message.Reply != nil && !message.Reply.Resolved {
			count++
		}
		for _, quote := range message.Quotes {
			if !quote.Resolved {
				count++
			}
		}
	}
	return count
}

func validateReferences(result chatwork.Result) error {
	add := func(value chatwork.Reference) error {
		if value.Kind == "" && value.Value == "" {
			return nil
		}
		if err := chatwork.ValidateReference(value.Kind, value.Value); err != nil {
			return fmt.Errorf("task projection reference: %w", err)
		}
		return nil
	}
	addAccount := func(account chatwork.Account) error {
		if err := add(account.Ref); err != nil {
			return err
		}
		return add(account.Room)
	}

	if result.Account != nil {
		if err := addAccount(*result.Account); err != nil {
			return err
		}
	}
	for _, room := range result.Rooms {
		if err := add(room.Ref); err != nil {
			return err
		}
	}
	for _, account := range result.Accounts {
		if err := addAccount(account); err != nil {
			return err
		}
	}
	for _, message := range result.Messages {
		for _, value := range []chatwork.Reference{message.Ref, message.Room, message.Sender.Ref, message.Sender.Room} {
			if err := add(value); err != nil {
				return err
			}
		}
		for _, value := range message.Recipients {
			if err := add(value); err != nil {
				return err
			}
		}
		if message.Reply != nil {
			if err := add(message.Reply.Target); err != nil {
				return err
			}
		}
		for _, quote := range message.Quotes {
			if err := add(quote.Target); err != nil {
				return err
			}
		}
	}
	for _, task := range result.Tasks {
		for _, value := range []chatwork.Reference{task.Ref, task.Room.Ref, task.Account.Ref, task.Account.Room,
			task.AssignedBy.Ref, task.AssignedBy.Room, task.Message} {
			if err := add(value); err != nil {
				return err
			}
		}
	}
	for _, file := range result.Files {
		for _, value := range []chatwork.Reference{file.Ref, file.Room, file.Account.Ref, file.Account.Room, file.Message} {
			if err := add(value); err != nil {
				return err
			}
		}
	}
	if result.InviteLink != nil {
		if err := add(result.InviteLink.Ref); err != nil {
			return err
		}
	}
	for _, request := range result.Requests {
		for _, value := range []chatwork.Reference{request.Ref, request.Account.Ref, request.Account.Room} {
			if err := add(value); err != nil {
				return err
			}
		}
	}
	for _, values := range [][]chatwork.Reference{result.Created, result.Affected} {
		for _, value := range values {
			if err := add(value); err != nil {
				return err
			}
		}
	}
	if result.CreatedInRoom != nil {
		if err := add(result.CreatedInRoom.ParentRoom); err != nil {
			return err
		}
		for _, value := range result.CreatedInRoom.Refs {
			if err := add(value); err != nil {
				return err
			}
		}
	}
	if result.Acknowledgement != nil {
		if err := add(result.Acknowledgement.Target); err != nil {
			return err
		}
	}
	return nil
}

func validateExternalText(result chatwork.Result) error {
	values := []string{result.Coverage.Kind, result.Coverage.Description}
	addAccount := func(account chatwork.Account) {
		values = append(values, account.Name, account.ChatworkID, account.OrganizationID, account.OrganizationName,
			account.Department, account.Title, account.URL, account.Introduction, account.Mail, account.Telephone,
			account.Extension, account.Mobile, account.Skype, account.Facebook, account.Twitter, account.AvatarURL,
			account.LoginMail, account.Role)
	}
	if result.Account != nil {
		addAccount(*result.Account)
	}
	for _, room := range result.Rooms {
		values = append(values, room.Name, room.Type, room.Role, room.IconURL, room.Description)
	}
	for _, account := range result.Accounts {
		addAccount(account)
	}
	for _, message := range result.Messages {
		addAccount(message.Sender)
		values = append(values, message.Body)
		if message.Reply != nil {
			values = append(values, message.Reply.Kind, message.Reply.ExternalID)
		}
		for _, quote := range message.Quotes {
			values = append(values, quote.Kind, quote.ExternalID)
		}
	}
	for _, task := range result.Tasks {
		addAccount(task.Account)
		addAccount(task.AssignedBy)
		values = append(values, task.Room.Name, task.Room.Type, task.Room.Role, task.Room.IconURL,
			task.Room.Description, task.Body, task.Status, task.LimitType)
	}
	for _, file := range result.Files {
		addAccount(file.Account)
		values = append(values, file.Name, file.DownloadURL)
	}
	if result.InviteLink != nil {
		values = append(values, result.InviteLink.URL, result.InviteLink.Description)
	}
	for _, request := range result.Requests {
		addAccount(request.Account)
		values = append(values, request.Message)
	}
	for _, value := range values {
		if !utf8.ValidString(value) {
			return fmt.Errorf("task projection external text must be valid UTF-8")
		}
	}
	return nil
}

func atom(value string) string {
	return quoted(value)
}

func quoted(value string) string {
	return strconv.Quote(safeExternalText(value))
}

// safeExternalText mirrors the CLI visible projection. Backslashes are
// escaped before controls, formats, and Unicode line separators become visible
// ASCII escape sequences.
func safeExternalText(value string) string {
	var output strings.Builder
	for _, r := range value {
		if r == '\\' {
			output.WriteString("\\\\")
			continue
		}
		if r == '\u2028' || r == '\u2029' {
			fmt.Fprintf(&output, "\\u%04X", r)
			continue
		}
		if unicode.Is(unicode.C, r) {
			switch r {
			case '\t':
				output.WriteString("\\t")
			case '\r':
				output.WriteString("\\r")
			case '\n':
				output.WriteString("\\n")
			default:
				if r <= 0xffff {
					fmt.Fprintf(&output, "\\u%04X", r)
				} else {
					fmt.Fprintf(&output, "\\U%08X", r)
				}
			}
			continue
		}
		output.WriteRune(r)
	}
	return output.String()
}

func line(output *strings.Builder, format string, args ...any) {
	fmt.Fprintf(output, format, args...)
	output.WriteByte('\n')
}
