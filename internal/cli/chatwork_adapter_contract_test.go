package cli

import (
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/apicall"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
)

type chatworkTaskContract struct {
	path   string
	task   chatwork.Task
	effect operation.Effect
	role   CommandRole
}

func TestChatworkAdapterContractBindsEveryTypedTask(t *testing.T) {
	want := []chatworkTaskContract{
		{"account show", chatwork.TaskAccountShow, operation.EffectRead, RoleDiscover},
		{"account status", chatwork.TaskAccountStatus, operation.EffectRead, RoleUtility},
		{"personal-tasks list", chatwork.TaskPersonalTasksList, operation.EffectRead, RoleDiscover},
		{"contacts list", chatwork.TaskContactsList, operation.EffectRead, RoleDiscover},
		{"rooms list", chatwork.TaskRoomsList, operation.EffectRead, RoleDiscover},
		{"rooms create", chatwork.TaskRoomsCreate, operation.EffectCreate, RoleAct},
		{"rooms show", chatwork.TaskRoomsShow, operation.EffectRead, RoleAct},
		{"rooms update", chatwork.TaskRoomsUpdate, operation.EffectWrite, RoleAct},
		{"rooms leave", chatwork.TaskRoomsLeave, operation.EffectWrite, RoleAct},
		{"rooms delete", chatwork.TaskRoomsDelete, operation.EffectWrite, RoleAct},
		{"members list", chatwork.TaskMembersList, operation.EffectRead, RoleAct},
		{"members replace", chatwork.TaskMembersReplace, operation.EffectWrite, RoleAct},
		{"messages list", chatwork.TaskMessagesList, operation.EffectRead, RoleAct},
		{"messages send", chatwork.TaskMessagesSend, operation.EffectCreate, RoleAct},
		{"messages mark-read", chatwork.TaskMessagesMarkRead, operation.EffectWrite, RoleAct},
		{"messages mark-unread", chatwork.TaskMessagesMarkUnread, operation.EffectWrite, RoleAct},
		{"messages show", chatwork.TaskMessagesShow, operation.EffectRead, RoleAct},
		{"messages update", chatwork.TaskMessagesUpdate, operation.EffectWrite, RoleAct},
		{"messages delete", chatwork.TaskMessagesDelete, operation.EffectWrite, RoleAct},
		{"room-tasks list", chatwork.TaskRoomTasksList, operation.EffectRead, RoleAct},
		{"room-tasks create", chatwork.TaskRoomTasksCreate, operation.EffectCreate, RoleAct},
		{"room-tasks show", chatwork.TaskRoomTasksShow, operation.EffectRead, RoleAct},
		{"room-tasks set-status", chatwork.TaskRoomTasksSetStatus, operation.EffectWrite, RoleAct},
		{"files list", chatwork.TaskFilesList, operation.EffectRead, RoleAct},
		{"files upload", chatwork.TaskFilesUpload, operation.EffectCreate, RoleAct},
		{"files show", chatwork.TaskFilesShow, operation.EffectRead, RoleAct},
		{"invite-link show", chatwork.TaskInviteLinkShow, operation.EffectRead, RoleAct},
		{"invite-link create", chatwork.TaskInviteLinkCreate, operation.EffectCreate, RoleAct},
		{"invite-link update", chatwork.TaskInviteLinkUpdate, operation.EffectWrite, RoleAct},
		{"invite-link delete", chatwork.TaskInviteLinkDelete, operation.EffectWrite, RoleAct},
		{"contact-requests list", chatwork.TaskContactRequestsList, operation.EffectRead, RoleDiscover},
		{"contact-requests accept", chatwork.TaskContactRequestsAccept, operation.EffectWrite, RoleAct},
		{"contact-requests reject", chatwork.TaskContactRequestsReject, operation.EffectWrite, RoleAct},
	}

	specs := chatworkCommandSpecs()
	if len(specs) != len(want) {
		t.Fatalf("Chatwork command specs = %d, want %d", len(specs), len(want))
	}
	byPath := make(map[string]CommandSpec, len(specs))
	for _, spec := range specs {
		if _, exists := byPath[spec.Path]; exists {
			t.Fatalf("Chatwork command path %q is bound more than once", spec.Path)
		}
		byPath[spec.Path] = spec
	}

	seenTasks := make(map[chatwork.Task]string, len(want))
	for _, expected := range want {
		expected := expected
		t.Run(expected.path, func(t *testing.T) {
			spec, exists := byPath[expected.path]
			if !exists {
				t.Fatalf("Chatwork command %q is missing", expected.path)
			}
			if spec.chatwork == nil {
				t.Fatal("Chatwork task binding is missing")
			}
			if spec.chatwork.Task != expected.task {
				t.Fatalf("task = %q, want %q", spec.chatwork.Task, expected.task)
			}
			if !spec.chatwork.Task.Valid() {
				t.Fatalf("task %q is not a valid typed Chatwork task", spec.chatwork.Task)
			}
			if previous, exists := seenTasks[spec.chatwork.Task]; exists {
				t.Fatalf("task %q is also bound by %q", spec.chatwork.Task, previous)
			}
			seenTasks[spec.chatwork.Task] = spec.Path
			if spec.Effect != expected.effect {
				t.Errorf("effect = %s, want %s", spec.Effect, expected.effect)
			}
			if spec.Role != expected.role {
				t.Errorf("role = %s, want %s", spec.Role, expected.role)
			}
			assertChatworkAuthenticationContract(t, spec)
			assertChatworkCallPolicy(t, spec)
			assertChatworkReconciliationContract(t, spec, byPath)
			assertChatworkInfraFaultContract(t, spec)
		})
	}
}

