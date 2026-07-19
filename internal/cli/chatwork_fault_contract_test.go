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
	specs := chatworkCommandSpecs()
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
	specs := chatworkCommandSpecs()
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

func TestChatworkRateLimitRecoverySeparatesReadsFromMutations(t *testing.T) {
	for _, spec := range chatworkCommandSpecs() {
		code := "chatwork_rate_limited"
		wantRetryable := true
		wantCommand := spec.Path
		if spec.Effect != operation.EffectRead {
			code = "chatwork_mutation_rate_limited"
			wantRetryable = false
			wantCommand = "help " + spec.Path
		}
		var matched *CommandError
		for index := range spec.Agent.Errors {
			if spec.Agent.Errors[index].Code == code {
				matched = &spec.Agent.Errors[index]
				break
			}
		}
		if matched == nil || matched.Kind != fault.KindRateLimited || matched.Retryable != wantRetryable ||
			len(matched.NextActions) != 1 || matched.NextActions[0].Command != wantCommand {
			t.Errorf("%s rate-limit contract = %+v, want code=%s retryable=%t recovery=%q", spec.Path, matched, code, wantRetryable, wantCommand)
		}
	}
}

func TestChatworkAPIFaultsCoverPAT(t *testing.T) {
	want := map[string]faultSignature{
		"chatwork_token_missing": {fault.KindAuthentication, false},
		"chatwork_token_invalid": {fault.KindAuthentication, false},
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
		_, hasReadRateLimit := declared["chatwork_rate_limited"]
		if hasReadRateLimit != !wantMutation {
			t.Errorf("%s read rate-limit fault present=%t, want %t", spec.Path, hasReadRateLimit, !wantMutation)
		}
		_, hasMutationRateLimit := declared["chatwork_mutation_rate_limited"]
		if hasMutationRateLimit != wantMutation {
			t.Errorf("%s mutation rate-limit fault present=%t, want %t", spec.Path, hasMutationRateLimit, wantMutation)
		}

		wantMessageRead := spec.chatwork.Task == chatwork.TaskMessagesList || spec.chatwork.Task == chatwork.TaskMessagesShow
		_, hasLimitation := declared["chatwork_message_limitation_invalid"]
		if hasLimitation != wantMessageRead {
			t.Errorf("%s limitation fault present=%t, want %t", spec.Path, hasLimitation, wantMessageRead)
		}
		_, hasRestricted := declared["chatwork_message_restricted"]
		wantRestricted := spec.chatwork.Task == chatwork.TaskMessagesShow
		if hasRestricted != wantRestricted {
			t.Errorf("%s restricted-message fault present=%t, want %t", spec.Path, hasRestricted, wantRestricted)
		}
		_, hasAccountVerification := declared["chatwork_account_verification_failed"]
		wantAccountVerification := spec.chatwork.Task == chatwork.TaskRoomsCreate
		if hasAccountVerification != wantAccountVerification {
			t.Errorf("%s account-verification fault present=%t, want %t", spec.Path, hasAccountVerification, wantAccountVerification)
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

func commandFaultSignatures(spec CommandSpec) map[string]faultSignature {
	result := make(map[string]faultSignature, len(spec.Agent.Errors))
	for _, declared := range spec.Agent.Errors {
		result[declared.Code] = faultSignature{kind: declared.Kind, retryable: declared.Retryable}
	}
	return result
}
