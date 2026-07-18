package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/execution"
	"github.com/tasuku43/cwk/internal/cli/capsule"
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
	arguments, err := parseChatworkArguments(command, args)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error()+"; usage: "+command.Usage(), "help "+command.Path, "Correct the declared command inputs.")
	}
	request, err := buildChatworkRequest(command, arguments)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error()+"; usage: "+command.Usage(), "help "+command.Path, "Correct the declared command inputs.")
	}
	if err := request.Validate(); err != nil {
		return c.failUsage(ctx, "invalid_arguments", "One or more Chatwork inputs are invalid; usage: "+command.Usage(), "help "+command.Path, "Correct the declared command inputs.")
	}

	var result chatwork.Result
	authenticatedAction := func(actionContext context.Context, session domainauthn.Session) error {
		if c.chatwork == nil {
			return fault.New(fault.KindContract, "missing_chatwork_port", "Chatwork task adapter is not configured", false)
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
			return c.failUsage(ctx, "invalid_arguments", buildErr.Error()+"; usage: "+command.Usage(), "help "+command.Path, "Correct the declared command inputs.")
		}
		policy := chatworkcmd.ConfirmationPolicy{
			Required: command.chatwork.Confirmation,
			Provided: arguments.first("--confirm"),
		}
		err = execution.New(policy).Invoke(ctx, executionRequest, func(actionContext context.Context, _ operation.Intent) error {
			actionErr := authenticated(actionContext)
			if unclassifiedMutationServiceError(actionErr) {
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

	output, err := capsule.Render(result)
	if err != nil {
		return c.fail(ctx, fault.New(fault.KindContract, "output_encoding_failed", "The Chatwork task projection could not be encoded.", false))
	}
	if len(output) > maxChatworkOutputBytes {
		return c.fail(ctx, fault.New(fault.KindContract, "output_contract_exceeded", "The Chatwork result exceeds the declared output bound.", false))
	}
	return c.emit(ctx, []byte(output))
}

type chatworkArguments map[string][]string

func (a chatworkArguments) first(name string) string {
	values := a[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func parseChatworkArguments(command CommandSpec, args []string) (chatworkArguments, error) {
	declared := make(map[string]CommandInput, len(command.Agent.Inputs))
	for _, input := range command.Agent.Inputs {
		if input.Source != InputSourceFlag || !strings.HasPrefix(input.Name, "--") {
			return nil, fmt.Errorf("Chatwork command has an unsupported input contract")
		}
		declared[input.Name] = input
	}

	parsed := make(chatworkArguments, len(declared))
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if !strings.HasPrefix(argument, "--") {
			return nil, fmt.Errorf("Chatwork command accepts values only through declared flags")
		}
		name, value, hasValue := argument, "", false
		if separator := strings.IndexByte(argument, '='); separator >= 0 {
			name, value, hasValue = argument[:separator], argument[separator+1:], true
		}
		input, ok := declared[name]
		if !ok {
			return nil, fmt.Errorf("unknown flag %q", name)
		}
		boolean := chatworkBooleanFlag(name)
		if boolean {
			if hasValue {
				return nil, fmt.Errorf("%s does not accept a value", name)
			}
			value, hasValue = "true", true
		} else if !hasValue {
			if index+1 >= len(args) || strings.HasPrefix(args[index+1], "--") {
				return nil, fmt.Errorf("%s requires a value", name)
			}
			index++
			value, hasValue = args[index], true
		}
		if !hasValue || value == "" {
			return nil, fmt.Errorf("%s requires a non-empty value", name)
		}
		if len(parsed[name]) > 0 && !input.Repeatable {
			return nil, fmt.Errorf("%s may be specified only once", name)
		}
		if name != "--confirm" && len(input.AllowedValues) > 0 && !containsExact(input.AllowedValues, value) {
			return nil, fmt.Errorf("%s must be one of %s", name, strings.Join(input.AllowedValues, " or "))
		}
		if input.ReferenceKind != "" {
			if _, err := chatwork.NewReference(chatwork.ReferenceKind(input.ReferenceKind), value); err != nil {
				return nil, fmt.Errorf("%s requires a canonical %s reference", name, input.ReferenceKind)
			}
		}
		parsed[name] = append(parsed[name], value)
	}
	for _, input := range command.Agent.Inputs {
		if input.Required && input.Name != "--confirm" && len(parsed[input.Name]) == 0 {
			return nil, fmt.Errorf("%s is required", input.Name)
		}
	}
	return parsed, nil
}

func buildChatworkRequest(command CommandSpec, arguments chatworkArguments) (chatwork.Request, error) {
	if command.chatwork == nil || !command.chatwork.Task.Valid() {
		return chatwork.Request{}, fmt.Errorf("Chatwork task declaration is invalid")
	}
	request := chatwork.Request{Task: command.chatwork.Task}
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
					return chatwork.Request{}, fmt.Errorf("%s requires a canonical reference", name)
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
			case "--account", "--owner":
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
				return chatwork.Request{}, fmt.Errorf("Chatwork reference binding is unsupported")
			}
			continue
		}

		value := values[0]
		switch name {
		case "--name":
			request.Name = value
		case "--description":
			request.Description = value
		case "--icon":
			request.Icon = value
		case "--body":
			request.Body = value
		case "--status":
			request.Status = value
		case "--limit":
			switch request.Task {
			case chatwork.TaskMessagesList:
				limit, err := strconv.Atoi(value)
				if err != nil || limit < 1 || limit > chatwork.MaxMessageSelectionLimit {
					return chatwork.Request{}, fmt.Errorf("--limit must be an integer from 1 to %d", chatwork.MaxMessageSelectionLimit)
				}
				request.MessageFilter.Limit = limit
			case chatwork.TaskRoomTasksCreate:
				limit, err := strconv.ParseInt(value, 10, 64)
				if err != nil || limit <= 0 {
					return chatwork.Request{}, fmt.Errorf("--limit must be a positive Unix time")
				}
				request.Limit = limit
			default:
				return chatwork.Request{}, fmt.Errorf("--limit is unsupported for this Chatwork task")
			}
		case "--limit-type":
			request.LimitType = value
		case "--window":
			request.ForceRecent = value == "recent"
		case "--context":
			request.MessageFilter.Context = chatwork.MessageContext(value)
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
		case "--invite-approval", "--approval":
			request.InviteApprovalSet = true
			request.InviteNeedsApproval = value == "required"
		case "--confirm":
			// Confirmation is invocation-local policy input, not provider data.
		default:
			return chatwork.Request{}, fmt.Errorf("Chatwork value binding is unsupported")
		}
	}
	if request.Task == chatwork.TaskRoomsCreate && (arguments.first("--invite-code") != "" || arguments.first("--invite-approval") != "") {
		request.InviteEnabled = true
	}
	if request.Task == chatwork.TaskMessagesList &&
		(len(request.MessageFilter.Senders) > 0 || request.MessageFilter.Limit > 0) &&
		request.MessageFilter.Context == "" {
		request.MessageFilter.Context = chatwork.MessageContextNone
	}
	return request, nil
}

func buildChatworkExecutionRequest(command CommandSpec, base operation.Intent, arguments chatworkArguments) (execution.Request, error) {
	if command.Agent.Mutation == nil || command.chatwork == nil {
		return execution.Request{}, fmt.Errorf("Chatwork mutation declaration is invalid")
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
		return execution.Request{}, fmt.Errorf("Chatwork mutation effect is invalid")
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
	return name == "--self-unread" || name == "--create-download-url"
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