func TestChatworkCorrectnessCriticalCatalogFields(t *testing.T) {
	want := map[string][]string{
		"personal-tasks list":     {"task_ref", "room_ref", "assigned_by_ref", "message_ref", "body", "status", "limit", "complete"},
		"contacts list":           {"account_ref", "room_ref", "name", "organization", "complete"},
		"rooms list":              {"room_ref", "name", "type", "role", "unread", "mentions", "tasks", "complete"},
		"members list":            {"account_ref", "name", "role", "complete"},
		"messages list":           {"message_ref", "room_ref", "sender_ref", "sender_name", "body", "send_time", "relations", "sequence", "actor_alias", "window", "limit", "complete", "unresolved_relations", "source_count", "filter_senders", "filter_context", "anchor_sequences"},
		"messages show":           {"message_ref", "room_ref", "sender_ref", "sender_name", "body", "send_time", "relations"},
		"messages send":           {"message_ref", "room_ref"},
		"room-tasks create":       {"task_ref", "room_ref"},
		"room-tasks list":         {"task_ref", "room_ref", "account_ref", "message_ref", "body", "status", "limit_time", "limit", "complete"},
		"files list":              {"file_ref", "room_ref", "account_ref", "message_ref", "name", "size", "limit", "complete"},
		"files upload":            {"file_ref", "room_ref"},
		"messages mark-read":      {"unread", "mentions"},
		"messages mark-unread":    {"unread", "mentions"},
		"rooms leave":             {"room_ref"},
		"rooms delete":            {"room_ref"},
		"contact-requests reject": {"request_ref"},
		"contact-requests list":   {"request_ref", "account_ref", "name", "message", "limit", "complete"},
		"members replace":         {"administrators", "members", "readonly"},
	}
	for _, spec := range chatworkCommandSpecs() {
		expected, applies := want[spec.Path]
		if !applies {
			continue
		}
		got := make([]string, len(spec.Agent.Output.Fields))
		for index, field := range spec.Agent.Output.Fields {
			got[index] = field.Name
		}
		if len(got) != len(expected) {
			t.Errorf("%s fields = %v, want %v", spec.Path, got, expected)
			continue
		}
		for index := range expected {
			if got[index] != expected[index] {
				t.Errorf("%s fields = %v, want %v", spec.Path, got, expected)
				break
			}
		}
		delete(want, spec.Path)
	}
	if len(want) != 0 {
		t.Fatalf("critical catalog commands are missing: %v", want)
	}
}

func TestFilesListCatalogExplainsPositionalRoundTripAndAbsentMessage(t *testing.T) {
	var files CommandSpec
	for _, spec := range chatworkCommandSpecs() {
		if spec.Path == "files list" {
			files = spec
			break
		}
	}
	if files.Path == "" {
		t.Fatal("files list catalog entry is missing")
	}
	if !strings.Contains(files.Agent.Outcome, "positions one and two unchanged to files show") {
		t.Fatalf("files list outcome does not explain its direct next action: %q", files.Agent.Outcome)
	}
	descriptions := map[string]string{}
	for _, field := range files.Agent.Output.Fields {
		descriptions[field.Name] = field.Description
	}
	for field, want := range map[string]string{
		"file_ref":    "position one; pass it unchanged to files show --file",
		"room_ref":    "position two; pass it unchanged to files show --room",
		"message_ref": "literal absent; never pass absent as a reference",
	} {
		if !strings.Contains(descriptions[field], want) {
			t.Errorf("files list %s description = %q, want phrase %q", field, descriptions[field], want)
		}
	}
}

