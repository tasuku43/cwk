// Package capsule renders the selected task-oriented text projection.
//
// The projection is deliberately presentation-only. It selects a fixed set of
// fields for each typed task result. Homogeneous list routes declare one fixed
// schema and trust boundary before positional records. The messages.list route
// additionally uses document-local aliases and sequence links while preserving
// provider order; it never derives relationships from external text.
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
	case chatwork.TaskMembersFind:
		renderMemberCandidates(&output, result.Accounts, result.Coverage, *result.MemberSelection)
	case chatwork.TaskMembersList:
		renderMembers(&output, result.Accounts, result.Coverage)
	case chatwork.TaskMembersReplace:
		renderMembershipCounts(&output, *result.MembershipCounts)
	case chatwork.TaskMessagesList:
		window, err := messageWindow(result.Coverage.Kind)
		if err != nil {
			return "", err
		}
		if err := renderMessages(&output, result.MessageRoom, result.Messages, result.Coverage, result.MessageAccess, window, result.MessageSelection, result.MessageReachability, result.MessageRelationResolution); err != nil {
			return "", err
		}
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

func renderCollectionPrelude(output *strings.Builder, noun string, count int, coverage chatwork.Coverage, schema string) {
	renderCollectionHeader(output, noun, count, coverage)
	line(output, "external-text=untrusted escaped")
	line(output, "schema: %s", schema)
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
	renderCollectionPrelude(output, "contacts", len(accounts), coverage, `account-ref room-ref "name" [organization]`)
	for _, account := range accounts {
		fmt.Fprintf(output, "%s %s %s", ref(account.Ref), ref(account.Room), quoted(account.Name))
		renderCollectionOrganization(output, account)
		output.WriteByte('\n')
	}
}

func renderMembers(output *strings.Builder, accounts []chatwork.Account, coverage chatwork.Coverage) {
	renderCollectionPrelude(output, "members", len(accounts), coverage, `account-ref "name" role`)
	for _, account := range accounts {
		line(output, "%s %s %s",
			ref(account.Ref), quoted(account.Name), atom(account.Role))
	}
}

func renderMemberCandidates(output *strings.Builder, accounts []chatwork.Account, coverage chatwork.Coverage, selection chatwork.MemberSelection) {
	fmt.Fprintf(output, "member-candidates query=%s source-count=%d candidate-count=%d complete=%t\n",
		quoted(selection.Query), selection.SourceCount, len(accounts), coverage.Complete)
	line(output, "external-text=untrusted escaped")
	line(output, `schema: account-ref "name" role`)
	for _, account := range accounts {
		line(output, "%s %s %s", ref(account.Ref), quoted(account.Name), atom(account.Role))
	}
}

func renderRooms(output *strings.Builder, rooms []chatwork.Room, coverage chatwork.Coverage) {
	renderCollectionPrelude(output, "rooms", len(rooms), coverage, `room-ref "name" type role unread mentions tasks`)
	for _, room := range rooms {
		line(output, "%s %s %s %s %d %d %d",
			ref(room.Ref), quoted(room.Name), atom(room.Type), atom(room.Role), room.Unread, room.Mentions, room.Tasks)
	}
}

func renderRoom(output *strings.Builder, room chatwork.Room) {
	renderRoomLine(output, "room ", room)
}

func renderRoomLine(output *strings.Builder, prefix string, room chatwork.Room) {
	line(output, "%sroom-ref=%s name=untrusted:%s type=%s role=%s unread=%d mentions=%d tasks=%d",
		prefix, ref(room.Ref), quoted(room.Name), atom(room.Type), atom(room.Role), room.Unread, room.Mentions, room.Tasks)
}

