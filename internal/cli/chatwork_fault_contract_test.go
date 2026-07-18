package cli

import (
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

type faultSignature struct {
	kind      fault.Kind
	retryable bool
}

func TestChatworkPublicFaultCodesHaveOneExactSignature(t *testing.T) {
	specs := append(chatworkAuthCommandSpecs(), chatworkCommandSpecs()...)
	seen := make(map[string]faultSignature)
	for _, spec := range specs {
		for _, declared := range spec.Agent.Errors {
			got := faultSignature{kind: declared.Kind, retryable: declared.Retryable}
			if previous, exists := seen[declared.Code]; exists && previous != got {
				t.Errorf("fault %q has conflicting signatures: %+v and %+v", declared.Code, previous, got)
			}
			seen[declared.Code] = got
		}
	}
}

func TestChatworkMutationRecoveryNeverReplaysAWrite(t *testing.T) {
	specs := append(chatworkAuthCommandSpecs(), chatworkCommandSpecs()...)
	byPath := make(map[string]CommandSpec, len(specs))
	for _, spec := range specs {
		byPath[spec.Path] = spec
	}
	for _, spec := range specs {
		if spec.Effect == operation.EffectRead {
			continue
		}
		for _, declared := range spec.Agent.Errors {
			for _, next := range declared.NextActions {
				if next.Command == "help" || strings.HasPrefix(next.Command, "help ") {
					continue
				}
				target, exists := byPath[next.Command]
				if !exists {
					t.Errorf("%s fault %s has unknown recovery %q", spec.Path, declared.Code, next.Command)
					continue
				}
				if target.Effect != operation.EffectRead {
					t.Errorf("%s fault %s recovers through mutating command %q", spec.Path, declared.Code, next.Command)
				}
			}
		}
	}
}

func TestChatworkAPIFaultsCoverSelectionPATAndOAuth(t *testing.T) {
	want := map[string]faultSignature{
		"chatwork_auth_method_missing":            {fault.KindAuthentication, false},
		"chatwork_auth_method_invalid":            {fault.KindAuthentication, false},
		"chatwork_token_missing":                  {fault.KindAuthentication, false},
		"chatwork_token_invalid":                  {fault.KindAuthentication, false},
		"oauth_client_configuration_missing":      {fault.KindInvalidInput, false},
		"oauth_client_configuration_invalid":      {fault.KindInvalidInput, false},
		"oauth_configuration_invalid":             {fault.KindInvalidInput, false},
		"oauth_credential_missing":                {fault.KindAuthentication, false},
		"oauth_credential_store_unavailable":      {fault.KindUnavailable, true},
		"oauth_refresh_failed":                    {fault.KindAuthentication, false},
		"oauth_credential_too_large":              {fault.KindContract, false},
		"oauth_identity_request_invalid":          {fault.KindContract, false},
		"oauth_identity_verification_unavailable": {fault.KindUnavailable, true},
		"oauth_identity_response_invalid":         {fault.KindContract, false},
		"oauth_identity_verification_failed":      {fault.KindAuthentication, false},
	}
	for _, spec := range chatworkCommandSpecs() {
		declared := commandFaultSignatures(spec)
		for code, expected := range want {
			if got, exists := declared[code]; !exists || got != expected {
				t.Errorf("%s fault %s = %+v, present=%t; want %+v", spec.Path, code, got, exists, expected)
			}
		}
	}
}

func TestChatworkTaskSpecificFaultsAreNotAdvertisedElsewhere(t *testing.T) {
	for _, spec := range chatworkCommandSpecs() {
		declared := commandFaultSignatures(spec)
		wantMutation := spec.Effect != operation.EffectRead
		_, hasUnknown := declared["chatwork_mutation_outcome_unknown"]
		if hasUnknown != wantMutation {
			t.Errorf("%s mutation outcome fault present=%t, want %t", spec.Path, hasUnknown, wantMutation)
		}
		_, hasUnavailable := declared["chatwork_unavailable"]
		if hasUnavailable != !wantMutation {
			t.Errorf("%s provider unavailable fault present=%t, want %t", spec.Path, hasUnavailable, !wantMutation)
		}

		wantNotation := spec.chatwork.Task == chatwork.TaskMessagesList || spec.chatwork.Task == chatwork.TaskMessagesShow
		_, hasNotation := declared["chatwork_notation_malformed"]
		if hasNotation != wantNotation {
			t.Errorf("%s notation fault present=%t, want %t", spec.Path, hasNotation, wantNotation)
		}

		wantUpload := spec.chatwork.Task == chatwork.TaskFilesUpload
		for _, code := range []string{"chatwork_file_name_invalid", "chatwork_file_unreadable", "chatwork_file_too_large", "chatwork_upload_contract_invalid"} {
			_, exists := declared[code]
			if exists != wantUpload {
				t.Errorf("%s upload fault %s present=%t, want %t", spec.Path, code, exists, wantUpload)
			}
		}
		if _, invented := declared["chatwork_file_invalid"]; invented {
			t.Errorf("%s advertises non-emitted chatwork_file_invalid", spec.Path)
		}
	}
}

func TestChatworkOAuthLifecycleFaultsMatchReachablePhases(t *testing.T) {
	byPath := make(map[string]map[string]faultSignature)
	for _, spec := range chatworkAuthCommandSpecs() {
		byPath[spec.Path] = commandFaultSignatures(spec)
	}

	assertFaultSignature(t, byPath["auth login"], "oauth_authorization_denied", fault.KindRejected, false)
	assertFaultSignature(t, byPath["auth login"], "oauth_credential_store_unavailable", fault.KindUnavailable, true)
	assertFaultSignature(t, byPath["auth login"], "oauth_identity_verification_unavailable", fault.KindUnavailable, true)
	assertFaultSignature(t, byPath["auth status"], "oauth_credential_store_unavailable", fault.KindUnavailable, true)
	assertFaultSignature(t, byPath["auth logout"], "oauth_credential_store_unavailable", fault.KindUnavailable, true)

	for _, code := range []string{"oauth_redirect_uri_invalid", "oauth_credential_not_found"} {
		for path, declared := range byPath {
			if _, exists := declared[code]; exists {
				t.Errorf("%s advertises non-emitted fault %s", path, code)
			}
		}
	}
	for _, code := range []string{"oauth_callback_missing", "oauth_redirect_invalid", "oauth_state_mismatch", "oauth_code_exchange_failed", "oauth_identity_response_invalid"} {
		if _, exists := byPath["auth login"][code]; !exists {
			t.Errorf("auth login is missing phase fault %s", code)
		}
		for _, path := range []string{"auth profiles", "auth status", "auth logout"} {
			if _, exists := byPath[path][code]; exists {
				t.Errorf("%s advertises login-only fault %s", path, code)
			}
		}
	}
	for _, path := range []string{"auth login", "auth logout"} {
		var unknown *CommandError
		for _, spec := range chatworkAuthCommandSpecs() {
			if spec.Path != path {
				continue
			}
			for index := range spec.Agent.Errors {
				if spec.Agent.Errors[index].Code == "unclassified_mutation_outcome" {
					unknown = &spec.Agent.Errors[index]
				}
			}
		}
		if unknown == nil || unknown.Kind != fault.KindContract || unknown.Retryable || len(unknown.NextActions) != 1 || unknown.NextActions[0].Command != "auth status" {
			t.Errorf("%s unknown outcome contract = %+v, want non-retryable contract reconciliation through auth status", path, unknown)
		}
	}
}

func commandFaultSignatures(spec CommandSpec) map[string]faultSignature {
	result := make(map[string]faultSignature, len(spec.Agent.Errors))
	for _, declared := range spec.Agent.Errors {
		result[declared.Code] = faultSignature{kind: declared.Kind, retryable: declared.Retryable}
	}
	return result
}

func assertFaultSignature(t *testing.T, declared map[string]faultSignature, code string, kind fault.Kind, retryable bool) {
	t.Helper()
	want := faultSignature{kind: kind, retryable: retryable}
	if got, exists := declared[code]; !exists || got != want {
		t.Errorf("fault %s = %+v, present=%t; want %+v", code, got, exists, want)
	}
}
