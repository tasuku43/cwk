package cli

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
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
	if len(seen) != 36 {
		t.Fatalf("typed Chatwork task bindings = %d, want 36", len(seen))
	}
}

func TestMessagesResourcesCatalogDeclaresExactTypedInputsAndShowReferences(t *testing.T) {
	var resources CommandSpec
	for _, command := range chatworkCommandSpecs() {
		if command.Path == "messages resources" {
			resources = command
			break
		}
	}
	if resources.Path == "" {
		t.Fatal("messages resources command is missing")
	}
	if resources.Effect != operation.EffectRead || resources.Role != RoleAct || resources.chatwork.Task != chatwork.TaskMessagesResources {
		t.Fatalf("task role/effect/binding = %s/%s/%s", resources.Role, resources.Effect, resources.chatwork.Task)
	}
	inputs := make(map[string]CommandInput, len(resources.Agent.Inputs))
	for _, input := range resources.Agent.Inputs {
		inputs[input.Name] = input
	}
	for flag, kind := range map[string]string{"--room": string(chatwork.ReferenceRoom), "--message": string(chatwork.ReferenceMessage)} {
		input := inputs[flag]
		if !input.Required || input.ReferenceKind != kind {
			t.Errorf("%s input = %+v, want required %s reference", flag, input, kind)
		}
	}
	kind := inputs["--kind"]
	if !kind.Required || !reflect.DeepEqual(kind.AllowedValues, []string{"files", "tasks", "all"}) {
		t.Errorf("kind input = %+v, want required files|tasks|all", kind)
	}
	fields := make(map[string]OutputField, len(resources.Agent.Output.Fields))
	for _, field := range resources.Agent.Output.Fields {
		fields[field.Name] = field
	}
	for field, referenceKind := range map[string]string{"room_ref": string(chatwork.ReferenceRoom), "message_ref": string(chatwork.ReferenceMessage), "file_ref": string(chatwork.ReferenceFile), "task_ref": string(chatwork.ReferenceTask)} {
		if fields[field].ReferenceKind != referenceKind {
			t.Errorf("%s reference kind = %q, want %q", field, fields[field].ReferenceKind, referenceKind)
		}
	}
	if fields["file_ref"].Type != OutputFieldTypeArray || fields["task_ref"].Type != OutputFieldTypeArray || resources.Agent.Output.Completeness != OutputCompletenessComplete {
		t.Fatalf("resource result contract = %+v", resources.Agent.Output)
	}
	if _, _, err := DefaultCatalog().ActiveView([]string{"rooms list", "messages resources"}); err == nil || !strings.Contains(err.Error(), `input "--message" requires`) {
		t.Fatalf("message resources without a visible message producer error = %v", err)
	}
	view, _, err := DefaultCatalog().ActiveView([]string{"rooms list", "messages list", "messages resources"})
	if err != nil {
		t.Fatalf("resource command reference closure failed: %v", err)
	}
	for _, path := range []string{"messages resources", "rooms list", "messages list"} {
		if _, exists := view.Lookup(path); !exists {
			t.Errorf("active view is missing %q", path)
		}
	}
}

