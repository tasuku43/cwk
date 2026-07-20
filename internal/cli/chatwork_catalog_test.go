package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
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
	if len(seen) != 34 {
		t.Fatalf("typed Chatwork task bindings = %d, want 34", len(seen))
	}
}

func TestRoomsCreateCatalogBindsVerifiedAuthenticatedAccountScope(t *testing.T) {
	var create CommandSpec
	for _, command := range chatworkCommandSpecs() {
		if command.Path == "rooms create" {
			create = command
			break
		}
	}
	if create.Path == "" {
		t.Fatal("rooms create is absent from the Chatwork catalog")
	}
	if strings.Contains(create.Usage(), "--owner") || !strings.Contains(create.Usage(), "--account <account-ref>") {
		t.Fatalf("rooms create usage = %q", create.Usage())
	}
	inputs := make(map[string]CommandInput, len(create.Agent.Inputs))
	for _, input := range create.Agent.Inputs {
		inputs[input.Name] = input
	}
	account := inputs["--account"]
	if !account.Required || account.ReferenceKind != "chatwork-account" || strings.Contains(account.Description, "所有者") {
		t.Fatalf("authenticated account input = %+v", account)
	}
	if !strings.Contains(inputs["--name"].Description, "1〜255") ||
		!reflect.DeepEqual(inputs["--icon"].AllowedValues, chatwork.RoomIconPresetValues()) ||
		!strings.Contains(inputs["--invite-code"].Description, "1〜50") {
		t.Fatalf("rooms create official field constraints = name %+v icon %+v code %+v", inputs["--name"], inputs["--icon"], inputs["--invite-code"])
	}
	if _, exists := inputs["--owner"]; exists {
		t.Fatal("rooms create still publishes --owner")
	}
	mutation := create.Agent.Mutation
	if mutation == nil || mutation.ParentInput != "--account" || !reflect.DeepEqual(mutation.TargetInputs, []string{"--account"}) {
		t.Fatalf("rooms create mutation = %+v", mutation)
	}
	var verification *CommandError
	for index := range create.Agent.Errors {
		if create.Agent.Errors[index].Code == "chatwork_account_verification_failed" {
			verification = &create.Agent.Errors[index]
			break
		}
	}
	if verification == nil || verification.Retryable || len(verification.NextActions) != 1 ||
		verification.NextActions[0].Command != "help rooms create" {
		t.Fatalf("rooms create account-verification recovery = %+v", verification)
	}
}

func TestInviteLinkCatalogPublishesCompleteReplacementAndDescription(t *testing.T) {
	commands := make(map[string]CommandSpec)
	for _, command := range chatworkCommandSpecs() {
		commands[command.Path] = command
	}
	create := commands["invite-link create"]
	update := commands["invite-link update"]
	if create.Path == "" || update.Path == "" {
		t.Fatal("invite-link mutation commands are absent from the Chatwork catalog")
	}

	createInputs := make(map[string]CommandInput, len(create.Agent.Inputs))
	for _, input := range create.Agent.Inputs {
		createInputs[input.Name] = input
	}
	if description := createInputs["--description"]; description.Name == "" || description.Required {
		t.Fatalf("invite-link create description = %+v", description)
	}

	updateInputs := make(map[string]CommandInput, len(update.Agent.Inputs))
	for _, input := range update.Agent.Inputs {
		updateInputs[input.Name] = input
	}
	if updateInputs["--code"].Required || updateInputs["--regenerate-code"].Required ||
		!updateInputs["--approval"].Required || !updateInputs["--description"].Required {
		t.Fatalf("invite-link update inputs = %+v", updateInputs)
	}
	if !strings.Contains(update.Usage(), "[--code <code>] [--regenerate-code]") ||
		!strings.Contains(update.Agent.Outcome, "すべて指定") {
		t.Fatalf("invite-link update contract = usage %q outcome %q", update.Usage(), update.Agent.Outcome)
	}
	if update.Agent.Mutation == nil || update.Agent.Mutation.Impact.AccessChange != yes {
		t.Fatalf("invite-link update mutation impact = %+v", update.Agent.Mutation)
	}
}

