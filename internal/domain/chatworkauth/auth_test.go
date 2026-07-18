package chatworkauth

import (
	"reflect"
	"testing"
)

func TestProfileReferenceAcceptsOnlyDiscoveredExactValue(t *testing.T) {
	ref, err := NewProfileReference(PublicClientProfileReference)
	if err != nil || ref.Value() != PublicClientProfileReference || !ref.Valid() {
		t.Fatalf("NewProfileReference() = (%q, %v), want exact valid reference", ref.Value(), err)
	}
	for _, value := range []string{"", "oauth2", "CWK_CHATWORK_OAUTH_PUBLIC_V1", PublicClientProfileReference + " ", "cwk_chatwork_oauth_public_v2"} {
		if _, err := NewProfileReference(value); err == nil {
			t.Errorf("NewProfileReference(%q) succeeded", value)
		}
	}
}

func TestTasksAndProfileStatesFailClosed(t *testing.T) {
	for _, task := range []Task{TaskProfilesList, TaskLogin, TaskStatus, TaskLogout} {
		if !task.Valid() {
			t.Errorf("task %q is invalid", task)
		}
	}
	if Task("").Valid() || Task("oauth.token.exchange").Valid() {
		t.Fatal("unknown authentication task was accepted")
	}
	for _, state := range []ProfileState{ProfileStateUnconfigured, ProfileStateReady, ProfileStateExpired} {
		if !state.Valid() {
			t.Errorf("state %q is invalid", state)
		}
	}
	if ProfileState("").Valid() || ProfileState("refreshing").Valid() {
		t.Fatal("unknown profile state was accepted")
	}
}

func TestProfileValidationKeepsProjectionSecretFreeAndConsistent(t *testing.T) {
	ref, err := NewProfileReference(PublicClientProfileReference)
	if err != nil {
		t.Fatal(err)
	}
	valid := Profile{Ref: ref, Method: "oauth2", State: ProfileStateReady, ExpiresAt: 1_800_000_000}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}

	tests := []Profile{
		{},
		{Ref: ref, Method: "pat", State: ProfileStateReady},
		{Ref: ref, Method: "oauth2", State: "unknown"},
		{Ref: ref, Method: "oauth2", State: ProfileStateExpired, ExpiresAt: -1},
		{Ref: ref, Method: "oauth2", State: ProfileStateUnconfigured, ExpiresAt: 1},
	}
	for _, profile := range tests {
		if err := profile.Validate(); err == nil {
			t.Errorf("invalid profile accepted: %+v", profile)
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