func assertChatworkAuthenticationContract(t *testing.T, spec CommandSpec) {
	t.Helper()
	requirement := spec.Agent.Authentication
	if requirement == nil {
		t.Fatal("Chatwork authentication requirement is missing")
	}
	if err := requirement.Validate(); err != nil {
		t.Fatalf("Chatwork authentication requirement is invalid: %v", err)
	}
	if len(requirement.Methods) != 1 || requirement.Methods[0] != authn.MethodPAT {
		t.Errorf("authentication methods = %v, want [pat]", requirement.Methods)
	}
	if requirement.Authority != chatwork.AuthenticationAuthority {
		t.Errorf("authentication authority = %q, want %q", requirement.Authority, chatwork.AuthenticationAuthority)
	}
	if requirement.Audience != chatwork.AuthenticationAudience {
		t.Errorf("authentication audience = %q, want %q", requirement.Audience, chatwork.AuthenticationAudience)
	}
	if requirement.AccountID != "" {
		t.Errorf("authentication account binding = %q, want the configured single account", requirement.AccountID)
	}
	if len(requirement.RequiredCapabilities) != 1 || requirement.RequiredCapabilities[0] != chatwork.AuthenticationCapability {
		t.Errorf("authentication capabilities = %v, want [%s]", requirement.RequiredCapabilities, chatwork.AuthenticationCapability)
	}
}

func assertChatworkCallPolicy(t *testing.T, spec CommandSpec) {
	t.Helper()
	policy, err := chatworkapi.CallPolicy(spec.chatwork.Task)
	if err != nil {
		t.Fatalf("adapter has no valid call policy: %v", err)
	}
	if err := policy.Validate(spec.Effect); err != nil {
		t.Fatalf("adapter call policy does not validate for %s: %v", spec.Effect, err)
	}
	if policy.MaxAttempts != chatworkapi.MaxAttempts {
		t.Errorf("adapter attempts = %d, want %d", policy.MaxAttempts, chatworkapi.MaxAttempts)
	}
	wantTimeout := chatworkapi.RequestTimeout
	if spec.chatwork.Task == chatwork.TaskFilesUpload {
		wantTimeout = chatworkapi.UploadTimeout
	}
	if policy.Timeout != wantTimeout {
		t.Errorf("adapter timeout = %s, want %s", policy.Timeout, wantTimeout)
	}
	wantIdempotency := apicall.IdempotencySafe
	if spec.Effect != operation.EffectRead {
		wantIdempotency = apicall.IdempotencyUnsafe
	}
	if policy.Idempotency != wantIdempotency {
		t.Errorf("adapter idempotency = %s, want %s", policy.Idempotency, wantIdempotency)
	}
}

func assertChatworkReconciliationContract(t *testing.T, spec CommandSpec, byPath map[string]CommandSpec) {
	t.Helper()
	if spec.Effect == operation.EffectRead {
		if spec.Agent.Mutation != nil || spec.chatwork.Reconcile != "" {
			t.Fatal("read task declares mutation reconciliation")
		}
		return
	}
	if spec.Agent.Mutation == nil {
		t.Fatal("mutation contract is missing")
	}
	if spec.chatwork.Reconcile == "" {
		t.Fatal("mutation reconciliation task is missing")
	}
	reconciliation, exists := byPath[spec.chatwork.Reconcile]
	if !exists {
		t.Fatalf("reconciliation task %q is not a Chatwork command", spec.chatwork.Reconcile)
	}
	if reconciliation.Effect != operation.EffectRead || reconciliation.Agent.Mutation != nil {
		t.Fatalf("reconciliation task %q is not read-only", reconciliation.Path)
	}

	unknownCodes := map[string]struct{}{
		"unclassified_mutation_outcome":     {},
		"chatwork_mutation_outcome_unknown": {},
	}
	targets := make(map[string]struct{})
	for _, declared := range spec.Agent.Errors {
		if _, applies := unknownCodes[declared.Code]; !applies {
			continue
		}
		delete(unknownCodes, declared.Code)
		if len(declared.NextActions) != 1 {
			t.Errorf("fault %q next actions = %d, want 1", declared.Code, len(declared.NextActions))
			continue
		}
		targets[declared.NextActions[0].Command] = struct{}{}
	}
	if len(unknownCodes) != 0 {
		t.Errorf("mutation outcome faults are missing: %v", unknownCodes)
	}
	if len(targets) != 1 {
		t.Fatalf("mutation has %d reconciliation targets, want 1", len(targets))
	}
	if _, exists := targets[spec.chatwork.Reconcile]; !exists {
		t.Errorf("mutation outcome faults do not reconcile through %q", spec.chatwork.Reconcile)
	}
}

