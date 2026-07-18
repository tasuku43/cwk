// Package capsule renders the candidate-C agent context capsule.
//
// It is deliberately a presentation-only package: callers provide a complete
// provider-independent chatwork.Result, and the renderer neither fetches
// missing context nor infers relations from text, names, or ordering.
package capsule

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

const Schema = "cwk-context-capsule/1"

// Render returns the deterministic candidate-C projection of result.
func Render(result chatwork.Result) (string, error) {
	if err := result.Validate(); err != nil {
		return "", fmt.Errorf("context capsule result: %w", err)
	}

	references, err := collectReferences(result)
	if err != nil {
		return "", err
	}
	aliases := newAliases(references)
	if err := validateExternalText(result); err != nil {
		return "", err
	}

	var output strings.Builder
	line(&output, Schema)
	line(&output, "task %s", result.Task)
	line(&output, "alias-policy display-only; command-input=canonical-reference")
	line(&output, "coverage kind=%s limit=%d complete=%t unresolved-relations=%d description=%s",
		atom(result.Coverage.Kind), result.Coverage.Limit, result.Coverage.Complete,
		countUnresolved(result.Messages), authored(result.Coverage.Description))

	line(&output, "refs %d", len(references))
	for _, ref := range references {
		line(&output, "  %s kind=%s canonical=%s", aliases.forRef(ref), ref.Kind, ref.Value)
	}

	line(&output, "result")
	renderResult(&output, result, aliases)
	return output.String(), nil
}

func renderResult(output *strings.Builder, result chatwork.Result, aliases aliasTable) {
	if result.Account != nil {
		line(output, "  account")
		renderAccount(output, "    ", *result.Account, aliases)
	}
	if result.Status != nil {
		status := result.Status
		line(output, "  status unread-rooms=%d mention-rooms=%d task-rooms=%d unread=%d mentions=%d tasks=%d",
			status.UnreadRooms, status.MentionRooms, status.TaskRooms, status.Unread, status.Mentions, status.Tasks)
	}
	if result.Rooms != nil {
		line(output, "  rooms %d", len(result.Rooms))
		for _, room := range result.Rooms {
			renderRoom(output, "    ", room, aliases)
		}
	}
	if result.Accounts != nil {
		line(output, "  accounts %d", len(result.Accounts))
		for _, account := range result.Accounts {
			renderAccount(output, "    ", account, aliases)
		}
	}
	if result.Messages != nil {
		line(output, "  messages %d", len(result.Messages))
		for _, message := range result.Messages {
			renderMessage(output, "    ", message, aliases)
		}
	}
	if result.Tasks != nil {
		line(output, "  tasks %d", len(result.Tasks))
		for _, task := range result.Tasks {
			line(output, "    %s room=%s account=%s assigned-by=%s message=%s limit-time=%d status=%s limit-type=%s",
				aliases.forRef(task.Ref), aliases.forRef(task.Room.Ref), aliases.forRef(task.Account.Ref),
				aliases.forRef(task.AssignedBy.Ref), aliases.forRef(task.Message), task.LimitTime,
				atom(task.Status), atom(task.LimitType))
			line(output, "      body untrusted=%s", external(task.Body))
		}
	}
	if result.Files != nil {
		line(output, "  files %d", len(result.Files))
		for _, file := range result.Files {
			line(output, "    %s room=%s account=%s message=%s size=%d uploaded=%d",
				aliases.forRef(file.Ref), aliases.forRef(file.Room), aliases.forRef(file.Account.Ref),
				aliases.forRef(file.Message), file.Size, file.UploadTime)
			line(output, "      name untrusted=%s", external(file.Name))
			line(output, "      download-url untrusted=%s", external(file.DownloadURL))
		}
	}
	if result.InviteLink != nil {
		link := result.InviteLink
		line(output, "  invite-link %s public=%t needs-approval=%t", aliases.forRef(link.Ref), link.Public, link.NeedsApproval)
		line(output, "    url untrusted=%s", external(link.URL))
		line(output, "    description untrusted=%s", external(link.Description))
	}
	if result.Requests != nil {
		line(output, "  contact-requests %d", len(result.Requests))
		for _, request := range result.Requests {
			line(output, "    %s account=%s", aliases.forRef(request.Ref), aliases.forRef(request.Account.Ref))
			line(output, "      name untrusted=%s", external(request.Account.Name))
			line(output, "      message untrusted=%s", external(request.Message))
		}
	}
	if result.Created != nil {
		line(output, "  created %s", refList(result.Created, aliases))
	}
	if result.Affected != nil {
		line(output, "  affected %s", refList(result.Affected, aliases))
	}
	if result.CreatedInRoom != nil {
		renderRoomScopedCreation(output, result, aliases)
	}
	if result.ReadState != nil {
		line(output, "  read-state unread=%d mentions=%d", result.ReadState.Unread, result.ReadState.Mentions)
	}
	if result.Acknowledgement != nil {
		line(output, "  acknowledgement acknowledged=%t target-ref=%s",
			result.Acknowledgement.Acknowledged, aliases.forRef(result.Acknowledgement.Target))
	}
	if result.MembershipCounts != nil {
		line(output, "  membership-counts administrators=%d members=%d readonly=%d",
			result.MembershipCounts.Administrators, result.MembershipCounts.Members, result.MembershipCounts.Readonly)
	}
}