func TestMessageSendUnknownRecoveryIsExactReadOnlyReconciliation(t *testing.T) {
	var send, reconcile CommandSpec
	for _, command := range chatworkCommandSpecs() {
		switch command.Path {
		case "messages send":
			send = command
		case "messages reconcile-send":
			reconcile = command
		}
	}
	if send.Path == "" || reconcile.Path == "" || send.chatwork.Reconcile != reconcile.Path {
		t.Fatalf("send/reconciliation catalog entries = %q/%q, recovery=%q", send.Path, reconcile.Path, send.chatwork.Reconcile)
	}
	if reconcile.Effect != operation.EffectRead || reconcile.Role != RoleDiscover || reconcile.Agent.Mutation != nil ||
		reconcile.chatwork.Task != chatwork.TaskMessagesReconcileSend {
		t.Fatalf("reconciliation command is not a read-only discover task: %+v", reconcile)
	}
	for _, declared := range send.Agent.Errors {
		if declared.Code != "unclassified_mutation_outcome" && declared.Code != "chatwork_mutation_outcome_unknown" {
			continue
		}
		if len(declared.NextActions) != 1 || declared.NextActions[0].Command != "messages reconcile-send" || declared.Retryable {
			t.Errorf("send outcome fault %q = %+v", declared.Code, declared)
		}
	}
	if len(reconcile.ExclusiveInputGroups) != 1 || !reflect.DeepEqual(reconcile.ExclusiveInputGroups[0], []string{"--body", "--body-file"}) {
		t.Fatalf("reconciliation body source constraint = %v", reconcile.ExclusiveInputGroups)
	}
	for _, input := range reconcile.Agent.Inputs {
		if input.Name == "--room" && (!input.Required || input.ReferenceKind != "chatwork-room") {
			t.Fatalf("reconciliation room input = %+v", input)
		}
	}
	for _, field := range reconcile.Agent.Output.Fields {
		if field.Name == "body" || field.Name == "body_file" || field.Name == "path" {
			t.Fatalf("reconciliation output exposes private input %q", field.Name)
		}
	}
	if !strings.Contains(reconcile.Agent.Outcome, "この呼び出しの送信結果を証明しない") {
		t.Fatalf("reconciliation outcome omits its uncertainty: %q", reconcile.Agent.Outcome)
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

func TestRoomRefHelpExplainsUnchangedDiscoveryReference(t *testing.T) {
	catalog := DefaultCatalog()
	if err := catalog.Validate(); err != nil {
		t.Fatalf("default catalog is invalid: %v", err)
	}
	rooms, found := catalog.Lookup("rooms list")
	if !found {
		t.Fatal("rooms list is absent from the catalog")
	}
	wantRoomRef := ProducedRef{Kind: string(chatwork.ReferenceRoom), Field: "room_ref"}
	if !reflect.DeepEqual(rooms.ProducedRefs(), []ProducedRef{wantRoomRef}) {
		t.Fatalf("rooms list produced references = %+v, want %+v", rooms.ProducedRefs(), wantRoomRef)
	}
	messages, found := catalog.Lookup("messages list")
	if !found {
		t.Fatal("messages list is absent from the catalog")
	}
	wantRoomInput := ConsumedRef{Kind: string(chatwork.ReferenceRoom), Argument: "--room"}
	var roomInputFound bool
	for _, input := range messages.ConsumedRefs() {
		roomInputFound = roomInputFound || input == wantRoomInput
	}
	if !roomInputFound {
		t.Fatalf("messages list does not consume the discovered room_ref: %+v", messages.ConsumedRefs())
	}

	consumers := 0
	for _, command := range catalog.Commands() {
		for _, input := range command.Agent.Inputs {
			if input.Name != "--room" {
				continue
			}
			consumers++
			if !input.Required || input.ReferenceKind != string(chatwork.ReferenceRoom) {
				t.Errorf("%s --room = %+v, want a required room reference", command.Path, input)
			}
			for _, phrase := range []string{
				"rooms list の room-ref をそのまま",
				"正の10進数、1〜32桁、先頭0なし",
			} {
				if !strings.Contains(input.Description, phrase) {
					t.Errorf("%s --room description lacks %q: %q", command.Path, phrase, input.Description)
				}
			}
			if strings.Contains(input.Description, "room_ref") {
				t.Errorf("%s --room description uses the internal room_ref name: %q", command.Path, input.Description)
			}
			help := string(renderCommandHelp(command, catalog.Commands()))
			if len(help) > maxExactHelpBytes {
				t.Errorf("%s exact help exceeds %d bytes: %d", command.Path, maxExactHelpBytes, len(help))
			}
		}
	}
	if consumers == 0 {
		t.Fatal("catalog has no --room reference consumers")
	}

	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	for _, phrase := range []string{
		"`rooms list` が表示する `room-ref` を、そのまま",
		"`--room <room-id>` に渡します。値は先頭ゼロなしの1〜32桁の正の10進数です。",
		"候補に表示される `account-ref` は、`--sender` にそのまま渡せます。",
	} {
		if !strings.Contains(string(readme), phrase) {
			t.Errorf("README.md does not explain the room reference round trip %q", phrase)
		}
	}
	if strings.Contains(string(readme), "room_ref") || strings.Contains(string(readme), "account_ref") {
		t.Errorf("README.md exposes internal reference field names instead of public text labels")
	}
}

func TestHumanCatalogKeepsExactPredicatesAndKnownMaximums(t *testing.T) {
	catalog := DefaultCatalog()
	wantSummaries := map[string]string{
		"personal-tasks list": "最大100件",
		"messages list":       "最大100件",
		"room-tasks list":     "最大100件",
		"files list":          "最大100件",
		"files upload":        "最大5 MiB",
	}
	for path, phrase := range wantSummaries {
		command, found := catalog.Lookup(path)
		if !found || !strings.Contains(command.Summary, phrase) {
			t.Errorf("%s summary = %q, want supported maximum %q", path, command.Summary, phrase)
		}
	}

	messages, _ := catalog.Lookup("messages list")
	var senderDescription string
	for _, input := range messages.Agent.Inputs {
		if input.Name == "--sender" {
			senderDescription = input.Description
			break
		}
	}
	if !strings.Contains(senderDescription, "完全一致") || !strings.Contains(senderDescription, "OR") {
		t.Errorf("sender help no longer states its exact OR predicate: %q", senderDescription)
	}
	reconcile, _ := catalog.Lookup("messages reconcile-send")
	if !strings.Contains(reconcile.Summary, "完全一致") || !strings.Contains(reconcile.Agent.Outcome, "完全一致") {
		t.Errorf("send reconciliation lost its complete-payload match wording: summary=%q outcome=%q", reconcile.Summary, reconcile.Agent.Outcome)
	}
}

func TestRoomsUpdatePublishesCreateIconPresetsWithoutChangingInputValidation(t *testing.T) {
	commands := make(map[string]CommandSpec)
	for _, command := range chatworkCommandSpecs() {
		commands[command.Path] = command
	}
	create, createFound := commands["rooms create"]
	update, updateFound := commands["rooms update"]
	if !createFound || !updateFound {
		t.Fatal("room create or update is absent from the Chatwork catalog")
	}
	iconInput := func(command CommandSpec) CommandInput {
		for _, input := range command.Agent.Inputs {
			if input.Name == "--icon" {
				return input
			}
		}
		t.Fatalf("%s has no --icon input", command.Path)
		return CommandInput{}
	}
	createIcon := iconInput(create)
	updateIcon := iconInput(update)
	presets := chatwork.RoomIconPresetValues()
	presetList := strings.Join(presets, "|")
	if !reflect.DeepEqual(createIcon.AllowedValues, presets) ||
		!strings.Contains(create.Usage(), "--icon "+presetList) {
		t.Fatalf("rooms create icon syntax and values diverge: usage=%q input=%+v presets=%v", create.Usage(), createIcon, presets)
	}
	if updateIcon.Source != InputSourceFlag || updateIcon.Required || len(updateIcon.AllowedValues) != 0 ||
		!strings.Contains(update.Usage(), "[--icon <preset>]") {
		t.Fatalf("rooms update icon input validation changed: usage=%q input=%+v", update.Usage(), updateIcon)
	}
	wantDescription := "変更後の定義済みアイコンプリセット。Chatwork公式の一覧: " + presetList + "。"
	if updateIcon.Description != wantDescription {
		t.Fatalf("rooms update icon description = %q, want %q", updateIcon.Description, wantDescription)
	}
	help := string(renderCommandHelp(update, DefaultCatalog().Commands()))
	if !strings.Contains(help, "--icon preset 任意: "+wantDescription) {
		t.Fatalf("rooms update exact help omits the complete preset list:\n%s", help)
	}
	if len(help) > maxExactHelpBytes {
		t.Fatalf("rooms update exact help exceeds %d bytes: %d", maxExactHelpBytes, len(help))
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
	if find.Role != RoleDiscover || find.Effect.String() != "read" || find.Usage() != "cwk members find --room <room-id> --query <text>" {
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
	if !strings.Contains(find.Agent.Outcome, "自動選択しない") {
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
	wantUsage := "cwk messages list --room <room-id> [--window recent|changes] [--since <RFC3339>] [--until <RFC3339>] [--on <day>] [--start-index <index>] [--count <count>] [--sender <account-ref>] [--contains <text>] [--context none|replies] [--resolve-relations <count>] [--max-body-bytes <bytes>] [--max-bytes <bytes>] [--jst]"
	if messages.Usage() != wantUsage {
		t.Fatalf("messages list usage = %q, want %q", messages.Usage(), wantUsage)
	}

	inputs := make(map[string]CommandInput, len(messages.Agent.Inputs))
	for _, input := range messages.Agent.Inputs {
		inputs[input.Name] = input
	}
	outputFields := make(map[string]OutputField, len(messages.Agent.Output.Fields))
	for _, field := range messages.Agent.Output.Fields {
		outputFields[field.Name] = field
	}
	if !strings.Contains(inputs["--room"].Description, "rooms list の room-ref をそのまま") ||
		!strings.Contains(inputs["--room"].Description, "正の10進数、1〜32桁、先頭0なし") {
		t.Fatalf("room input omits the canonical decimal reference format: %+v", inputs["--room"])
	}
	count := inputs["--count"]
	if count.Name == "" || count.Required || count.Repeatable || count.Source != InputSourceFlag ||
		len(count.AllowedValues) != 0 || count.ReferenceKind != "" ||
		!strings.Contains(count.Description, "1") || !strings.Contains(count.Description, "100") || !strings.Contains(count.Description, "主要アンカー") ||
		!strings.Contains(count.Description, "--start-index起点") ||
		!strings.Contains(count.Description, "返信文脈") {
		t.Fatalf("count input contract = %+v", count)
	}
	start := inputs["--start-index"]
	if start.Name == "" || start.Required || start.Repeatable || start.Source != InputSourceFlag ||
		len(start.AllowedValues) != 0 || start.ReferenceKind != "" ||
		!strings.Contains(start.Description, "1始まり") || !strings.Contains(start.Description, "1〜100") ||
		!strings.Contains(start.Description, "既定1") {
		t.Fatalf("start-index input contract = %+v", start)
	}
	sender := inputs["--sender"]
	if sender.Required || !sender.Repeatable || sender.ReferenceKind != "chatwork-account" ||
		!strings.Contains(sender.Description, "反復") || !strings.Contains(sender.Description, "OR") ||
		!strings.Contains(sender.Description, "最大100") ||
		!strings.Contains(sender.Description, "members find の account-ref をそのまま") ||
		strings.Contains(sender.Description, "account_ref") {
		t.Fatalf("sender input contract = %+v", sender)
	}
	contains := inputs["--contains"]
	if contains.Name == "" || contains.Required || !contains.Repeatable || contains.Source != InputSourceFlag ||
		contains.ReferenceKind != "" || len(contains.AllowedValues) != 0 ||
		!strings.Contains(contains.Description, "大小文字区別") || !strings.Contains(contains.Description, "リテラル部分一致") ||
		!strings.Contains(contains.Description, "語句OR") || !strings.Contains(contains.Description, "1〜1024 UTF-8バイト") ||
		!strings.Contains(contains.Description, "100") || !strings.Contains(contains.Description, "trim/Unicode正規化なし") {
		t.Fatalf("contains input contract = %+v", contains)
	}
	context := inputs["--context"]
	if context.Required || !reflect.DeepEqual(context.AllowedValues, []string{"none", "replies"}) ||
		!strings.Contains(context.Description, "1ホップ") ||
		!strings.Contains(context.Description, "replies") || !strings.Contains(context.Description, "none") ||
		!strings.Contains(context.Description, "型付き同室返信元/先") || !strings.Contains(context.Description, "範囲内") {
		t.Fatalf("context input contract = %+v", context)
	}
	resolution := inputs["--resolve-relations"]
	if resolution.Name == "" || resolution.Required || resolution.Repeatable || resolution.Source != InputSourceFlag ||
		len(resolution.AllowedValues) != 0 || resolution.ReferenceKind != "" ||
		!strings.Contains(resolution.Description, "0〜100") || !strings.Contains(resolution.Description, "再帰") ||
		!strings.Contains(resolution.Description, "返信元") || !strings.Contains(resolution.Description, "既定5") ||
		!strings.Contains(resolution.Description, "0=無効") {
		t.Fatalf("resolve-relations input contract = %+v", resolution)
	}
	maxBytes := inputs["--max-bytes"]
	if maxBytes.Name == "" || maxBytes.Required || maxBytes.Repeatable || maxBytes.Source != InputSourceFlag ||
		len(maxBytes.AllowedValues) != 0 || maxBytes.ReferenceKind != "" ||
		!strings.Contains(maxBytes.Description, "1〜16777216") || !strings.Contains(maxBytes.Description, "標準出力") ||
		!strings.Contains(maxBytes.Description, "本文分割なし") {
		t.Fatalf("max-bytes input contract = %+v", maxBytes)
	}
	maxBodyBytes := inputs["--max-body-bytes"]
	if maxBodyBytes.Name == "" || maxBodyBytes.Required || maxBodyBytes.Repeatable || maxBodyBytes.Source != InputSourceFlag ||
		len(maxBodyBytes.AllowedValues) != 0 || maxBodyBytes.ReferenceKind != "" ||
		!strings.Contains(maxBodyBytes.Description, "0〜65535") ||
		!strings.Contains(maxBodyBytes.Description, "生UTF-8") ||
		!strings.Contains(maxBodyBytes.Description, "元バイト数") {
		t.Fatalf("max-body-bytes input contract = %+v", maxBodyBytes)
	}
	if outputFields["body_truncated"].Type != OutputFieldTypeBoolean ||
		!strings.Contains(outputFields["body_truncated"].Description, "body-truncated=true") ||
		outputFields["original_body_bytes"].Type != OutputFieldTypeInteger ||
		!strings.Contains(outputFields["original_body_bytes"].Description, "UTF-8本文バイト数") {
		t.Fatalf("message body truncation output facts = %+v / %+v", outputFields["body_truncated"], outputFields["original_body_bytes"])
	}
	window := inputs["--window"]
	if !reflect.DeepEqual(window.AllowedValues, []string{"recent", "changes"}) ||
		!strings.Contains(window.Description, "recent") || !strings.Contains(window.Description, "既定") ||
		!strings.Contains(window.Description, "差分") {
		t.Fatalf("window input contract = %+v", window)
	}
	since, until, on := inputs["--since"], inputs["--until"], inputs["--on"]
	if since.Required || since.Repeatable || !strings.Contains(since.Description, "含む") ||
		!strings.Contains(since.Description, "RFC3339") || !strings.Contains(since.Description, "オフセット必須") {
		t.Fatalf("since input contract = %+v", since)
	}
	if until.Required || until.Repeatable || !strings.Contains(until.Description, "含まない") ||
		!strings.Contains(until.Description, "RFC3339") || !strings.Contains(until.Description, "オフセット必須") {
		t.Fatalf("until input contract = %+v", until)
	}
	if on.Required || on.Repeatable || !strings.Contains(on.Description, "Asia/Tokyo") ||
		!strings.Contains(on.Description, "YYYY-MM-DD") || !strings.Contains(on.Description, "today") ||
		!strings.Contains(on.Description, "yesterday") || !strings.Contains(on.Description, "開始時確定") {
		t.Fatalf("on input contract = %+v", on)
	}
	if strings.Contains(count.Description, "Use --window recent") {
		t.Fatalf("count input repeats the new default window: %+v", count)
	}
	declaredErrors := make(map[string]CommandError, len(messages.Agent.Errors))
	for _, commandError := range messages.Agent.Errors {
		declaredErrors[commandError.Code] = commandError
	}
	if budgetFault := declaredErrors["output_budget_too_small"]; budgetFault.Kind != fault.KindInvalidInput || budgetFault.Retryable {
		t.Fatalf("output budget fault = %+v", budgetFault)
	}
}

func TestMessagesShowCatalogDeclaresOptionalBoundedReplyContext(t *testing.T) {
	var show CommandSpec
	for _, command := range chatworkCommandSpecs() {
		if command.Path == "messages show" {
			show = command
			break
		}
	}
	if show.Path == "" {
		t.Fatal("messages show is absent from the Chatwork catalog")
	}
	if want := "cwk messages show --room <room-id> --message <message-ref> [--context none|replies] [--around <count>] [--resolve-relations <count>] [--jst]"; show.Usage() != want {
		t.Fatalf("messages show usage = %q, want %q", show.Usage(), want)
	}
	inputs := make(map[string]CommandInput, len(show.Agent.Inputs))
	for _, input := range show.Agent.Inputs {
		inputs[input.Name] = input
	}
	if !reflect.DeepEqual(inputs["--context"].AllowedValues, []string{"none", "replies"}) ||
		!strings.Contains(inputs["--context"].Description, "最新100件") || !strings.Contains(inputs["--context"].Description, "直接の子") {
		t.Fatalf("messages show context = %+v", inputs["--context"])
	}
	if inputs["--resolve-relations"].Required || !strings.Contains(inputs["--resolve-relations"].Description, "値を省略すると5") ||
		!strings.Contains(inputs["--resolve-relations"].Description, "0でも") {
		t.Fatalf("messages show resolution = %+v", inputs["--resolve-relations"])
	}
	if inputs["--around"].Required || inputs["--around"].Repeatable ||
		!strings.Contains(inputs["--around"].Description, "0〜100") || !strings.Contains(inputs["--around"].Description, "一度") {
		t.Fatalf("messages show around input = %+v", inputs["--around"])
	}
	constraints := strings.Join(show.HelpConstraints, "\n")
	if !strings.Contains(constraints, "--context none") || !strings.Contains(constraints, "型付き直接子") || !strings.Contains(constraints, "--around") {
		t.Fatalf("messages show constraints = %q", constraints)
	}
}

func TestMessagesListHelpStatesStartIndexAndGlobalSearchLimits(t *testing.T) {
	var list CommandSpec
	for _, command := range chatworkCommandSpecs() {
		if command.Path == "messages list" {
			list = command
			break
		}
	}
	constraints := strings.Join(list.HelpConstraints, "\n")
	for _, want := range []string{"--on", "--since/--until", "--contains", "取得済み範囲", "ローカル", "API検索なし"} {
		if !strings.Contains(constraints, want) {
			t.Errorf("messages list constraints lack %q: %s", want, constraints)
		}
	}
	inputs := make(map[string]CommandInput, len(list.Agent.Inputs))
	for _, input := range list.Agent.Inputs {
		inputs[input.Name] = input
	}
	if description := inputs["--max-body-bytes"].Description; !strings.Contains(description, "文字境界") || !strings.Contains(description, "body-truncated") || !strings.Contains(description, "元バイト数") {
		t.Errorf("max body length help omits truncation details: %q", description)
	}
}

func TestMessageMutationCatalogPublishesExclusiveBoundedBodySources(t *testing.T) {
	commands := make(map[string]CommandSpec)
	allCommands := chatworkCommandSpecs()
	for _, command := range allCommands {
		commands[command.Path] = command
	}
	wantUsage := map[string]string{
		"messages send":   "cwk messages send --room <room-id> [--to <account-ref>] [--reply-to <message-ref>] [--body <text>] [--body-file <path>] [--self-unread]",
		"messages update": "cwk messages update --room <room-id> --message <message-ref> [--body <text>] [--body-file <path>]",
	}
	wantFaults := []string{
		"chatwork_message_body_stdin_unreadable",
		"chatwork_message_body_file_unreadable",
		"chatwork_message_body_too_large",
		"chatwork_message_body_invalid",
	}
	for path, usage := range wantUsage {
		t.Run(path, func(t *testing.T) {
			command := commands[path]
			if command.Usage() != usage {
				t.Fatalf("usage = %q, want %q", command.Usage(), usage)
			}
			inputs := make(map[string]CommandInput, len(command.Agent.Inputs))
			for _, input := range command.Agent.Inputs {
				inputs[input.Name] = input
			}
			if path == "messages send" {
				if !inputs["--to"].Repeatable || inputs["--to"].Required || inputs["--to"].ReferenceKind != string(chatwork.ReferenceAccount) {
					t.Fatalf("typed --to input = %+v", inputs["--to"])
				}
				if inputs["--reply-to"].Repeatable || inputs["--reply-to"].Required || inputs["--reply-to"].ReferenceKind != string(chatwork.ReferenceMessage) {
					t.Fatalf("typed --reply-to input = %+v", inputs["--reply-to"])
				}
				mutation := command.Agent.Mutation
				if mutation == nil || mutation.ParentInput != "--room" || mutation.TargetIDInput != "" ||
					!reflect.DeepEqual(mutation.TargetInputs, []string{"--room"}) ||
					!reflect.DeepEqual(mutation.PreflightInputs, []string{"--to", "--reply-to"}) ||
					mutation.Impact.Notification != operation.DeclarationYes {
					t.Fatalf("message-send mutation binding = %+v", mutation)
				}
			}
			if inputs["--body"].Required || inputs["--body-file"].Required ||
				!strings.Contains(inputs["--body"].Description, "標準入力") ||
				!strings.Contains(inputs["--body-file"].Description, "65535") {
				t.Fatalf("body inputs = body %+v file %+v", inputs["--body"], inputs["--body-file"])
			}
			constraints := strings.Join(command.HelpConstraints, "\n")
			if !strings.Contains(constraints, "どちらか一方") || !strings.Contains(constraints, "--body -") {
				t.Fatalf("constraints = %q", constraints)
			}
			declared := make(map[string]CommandError, len(command.Agent.Errors))
			for _, commandError := range command.Agent.Errors {
				declared[commandError.Code] = commandError
			}
			for _, code := range wantFaults {
				commandError, ok := declared[code]
				if !ok || commandError.Kind != fault.KindInvalidInput || commandError.Retryable {
					t.Errorf("fault %s = %+v, exists=%t", code, commandError, ok)
				}
			}
			output := renderCommandHelp(command, allCommands)
			if len(output) > maxExactHelpBytes {
				t.Fatalf("exact help is %d bytes: %s", len(output), output)
			}
			for _, want := range []string{"--body text 選択必須:", "--body-file path 選択必須:"} {
				if !strings.Contains(string(output), want) {
					t.Errorf("exact help lacks conditional requirement %q:\n%s", want, output)
				}
			}
		})
	}
}
