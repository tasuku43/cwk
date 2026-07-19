package commandselection

import (
	"fmt"
	"strings"
	"testing"
)

func TestProfilePreservesOrderAndCopiesAtEveryBoundary(t *testing.T) {
	input := []string{"messages list", "rooms list", "stale command"}
	profile, err := New(input)
	if err != nil {
		t.Fatal(err)
	}
	input[0] = "contacts list"

	first := profile.EnabledCommands()
	if got, want := fmt.Sprint(first), "[messages list rooms list stale command]"; got != want {
		t.Fatalf("EnabledCommands() = %s, want %s", got, want)
	}
	first[0] = "contacts list"
	if got := profile.EnabledCommands()[0]; got != "messages list" {
		t.Fatalf("profile changed through returned slice: %q", got)
	}
	if !profile.Contains("stale command") || profile.Contains("contact-requests list") {
		t.Fatalf("Contains returned an unexpected result")
	}
}

func TestProfileAcceptsEmptyAndMaximumSelection(t *testing.T) {
	if profile, err := New(nil); err != nil || len(profile.EnabledCommands()) != 0 {
		t.Fatalf("New(nil) = %#v, %v", profile, err)
	}

	paths := make([]string, MaxEnabledCommands)
	for index := range paths {
		paths[index] = fmt.Sprintf("command-%d show", index)
	}
	profile, err := New(paths)
	if err != nil || len(profile.EnabledCommands()) != MaxEnabledCommands {
		t.Fatalf("maximum profile: len=%d err=%v", len(profile.EnabledCommands()), err)
	}
}

func TestProfileRejectsInvalidDuplicateOrOversizedSelection(t *testing.T) {
	tests := map[string][]string{
		"too many":      make([]string, MaxEnabledCommands+1),
		"duplicate":     {"rooms list", "rooms list"},
		"empty":         {""},
		"leading space": {" rooms list"},
		"double space":  {"rooms  list"},
		"uppercase":     {"Rooms list"},
		"control":       {"rooms\nlist"},
		"oversized":     {"a" + strings.Repeat("b", MaxCommandPathBytes)},
	}
	for name, paths := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := New(paths); err == nil {
				t.Fatalf("New(%q) succeeded", paths)
			}
		})
	}
}