func renderMessages(output *strings.Builder, room chatwork.Reference, messages []chatwork.Message, coverage chatwork.Coverage, access chatwork.MessageAccessLimitation, window string, selection *chatwork.MessageSelection, reachability *chatwork.MessageReachability, resolution *chatwork.MessageRelationResolution) error {
	actors, actorByRef, err := messageActors(messages)
	if err != nil {
		return err
	}
	sequences := make([]int, len(messages))
	if selection != nil {
		if len(selection.SourceSequences) != len(messages) {
			return fmt.Errorf("task projection message selection sequence count does not match the selected messages")
		}
		copy(sequences, selection.SourceSequences)
	} else {
		for index := range messages {
			sequences[index] = index + 1
		}
	}
	sequenceByRef := make(map[string]int, len(messages))
	resolvedContextByRef := make(map[string]struct{})
	if resolution != nil {
		for _, target := range resolution.Targets {
			if target.Message != nil {
				resolvedContextByRef[target.Message.Ref.Value] = struct{}{}
			}
		}
	}
	seenSequences := make(map[int]struct{}, len(messages))
	for index, message := range messages {
		if _, exists := sequenceByRef[message.Ref.Value]; exists {
			return fmt.Errorf("task projection message window contains a duplicate canonical message reference")
		}
		if sequences[index] <= 0 {
			return fmt.Errorf("task projection message selection contains an invalid source sequence")
		}
		if _, exists := seenSequences[sequences[index]]; exists {
			return fmt.Errorf("task projection message selection contains a duplicate source sequence")
		}
		seenSequences[sequences[index]] = struct{}{}
		sequenceByRef[message.Ref.Value] = sequences[index]
	}

	fmt.Fprintf(output, "messages room-ref=%s count=%d window=%s", ref(room), len(messages), window)
	if coverage.Limit > 0 {
		fmt.Fprintf(output, " source-limit=%d", coverage.Limit)
	}
	accessValue, err := messageAccess(access)
	if err != nil {
		return err
	}
	relationMessages := appendResolutionMessages(messages, resolution)
	fmt.Fprintf(output, " complete=%t access-limitation=%s unresolved-relations=%d unknown-relation-sets=%d",
		coverage.Complete, accessValue, countUnresolved(relationMessages), countUnknownRelations(relationMessages))
	if reachability != nil && reachability.OldestMessage != (chatwork.Reference{}) {
		fmt.Fprintf(output, " oldest-reachable-message-ref=%s oldest-reachable-send-time=%d", ref(reachability.OldestMessage), reachability.OldestSendTime)
	}
	if reachability != nil && reachability.PeriodReachability != "" {
		fmt.Fprintf(output, " period-reachability=%s", string(reachability.PeriodReachability))
	}
	output.WriteByte('\n')
	if selection != nil {
		fmt.Fprintf(output, "selection source-count=%d", selection.SourceCount)
		if selection.Filter.Period.Since > 0 {
			fmt.Fprintf(output, " since=%d", selection.Filter.Period.Since)
		}
		if selection.Filter.Period.Until > 0 {
			fmt.Fprintf(output, " until=%d", selection.Filter.Period.Until)
		}
		if selection.Filter.Period.Day != "" {
			fmt.Fprintf(output, " on=%s time-zone=%s", selection.Filter.Period.Day, selection.Filter.Period.TimeZone)
		}
		if selection.Filter.Period != (chatwork.MessagePeriod{}) || selection.Filter.StartIndex > 0 || selection.Filter.Count > 0 {
			fmt.Fprintf(output, " candidate-count=%d", selection.CandidateCount)
		}
		if selection.Filter.StartIndex > 0 || selection.Filter.Count > 0 {
			fmt.Fprintf(output, " start-index=%d", selection.Filter.StartIndex)
		}
		if selection.Filter.Count > 0 {
			fmt.Fprintf(output, " count=%d", selection.Filter.Count)
		}
		if selection.Filter.StartIndex > 0 || selection.Filter.Count > 0 {
			fmt.Fprintf(output, " items-per-page=%d", selection.ItemsPerPage)
		}
		if selection.NextStartIndex > 0 {
			fmt.Fprintf(output, " next-start-index=%d", selection.NextStartIndex)
		}
		if len(selection.Filter.Senders) > 0 {
			fmt.Fprintf(output, " senders=%s", bracketedReferences(selection.Filter.Senders))
		}
		if len(selection.Filter.Senders) > 0 || selection.Filter.Context == chatwork.MessageContextReplies {
			fmt.Fprintf(output, " context=%s", string(selection.Filter.Context))
		}
		fmt.Fprintf(output, " anchors=%s\n", bracketedSequences(selection.AnchorSequences))
	}
	if resolution != nil {
		line(output, "relation-resolution fetch-limit=%d fetch-attempts=%d targets=%d", resolution.FetchLimit, resolution.FetchAttempts, len(resolution.Targets))
	}
	line(output, "external-text=untrusted escaped")
	line(output, "schema: #sequence message-ref actor sent [reply] [to] [quote] [relation-state] \"body\"")
	line(output, "actors")
	for index, actor := range actors {
		line(output, "  a%d account-ref=%s name=%s", index+1, ref(actor.Ref), quoted(actor.Name))
	}
	for index, message := range messages {
		fmt.Fprintf(output, "#%d %s %s %d", sequences[index], ref(message.Ref), actorByRef[message.Sender.Ref.Value], message.SendTime)
		if len(message.Replies) > 0 {
			values := make([]string, 0, len(message.Replies))
			for _, reply := range message.Replies {
				value, relationErr := messageReply(reply, sequenceByRef, resolvedContextByRef)
				if relationErr != nil {
					return relationErr
				}
				values = append(values, value)
			}
			fmt.Fprintf(output, " reply=%s", compactValues(values))
		}
		if len(message.Recipients) > 0 {
			fmt.Fprintf(output, " to=%s", accountTargets(message.Recipients, actorByRef))
		}
		if len(message.Quotes) > 0 {
			values := make([]string, len(message.Quotes))
			for quoteIndex, quote := range message.Quotes {
				values[quoteIndex] = accountRelation(quote, actorByRef)
			}
			fmt.Fprintf(output, " quote=%s", compactValues(values))
		}
		if message.RelationState == chatwork.MessageRelationsUnknown {
			fmt.Fprintf(output, " relation-state=unknown")
		}
		line(output, " %s", quoted(message.Body))
	}
	if resolution != nil {
		for _, target := range resolution.Targets {
			if target.Message == nil {
				line(output, "relation-gap target-ref=%s state=%s", ref(target.Target), string(target.State))
				continue
			}
			renderMessageLine(output, fmt.Sprintf("relation-context provenance=%s ", string(target.State)), *target.Message)
		}
	}
	return nil
}