func renderRoomScopedCreation(output *strings.Builder, result chatwork.Result, aliases aliasTable) {
	creation := result.CreatedInRoom
	switch result.Task {
	case chatwork.TaskMessagesSend:
		line(output, "  creation message-ref=%s room-ref=%s",
			aliases.forRef(creation.Refs[0]), aliases.forRef(creation.ParentRoom))
	case chatwork.TaskRoomTasksCreate:
		for _, ref := range creation.Refs {
			line(output, "  creation task-ref=%s room-ref=%s",
				aliases.forRef(ref), aliases.forRef(creation.ParentRoom))
		}
	case chatwork.TaskFilesUpload:
		line(output, "  creation file-ref=%s room-ref=%s",
			aliases.forRef(creation.Refs[0]), aliases.forRef(creation.ParentRoom))
	}
}

func renderMessage(output *strings.Builder, indent string, message chatwork.Message, aliases aliasTable) {
	line(output, "%s%s room=%s sender=%s sent=%d updated=%d", indent, aliases.forRef(message.Ref),
		aliases.forRef(message.Room), aliases.forRef(message.Sender.Ref), message.SendTime, message.UpdateTime)
	line(output, "%s  sender-name untrusted=%s", indent, external(message.Sender.Name))
	line(output, "%s  to %s", indent, refList(message.Recipients, aliases))
	if message.Reply == nil {
		line(output, "%s  reply absent", indent)
	} else {
		renderRelation(output, indent+"  ", "reply", *message.Reply, aliases)
	}
	line(output, "%s  quotes %d", indent, len(message.Quotes))
	for index, relation := range message.Quotes {
		renderRelation(output, indent+"    ", fmt.Sprintf("quote[%d]", index+1), relation, aliases)
	}
	line(output, "%s  body untrusted=%s", indent, external(message.Body))
}

func renderRelation(output *strings.Builder, indent, label string, relation chatwork.Relation, aliases aliasTable) {
	state := "unresolved"
	if relation.Resolved {
		state = "resolved"
	}
	target := "absent"
	if relation.Target.Value != "" {
		target = aliases.forRef(relation.Target)
	}
	line(output, "%s%s kind=%s state=%s target=%s external-id=untrusted:%s",
		indent, label, atom(relation.Kind), state, target, external(relation.ExternalID))
}

func renderAccount(output *strings.Builder, indent string, account chatwork.Account, aliases aliasTable) {
	line(output, "%s%s room=%s role=%s", indent, aliases.forRef(account.Ref), aliases.forRef(account.Room), atom(account.Role))
	for _, field := range []struct {
		name  string
		value string
	}{
		{"name", account.Name}, {"chatwork-id", account.ChatworkID},
		{"organization-id", account.OrganizationID}, {"organization-name", account.OrganizationName},
		{"department", account.Department}, {"title", account.Title}, {"url", account.URL},
		{"introduction", account.Introduction}, {"mail", account.Mail}, {"telephone", account.Telephone},
		{"extension", account.Extension}, {"mobile", account.Mobile}, {"skype", account.Skype},
		{"facebook", account.Facebook}, {"twitter", account.Twitter}, {"avatar-url", account.AvatarURL},
		{"login-mail", account.LoginMail},
	} {
		if field.value != "" {
			line(output, "%s  %s untrusted=%s", indent, field.name, external(field.value))
		}
	}
}

func renderRoom(output *strings.Builder, indent string, room chatwork.Room, aliases aliasTable) {
	line(output, "%s%s type=%s role=%s sticky=%t unread=%d mentions=%d my-tasks=%d messages=%d files=%d tasks=%d updated=%d",
		indent, aliases.forRef(room.Ref), atom(room.Type), atom(room.Role), room.Sticky, room.Unread,
		room.Mentions, room.MyTasks, room.Messages, room.Files, room.Tasks, room.LastUpdateTime)
	line(output, "%s  name untrusted=%s", indent, external(room.Name))
	line(output, "%s  description untrusted=%s", indent, external(room.Description))
	line(output, "%s  icon-url untrusted=%s", indent, external(room.IconURL))
}

type aliasTable map[string]string

func newAliases(references []chatwork.Reference) aliasTable {
	table := make(aliasTable, len(references))
	counts := make(map[chatwork.ReferenceKind]int)
	for _, ref := range references {
		counts[ref.Kind]++
		table[refKey(ref)] = aliasPrefix(ref.Kind) + strconv.Itoa(counts[ref.Kind])
	}
	return table
}

func (a aliasTable) forRef(ref chatwork.Reference) string {
	if ref.Value == "" {
		return "absent"
	}
	return a[refKey(ref)]
}

