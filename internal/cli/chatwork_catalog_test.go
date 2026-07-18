package cli

import (
	"reflect"
	"testing"
)

func TestChatworkCatalogSpecsValidateWithPublicCatalog(t *testing.T) {
	if err := DefaultCatalog().Validate(); err != nil {
		t.Fatalf("Chatwork catalog validation failed: %v", err)
	}
}

func TestChatworkCatalogContainsEveryTypedTaskOnce(t *testing.T) {
	seen := make(map[string]string)
	for _, command := range chatworkCommandSpecs() {
		if command.chatwork == nil {
			t.Fatalf("command %q has no Chatwork task binding", command.Path)
		}
		task := string(command.chatwork.Task)
		if previous, exists := seen[task]; exists {
			t.Fatalf("task %q is bound by %q and %q", task, previous, command.Path)
		}
		seen[task] = command.Path
	}
	if len(seen) != 33 {
		t.Fatalf("typed Chatwork task bindings = %d, want 33", len(seen))
	}
}

func TestPresentationChangesKeepSuccessFormatsTextOnly(t *testing.T) {
	changed := map[string]bool{
		"contacts list": true, "rooms list": true, "members list": true,
		"personal-tasks list": true, "room-tasks list": true, "files list": true,
		"contact-requests list": true, "messages list": true, "messages show": true,
	}
	for _, command := range chatworkCommandSpecs() {
		if !changed[command.Path] {
			continue
		}
		if !reflect.DeepEqual(command.Agent.Output.Formats, []OutputFormat{OutputFormatText}) ||
			command.Agent.Output.DefaultFormat != OutputFormatText ||
			command.Agent.Output.JSONSchemaVersion != 0 || command.Agent.Output.JSONEnvelope != "" {
			t.Fatalf("%s success output contract changed: %+v", command.Path, command.Agent.Output)
		}
	}
}
