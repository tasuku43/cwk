package cli

import (
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

type chatworkCommandDefinition struct {
	Task         chatwork.Task
	Confirmation string
	Reconcile    string
}

const (
	confirmAccessChange = "access-change"
	confirmDestructive  = "destructive"
)

var chatworkAuthentication = &authn.Requirement{
	Methods:              []authn.Method{authn.MethodPAT},
	Authority:            chatwork.AuthenticationAuthority,
	Audience:             chatwork.AuthenticationAudience,
	RequiredCapabilities: []string{chatwork.AuthenticationCapability},
}

func chatworkCommandSpecs() []CommandSpec {
	room := string(chatwork.ReferenceRoom)
	account := string(chatwork.ReferenceAccount)
	message := string(chatwork.ReferenceMessage)
	task := string(chatwork.ReferenceTask)
	file := string(chatwork.ReferenceFile)
	invite := string(chatwork.ReferenceInvite)
	request := string(chatwork.ReferenceRequest)

	return []CommandSpec{
		chatworkRead("account show", "Show the authenticated Chatwork account", "", RoleDiscover,
			"chatwork.account.inspect", "Read the exact account bound to the configured Chatwork token",
			nil, fields(refField("account_ref", account, "Canonical account reference accepted by room creation and account filters."), textField("name", "Account display name."), textField("organization", "Non-empty human-readable organization and department facts, when present.")), chatwork.TaskAccountShow),
		chatworkRead("account status", "Show unread, mention, and task counts", "", RoleUtility,
			"chatwork.account.inspect", "Read the authenticated account's aggregate Chatwork status",
			nil, fields(integerField("unread", "Total unread messages."), integerField("mentions", "Total unread mentions."), integerField("tasks", "Total incomplete tasks.")), chatwork.TaskAccountStatus),
		chatworkRead("personal-tasks list", "List tasks assigned to the authenticated account", "[--assigned-by <account-ref>] [--status open|done]", RoleDiscover,
			"chatwork.personal-tasks.inspect", "List the bounded personal task collection in one fixed positional schema with canonical task, room, account, and message references",
			[]CommandInput{refFlag("--assigned-by", false, account, "Filter by one exact assigning account reference."), enumFlag("--status", false, "Filter by task status.", "open", "done")},
			fields(refField("task_ref", task, "Canonical task reference in position one."), refField("room_ref", room, "Canonical room reference in position two."), refField("assigned_by_ref", account, "Canonical assigning account reference in position three."), refField("message_ref", message, "Canonical task-message reference in position four."), textField("body", "Terminal-safe quoted task body in position five."), textField("status", "Task status in position six."), limitField(), completeField()), chatwork.TaskPersonalTasksList),
		chatworkRead("contacts list", "Discover Chatwork contacts", "", RoleDiscover,
			"chatwork.contacts.discover", "List contacts in one fixed positional schema with exact account and direct-room references",
			nil, fields(refField("account_ref", account, "Canonical contact account reference in position one."), refField("room_ref", room, "Canonical direct-room reference in position two."), textField("name", "Terminal-safe quoted contact display name in position three."), textField("organization", "Optional final organization suffix with non-empty name or department facts."), completeField()), chatwork.TaskContactsList),
		chatworkRead("rooms list", "Discover joined Chatwork rooms", "", RoleDiscover,
			"chatwork.rooms.manage", "List joined rooms in one fixed positional schema with exact room references and task-relevant status",
			nil, roomFields(room, true), chatwork.TaskRoomsList),
		chatworkMutation("rooms create", "Create a group room with exact members", "--owner <account-ref> --name <text> --admin <account-ref> [--member <account-ref>] [--readonly <account-ref>] [--description <text>] [--icon <preset>] [--invite-code <code>] [--invite-approval required|not-required] --confirm=access-change", RoleAct,
			"chatwork.rooms.manage", "Create one group room in the authenticated account scope with explicit membership and access impact",
			[]CommandInput{refFlag("--owner", true, account, "Bind room creation to the exact authenticated account reference."), textFlag("--name", true, "Room name."), repeatedRefFlag("--admin", true, account, "Add one administrator; repeat for additional administrators."), repeatedRefFlag("--member", false, account, "Add one member; repeat for additional members."), repeatedRefFlag("--readonly", false, account, "Add one read-only member; repeat for additional members."), textFlag("--description", false, "Room description."), textFlag("--icon", false, "Documented Chatwork icon preset."), textFlag("--invite-code", false, "Create an invitation link atomically with this optional code."), enumFlag("--invite-approval", false, "Create an invitation link with this approval requirement.", "required", "not-required"), confirmFlag(confirmAccessChange)},
			fields(refField("room_ref", room, "Canonical reference of the created room.")), chatwork.TaskRoomsCreate, confirmAccessChange, "rooms list",
			createMutation("chatwork-room", "--owner", operation.CardinalityMany, yes, yes, no)),
		chatworkRead("rooms show", "Show one exact room", "--room <room-ref>", RoleAct,
			"chatwork.rooms.manage", "Read one exact room without display-name rediscovery",
			[]CommandInput{refFlag("--room", true, room, "Pass a room_ref from room discovery unchanged.")}, roomFields(room, false), chatwork.TaskRoomsShow),
		chatworkMutation("rooms update", "Update one exact room's descriptive facts", "--room <room-ref> [--name <text>] [--description <text>] [--icon <preset>]", RoleAct,
			"chatwork.rooms.manage", "Update the selected room name, description, or icon without changing membership",
			[]CommandInput{refFlag("--room", true, room, "Exact room to update."), textFlag("--name", false, "Replacement room name."), textFlag("--description", false, "Replacement room description."), textFlag("--icon", false, "Replacement documented icon preset.")},
			fields(refField("room_ref", room, "Canonical updated room reference.")), chatwork.TaskRoomsUpdate, "", "rooms show",
			writeMutation("chatwork-room", "--room", "", operation.CardinalityOne, no, no, no)),
		chatworkMutation("rooms leave", "Leave one exact group room", "--room <room-ref> --confirm=destructive", RoleAct,
			"chatwork.rooms.manage", "Leave the selected group room with explicit destructive and access impact",
			[]CommandInput{refFlag("--room", true, room, "Exact room to leave."), confirmFlag(confirmDestructive)}, fields(refField("room_ref", room, "Canonical room that was left.")), chatwork.TaskRoomsLeave, confirmDestructive, "rooms show",
			writeMutation("chatwork-room", "--room", "", operation.CardinalityMany, no, yes, yes)),
		chatworkMutation("rooms delete", "Permanently delete one exact group room", "--room <room-ref> --confirm=destructive", RoleAct,
			"chatwork.rooms.manage", "Permanently delete the selected group room and its contained data",
			[]CommandInput{refFlag("--room", true, room, "Exact room to delete."), confirmFlag(confirmDestructive)}, fields(refField("room_ref", room, "Canonical room that was deleted.")), chatwork.TaskRoomsDelete, confirmDestructive, "rooms show",
			writeMutation("chatwork-room", "--room", "", operation.CardinalityUnbounded, no, yes, yes)),
		chatworkRead("members list", "List members of one exact room", "--room <room-ref>", RoleAct,
			"chatwork.members.manage", "List member identities and roles in one fixed positional schema for one exact room",
			[]CommandInput{refFlag("--room", true, room, "Exact room whose membership is read.")}, fields(refField("account_ref", account, "Canonical member account reference in position one."), textField("name", "Terminal-safe quoted member display name in position two."), textField("role", "Member role in position three."), completeField()), chatwork.TaskMembersList),
		chatworkMutation("members replace", "Replace one room's complete membership", "--room <room-ref> --admin <account-ref> [--member <account-ref>] [--readonly <account-ref>] --confirm=access-change", RoleAct,
			"chatwork.members.manage", "Replace the selected room's complete role membership with explicit access impact",
			[]CommandInput{refFlag("--room", true, room, "Exact room whose membership is replaced."), repeatedRefFlag("--admin", true, account, "Administrator account; repeat for more."), repeatedRefFlag("--member", false, account, "Member account; repeat for more."), repeatedRefFlag("--readonly", false, account, "Read-only account; repeat for more."), confirmFlag(confirmAccessChange)},
			fields(integerField("administrators", "Resulting administrator count."), integerField("members", "Resulting member count."), integerField("readonly", "Resulting read-only count.")), chatwork.TaskMembersReplace, confirmAccessChange, "members list",
			writeMutation(room, "--room", "", operation.CardinalityMany, yes, yes, yes)),
		chatworkRead("messages list", "Get a bounded, selectable message window", "--room <room-ref> [--window changes|recent] [--limit <count>] [--sender <account-ref>] [--context none|replies]", RoleAct,
			"chatwork.messages.manage", "Get this room's bounded provider-order message window, optionally limiting the newest primary messages and filtering by exact senders before one-hop explicit reply context, while preserving canonical references and typed To, reply, quote, and coverage semantics",
			[]CommandInput{
				refFlag("--room", true, room, "Exact room whose messages are read."),
				enumFlag("--window", false, "Choose provider differential changes (default) or the latest bounded window.", "changes", "recent"),
				integerFlag("--limit", false, "Keep the newest 1..100 primary messages by typed send time after optional sender matching; direct reply context may add records beyond this count. Use --window recent for the room's current latest window."),
				repeatedRefFlag("--sender", false, account, "Filter by an exact sender account within the bounded provider window; repeat to match any listed sender (OR), up to 100 exact references."),
				enumFlag("--context", false, "With --sender or --limit, include no related records (none, default) or one-hop explicit reply parents and children from the bounded provider window (replies).", "none", "replies"),
			}, messageFields(room, message, account, true), chatwork.TaskMessagesList),
		chatworkMutation("messages send", "Send a message to one exact room", "--room <room-ref> --body <text> [--self-unread]", RoleAct,
			"chatwork.messages.manage", "Send one exact message body to the selected room",
			[]CommandInput{refFlag("--room", true, room, "Exact destination room."), textFlag("--body", true, "Message body, including reviewed Chatwork notation when intended."), boolFlag("--self-unread", "Leave the sent message unread for the authenticated account.")},
			fields(refField("message_ref", message, "Canonical created message reference."), refField("room_ref", room, "Canonical parent room reference.")), chatwork.TaskMessagesSend, "", "messages list",
			createMutation("chatwork-message", "--room", operation.CardinalityOne, yes, no, no)),
		chatworkMutation("messages mark-read", "Mark through one exact message as read", "--room <room-ref> --message <message-ref>", RoleAct,
			"chatwork.messages.manage", "Mark messages through the selected exact message as read in one room",
			[]CommandInput{refFlag("--room", true, room, "Exact parent room."), refFlag("--message", true, message, "Exact message boundary to mark read.")}, unreadFields(), chatwork.TaskMessagesMarkRead, "", "messages show",
			writeMutation(message, "--message", "--room", operation.CardinalityMany, no, no, no)),
		chatworkMutation("messages mark-unread", "Mark from one exact message as unread", "--room <room-ref> --message <message-ref>", RoleAct,
			"chatwork.messages.manage", "Mark messages from the selected exact message as unread in one room",
			[]CommandInput{refFlag("--room", true, room, "Exact parent room."), refFlag("--message", true, message, "Exact message boundary to mark unread.")}, unreadFields(), chatwork.TaskMessagesMarkUnread, "", "messages show",
			writeMutation(message, "--message", "--room", operation.CardinalityMany, no, no, no)),
		chatworkRead("messages show", "Show one exact message", "--room <room-ref> --message <message-ref>", RoleAct,
			"chatwork.messages.manage", "Read one exact message with typed relationship facts",
			[]CommandInput{refFlag("--room", true, room, "Exact parent room."), refFlag("--message", true, message, "Exact message to read.")}, messageFields(room, message, account, false), chatwork.TaskMessagesShow),
		chatworkMutation("messages update", "Update one exact message", "--room <room-ref> --message <message-ref> --body <text>", RoleAct,
			"chatwork.messages.manage", "Replace the body of one exact message",
			[]CommandInput{refFlag("--room", true, room, "Exact parent room."), refFlag("--message", true, message, "Exact message to update."), textFlag("--body", true, "Replacement message body.")}, fields(refField("message_ref", message, "Canonical updated message reference.")), chatwork.TaskMessagesUpdate, "", "messages show",
			writeMutation("chatwork-message", "--message", "--room", operation.CardinalityOne, no, no, no)),
		chatworkMutation("messages delete", "Delete one exact message", "--room <room-ref> --message <message-ref> --confirm=destructive", RoleAct,
			"chatwork.messages.manage", "Delete the selected exact message with explicit destructive impact",
			[]CommandInput{refFlag("--room", true, room, "Exact parent room."), refFlag("--message", true, message, "Exact message to delete."), confirmFlag(confirmDestructive)}, fields(refField("message_ref", message, "Canonical deleted message reference.")), chatwork.TaskMessagesDelete, confirmDestructive, "messages show",
			writeMutation("chatwork-message", "--message", "--room", operation.CardinalityOne, no, no, yes)),
		chatworkRead("room-tasks list", "List tasks in one exact room", "--room <room-ref> [--account <account-ref>] [--assigned-by <account-ref>] [--status open|done]", RoleAct,
			"chatwork.room-tasks.manage", "List a bounded room task collection in one fixed positional schema with exact task and account references",
			[]CommandInput{refFlag("--room", true, room, "Exact room whose tasks are read."), refFlag("--account", false, account, "Filter by exact assignee account."), refFlag("--assigned-by", false, account, "Filter by exact assigning account."), enumFlag("--status", false, "Filter by task status.", "open", "done")}, taskFields(room, task, account, message, true), chatwork.TaskRoomTasksList),
		chatworkMutation("room-tasks create", "Create tasks for exact assignees in one room", "--room <room-ref> --body <text> --assignee <account-ref> [--limit <unix-time>] [--limit-type date|time]", RoleAct,
			"chatwork.room-tasks.manage", "Create the selected task body for exact account assignees in one room",
			[]CommandInput{refFlag("--room", true, room, "Exact parent room."), textFlag("--body", true, "Task body."), repeatedRefFlag("--assignee", true, account, "Exact assignee; repeat for additional assignees."), integerFlag("--limit", false, "Optional Unix deadline."), enumFlag("--limit-type", false, "Interpret the deadline as a date or time.", "date", "time")}, fields(refField("task_ref", task, "Canonical created task reference."), refField("room_ref", room, "Canonical parent room reference.")), chatwork.TaskRoomTasksCreate, "", "room-tasks list",
			createMutation("chatwork-task", "--room", operation.CardinalityMany, yes, no, no)),
		chatworkRead("room-tasks show", "Show one exact room task", "--room <room-ref> --task <task-ref>", RoleAct,
			"chatwork.room-tasks.manage", "Read one exact task without rediscovery",
			[]CommandInput{refFlag("--room", true, room, "Exact parent room."), refFlag("--task", true, task, "Exact task to read.")}, taskFields(room, task, account, message, false), chatwork.TaskRoomTasksShow),
		chatworkMutation("room-tasks set-status", "Set one exact task's completion status", "--room <room-ref> --task <task-ref> --status open|done", RoleAct,
			"chatwork.room-tasks.manage", "Set the selected exact task to open or done",
			[]CommandInput{refFlag("--room", true, room, "Exact parent room."), refFlag("--task", true, task, "Exact task to change."), enumFlag("--status", true, "Replacement task status.", "open", "done")}, fields(refField("task_ref", task, "Canonical updated task reference.")), chatwork.TaskRoomTasksSetStatus, "", "room-tasks show",
			writeMutation("chatwork-task", "--task", "--room", operation.CardinalityOne, yes, no, no)),
		chatworkRead("files list", "List files in one exact room", "--room <room-ref> [--account <account-ref>]", RoleAct,
			"chatwork.files.manage", "List a bounded room file collection in one fixed positional schema; pass positions one and two unchanged to files show",
			[]CommandInput{refFlag("--room", true, room, "Exact room whose files are read."), refFlag("--account", false, account, "Filter by exact uploader account.")}, fileFields(room, file, account, message, true), chatwork.TaskFilesList),
		chatworkMutation("files upload", "Upload one bounded file to one exact room", "--room <room-ref> --path <file> [--message <text>]", RoleAct,
			"chatwork.files.manage", "Upload one local file of at most 5 MiB to the selected room",
			[]CommandInput{refFlag("--room", true, room, "Exact destination room."), textFlag("--path", true, "Local file path; bounded and validated before upload."), textFlag("--message", false, "Optional message attached to the file.")}, fields(refField("file_ref", file, "Canonical uploaded file reference."), refField("room_ref", room, "Canonical parent room reference.")), chatwork.TaskFilesUpload, "", "files list",
			createMutation("chatwork-file", "--room", operation.CardinalityOne, yes, no, no)),
		chatworkRead("files show", "Show one exact room file", "--room <room-ref> --file <file-ref> [--create-download-url]", RoleAct,
			"chatwork.files.manage", "Read one exact file and optionally request its bounded provider download URL",
			[]CommandInput{refFlag("--room", true, room, "Exact parent room."), refFlag("--file", true, file, "Exact file to read."), boolFlag("--create-download-url", "Request a provider download URL in this result.")}, fileFields(room, file, account, message, false), chatwork.TaskFilesShow),
		chatworkRead("invite-link show", "Show one room's invitation-link state", "--room <room-ref>", RoleAct,
			"chatwork.invite-links.manage", "Read the invitation-link state for one exact room",
			[]CommandInput{refFlag("--room", true, room, "Exact room whose invitation-link state is read.")}, inviteFields(invite), chatwork.TaskInviteLinkShow),
		chatworkMutation("invite-link create", "Create an invitation link for one room", "--room <room-ref> [--code <code>] [--approval required|not-required] --confirm=access-change", RoleAct,
			"chatwork.invite-links.manage", "Create a room invitation link with explicit access impact",
			[]CommandInput{refFlag("--room", true, room, "Exact parent room."), textFlag("--code", false, "Optional documented invitation-link code."), enumFlag("--approval", false, "Whether joining requires administrator approval.", "required", "not-required"), confirmFlag(confirmAccessChange)}, inviteFields(invite), chatwork.TaskInviteLinkCreate, confirmAccessChange, "invite-link show",
			createMutation("chatwork-invite-link", "--room", operation.CardinalityOne, no, yes, no)),
		chatworkMutation("invite-link update", "Update one exact invitation link", "--invite <invite-ref> [--code <code>] [--approval required|not-required] --confirm=access-change", RoleAct,
			"chatwork.invite-links.manage", "Update the selected invitation link's code or approval requirement",
			[]CommandInput{refFlag("--invite", true, invite, "Exact invitation-link reference emitted by invite-link show or create."), textFlag("--code", false, "Replacement documented invitation-link code."), enumFlag("--approval", false, "Replacement approval requirement.", "required", "not-required"), confirmFlag(confirmAccessChange)}, inviteFields(invite), chatwork.TaskInviteLinkUpdate, confirmAccessChange, "invite-link show",
			writeMutation("chatwork-invite-link", "--invite", "", operation.CardinalityOne, no, yes, no)),
		chatworkMutation("invite-link delete", "Disable one exact invitation link", "--invite <invite-ref> --confirm=destructive", RoleAct,
			"chatwork.invite-links.manage", "Disable the selected invitation link with explicit destructive access impact",
			[]CommandInput{refFlag("--invite", true, invite, "Exact invitation-link reference."), confirmFlag(confirmDestructive)}, inviteFields(invite), chatwork.TaskInviteLinkDelete, confirmDestructive, "invite-link show",
			writeMutation("chatwork-invite-link", "--invite", "", operation.CardinalityOne, no, yes, yes)),
		chatworkRead("contact-requests list", "Discover incoming contact requests", "", RoleDiscover,
			"chatwork.contact-requests.manage", "List incoming contact requests in one fixed positional schema with exact request and account references",
			nil, fields(refField("request_ref", request, "Canonical incoming-request reference in position one."), refField("account_ref", account, "Canonical requesting account reference in position two."), textField("name", "Terminal-safe quoted requesting account name in position three."), textField("message", "Optional terminal-safe quoted request message in the final position."), limitField(), completeField()), chatwork.TaskContactRequestsList),
		chatworkMutation("contact-requests accept", "Accept one exact contact request", "--request <request-ref>", RoleAct,
			"chatwork.contact-requests.manage", "Accept the selected incoming contact request",
			[]CommandInput{refFlag("--request", true, request, "Exact incoming-request reference.")}, fields(refField("account_ref", account, "Canonical accepted contact account reference."), refField("room_ref", room, "Canonical direct-room reference.")), chatwork.TaskContactRequestsAccept, "", "contact-requests list",
			writeMutation("chatwork-contact-request", "--request", "", operation.CardinalityOne, yes, yes, no)),
		chatworkMutation("contact-requests reject", "Reject one exact contact request", "--request <request-ref> --confirm=destructive", RoleAct,
			"chatwork.contact-requests.manage", "Reject the selected incoming contact request",
			[]CommandInput{refFlag("--request", true, request, "Exact incoming-request reference."), confirmFlag(confirmDestructive)}, fields(refField("request_ref", request, "Canonical contact request that was rejected.")), chatwork.TaskContactRequestsReject, confirmDestructive, "contact-requests list",
			writeMutation("chatwork-contact-request", "--request", "", operation.CardinalityOne, no, yes, yes)),
	}
}

const (
	no  = operation.DeclarationNo
	yes = operation.DeclarationYes
)

func chatworkRead(path, summary, args string, role CommandRole, capability, outcome string, inputs []CommandInput, output []OutputField, task chatwork.Task) CommandSpec {
	if inputs == nil {
		inputs = []CommandInput{}
	}
	return chatworkBase(path, summary, args, operation.EffectRead, role, capability, outcome, inputs, output, task, "", "", nil)
}

func chatworkMutation(path, summary, args string, role CommandRole, capability, outcome string, inputs []CommandInput, output []OutputField, task chatwork.Task, confirmation, reconcile string, mutation MutationContract) CommandSpec {
	effect := operation.EffectWrite
	if mutation.TargetIDInput == "" {
		effect = operation.EffectCreate
	}
	return chatworkBase(path, summary, args, effect, role, capability, outcome, inputs, output, task, confirmation, reconcile, &mutation)
}

func chatworkBase(path, summary, args string, effect operation.Effect, role CommandRole, capability, outcome string, inputs []CommandInput, output []OutputField, task chatwork.Task, confirmation, reconcile string, mutation *MutationContract) CommandSpec {
	return CommandSpec{
		Path: path, Summary: summary, Args: args, Effect: effect, Role: role, Configurable: true,
		Agent: AgentContract{
			CapabilityID: capability, Outcome: outcome, Inputs: inputs,
			Output:         CommandOutput{Formats: []OutputFormat{OutputFormatText}, DefaultFormat: OutputFormatText, Fields: output, Completeness: OutputCompletenessComplete},
			Prerequisites:  []string{"Set CWK_API_TOKEN only for the command process; do not pass the token through argv or a project file."},
			Authentication: chatworkAuthentication,
			Errors:         chatworkCommandErrors(path, task, reconcile, mutation != nil), Mutation: mutation,
		},
		handler:  runChatwork,
		chatwork: &chatworkCommandDefinition{Task: task, Confirmation: confirmation, Reconcile: reconcile},
	}
}

func chatworkCommandErrors(path string, task chatwork.Task, reconcile string, mutation bool) []CommandError {
	help := "help " + path
	retry := path
	if mutation {
		// Mutation recovery never suggests replaying a write. Even failures that
		// are retryable at the fault level route through scoped help; uncertain
		// outcomes use the exact read-only reconciliation task below.
		retry = help
	}
	errors := []CommandError{
		declaredCommandError(fault.KindInvalidInput, "invalid_arguments", false, help, "Correct the declared command inputs."),
		declaredCommandError(fault.KindInvalidInput, "invalid_chatwork_task", false, help, "Correct the typed Chatwork task inputs."),
		declaredCommandError(fault.KindInvalidInput, "invalid_chatwork_request", false, help, "Correct the typed Chatwork adapter request."),
		declaredCommandError(fault.KindContract, "missing_context", false, help, "Repair the context-aware command invocation."),
		declaredCommandError(fault.KindContract, "missing_chatwork_port", false, help, "Repair the Chatwork adapter composition."),
		declaredCommandError(fault.KindContract, "chatwork_result_mismatch", false, help, "Repair the typed Chatwork adapter result contract."),
		declaredCommandError(fault.KindContract, "chatwork_result_invalid", false, help, "Repair the task-specific typed Chatwork result contract."),
		declaredCommandError(fault.KindAuthentication, "invalid_authentication_binding", false, help, "Re-establish the configured Chatwork authentication."),
		declaredCommandError(fault.KindInternal, "unclassified_chatwork_error", false, help, "Inspect the Chatwork adapter classification."),
		declaredCommandError(fault.KindInvalidInput, "chatwork_invalid_request", false, help, "Correct the task inputs accepted by Chatwork."),
		declaredCommandError(fault.KindAuthentication, "chatwork_token_missing", false, help, "Set CWK_API_TOKEN for this command process."),
		declaredCommandError(fault.KindAuthentication, "chatwork_token_invalid", false, help, "Replace CWK_API_TOKEN with a valid process-local token."),
		declaredCommandError(fault.KindAuthentication, "chatwork_authentication_failed", false, help, "Replace the configured Chatwork token."),
		declaredCommandError(fault.KindPermission, "chatwork_permission_denied", false, help, "Use an account permitted to perform this task."),
		declaredCommandError(fault.KindNotFound, "chatwork_not_found", false, help, "Rediscover a current canonical reference."),
		declaredCommandError(fault.KindRateLimited, "chatwork_rate_limited", true, retry, "Retry after the declared provider reset time."),
		declaredCommandError(fault.KindContract, "chatwork_response_too_large", false, help, "Narrow the task or review the fixed response bound."),
		declaredCommandError(fault.KindContract, "chatwork_response_invalid", false, help, "Review provider schema drift before retrying."),
		declaredCommandError(fault.KindContract, "chatwork_response_malformed", false, help, "Review provider schema drift before retrying."),
		declaredCommandError(fault.KindContract, "chatwork_response_unmapped", false, help, "Repair the typed response mapping."),
		declaredCommandError(fault.KindUnavailable, "chatwork_response_unavailable", true, retry, "Retry only after reviewing the bounded response failure."),
		declaredCommandError(fault.KindContract, "chatwork_request_contract_invalid", false, help, "Repair the typed request mapping."),
		declaredCommandError(fault.KindContract, "chatwork_transport_missing", false, help, "Repair the Chatwork transport composition."),
		declaredCommandError(fault.KindContract, "chatwork_unexpected_response", false, help, "Review undocumented provider behavior before retrying."),
		declaredCommandError(fault.KindContract, "output_contract_exceeded", false, help, "Narrow the result or review the fixed output bound."),
		declaredCommandError(fault.KindContract, "output_encoding_failed", false, help, "Repair the task projection."),
		declaredCommandError(fault.KindInternal, "output_write_failed", true, retry, "Retry with a writable output stream."),
		declaredCommandError(fault.KindCanceled, "operation_canceled", true, retry, "Retry when the caller is ready."),
	}
	if !mutation {
		errors = append(errors, declaredCommandError(fault.KindUnavailable, "chatwork_unavailable", true, path, "Retry after Chatwork becomes available."))
	}
	if task == chatwork.TaskMessagesList || task == chatwork.TaskMessagesShow {
		errors = append(errors, declaredCommandError(fault.KindContract, "chatwork_notation_malformed", false, help, "Review malformed or unsupported message notation."))
	}
	if task == chatwork.TaskFilesUpload {
		errors = append(errors,
			declaredCommandError(fault.KindInvalidInput, "chatwork_file_name_invalid", false, help, "Choose a file with a valid base name."),
			declaredCommandError(fault.KindInvalidInput, "chatwork_file_unreadable", false, help, "Choose a readable local file."),
			declaredCommandError(fault.KindInvalidInput, "chatwork_file_too_large", false, help, "Choose a file no larger than 5 MiB."),
			declaredCommandError(fault.KindContract, "chatwork_upload_contract_invalid", false, help, "Repair the bounded multipart request mapping."),
		)
	}
	for _, required := range []struct {
		kind fault.Kind
		code string
	}{
		{fault.KindContract, "missing_authentication_context"}, {fault.KindContract, "missing_authenticated_action"},
		{fault.KindContract, "invalid_authentication_requirement"}, {fault.KindAuthentication, "missing_authenticator"},
		{fault.KindContract, "missing_authentication_clock"}, {fault.KindAuthentication, "invalid_authentication_session"},
		{fault.KindContract, "authentication_evaluation_failed"}, {fault.KindPermission, "insufficient_authentication_capability"},
		{fault.KindAuthentication, "authentication_expired"}, {fault.KindAuthentication, "authentication_context_mismatch"},
		{fault.KindAuthentication, "authentication_failed"}, {fault.KindCanceled, "authentication_canceled"},
		{fault.KindInternal, "unclassified_authenticated_action_error"},
	} {
		errors = append(errors, declaredCommandError(required.kind, required.code, false, help, "Repair or re-establish the declared Chatwork authentication context."))
	}
	if mutation {
		errors = append(errors,
			declaredCommandError(fault.KindContract, "invalid_mutation_contract", false, help, "Repair the mutation target and impact declaration."),
			declaredCommandError(fault.KindContract, "missing_mutation_action", false, help, "Repair mutation action composition."),
			declaredCommandError(fault.KindRejected, "missing_mutation_policy", false, help, "Configure the declared Chatwork mutation policy."),
			declaredCommandError(fault.KindRejected, "mutation_rejected", false, help, "Supply the required explicit confirmation without changing the target."),
			declaredCommandError(fault.KindContract, "unclassified_mutation_outcome", false, reconcile, "Reconcile through this read-only task before another mutation."),
			declaredCommandError(fault.KindContract, "chatwork_mutation_outcome_unknown", false, reconcile, "Reconcile through this read-only task before another mutation."),
		)
	}
	return errors
}

func createMutation(kind, parent string, cardinality operation.Cardinality, notification, access, destructive operation.Declaration) MutationContract {
	return MutationContract{TargetKind: kind, TargetInputs: []string{parent}, ParentInput: parent, Impact: operation.Impact{Cardinality: cardinality, Notification: notification, AccessChange: access, Destructive: destructive}}
}

func writeMutation(kind, target, parent string, cardinality operation.Cardinality, notification, access, destructive operation.Declaration) MutationContract {
	targets := []string{target}
	if parent != "" {
		targets = append(targets, parent)
	}
	return MutationContract{TargetKind: kind, TargetInputs: targets, ParentInput: parent, TargetIDInput: target, Impact: operation.Impact{Cardinality: cardinality, Notification: notification, AccessChange: access, Destructive: destructive}}
}

func refFlag(name string, required bool, kind, description string) CommandInput {
	return CommandInput{Name: name, Source: InputSourceFlag, Required: required, Description: description, AllowedValues: []string{}, ReferenceKind: kind}
}
func repeatedRefFlag(name string, required bool, kind, description string) CommandInput {
	input := refFlag(name, required, kind, description)
	input.Repeatable = true
	return input
}
func textFlag(name string, required bool, description string) CommandInput {
	return CommandInput{Name: name, Source: InputSourceFlag, Required: required, Description: description, AllowedValues: []string{}}
}
func integerFlag(name string, required bool, description string) CommandInput {
	return textFlag(name, required, description)
}
func boolFlag(name, description string) CommandInput {
	return CommandInput{Name: name, Source: InputSourceFlag, Required: false, Description: description, AllowedValues: []string{}}
}
func enumFlag(name string, required bool, description string, values ...string) CommandInput {
	return CommandInput{Name: name, Source: InputSourceFlag, Required: required, Description: description, AllowedValues: values}
}
func confirmFlag(value string) CommandInput {
	return enumFlag("--confirm", true, "Explicitly confirm the declared high-impact mutation class.", value)
}

func fields(values ...OutputField) []OutputField { return values }
func refField(name, kind, description string) OutputField {
	return OutputField{Name: name, Type: OutputFieldTypeString, Description: description, ReferenceKind: kind}
}
func textField(name, description string) OutputField {
	return OutputField{Name: name, Type: OutputFieldTypeString, Description: description}
}
func integerField(name, description string) OutputField {
	return OutputField{Name: name, Type: OutputFieldTypeInteger, Description: description}
}
func booleanField(name, description string) OutputField {
	return OutputField{Name: name, Type: OutputFieldTypeBoolean, Description: description}
}
func limitField() OutputField {
	return integerField("limit", "Maximum provider items in this result window.")
}
func completeField() OutputField {
	return booleanField("complete", "Whether this output is the complete documented collection for the operation.")
}
func roomFields(room string, collection bool) []OutputField {
	roomDescription := "Canonical room reference accepted unchanged by room actions."
	nameDescription := "Room name as untrusted external text."
	if collection {
		roomDescription = "Canonical room reference in position one; pass it unchanged to room actions."
		nameDescription = "Terminal-safe quoted room name in position two."
	}
	result := fields(refField("room_ref", room, roomDescription), textField("name", nameDescription), textField("type", "Room type."), textField("role", "Authenticated account role."), integerField("unread", "Unread message count."), integerField("mentions", "Unread mention count."), integerField("tasks", "Incomplete task count."))
	if collection {
		result = append(result, completeField())
	}
	return result
}
func messageFields(room, message, account string, collection bool) []OutputField {
	messageDescription := "Canonical message reference."
	bodyDescription := "Message body as structurally framed untrusted text."
	sendDescription := "Unix send time."
	if collection {
		messageDescription = "Canonical message reference in the second positional record field; pass it unchanged to message actions."
		sendDescription = "Unix send time in the fourth positional record field."
		bodyDescription = "Terminal-safe quoted message body in the final positional record field."
	}
	result := fields(refField("message_ref", message, messageDescription), refField("room_ref", room, "Canonical parent room reference."), refField("sender_ref", account, "Canonical sender account reference."), textField("sender_name", "Sender display name as structurally framed untrusted text."), textField("body", bodyDescription), integerField("send_time", sendDescription), OutputField{Name: "relations", Type: OutputFieldTypeArray, Description: "Typed To, reply, and quote relations with resolved or unresolved state."})
	if collection {
		result = append(result,
			integerField("sequence", "One-based position in the original provider-returned message window; selected output may contain gaps."),
			textField("actor_alias", "Deterministic document-local sender alias; never a command identity."),
			textField("window", "Requested recent or differential message window."),
			integerField("source_limit", "Provider source-window maximum before local message selection."), completeField(),
			integerField("unresolved_relations", "Typed relations whose canonical target could not be resolved."),
			integerField("source_count", "Provider-returned message count before active local selection; absent when no selection was requested."),
			integerField("candidate_count", "Primary-message candidate count after optional sender matching and before --limit; absent when --limit was not supplied."),
			integerField("selection_limit", "Requested newest-primary-message limit; absent when --limit was not supplied."),
			OutputField{Name: "filter_senders", Type: OutputFieldTypeArray, Description: "Exact canonical sender account references used as OR anchors; absent when no filter was requested."},
			textField("filter_context", "Effective none or replies context policy; omitted for limit-only selection with default none."),
			OutputField{Name: "anchor_sequences", Type: OutputFieldTypeArray, Description: "Original provider sequences selected as primary messages; displayed non-anchor sequences are reply context."},
		)
	}
	return result
}
func taskFields(room, task, account, message string, collection bool) []OutputField {
	taskDescription := "Canonical task reference."
	roomDescription := "Canonical parent room reference."
	accountDescription := "Canonical assignee account reference."
	messageDescription := "Canonical task-message reference."
	bodyDescription := "Task body as untrusted external text."
	if collection {
		taskDescription = "Canonical task reference in position one."
		roomDescription = "Canonical parent room reference in position two."
		accountDescription = "Canonical assignee account reference in position three."
		messageDescription = "Canonical task-message reference in position four."
		bodyDescription = "Terminal-safe quoted task body in position five."
	}
	result := fields(refField("task_ref", task, taskDescription), refField("room_ref", room, roomDescription), refField("account_ref", account, accountDescription), refField("message_ref", message, messageDescription), textField("body", bodyDescription), textField("status", "Task completion status."), integerField("limit_time", "Unix deadline or zero."))
	if collection {
		result = append(result, limitField(), completeField())
	}
	return result
}
func fileFields(room, file, account, message string, collection bool) []OutputField {
	fileDescription := "Canonical file reference."
	roomDescription := "Canonical parent room reference."
	accountDescription := "Canonical uploader account reference."
	messageDescription := "Canonical file-message reference."
	nameDescription := "File name as untrusted external text."
	if collection {
		fileDescription = "Canonical file reference in position one; pass it unchanged to files show --file."
		roomDescription = "Canonical parent room reference in position two; pass it unchanged to files show --room."
		accountDescription = "Canonical uploader account reference in position three."
		messageDescription = "Position four is a canonical file-message reference when present or the literal absent; never pass absent as a reference."
		nameDescription = "Terminal-safe quoted file name in position five."
	}
	result := fields(refField("file_ref", file, fileDescription), refField("room_ref", room, roomDescription), refField("account_ref", account, accountDescription), refField("message_ref", message, messageDescription), textField("name", nameDescription), integerField("size", "File size in bytes."))
	if collection {
		return append(result, limitField(), completeField())
	}
	return append(result, textField("download_url", "Requested provider download URL when one is returned."))
}
func inviteFields(invite string) []OutputField {
	return fields(refField("invite_ref", invite, "Canonical invitation-link reference accepted unchanged by update/delete."), booleanField("public", "Whether the invitation link is enabled."), textField("url", "Invitation URL when enabled and non-empty."), booleanField("needs_approval", "Whether an administrator must approve joining; emitted when enabled."), textField("description", "Non-empty provider invitation description when enabled."))
}
func unreadFields() []OutputField {
	return fields(integerField("unread", "Resulting room unread count."), integerField("mentions", "Resulting room unread mention count."))
}
