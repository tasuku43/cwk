package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/execution"
	domainauthn "github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

const maxChatworkOutputBytes = 16 * 1024 * 1024

// runChatwork is the one CLI boundary shared by the task-oriented Chatwork
// catalog. The catalog limits the accepted argv surface; this function only
// maps those declared values into the provider-independent request union.
func runChatwork(ctx context.Context, c *CLI, command CommandSpec, intent operation.Intent, args []string) int {
	arguments, deprecatedFlags, err := parseChatworkInvocationArguments(command, args)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error()+"; 使い方: "+command.Usage(), "help "+command.Path, "宣言されているコマンド入力を修正してください。")
	}
	requestArguments, presentation, err := splitChatworkPresentationArguments(command, arguments)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error()+"; 使い方: "+command.Usage(), "help "+command.Path, "宣言されているコマンド入力を修正してください。")
	}
	var now func() time.Time
	if c != nil {
		now = c.now
	}
	request, err := buildChatworkRequest(command, requestArguments, now)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error()+"; 使い方: "+command.Usage(), "help "+command.Path, "宣言されているコマンド入力を修正してください。")
	}
	if err := resolveChatworkMessageStdin(ctx, c, &request); err != nil {
		return c.fail(ctx, err)
	}
	if err := request.Validate(); err != nil {
		return c.failUsage(ctx, "invalid_arguments", invalidChatworkRequestMessage(command, arguments)+"; 使い方: "+command.Usage(), "help "+command.Path, "名前を示した入力の値と併用条件を修正してください。")
	}

	var result chatwork.Result
	mutationPortStarted := false
	authenticatedAction := func(actionContext context.Context, session domainauthn.Session) error {
		if c.chatwork == nil {
			return fault.New(fault.KindContract, "missing_chatwork_port", "Chatwork タスクアダプターが設定されていません", false)
		}
		if command.Effect != operation.EffectRead {
			mutationPortStarted = true
		}
		value, executeErr := c.chatwork.Execute(actionContext, session.BindingID, request)
		if executeErr == nil {
			result = value
		}
		return executeErr
	}
	authenticated := func(actionContext context.Context) error {
		if err := c.ensureChatwork(actionContext); err != nil {
			return err
		}
		requirement := domainauthn.Requirement{}
		if command.Agent.Authentication != nil {
			requirement = command.Agent.Authentication.Clone()
		}
		if request.Task == chatwork.TaskRoomsCreate {
			requirement.AccountID = request.Account.Value
		}
		var gate *appauthn.Gate
		if c != nil {
			gate = c.chatworkAuth
		}
		return gate.Invoke(actionContext, requirement, authenticatedAction)
	}

	if command.Effect == operation.EffectRead {
		err = authenticated(ctx)
	} else {
		executionRequest, buildErr := buildChatworkExecutionRequest(command, intent, arguments)
		if buildErr != nil {
			return c.failUsage(ctx, "invalid_arguments", buildErr.Error()+"; 使い方: "+command.Usage(), "help "+command.Path, "宣言されているコマンド入力を修正してください。")
		}
		policy := chatworkcmd.ConfirmationPolicy{
			Required: command.chatwork.Confirmation,
			Provided: arguments.first("--confirm"),
		}
		err = execution.New(policy).Invoke(ctx, executionRequest, func(actionContext context.Context, _ operation.Intent) error {
			actionErr := authenticated(actionContext)
			if unclassifiedMutationServiceError(actionErr) && (mutationPortStarted || !operationCanceledServiceError(actionErr)) {
				// Service-level cancellation or fallback classification cannot
				// prove whether a called mutation reached Chatwork. Return an
				// unstructured private sentinel so execution.Invoker applies its
				// conservative, non-retryable unknown-outcome contract.
				return fmt.Errorf("Chatwork mutation outcome is not classified")
			}
			return actionErr
		})
	}
	if err != nil {
		return c.fail(ctx, err)
	}

	output, err := renderChatworkResult(result, presentation)
	if err != nil {
		if _, structured := fault.PublicCopy(err); structured {
			return c.fail(ctx, err)
		}
		return c.fail(ctx, fault.New(fault.KindContract, "output_encoding_failed", "Chatwork タスクの投影をエンコードできませんでした。", false))
	}
	if len(output) > maxChatworkOutputBytes {
		return c.fail(ctx, fault.New(fault.KindContract, "output_contract_exceeded", "Chatwork の結果が宣言済みの出力上限を超えています。", false))
	}
	if len(deprecatedFlags) > 0 {
		var warning strings.Builder
		for _, used := range deprecatedFlags {
			fmt.Fprintf(&warning, "warning: %s は %s に改名されました\n", used.Alias, used.Canonical)
		}
		if _, writeErr := writeOnce(c.Err, []byte(warning.String())); writeErr != nil {
			return c.fail(ctx, fault.Wrap(
				fault.KindInternal,
				"output_write_failed",
				"非推奨フラグの移行案内を完全に書き込めませんでした。",
				true,
				writeErr,
				fault.NextAction{Command: invocationCommandPath(ctx), Reason: "書き込み可能な標準エラー出力で再試行してください。"},
			))
		}
	}
	if command.Effect == operation.EffectRead {
		return c.emit(ctx, []byte(output))
	}
	return c.emitMutationResult(ctx, []byte(output))
}