type chatworkFaultContract struct {
	code      string
	kind      fault.Kind
	retryable bool
	applies   func(CommandSpec) bool
}

func assertChatworkInfraFaultContract(t *testing.T, spec CommandSpec) {
	t.Helper()
	all := func(CommandSpec) bool { return true }
	mutation := func(spec CommandSpec) bool { return spec.Effect != operation.EffectRead }
	read := func(spec CommandSpec) bool { return spec.Effect == operation.EffectRead }
	upload := func(spec CommandSpec) bool { return spec.chatwork.Task == chatwork.TaskFilesUpload }
	messageRead := func(spec CommandSpec) bool {
		return spec.chatwork.Task == chatwork.TaskMessagesList || spec.chatwork.Task == chatwork.TaskMessagesShow
	}

	want := []chatworkFaultContract{
		{"missing_authentication_context", fault.KindContract, false, all},
		{"authentication_canceled", fault.KindCanceled, false, all},
		{"invalid_authentication_requirement", fault.KindContract, false, all},
		{"missing_authenticator", fault.KindAuthentication, false, all},
		{"authentication_context_mismatch", fault.KindAuthentication, false, all},
		{"insufficient_authentication_capability", fault.KindPermission, false, all},
		{"chatwork_token_missing", fault.KindAuthentication, false, all},
		{"chatwork_token_invalid", fault.KindAuthentication, false, all},
		{"authentication_failed", fault.KindAuthentication, false, all},
		{"invalid_authentication_session", fault.KindAuthentication, false, all},
		{"invalid_authentication_binding", fault.KindAuthentication, false, all},
		{"missing_context", fault.KindContract, false, all},
		{"operation_canceled", fault.KindCanceled, true, all},
		{"invalid_chatwork_request", fault.KindInvalidInput, false, all},
		{"chatwork_request_contract_invalid", fault.KindContract, false, all},
		{"chatwork_transport_missing", fault.KindContract, false, all},
		{"chatwork_response_invalid", fault.KindContract, false, all},
		{"chatwork_unexpected_response", fault.KindContract, false, all},
		{"chatwork_invalid_request", fault.KindInvalidInput, false, all},
		{"chatwork_authentication_failed", fault.KindAuthentication, false, all},
		{"chatwork_permission_denied", fault.KindPermission, false, all},
		{"chatwork_not_found", fault.KindNotFound, false, all},
		{"chatwork_rate_limited", fault.KindRateLimited, true, all},
		{"chatwork_response_unavailable", fault.KindUnavailable, true, all},
		{"chatwork_response_too_large", fault.KindContract, false, all},
		{"chatwork_response_malformed", fault.KindContract, false, all},
		{"chatwork_response_unmapped", fault.KindContract, false, all},
		{"chatwork_result_invalid", fault.KindContract, false, all},
		{"chatwork_unavailable", fault.KindUnavailable, true, read},
		{"chatwork_mutation_outcome_unknown", fault.KindContract, false, mutation},
		{"chatwork_notation_malformed", fault.KindContract, false, messageRead},
		{"chatwork_file_unreadable", fault.KindInvalidInput, false, upload},
		{"chatwork_file_too_large", fault.KindInvalidInput, false, upload},
		{"chatwork_file_name_invalid", fault.KindInvalidInput, false, upload},
		{"chatwork_upload_contract_invalid", fault.KindContract, false, upload},
	}

	declared := make(map[string]CommandError, len(spec.Agent.Errors))
	for _, candidate := range spec.Agent.Errors {
		declared[candidate.Code] = candidate
	}
	for _, expected := range want {
		if !expected.applies(spec) {
			continue
		}
		candidate, exists := declared[expected.code]
		if !exists {
			t.Errorf("infra fault %q is not declared", expected.code)
			continue
		}
		if candidate.Kind != expected.kind || candidate.Retryable != expected.retryable {
			t.Errorf("infra fault %q = (%s, retryable=%t), want (%s, retryable=%t)", expected.code, candidate.Kind, candidate.Retryable, expected.kind, expected.retryable)
		}
	}
}