func appendResolutionMessages(messages []chatwork.Message, resolution *chatwork.MessageRelationResolution) []chatwork.Message {
	combined := append([]chatwork.Message(nil), messages...)
	if resolution == nil {
		return combined
	}
	for _, target := range resolution.Targets {
		if target.Message != nil {
			combined = append(combined, *target.Message)
		}
	}
	return combined
}

func bracketedReferences(values []chatwork.Reference) string {
	items := make([]string, len(values))
	for index, value := range values {
		items[index] = ref(value)
	}
	return "[" + strings.Join(items, ",") + "]"
}

func bracketedSequences(values []int) string {
	items := make([]string, len(values))
	for index, value := range values {
		items[index] = "#" + strconv.Itoa(value)
	}
	return "[" + strings.Join(items, ",") + "]"
}

func messageActors(messages []chatwork.Message) ([]chatwork.Account, map[string]string, error) {
	actors := make([]chatwork.Account, 0)
	byReference := make(map[string]string)
	names := make(map[string]string)
	for _, message := range messages {
		key := message.Sender.Ref.Value
		if name, exists := names[key]; exists {
			if name != message.Sender.Name {
				return nil, nil, fmt.Errorf("task projection message sender name is inconsistent for one canonical account reference")
			}
			continue
		}
		names[key] = message.Sender.Name
		actors = append(actors, message.Sender)
		byReference[key] = fmt.Sprintf("a%d", len(actors))
	}
	return actors, byReference, nil
}

func messageReply(value chatwork.Relation, sequenceByRef map[string]int, resolvedContextByRef map[string]struct{}) (string, error) {
	if !value.Resolved {
		if value.Target == (chatwork.Reference{}) {
			return "?", nil
		}
		return "?" + ref(value.Target), nil
	}
	sequence, found := sequenceByRef[value.Target.Value]
	if !found {
		if _, contextResolved := resolvedContextByRef[value.Target.Value]; contextResolved {
			return "message-ref:" + ref(value.Target), nil
		}
		return "", fmt.Errorf("task projection resolved reply target is outside the message window and relation context")
	}
	return fmt.Sprintf("#%d", sequence), nil
}

func accountTargets(values []chatwork.Reference, actorByRef map[string]string) string {
	targets := make([]string, len(values))
	for index, value := range values {
		targets[index] = accountTarget(value, actorByRef)
	}
	return compactValues(targets)
}

func accountTarget(value chatwork.Reference, actorByRef map[string]string) string {
	if alias, found := actorByRef[value.Value]; found {
		return alias
	}
	return "account-ref:" + ref(value)
}

func accountRelation(value chatwork.Relation, actorByRef map[string]string) string {
	if value.Target == (chatwork.Reference{}) {
		return "?"
	}
	target := accountTarget(value.Target, actorByRef)
	if !value.Resolved {
		return "?" + target
	}
	return target
}