type chatworkArguments map[string][]string

type deprecatedFlagUse struct {
	Alias     string
	Canonical string
}

func (a chatworkArguments) first(name string) string {
	values := a[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func parseChatworkArguments(command CommandSpec, args []string) (chatworkArguments, error) {
	parsed, _, err := parseChatworkInvocationArguments(command, args)
	return parsed, err
}

func parseChatworkInvocationArguments(command CommandSpec, args []string) (chatworkArguments, []deprecatedFlagUse, error) {
	declared := make(map[string]CommandInput, len(command.Agent.Inputs))
	aliases := make(map[string]string)
	candidates := make([]string, 0, len(command.Agent.Inputs))
	for _, input := range command.Agent.Inputs {
		if input.Source != InputSourceFlag || !strings.HasPrefix(input.Name, "--") {
			return nil, nil, fmt.Errorf("Chatwork コマンドに未対応の入力契約があります")
		}
		declared[input.Name] = input
		candidates = append(candidates, input.Name)
		for _, alias := range input.DeprecatedAliases {
			aliases[alias] = input.Name
			candidates = append(candidates, alias)
		}
	}

	parsed := make(chatworkArguments, len(declared))
	deprecatedFlags := make([]deprecatedFlagUse, 0)
	seenDeprecatedFlags := make(map[string]struct{})
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if !strings.HasPrefix(argument, "--") {
			return nil, nil, fmt.Errorf("Chatwork コマンドの値は宣言済みフラグからのみ指定できます")
		}
		providedName, value, hasValue := argument, "", false
		if separator := strings.IndexByte(argument, '='); separator >= 0 {
			providedName, value, hasValue = argument[:separator], argument[separator+1:], true
		}
		name := providedName
		input, ok := declared[name]
		if !ok {
			canonical, aliasOK := aliases[name]
			if !aliasOK {
				if command.Path == "members list" && name == "--query" {
					return nil, nil, fmt.Errorf("members list は --query に対応していません。cwk members find --room <room-id> --query <text> を使ってください")
				}
				return nil, nil, unknownChatworkFlagError(name, candidates, aliases)
			}
			name = canonical
			input = declared[canonical]
			if _, seen := seenDeprecatedFlags[providedName]; !seen {
				deprecatedFlags = append(deprecatedFlags, deprecatedFlagUse{Alias: providedName, Canonical: canonical})
				seenDeprecatedFlags[providedName] = struct{}{}
			}
		}
		boolean := chatworkBooleanFlag(name)
		if boolean {
			if hasValue {
				return nil, nil, fmt.Errorf("%s は値を受け付けません", providedName)
			}
			value, hasValue = "true", true
		} else if !hasValue {
			if index+1 >= len(args) || strings.HasPrefix(args[index+1], "--") {
				if optionalChatworkFlagValue(command, name) {
					value, hasValue = strconv.Itoa(chatwork.DefaultMessageRelationFetches), true
				} else {
					return nil, nil, fmt.Errorf("%s には値が必要です", providedName)
				}
			} else {
				index++
				value, hasValue = args[index], true
			}
		}
		if !hasValue || value == "" {
			return nil, nil, fmt.Errorf("%s には空でない値が必要です", providedName)
		}
		if len(parsed[name]) > 0 && !input.Repeatable {
			return nil, nil, fmt.Errorf("%s は1回だけ指定できます", name)
		}
		if name != "--confirm" && len(input.AllowedValues) > 0 && !containsExact(input.AllowedValues, value) {
			return nil, nil, fmt.Errorf("%s は %s のいずれかで指定してください", providedName, strings.Join(input.AllowedValues, "、"))
		}
		if input.ReferenceKind != "" {
			if _, err := chatwork.NewReference(chatwork.ReferenceKind(input.ReferenceKind), value); err != nil {
				return nil, nil, invalidChatworkReferenceInput(providedName, input.ReferenceKind)
			}
		}
		parsed[name] = append(parsed[name], value)
	}
	for _, input := range command.Agent.Inputs {
		if input.Required && input.Name != "--confirm" && len(parsed[input.Name]) == 0 {
			return nil, nil, fmt.Errorf("%s は必須です", input.Name)
		}
	}
	for _, group := range command.ExclusiveInputGroups {
		provided := make([]string, 0, len(group))
		for _, name := range group {
			if len(parsed[name]) > 0 {
				provided = append(provided, name)
			}
		}
		if len(provided) != 1 {
			return nil, nil, fmt.Errorf("%s はどちらか一方だけを指定してください", strings.Join(group, " と "))
		}
	}
	return parsed, deprecatedFlags, nil
}

func optionalChatworkFlagValue(command CommandSpec, name string) bool {
	if name != "--resolve-relations" || command.chatwork == nil {
		return false
	}
	return command.chatwork.Task == chatwork.TaskMessagesList || command.chatwork.Task == chatwork.TaskMessagesShow
}

func unknownChatworkFlagError(name string, candidates []string, aliases map[string]string) error {
	suggestion := closestFlag(name, candidates)
	if canonical, deprecated := aliases[suggestion]; deprecated {
		suggestion = canonical
	}
	if suggestion != "" {
		return fmt.Errorf("フラグ %q は不明です。もしかして %q ですか？", name, suggestion)
	}
	return fmt.Errorf("フラグ %q は不明です", name)
}

func invalidChatworkReferenceInput(name, kind string) error {
	if name == "--room" && kind == string(chatwork.ReferenceRoom) {
		return fmt.Errorf("--room には先頭ゼロなし1〜32桁の正の10進 room_id が必要です。rid接頭辞、chatwork-room:接頭辞、URL、ルーム名は受理しません。名前しか分からない場合は cwk rooms list --query <text> を使ってください")
	}
	return fmt.Errorf("%s には正規の %s 参照が必要です", name, kind)
}

func buildChatworkRequest(command CommandSpec, arguments chatworkArguments, now func() time.Time) (chatwork.Request, error) {
	if command.chatwork == nil || !command.chatwork.Task.Valid() {
		return chatwork.Request{}, fmt.Errorf("Chatwork タスク宣言は無効です")
	}
	request := chatwork.Request{Task: command.chatwork.Task}
	if request.Task == chatwork.TaskMessagesList {
		// The public task defaults to the latest bounded provider window. Provider
		// differential state remains available only through explicit
		// --window changes, which the binding loop applies below.
		request.ForceRecent = true
		// The shortest task owns a small exact-message budget so ordinary missing
		// reply chains are self-contained. Zero remains an explicit opt-out.
		request.MessageRelationFetchLimit = chatwork.DefaultMessageRelationFetches
	}
	inputs := make(map[string]CommandInput, len(command.Agent.Inputs))
	for _, input := range command.Agent.Inputs {
		inputs[input.Name] = input
	}

	for name, values := range arguments {
		input := inputs[name]
		if input.ReferenceKind != "" {
			refs := make([]chatwork.Reference, 0, len(values))
			for _, value := range values {
				ref, err := chatwork.NewReference(chatwork.ReferenceKind(input.ReferenceKind), value)
				if err != nil {
					return chatwork.Request{}, fmt.Errorf("%s には正規参照が必要です", name)
				}
				refs = append(refs, ref)
			}
			switch name {
			case "--room":
				request.Room = refs[0]
			case "--message":
				request.Message = refs[0]
			case "--task":
				request.TaskRef = refs[0]
			case "--file":
				request.File = refs[0]
			case "--invite":
				request.Invite = refs[0]
			case "--request":
				request.Request = refs[0]
			case "--account":
				request.Account = refs[0]
			case "--sender":
				request.MessageFilter.Senders = refs
			case "--assigned-by":
				request.AssignedBy = refs[0]
			case "--admin":
				request.Admins = refs
			case "--member":
				request.Members = refs
			case "--readonly":
				request.ReadonlyMembers = refs
			case "--assignee":
				request.Assignees = refs
			default:
				return chatwork.Request{}, fmt.Errorf("Chatwork 参照のバインドには対応していません")
			}
			continue
		}

		value := values[0]
		switch name {
		case "--name":
			request.Name = value
		case "--query":
			switch request.Task {
			case chatwork.TaskRoomsList:
				request.RoomQuery = value
			case chatwork.TaskMembersFind:
				request.MemberQuery = value
			default:
				return chatwork.Request{}, fmt.Errorf("この Chatwork タスクでは --query に対応していません")
			}
		case "--description":
			request.Description = value
			request.DescriptionSet = true
		case "--icon":
			request.Icon = value
		case "--body":
			request.Body = value
		case "--body-file":
			if request.Task != chatwork.TaskMessagesSend && request.Task != chatwork.TaskMessagesUpdate {
				return chatwork.Request{}, fmt.Errorf("この Chatwork タスクでは --body-file に対応していません")
			}
			request.BodyFile = value
		case "--status":
			request.Status = value
		case "--limit":
			switch request.Task {
			case chatwork.TaskRoomTasksCreate:
				limit, err := strconv.ParseInt(value, 10, 64)
				if err != nil || limit <= 0 {
					return chatwork.Request{}, fmt.Errorf("--limit は正の Unix 時刻で指定してください")
				}
				request.Limit = limit
			default:
				return chatwork.Request{}, fmt.Errorf("この Chatwork タスクでは --limit に対応していません")
			}
		case "--start-index":
			if request.Task != chatwork.TaskMessagesList {
				return chatwork.Request{}, fmt.Errorf("この Chatwork タスクでは --start-index に対応していません")
			}
			start, err := strconv.Atoi(value)
			if err == nil && start > chatwork.MaxMessageSelectionCount {
				return chatwork.Request{}, fmt.Errorf("--start-index の値 %d は上限 %d を超えています。このAPIでは100件を超えて遡れません。--since または --until で期間を絞ってください", start, chatwork.MaxMessageSelectionCount)
			}
			if err != nil || start < 1 {
				return chatwork.Request{}, fmt.Errorf("--start-index は 1 から %d までの整数で指定してください", chatwork.MaxMessageSelectionCount)
			}
			request.MessageFilter.StartIndex = start
		case "--count":
			if request.Task != chatwork.TaskMessagesList {
				return chatwork.Request{}, fmt.Errorf("この Chatwork タスクでは --count に対応していません")
			}
			count, err := strconv.Atoi(value)
			if err != nil || count < 1 || count > chatwork.MaxMessageSelectionCount {
				return chatwork.Request{}, fmt.Errorf("--count は 1 から %d までの整数で指定してください", chatwork.MaxMessageSelectionCount)
			}
			request.MessageFilter.Count = count
		case "--resolve-relations":
			if request.Task != chatwork.TaskMessagesList && request.Task != chatwork.TaskMessagesShow {
				return chatwork.Request{}, fmt.Errorf("この Chatwork タスクでは --resolve-relations に対応していません")
			}
			limit, err := strconv.Atoi(value)
			if err != nil || limit < 0 || limit > chatwork.MaxMessageRelationFetches {
				return chatwork.Request{}, fmt.Errorf("--resolve-relations は 0 から %d までの整数で指定してください", chatwork.MaxMessageRelationFetches)
			}
			request.MessageRelationFetchLimit = limit
		case "--limit-type":
			request.LimitType = value
		case "--window":
			request.ForceRecent = value == "recent"
		case "--context":
			if request.Task != chatwork.TaskMessagesList && request.Task != chatwork.TaskMessagesShow {
				return chatwork.Request{}, fmt.Errorf("この Chatwork タスクでは --context に対応していません")
			}
			request.MessageFilter.Context = chatwork.MessageContext(value)
		case "--since", "--until", "--on":
			// Period syntax is normalized together after all flags have been bound
			// so mutually exclusive and two-bound invariants are order independent.
		case "--self-unread":
			request.SelfUnread = true
		case "--create-download-url":
			request.CreateDownloadURL = true
		case "--path":
			request.FilePath = value
		case "--message":
			request.FileMessage = value
		case "--invite-code", "--code":
			request.InviteCode = value
		case "--regenerate-code":
			request.InviteRegenerateCode = true
		case "--invite-approval", "--approval":
			request.InviteApprovalSet = true
			request.InviteNeedsApproval = value == "required"
		case "--confirm":
			// Confirmation is invocation-local policy input, not provider data.
		default:
			return chatwork.Request{}, fmt.Errorf("Chatwork 値のバインドには対応していません")
		}
	}
	if request.Task == chatwork.TaskRoomsCreate && (arguments.first("--invite-code") != "" || arguments.first("--invite-approval") != "") {
		request.InviteEnabled = true
	}
	if request.Task == chatwork.TaskMessagesList &&
		request.MessageFilter.Count > 0 && request.MessageFilter.StartIndex == 0 {
		request.MessageFilter.StartIndex = 1
	}
	if request.Task == chatwork.TaskMessagesList {
		period, periodErr := buildMessagePeriod(arguments, now)
		if periodErr != nil {
			return chatwork.Request{}, periodErr
		}
		request.MessageFilter.Period = period
	}
	if request.Task == chatwork.TaskMessagesList {
		selectionActive := len(request.MessageFilter.Senders) > 0 || request.MessageFilter.Period != (chatwork.MessagePeriod{}) ||
			request.MessageFilter.StartIndex > 0 || request.MessageFilter.Count > 0
		_, contextSpecified := arguments["--context"]
		if contextSpecified && !selectionActive {
			return chatwork.Request{}, fmt.Errorf("--context は単独で指定できません。--sender、--on/--since/--until、--start-index、--count のいずれかと併用してください")
		}
		if selectionActive && request.MessageFilter.Context == "" {
			request.MessageFilter.Context = chatwork.MessageContextReplies
		}
	}
	if request.Task == chatwork.TaskMessagesShow {
		_, contextSpecified := arguments["--context"]
		_, resolutionSpecified := arguments["--resolve-relations"]
		if resolutionSpecified && !contextSpecified {
			request.MessageFilter.Context = chatwork.MessageContextReplies
		}
		if request.MessageFilter.Context == chatwork.MessageContextReplies && !resolutionSpecified {
			request.MessageRelationFetchLimit = chatwork.DefaultMessageRelationFetches
		}
		if request.MessageFilter.Context == chatwork.MessageContextNone && request.MessageRelationFetchLimit > 0 {
			return chatwork.Request{}, fmt.Errorf("--context none は正の --resolve-relations と併用できません。返信関係を取得する場合は --context replies を使ってください")
		}
	}
	if request.Task == chatwork.TaskMessagesSend || request.Task == chatwork.TaskMessagesUpdate {
		_, bodySet := arguments["--body"]
		_, bodyFileSet := arguments["--body-file"]
		if bodySet == bodyFileSet {
			return chatwork.Request{}, fmt.Errorf("--body と --body-file はどちらか一方だけを指定してください")
		}
	}
	return request, nil
}

func invalidChatworkRequestMessage(command CommandSpec, arguments chatworkArguments) string {
	names := make([]string, 0, len(arguments))
	for _, input := range command.Agent.Inputs {
		if len(arguments[input.Name]) > 0 {
			names = append(names, input.Name)
		}
	}
	if len(names) == 0 {
		return "Chatwork コマンドの必須入力または既定値が無効です"
	}
	return fmt.Sprintf("Chatwork 入力 %s の値または併用条件が無効です", strings.Join(names, "、"))
}

func resolveChatworkMessageStdin(ctx context.Context, c *CLI, request *chatwork.Request) error {
	if request == nil || (request.Task != chatwork.TaskMessagesSend && request.Task != chatwork.TaskMessagesUpdate) || request.BodyFile != "" {
		return nil
	}
	if request.Body != "-" {
		return messageBodyInputFault(request.Body)
	}
	if c == nil || c.In == nil {
		return fault.New(fault.KindInvalidInput, "chatwork_message_body_stdin_unreadable", "標準入力からメッセージ本文を読み取れませんでした", false)
	}
	body, err := readContextBounded(ctx, c.In, chatwork.MaxMessageBodyBytes+1)
	if err != nil {
		if ctx != nil && ctx.Err() != nil {
			return fault.Wrap(fault.KindCanceled, "operation_canceled", "標準入力からのメッセージ本文読み取りをキャンセルしました", true, ctx.Err())
		}
		return fault.New(fault.KindInvalidInput, "chatwork_message_body_stdin_unreadable", "標準入力からメッセージ本文を読み取れませんでした", false)
	}
	request.Body = string(body)
	return messageBodyInputFault(request.Body)
}

func messageBodyInputFault(body string) error {
	if len(body) > chatwork.MaxMessageBodyBytes {
		return fault.New(fault.KindInvalidInput, "chatwork_message_body_too_large", "メッセージ本文が 65535 バイトの上限を超えています", false)
	}
	if err := chatwork.ValidateMessageBody(body); err != nil {
		return fault.New(fault.KindInvalidInput, "chatwork_message_body_invalid", "メッセージ本文は空でない65535バイト以下の有効なUTF-8で指定してください", false)
	}
	return nil
}

func buildMessagePeriod(arguments chatworkArguments, now func() time.Time) (chatwork.MessagePeriod, error) {
	sinceValue := arguments.first("--since")
	untilValue := arguments.first("--until")
	onValue := arguments.first("--on")
	if onValue != "" && (sinceValue != "" || untilValue != "") {
		return chatwork.MessagePeriod{}, fmt.Errorf("--on は --since または --until と同時に指定できません")
	}
	if onValue != "" {
		day := onValue
		switch onValue {
		case "today", "yesterday":
			if now == nil {
				return chatwork.MessagePeriod{}, fmt.Errorf("相対日の解決に必要なコマンド時計が設定されていません")
			}
			current := now()
			if current.IsZero() {
				return chatwork.MessagePeriod{}, fmt.Errorf("相対日の解決に必要なコマンド時計が無効です")
			}
			location := time.FixedZone(chatwork.MessageDayTimeZone, 9*60*60)
			current = current.In(location)
			if onValue == "yesterday" {
				current = current.AddDate(0, 0, -1)
			}
			day = current.Format("2006-01-02")
		}
		period, err := chatwork.NewMessageDayPeriod(day)
		if err != nil {
			return chatwork.MessagePeriod{}, fmt.Errorf("--on は YYYY-MM-DD、today、yesterday のいずれかで指定してください")
		}
		return period, nil
	}

	since, err := parseMessagePeriodBound("--since", sinceValue)
	if err != nil {
		return chatwork.MessagePeriod{}, err
	}
	until, err := parseMessagePeriodBound("--until", untilValue)
	if err != nil {
		return chatwork.MessagePeriod{}, err
	}
	period, err := chatwork.NewMessagePeriod(since, until)
	if err != nil {
		return chatwork.MessagePeriod{}, fmt.Errorf("--since と --until は空でない昇順の期間を指定してください")
	}
	return period, nil
}

func parseMessagePeriodBound(name, value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || parsed.Nanosecond() != 0 || parsed.Unix() <= 0 {
		return 0, fmt.Errorf("%s は明示的なオフセットを持つ秒精度の RFC3339 時刻で指定してください", name)
	}
	return parsed.Unix(), nil
}

func buildChatworkExecutionRequest(command CommandSpec, base operation.Intent, arguments chatworkArguments) (execution.Request, error) {
	if command.Agent.Mutation == nil || command.chatwork == nil {
		return execution.Request{}, fmt.Errorf("Chatwork 変更宣言は無効です")
	}
	mutation := *command.Agent.Mutation
	intent := base
	intent.Target = operation.TargetRef{Kind: mutation.TargetKind}
	intent.Impact = mutation.Impact
	if command.Effect == operation.EffectCreate {
		intent.Target.ParentID = arguments.first(mutation.ParentInput)
	} else if command.Effect == operation.EffectWrite {
		intent.Target.ID = arguments.first(mutation.TargetIDInput)
		if mutation.ParentInput != "" {
			intent.Target.ParentID = arguments.first(mutation.ParentInput)
		}
	} else {
		return execution.Request{}, fmt.Errorf("Chatwork 変更の効果は無効です")
	}
	request := execution.Request{
		Intent:          intent,
		ExpectedCommand: command.Path,
		ExpectedEffect:  command.Effect,
		ExpectedTarget:  intent.Target,
		ExpectedImpact:  mutation.Impact,
	}
	return request, nil
}

func chatworkBooleanFlag(name string) bool {
	return name == "--self-unread" || name == "--create-download-url" || name == "--regenerate-code"
}

func containsExact(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func unclassifiedMutationServiceError(err error) bool {
	public, ok := fault.PublicCopy(err)
	if !ok {
		return err != nil
	}
	return public.Code == "unclassified_chatwork_error" || public.Code == "operation_canceled"
}

func operationCanceledServiceError(err error) bool {
	public, ok := fault.PublicCopy(err)
	return ok && public.Code == "operation_canceled"
}