func aliasPrefix(kind chatwork.ReferenceKind) string {
	switch kind {
	case chatwork.ReferenceAccount:
		return "a"
	case chatwork.ReferenceRoom:
		return "r"
	case chatwork.ReferenceMessage:
		return "m"
	case chatwork.ReferenceTask:
		return "t"
	case chatwork.ReferenceFile:
		return "f"
	case chatwork.ReferenceInvite:
		return "i"
	case chatwork.ReferenceRequest:
		return "q"
	default:
		panic("validated reference kind has no alias prefix")
	}
}

func collectReferences(result chatwork.Result) ([]chatwork.Reference, error) {
	unique := make(map[string]chatwork.Reference)
	add := func(ref chatwork.Reference) error {
		if ref.Kind == "" && ref.Value == "" {
			return nil
		}
		if err := chatwork.ValidateReference(ref.Kind, ref.Value); err != nil {
			return fmt.Errorf("context capsule reference: %w", err)
		}
		unique[refKey(ref)] = ref
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
			return nil, err
		}
	}
	for _, room := range result.Rooms {
		if err := add(room.Ref); err != nil {
			return nil, err
		}
	}
	for _, account := range result.Accounts {
		if err := addAccount(account); err != nil {
			return nil, err
		}
	}
	for _, message := range result.Messages {
		for _, ref := range []chatwork.Reference{message.Ref, message.Room, message.Sender.Ref, message.Sender.Room} {
			if err := add(ref); err != nil {
				return nil, err
			}
		}
		for _, ref := range message.Recipients {
			if err := add(ref); err != nil {
				return nil, err
			}
		}
		if message.Reply != nil {
			if err := add(message.Reply.Target); err != nil {
				return nil, err
			}
		}
		for _, quote := range message.Quotes {
			if err := add(quote.Target); err != nil {
				return nil, err
			}
		}
	}
	for _, task := range result.Tasks {
		for _, ref := range []chatwork.Reference{task.Ref, task.Room.Ref, task.Account.Ref, task.Account.Room,
			task.AssignedBy.Ref, task.AssignedBy.Room, task.Message} {
			if err := add(ref); err != nil {
				return nil, err
			}
		}
	}
	for _, file := range result.Files {
		for _, ref := range []chatwork.Reference{file.Ref, file.Room, file.Account.Ref, file.Account.Room, file.Message} {
			if err := add(ref); err != nil {
				return nil, err
			}
		}
	}
	if result.InviteLink != nil {
		if err := add(result.InviteLink.Ref); err != nil {
			return nil, err
		}
	}
	for _, request := range result.Requests {
		for _, ref := range []chatwork.Reference{request.Ref, request.Account.Ref, request.Account.Room} {
			if err := add(ref); err != nil {
				return nil, err
			}
		}
	}
	for _, refs := range [][]chatwork.Reference{result.Created, result.Affected} {
		for _, ref := range refs {
			if err := add(ref); err != nil {
				return nil, err
			}
		}
	}
	if result.CreatedInRoom != nil {
		if err := add(result.CreatedInRoom.ParentRoom); err != nil {
			return nil, err
		}
		for _, ref := range result.CreatedInRoom.Refs {
			if err := add(ref); err != nil {
				return nil, err
			}
		}
	}
	if result.Acknowledgement != nil {
		if err := add(result.Acknowledgement.Target); err != nil {
			return nil, err
		}
	}

	references := make([]chatwork.Reference, 0, len(unique))
	for _, ref := range unique {
		references = append(references, ref)
	}
	sort.Slice(references, func(i, j int) bool {
		left, right := references[i], references[j]
		if left.Kind != right.Kind {
			return referenceKindOrder(left.Kind) < referenceKindOrder(right.Kind)
		}
		if len(left.Value) != len(right.Value) {
			return len(left.Value) < len(right.Value)
		}
		return left.Value < right.Value
	})
	return references, nil
}

func referenceKindOrder(kind chatwork.ReferenceKind) int {
	switch kind {
	case chatwork.ReferenceAccount:
		return 0
	case chatwork.ReferenceRoom:
		return 1
	case chatwork.ReferenceMessage:
		return 2
	case chatwork.ReferenceTask:
		return 3
	case chatwork.ReferenceFile:
		return 4
	case chatwork.ReferenceInvite:
		return 5
	case chatwork.ReferenceRequest:
		return 6
	default:
		return 7
	}
}

func refKey(ref chatwork.Reference) string {
	return string(ref.Kind) + "\x00" + ref.Value
}

func refList(refs []chatwork.Reference, aliases aliasTable) string {
	values := make([]string, len(refs))
	for index, ref := range refs {
		values[index] = aliases.forRef(ref)
	}
	return "[" + strings.Join(values, ",") + "]"
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
			return fmt.Errorf("context capsule external text must be valid UTF-8")
		}
	}
	return nil
}

func atom(value string) string {
	return strconv.Quote(safeExternalText(value))
}

func authored(value string) string {
	return strconv.Quote(safeExternalText(value))
}

func external(value string) string {
	return strconv.Quote(safeExternalText(value))
}

// safeExternalText mirrors the CLI's visible projection without importing its
// parent package. Backslashes are escaped first, then controls, format runes,
// and Unicode line separators are rendered as visible ASCII escapes.
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