func compactValues(values []string) string {
	if len(values) == 1 {
		return values[0]
	}
	return "[" + strings.Join(values, ",") + "]"
}

func renderMessage(output *strings.Builder, message chatwork.Message) {
	renderMessageLine(output, "message ", message)
}

func renderMessageLine(output *strings.Builder, prefix string, message chatwork.Message) {
	if message.RelationState == chatwork.MessageRelationsUnknown {
		line(output, "%smessage-ref=%s room-ref=%s sender-ref=%s sender-name=untrusted:%s send-time=%d relation-state=unknown body=untrusted:%s",
			prefix, ref(message.Ref), ref(message.Room), ref(message.Sender.Ref), quoted(message.Sender.Name),
			message.SendTime, quoted(message.Body))
		return
	}
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
	if message.RelationState == chatwork.MessageRelationsUnknown {
		return "unknown"
	}
	values := make([]string, 0, len(message.Recipients)+len(message.Replies)+len(message.Quotes))
	for _, recipient := range message.Recipients {
		values = append(values, fmt.Sprintf("to{target-ref=%s}", ref(recipient)))
	}
	for _, reply := range message.Replies {
		values = append(values, relation("reply", reply))
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
	renderCollectionPrelude(output, "personal-tasks", len(tasks), coverage, `task-ref room-ref assigned-by-ref message-ref "body" status`)
	for _, task := range tasks {
		line(output, "%s %s %s %s %s %s",
			ref(task.Ref), ref(task.Room.Ref), ref(task.AssignedBy.Ref), ref(task.Message), quoted(task.Body), atom(task.Status))
	}
}

func renderRoomTasks(output *strings.Builder, tasks []chatwork.WorkTask, coverage chatwork.Coverage) {
	renderCollectionPrelude(output, "room-tasks", len(tasks), coverage, `task-ref room-ref account-ref message-ref "body" status limit-time`)
	for _, task := range tasks {
		line(output, "%s %s %s %s %s %s %d",
			ref(task.Ref), ref(task.Room.Ref), ref(task.Account.Ref), ref(task.Message), quoted(task.Body), atom(task.Status), task.LimitTime)
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
	renderCollectionPrelude(output, "files", len(files), coverage, `file-ref room-ref account-ref message-ref "name" size`)
	for _, file := range files {
		line(output, "%s %s %s %s %s %d",
			ref(file.Ref), ref(file.Room), ref(file.Account.Ref), ref(file.Message), quoted(file.Name), file.Size)
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
	renderCollectionPrelude(output, "contact-requests", len(requests), coverage, `request-ref account-ref "name" ["message"]`)
	for _, request := range requests {
		fmt.Fprintf(output, "%s %s %s", ref(request.Ref), ref(request.Account.Ref), quoted(request.Account.Name))
		if request.Message != "" {
			fmt.Fprintf(output, " %s", quoted(request.Message))
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

func renderCollectionOrganization(output *strings.Builder, account chatwork.Account) {
	if account.OrganizationName == "" && account.Department == "" {
		return
	}
	output.WriteString(" organization={")
	separator := ""
	if account.OrganizationName != "" {
		fmt.Fprintf(output, "name=%s", quoted(account.OrganizationName))
		separator = ","
	}
	if account.Department != "" {
		fmt.Fprintf(output, "%sdepartment=%s", separator, quoted(account.Department))
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
		for _, reply := range message.Replies {
			if !reply.Resolved {
				count++
			}
		}
		for _, quote := range message.Quotes {
			if !quote.Resolved {
				count++
			}
		}
	}
	return count
}

func countUnknownRelations(messages []chatwork.Message) int {
	count := 0
	for _, message := range messages {
		if message.RelationState == chatwork.MessageRelationsUnknown {
			count++
		}
	}
	return count
}

func messageAccess(value chatwork.MessageAccessLimitation) (string, error) {
	switch value {
	case chatwork.MessageAccessNone:
		return "none", nil
	case chatwork.MessageAccessPartial:
		return "partial", nil
	case chatwork.MessageAccessAll:
		return "all", nil
	default:
		return "", fmt.Errorf("task projection message access limitation is invalid")
	}
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
		for _, reply := range message.Replies {
			if err := add(reply.Target); err != nil {
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
		for _, reply := range message.Replies {
			values = append(values, reply.Kind, reply.ExternalID)
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