func TestPresentationChangesKeepSuccessFormatsTextOnly(t *testing.T) {
	changed := map[string]bool{
		"contacts list": true, "rooms list": true, "members find": true, "members list": true,
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

func TestMembersFindCatalogPublishesCandidateDiscoveryWithoutAutomaticSelection(t *testing.T) {
	var find CommandSpec
	for _, command := range chatworkCommandSpecs() {
		if command.Path == "members find" {
			find = command
			break
		}
	}
	if find.Path == "" {
		t.Fatal("members find is absent from the Chatwork catalog")
	}
	if find.Role != RoleDiscover || find.Effect.String() != "read" || find.Usage() != "cwk members find --room <room-ref> --query <text>" {
		t.Fatalf("members find command = %+v", find)
	}
	inputs := make(map[string]CommandInput, len(find.Agent.Inputs))
	for _, input := range find.Agent.Inputs {
		inputs[input.Name] = input
	}
	if !inputs["--room"].Required || inputs["--room"].ReferenceKind != "chatwork-room" ||
		!inputs["--query"].Required || inputs["--query"].ReferenceKind != "" ||
		!strings.Contains(inputs["--query"].Description, "複数候補を自動選択しません") {
		t.Fatalf("members find inputs = %+v", inputs)
	}
	if !strings.Contains(find.Agent.Outcome, "曖昧さを残した") {
		t.Fatalf("members find outcome = %q", find.Agent.Outcome)
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
	wantUsage := "cwk messages list --room <room-ref> [--window recent|changes] [--since <RFC3339>] [--until <RFC3339>] [--on <day>] [--start-index <index>] [--count <count>] [--sender <account-ref>] [--context none|replies] [--resolve-relations <count>]"
	if messages.Usage() != wantUsage {
		t.Fatalf("messages list usage = %q, want %q", messages.Usage(), wantUsage)
	}

	inputs := make(map[string]CommandInput, len(messages.Agent.Inputs))
	for _, input := range messages.Agent.Inputs {
		inputs[input.Name] = input
	}
	count := inputs["--count"]
	if count.Name == "" || count.Required || count.Repeatable || count.Source != InputSourceFlag ||
		len(count.AllowedValues) != 0 || count.ReferenceKind != "" ||
		!strings.Contains(count.Description, "1") || !strings.Contains(count.Description, "100") ||
		!strings.Contains(count.Description, "最大件数") || !strings.Contains(count.Description, "終了順位ではありません") ||
		!strings.Contains(count.Description, "11〜30") || !strings.Contains(count.Description, "返信コンテキスト") {
		t.Fatalf("count input contract = %+v", count)
	}
	start := inputs["--start-index"]
	if start.Name == "" || start.Required || start.Repeatable || start.Source != InputSourceFlag ||
		len(start.AllowedValues) != 0 || start.ReferenceKind != "" ||
		!strings.Contains(start.Description, "1始まり") || !strings.Contains(start.Description, "1〜100") ||
		!strings.Contains(start.Description, "省略時は1") {
		t.Fatalf("start-index input contract = %+v", start)
	}
	sender := inputs["--sender"]
	if sender.Required || !sender.Repeatable || sender.ReferenceKind != "chatwork-account" ||
		!strings.Contains(sender.Description, "繰り返し") || !strings.Contains(sender.Description, "OR") ||
		!strings.Contains(sender.Description, "プロバイダーの上限付き範囲") || !strings.Contains(sender.Description, "100") ||
		!strings.Contains(sender.Description, "members find") || !strings.Contains(sender.Description, "変更せず") {
		t.Fatalf("sender input contract = %+v", sender)
	}
	context := inputs["--context"]
	if context.Required || !reflect.DeepEqual(context.AllowedValues, []string{"none", "replies"}) ||
		!strings.Contains(context.Description, "1ホップ") || !strings.Contains(context.Description, "既定値") ||
		!strings.Contains(context.Description, "返信元・返信先") || !strings.Contains(context.Description, "上限付き範囲") {
		t.Fatalf("context input contract = %+v", context)
	}
	resolution := inputs["--resolve-relations"]
	if resolution.Name == "" || resolution.Required || resolution.Repeatable || resolution.Source != InputSourceFlag ||
		len(resolution.AllowedValues) != 0 || resolution.ReferenceKind != "" ||
		!strings.Contains(resolution.Description, "0〜100") || !strings.Contains(resolution.Description, "追加の一件取得") ||
		!strings.Contains(resolution.Description, "重複ID") || !strings.Contains(resolution.Description, "再帰的") ||
		!strings.Contains(resolution.Description, "既定値は5") || !strings.Contains(resolution.Description, "0は追加取得を無効化") {
		t.Fatalf("resolve-relations input contract = %+v", resolution)
	}
	window := inputs["--window"]
	if !reflect.DeepEqual(window.AllowedValues, []string{"recent", "changes"}) ||
		!strings.Contains(window.Description, "recent") || !strings.Contains(window.Description, "既定値") ||
		!strings.Contains(window.Description, "差分") {
		t.Fatalf("window input contract = %+v", window)
	}
	since, until, on := inputs["--since"], inputs["--until"], inputs["--on"]
	if since.Required || since.Repeatable || !strings.Contains(since.Description, "含む") ||
		!strings.Contains(since.Description, "RFC3339") || !strings.Contains(since.Description, "明示オフセット") {
		t.Fatalf("since input contract = %+v", since)
	}
	if until.Required || until.Repeatable || !strings.Contains(until.Description, "含まない") ||
		!strings.Contains(until.Description, "RFC3339") || !strings.Contains(until.Description, "明示オフセット") {
		t.Fatalf("until input contract = %+v", until)
	}
	if on.Required || on.Repeatable || !strings.Contains(on.Description, "Asia/Tokyo") ||
		!strings.Contains(on.Description, "YYYY-MM-DD") || !strings.Contains(on.Description, "today") ||
		!strings.Contains(on.Description, "yesterday") || !strings.Contains(on.Description, "一度だけ") {
		t.Fatalf("on input contract = %+v", on)
	}
	if strings.Contains(count.Description, "Use --window recent") {
		t.Fatalf("count input repeats the new default window: %+v", count)
	}
}
