package chatworkauth

import (
	"reflect"
	"testing"
)

func TestFixedAuthenticationTargetIsNonCredentialPolicyIdentity(t *testing.T) {
	if TargetKind != "chatwork-authentication" || TargetStableID != "single-account" {
		t.Fatalf("target = %q/%q", TargetKind, TargetStableID)
	}
}

func TestTasksAndStatesFailClosed(t *testing.T) {
	for _, task := range []Task{TaskLogin, TaskStatus, TaskLogout} {
		if !task.Valid() {
			t.Errorf("task %q is invalid", task)
		}
	}
	if Task("").Valid() || Task("oauth.token.exchange").Valid() {
		t.Fatal("unknown authentication task was accepted")
	}
	for _, state := range []State{StateUnconfigured, StateReady, StateExpired} {
		if !state.Valid() {
			t.Errorf("state %q is invalid", state)
		}
	}
	if State("").Valid() || State("refreshing").Valid() {
		t.Fatal("unknown authentication state was accepted")
	}
}

func TestSummaryValidationKeepsProjectionSecretFreeAndConsistent(t *testing.T) {
	valid := Summary{Method: "oauth2", State: StateReady, ExpiresAt: 1_800_000_000}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid summary rejected: %v", err)
	}

	tests := []Summary{
		{},
		{Method: "pat", State: StateReady},
		{Method: "oauth2", State: "unknown"},
		{Method: "oauth2", State: StateExpired, ExpiresAt: -1},
		{Method: "oauth2", State: StateUnconfigured, ExpiresAt: 1},
	}
	for _, summary := range tests {
		if err := summary.Validate(); err == nil {
			t.Errorf("invalid summary accepted: %+v", summary)
		}
	}
}

func TestCredentialStatusHasOnlyTheReviewedSecretFreeFields(t *testing.T) {
	statusType := reflect.TypeOf(CredentialStatus{})
	want := []string{"Authenticated", "ExpiresAt"}
	if statusType.NumField() != len(want) {
		t.Fatalf("CredentialStatus field count = %d, want %d", statusType.NumField(), len(want))
	}
	for index, name := range want {
		if statusType.Field(index).Name != name {
			t.Fatalf("CredentialStatus field %d = %q, want %q", index, statusType.Field(index).Name, name)
		}
	}
}
