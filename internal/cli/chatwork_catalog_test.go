package cli

import (
	"reflect"
	"strings"
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

func TestMessageListCatalogPublishesBoundedSelectionInputs(t *testing.T) {
	var messages CommandSpec
	for _, command := range chatworkCommandSpecs() {
		if command.Path == "messages list" {
			messages = command
			break
		}
	}
	if messages.Path == "" {
		t.Fatal("messages list is absent from the Chatwork catalog")
	}
	wantUsage := "cwk messages list --room <room-ref> [--window recent|changes] [--limit <count>] [--sender <account-ref>] [--context none|replies]"
	if messages.Usage() != wantUsage {
		t.Fatalf("messages list usage = %q, want %q", messages.Usage(), wantUsage)
	}

	inputs := make(map[string]CommandInput, len(messages.Agent.Inputs))
	for _, input := range messages.Agent.Inputs {
		inputs[input.Name] = input
	}
	limit := inputs["--limit"]
	if limit.Name == "" || limit.Required || limit.Repeatable || limit.Source != InputSourceFlag ||
		len(limit.AllowedValues) != 0 || limit.ReferenceKind != "" ||
		!strings.Contains(limit.Description, "1") || !strings.Contains(limit.Description, "100") ||
		!strings.Contains(limit.Description, "新しい") ||
		!strings.Contains(limit.Description, "返信コンテキスト") {
		t.Fatalf("limit input contract = %+v", limit)
	}
	sender := inputs["--sender"]
	if sender.Required || !sender.Repeatable || sender.ReferenceKind != "chatwork-account" ||
		!strings.Contains(sender.Description, "繰り返し") || !strings.Contains(sender.Description, "OR") ||
		!strings.Contains(sender.Description, "プロバイダーの上限付き範囲") || !strings.Contains(sender.Description, "100") {
		t.Fatalf("sender input contract = %+v", sender)
	}
	context := inputs["--context"]
	if context.Required || !reflect.DeepEqual(context.AllowedValues, []string{"none", "replies"}) ||
		!strings.Contains(context.Description, "1ホップ") || !strings.Contains(context.Description, "既定値") ||
		!strings.Contains(context.Description, "返信元・返信先") || !strings.Contains(context.Description, "上限付き範囲") {
		t.Fatalf("context input contract = %+v", context)
	}
	window := inputs["--window"]
	if !reflect.DeepEqual(window.AllowedValues, []string{"recent", "changes"}) ||
		!strings.Contains(window.Description, "recent") || !strings.Contains(window.Description, "既定値") ||
		!strings.Contains(window.Description, "差分") {
		t.Fatalf("window input contract = %+v", window)
	}
	if strings.Contains(limit.Description, "Use --window recent") {
		t.Fatalf("limit input repeats the new default window: %+v", limit)
	}
}
