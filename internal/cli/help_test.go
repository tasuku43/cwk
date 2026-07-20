package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/fault"
)

func newInternalSampleHelpCLI(out, errOut *bytes.Buffer) *CLI {
	commands := append(DefaultCatalog().Commands(), sampleTestCommandSpecs()...)
	return newCLI(strings.NewReader(""), out, errOut, NewCatalog(commands...), passingInspector("unused"))
}

func TestRootTextHelpIsACatalogDerivedNamespaceIndex(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help"}); code != ExitOK {
		t.Fatalf("Run(help) code = %d, stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	want := "Chatwork CLI\n\n" +
		"使い方:\n" +
		"  cwk [--error-format text|json] <command> [arguments]\n\n" +
		"グローバルオプション:\n" +
		"  --error-format text|json  構造化エラーの表示形式を選択します（既定: text）\n\n" +
		"コマンド:\n" +
		"  doctor   ローカル環境を読み取り専用で診断する\n" +
		"  help     人向けヘルプまたはエージェント向けコマンド仕様を表示する\n" +
		"  version  バージョン情報を表示する\n" +
		"  config   エージェントに表示するコマンドを選択する\n\n" +
		"名前空間:\n" +
		"  account           2 コマンド\n" +
		"  personal-tasks    1 コマンド\n" +
		"  contacts          1 コマンド\n" +
		"  rooms             6 コマンド\n" +
		"  members           3 コマンド\n" +
		"  messages          7 コマンド\n" +
		"  room-tasks        4 コマンド\n" +
		"  files             3 コマンド\n" +
		"  invite-link       4 コマンド\n" +
		"  contact-requests  3 コマンド\n\n" +
		"コマンドを選ぶには 'cwk <namespace> --help' を実行してください。\n" +
		"詳細を確認するには、正確なコマンドの末尾に '--help' を付けてください。\n" +
		"結果・エラー・復旧を含む機械可読契約には 'cwk help <exact-command> --format agent' を実行してください。\n"
	if output != want {
		t.Fatalf("root text help = %q, want %q", output, want)
	}

	directCommands := make([]CommandSpec, 0)
	namespaceCounts := make(map[string]int)
	namespaceOrder := make([]string, 0)
	for _, spec := range command.catalog.Commands() {
		if !strings.Contains(spec.Path, " ") {
			directCommands = append(directCommands, spec)
			continue
		}
		namespace := commandNamespace(spec.Path)
		if namespaceCounts[namespace] == 0 {
			namespaceOrder = append(namespaceOrder, namespace)
		}
		namespaceCounts[namespace]++
		if strings.Contains(output, "  "+spec.Path+"  ") || strings.Contains(output, spec.Summary) {
			t.Errorf("root help leaked leaf command %q\n%s", spec.Path, output)
		}
	}

	lastOffset := -1
	directWidth := 0
	for _, spec := range directCommands {
		if len(spec.Path) > directWidth {
			directWidth = len(spec.Path)
		}
	}
	for _, spec := range directCommands {
		line := fmt.Sprintf("  %-*s  %s\n", directWidth, spec.Path, spec.Summary)
		offset := strings.Index(output, line)
		if offset <= lastOffset || strings.Count(output, line) != 1 {
			t.Errorf("direct command line %q is missing, duplicated, or out of section order\n%s", line, output)
		}
		lastOffset = offset
	}
	namespaceWidth := 0
	for _, namespace := range namespaceOrder {
		if len(namespace) > namespaceWidth {
			namespaceWidth = len(namespace)
		}
	}
	for _, namespace := range namespaceOrder {
		line := fmt.Sprintf("  %-*s  %d コマンド\n", namespaceWidth, namespace, namespaceCounts[namespace])
		offset := strings.Index(output, line)
		if offset <= lastOffset || strings.Count(output, line) != 1 {
			t.Errorf("namespace line %q is missing, duplicated, or out of catalog order\n%s", line, output)
		}
		selected, exact := command.catalog.Select(namespace)
		if exact || len(selected) != namespaceCounts[namespace] {
			t.Errorf("namespace %q does not round-trip: exact=%t commands=%d", namespace, exact, len(selected))
		}
		lastOffset = offset
	}
	for _, want := range []string{
		"コマンドを選ぶには 'cwk <namespace> --help' を実行してください。",
		"詳細を確認するには、正確なコマンドの末尾に '--help' を付けてください。",
		"結果・エラー・復旧を含む機械可読契約には 'cwk help <exact-command> --format agent' を実行してください。",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("root help lacks navigation %q\n%s", want, output)
		}
	}
}

func TestCommandHelpUsesCatalogMetadataAndDerivedReferences(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := newInternalSampleHelpCLI(&stdout, &stderr)
	if code := runCLI(command, []string{"sample", "read", "--help"}); code != ExitOK {
		t.Fatalf("Run(sample read --help) code = %d, stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"使い方:\n  cwk sample read --id <sample-id> [--format tsv|json]",
		"Read exactly one offline sample by opaque ID.",
		"入力:\n  --id      必須 flag, reference=sample",
		"Pass an id from sample list byte-for-byte without parsing or transformation.",
		"  --format  任意 flag, values=tsv|json",
		"Select the single sample representation.",
		"効果: read",
		"役割: act",
		"使用する参照: sample（入力 --id）",
		"この名前空間の他のコマンドには 'cwk sample --help' を実行してください。",
		"機械可読契約には 'cwk help sample read --format agent' を実行してください。",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("command help lacks %q\n%s", want, output)
		}
	}
}

func TestRootAgentHelpIsACompactProjectionOfTheCatalog(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help", "--format", "agent"}); code != ExitOK {
		t.Fatalf("Run(agent help) code = %d, stderr = %q", code, stderr.String())
	}

	var document agentIndexDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatalf("agent help is not JSON: %v\n%s", err, stdout.String())
	}
	if document.SchemaVersion != agentHelpSchemaVersion || document.View != "index" || document.Program != ProgramName {
		t.Fatalf("agent document header = %+v", document)
	}
	if document.ScopeRequest.InvocationTemplate != "cwk help <exact-command> --format agent" ||
		!reflect.DeepEqual(document.ScopeRequest.SelectorFields, []string{"commands[].path"}) ||
		document.ScopeRequest.UnknownOutcomeMaxInvocations != 2 || document.ScopeRequest.KnownPathMaxInvocations != 1 {
		t.Fatalf("scope request = %+v", document.ScopeRequest)
	}
	specs := command.catalog.Commands()
	if len(document.Commands) != len(specs) {
		t.Fatalf("agent commands = %d, catalog commands = %d", len(document.Commands), len(specs))
	}
	for index, spec := range specs {
		got := document.Commands[index]
		if got.Path != spec.Path || got.Namespace != commandNamespace(spec.Path) || got.Summary != spec.Summary ||
			got.CapabilityID != spec.Agent.CapabilityID || got.Outcome != spec.Agent.Outcome ||
			got.Effect != spec.Effect.String() || got.Role != spec.Role.String() {
			t.Errorf("agent command %d = %+v, want catalog %+v", index, got, spec)
		}
	}
}

func TestScopedAgentHelpIsACompleteProjectionOfEveryCatalogCommand(t *testing.T) {
	command := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	for _, spec := range command.catalog.Commands() {
		t.Run(strings.ReplaceAll(spec.Path, " ", "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			selected := New(strings.NewReader(""), &stdout, &stderr)
			args := append([]string{"help"}, strings.Fields(spec.Path)...)
			args = append(args, "--format=agent")
			if code := runCLI(selected, args); code != ExitOK {
				t.Fatalf("Run(%v) code = %d, stderr = %q", args, code, stderr.String())
			}
			var document agentDocument
			if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
				t.Fatalf("agent help is not JSON: %v\n%s", err, stdout.String())
			}
			if document.SchemaVersion != agentHelpSchemaVersion || document.View != "scope" || document.Program != ProgramName ||
				document.Scope != (agentScope{Selector: spec.Path, Kind: "command"}) {
				t.Fatalf("agent document header = %+v", document)
			}
			if len(document.GlobalInputs) != 1 || document.GlobalInputs[0].Name != "--error-format" ||
				!reflect.DeepEqual(document.GlobalInputs[0].AllowedValues, []string{"text", "json"}) ||
				document.ErrorContract.CommandErrorsField != "commands[].contract.errors" || len(document.ErrorContract.ExitCodes) != 12 ||
				len(document.ErrorContract.GlobalErrors) != 9 || document.ErrorContract.JSONSchemaVersion != 1 {
				t.Fatalf("global agent contract = %+v / %+v", document.GlobalInputs, document.ErrorContract)
			}
			if document.IOContract.SuccessStream != "stdout" || document.IOContract.ErrorStream != "stderr" ||
				!document.IOContract.SuccessStatusRequiresCompleteWrite || document.IOContract.PartialOutputIsSuccess ||
				document.IOContract.ExternalTextTrust != "untrusted_data" ||
				document.IOContract.ExternalTextProjection != "visible_escape" ||
				document.IOContract.OpaqueReferencePolicy != "validated_exact_bytes" {
				t.Fatalf("I/O contract = %+v", document.IOContract)
			}
			if len(document.Commands) != 1 {
				t.Fatalf("selected commands = %+v", document.Commands)
			}
			got := document.Commands[0]
			if got.Path != spec.Path || got.Summary != spec.Summary || got.Usage != spec.Usage() || got.Args != spec.Args ||
				got.Effect != spec.Effect.String() || got.Role != spec.Role.String() ||
				!reflect.DeepEqual(got.Contract, spec.Agent) ||
				!reflect.DeepEqual(got.ProducesRefs, spec.ProducedRefs()) ||
				!reflect.DeepEqual(got.ConsumesRefs, spec.ConsumedRefs()) {
				t.Errorf("agent command = %+v, want catalog %+v", got, spec)
			}
			if got.Contract.Output.DefaultFormat == OutputFormatUnknown ||
				(containsOutputFormat(got.Contract.Output.Formats, OutputFormatJSON) && got.Contract.Output.JSONSchemaVersion <= 0) {
				t.Errorf("agent command %q has incomplete output metadata: %+v", got.Path, got.Contract.Output)
			}
		})
	}
}

func TestAgentErrorContractDefinesUnknownAndAdvisoryRateLimitTiming(t *testing.T) {
	contract := defaultAgentErrorContract()
	fields := make(map[string]string, len(contract.Fields))
	for _, field := range contract.Fields {
		fields[field.Name] = field.Description
	}
	if !strings.Contains(fields["retry_after"], "不明") ||
		!strings.Contains(fields["retry_after"], "再実行する許可ではありません") {
		t.Fatalf("retry_after contract = %q", fields["retry_after"])
	}
	if !strings.Contains(fields["retryable"], "自動再試行は常に行いません") {
		t.Fatalf("retryable contract = %q", fields["retryable"])
	}
}

func TestDoctorAgentHelpDeclaresCommandSelectionGrammarAndRuntimeRecovery(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help", "doctor", "--format=agent"}); code != ExitOK {
		t.Fatalf("Run(help doctor --format=agent) code = %d, stderr = %q", code, stderr.String())
	}
	var document agentDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatalf("doctor agent help is not JSON: %v\n%s", err, stdout.String())
	}
	if len(document.Commands) != 1 || document.Commands[0].Path != "doctor" {
		t.Fatalf("doctor commands = %+v", document.Commands)
	}
	var detailDescription string
	for _, field := range document.Commands[0].Contract.Output.Fields {
		if field.Name == "detail" {
			detailDescription = field.Description
			break
		}
	}
	if !strings.Contains(detailDescription, commandSelectionDoctorDetailGrammar) ||
		!strings.Contains(detailDescription, "count は負でない10進整数") {
		t.Fatalf("doctor detail contract does not publish the fixed grammar: %q", detailDescription)
	}

	for _, code := range []string{"command_selection_unsafe", "command_selection_unavailable"} {
		var declared *CommandError
		for index := range document.ErrorContract.GlobalErrors {
			candidate := &document.ErrorContract.GlobalErrors[index]
			if candidate.Code == code {
				declared = candidate
				break
			}
		}
		if declared == nil || len(declared.NextActions) != 1 || declared.NextActions[0].Command != "doctor" {
			t.Fatalf("declared %s recovery = %+v", code, declared)
		}

		runtimeFault := commandSelectionDispatchFault(fault.New(fault.KindUnavailable, code, code, false))
		structured, ok := fault.PublicCopy(runtimeFault)
		if !ok || len(structured.NextActions) != 1 || structured.NextActions[0] != declared.NextActions[0] {
			t.Fatalf("runtime %s recovery = %+v; declared = %+v", code, structured, declared)
		}
	}
}

func TestConfigAgentHelpDeclaresExactOutputFields(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help", "config", "--format=agent"}); code != ExitOK {
		t.Fatalf("Run(help config --format=agent) code = %d, stderr = %q", code, stderr.String())
	}
	var document agentDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatalf("config agent help is not JSON: %v\n%s", err, stdout.String())
	}
	if len(document.Commands) != 1 || document.Commands[0].Path != "config" {
		t.Fatalf("config commands = %+v", document.Commands)
	}
	fields := document.Commands[0].Contract.Output.Fields
	got := make([]string, len(fields))
	for index, field := range fields {
		got[index] = field.Name
	}
	want := []string{"status", "visible", "hidden", "changed", "cleaned"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("config output fields = %v, want %v", got, want)
	}
}

func TestAgentHelpRootNamespaceAndExactShapeSnapshots(t *testing.T) {
	root := runAgentHelpForTest(t, []string{"help", "--format=agent"})
	assertJSONKeys(t, root, []string{"commands", "program", "schema_version", "scope_request", "view"})
	var rootCommands []map[string]json.RawMessage
	if err := json.Unmarshal(root["commands"], &rootCommands); err != nil {
		t.Fatal(err)
	}
	assertJSONKeys(t, rootCommands[0], []string{"capability_id", "effect", "namespace", "outcome", "path", "role", "summary"})
	var scopeRequest map[string]json.RawMessage
	if err := json.Unmarshal(root["scope_request"], &scopeRequest); err != nil {
		t.Fatal(err)
	}
	assertJSONKeys(t, scopeRequest, []string{"invocation_template", "known_path_max_invocations", "selector_fields", "unknown_outcome_max_invocations"})

	namespace := runAgentHelpForTest(t, []string{"help", "sample", "--format=agent"})
	assertJSONKeys(t, namespace, []string{"commands", "program", "schema_version", "scope", "scope_request", "view"})
	var namespaceCommands []map[string]json.RawMessage
	if err := json.Unmarshal(namespace["commands"], &namespaceCommands); err != nil {
		t.Fatal(err)
	}
	assertJSONKeys(t, namespaceCommands[0], []string{"capability_id", "effect", "namespace", "outcome", "path", "role", "summary"})

	scoped := runAgentHelpForTest(t, []string{"help", "sample", "read", "--format=agent"})
	assertJSONKeys(t, scoped, []string{"commands", "error_contract", "global_inputs", "io_contract", "program", "schema_version", "scope", "view", "workflows"})
	var ioContract map[string]json.RawMessage
	if err := json.Unmarshal(scoped["io_contract"], &ioContract); err != nil {
		t.Fatal(err)
	}
	assertJSONKeys(t, ioContract, []string{"error_stream", "external_text_projection", "external_text_trust", "opaque_reference_policy", "partial_output_is_success", "success_status_requires_complete_write", "success_stream"})
	var scopedCommands []map[string]json.RawMessage
	if err := json.Unmarshal(scoped["commands"], &scopedCommands); err != nil {
		t.Fatal(err)
	}
	assertJSONKeys(t, scopedCommands[0], []string{"args", "consumes_refs", "contract", "effect", "next_actions", "path", "produces_refs", "role", "summary", "usage"})
}

func TestRootAgentHelpSizeGrowthContainsOnlyIndexFields(t *testing.T) {
	command := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	base := utilitySpec("base")
	makeCommands := func(count int) []CommandSpec {
		commands := make([]CommandSpec, 0, count)
		for index := 0; index < count; index++ {
			spec := cloneCommandSpec(base)
			spec.Path = fmt.Sprintf("area inspect%03d", index)
			spec.Summary = "Inspect one bounded synthetic area"
			spec.Agent.CapabilityID = fmt.Sprintf("area.inspect%03d", index)
			spec.Agent.Outcome = "Inspect one bounded synthetic area without external I/O"
			for errorIndex := range spec.Agent.Errors {
				for actionIndex := range spec.Agent.Errors[errorIndex].NextActions {
					spec.Agent.Errors[errorIndex].NextActions[actionIndex].Command = spec.Path
				}
			}
			commands = append(commands, spec)
		}
		return commands
	}
	one, err := command.renderAgentIndex(makeCommands(1))
	if err != nil {
		t.Fatal(err)
	}
	many, err := command.renderAgentIndex(makeCommands(101))
	if err != nil {
		t.Fatal(err)
	}
	namespace, err := command.renderAgentNamespaceIndex("area", makeCommands(101))
	if err != nil {
		t.Fatal(err)
	}
	perCommandGrowth := (len(many) - len(one)) / 100
	if perCommandGrowth > 320 {
		t.Fatalf("root index grew by %d bytes per command, want <= 320", perCommandGrowth)
	}
	if len(namespace) > len(many)+100 {
		t.Fatalf("namespace index bytes = %d, root index bytes = %d", len(namespace), len(many))
	}
	catalog := NewCatalog(makeCommands(100)...)
	if err := catalog.Validate(); err != nil {
		t.Fatalf("100-command catalog failed validation: %v", err)
	}
	if selected, exact := catalog.Select("area"); exact || len(selected) != 100 {
		t.Fatalf("100-command namespace selection exact=%t, commands=%d", exact, len(selected))
	}
	if selected, exact := catalog.Select("area inspect042"); !exact || len(selected) != 1 || selected[0].Path != "area inspect042" {
		t.Fatalf("exact selection exact=%t, commands=%+v", exact, selected)
	}
	if selected, exact := catalog.Select("are"); exact || len(selected) != 0 {
		t.Fatalf("non-boundary selector exact=%t, commands=%+v", exact, selected)
	}
	for _, forbidden := range []string{"global_inputs", "io_contract", "error_contract", "workflows", "contract", "usage", "args", "inputs", "output", "errors", "mutation"} {
		for label, output := range map[string][]byte{"root": many, "namespace": namespace} {
			if bytes.Contains(output, []byte(`"`+forbidden+`"`)) {
				t.Errorf("%s index leaked detailed field %q", label, forbidden)
			}
		}
	}

	oversized := cloneCommandSpec(base)
	oversized.Summary = strings.Repeat("s", maxAgentIndexEntryBytes)
	if err := NewCatalog(oversized).Validate(); err == nil || !strings.Contains(err.Error(), "agent index entry") {
		t.Fatalf("oversized root index entry error = %v", err)
	}
}

func TestCatalogSelectReturnsDeepCopiesForScopedProjection(t *testing.T) {
	commands := append(DefaultCatalog().Commands(), sampleTestCommandSpecs()...)
	catalog := NewCatalog(commands...)
	before := catalog.Commands()

	namespace, exact := catalog.Select("sample")
	if exact || len(namespace) != 2 {
		t.Fatalf("namespace selection exact=%t, commands=%+v", exact, namespace)
	}
	namespace[0].Agent.Inputs[0].AllowedValues[0] = "changed"
	namespace[0].Agent.Output.Fields[0].Name = "changed"
	namespace[0].Agent.Errors[0].NextActions[0].Command = "changed"

	selected, exact := catalog.Select("sample read")
	if !exact || len(selected) != 1 {
		t.Fatalf("exact selection exact=%t, commands=%+v", exact, selected)
	}
	selected[0].Agent.Inputs[0].ReferenceKind = "changed"
	selected[0].Agent.Output.Formats[0] = OutputFormatNone

	after := catalog.Commands()
	for index := range before {
		if before[index].Path != after[index].Path || before[index].Summary != after[index].Summary ||
			before[index].Args != after[index].Args || before[index].Effect != after[index].Effect ||
			before[index].Role != after[index].Role || !reflect.DeepEqual(before[index].Agent, after[index].Agent) {
			t.Fatalf("mutating scoped selections changed catalog command %q", before[index].Path)
		}
	}
}

func runAgentHelpForTest(t *testing.T, args []string) map[string]json.RawMessage {
	t.Helper()
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if len(args) > 1 && args[1] == "sample" {
		command = newInternalSampleHelpCLI(&stdout, &stderr)
	}
	if code := runCLI(command, args); code != ExitOK {
		t.Fatalf("Run(%v) code = %d, stderr = %q", args, code, stderr.String())
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatalf("agent help is not JSON: %v\n%s", err, stdout.String())
	}
	return document
}

func runSampleExactAgentHelp(t *testing.T, path string) agentDocument {
	t.Helper()
	var stdout, stderr bytes.Buffer
	command := newInternalSampleHelpCLI(&stdout, &stderr)
	args := append([]string{"help"}, strings.Fields(path)...)
	args = append(args, "--format=agent")
	if code := runCLI(command, args); code != ExitOK {
		t.Fatalf("Run(%v) code = %d, stderr = %q", args, code, stderr.String())
	}
	var document agentDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatalf("exact agent help is not JSON: %v\n%s", err, stdout.String())
	}
	return document
}

func assertJSONKeys(t *testing.T, document map[string]json.RawMessage, want []string) {
	t.Helper()
	got := make([]string, 0, len(document))
	for key := range document {
		got = append(got, key)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON keys = %v, want %v", got, want)
	}
}

func containsOutputFormat(formats []OutputFormat, wanted OutputFormat) bool {
	for _, format := range formats {
		if format == wanted {
			return true
		}
	}
	return false
}

func TestAgentHelpNamespaceIsACompactIndexWithExactCommandPointers(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := newInternalSampleHelpCLI(&stdout, &stderr)
	if code := runCLI(command, []string{"help", "sample", "--format=agent"}); code != ExitOK {
		t.Fatalf("Run(namespace agent help) code = %d, stderr = %q", code, stderr.String())
	}
	var document agentIndexDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.SchemaVersion != agentHelpSchemaVersion || document.View != "index" || document.Program != ProgramName ||
		document.Scope == nil || *document.Scope != (agentScope{Selector: "sample", Kind: "namespace"}) {
		t.Fatalf("namespace index header = %+v", document)
	}
	if len(document.Commands) != 2 || document.Commands[0].Path != "sample list" || document.Commands[1].Path != "sample read" {
		t.Fatalf("namespace commands = %+v", document.Commands)
	}
	if document.ScopeRequest.InvocationTemplate != "cwk help <exact-command> --format agent" ||
		!reflect.DeepEqual(document.ScopeRequest.SelectorFields, []string{"commands[].path"}) ||
		document.ScopeRequest.UnknownOutcomeMaxInvocations != 2 || document.ScopeRequest.KnownPathMaxInvocations != 1 {
		t.Fatalf("namespace scope request = %+v", document.ScopeRequest)
	}
	for _, entry := range document.Commands {
		if !strings.HasPrefix(entry.Path, "sample ") {
			t.Fatalf("unscoped command leaked into namespace help: %+v", entry)
		}
	}
}

func TestCompleteAgentHelpRejectsNamespaceAggregation(t *testing.T) {
	command := newInternalSampleHelpCLI(&bytes.Buffer{}, &bytes.Buffer{})
	commands, exact := command.catalog.Select("sample")
	if exact || len(commands) != 2 {
		t.Fatalf("sample namespace selection exact=%t commands=%d", exact, len(commands))
	}
	if _, err := command.renderAgentHelp("sample", exact, commands); err == nil ||
		!strings.Contains(err.Error(), "正確なコマンドを一つ") {
		t.Fatalf("namespace complete-contract error = %v", err)
	}
}

func TestTextHelpCanSelectNamespace(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := newInternalSampleHelpCLI(&stdout, &stderr)
	if code := runCLI(command, []string{"help", "sample"}); code != ExitOK {
		t.Fatalf("Run(namespace help) code = %d, stderr = %q", code, stderr.String())
	}
	want := "Chatwork CLI\n\n" +
		"使い方:\n" +
		"  cwk sample <command> [arguments]\n\n" +
		"コマンド:\n" +
		"  list  Discover offline samples and their opaque IDs\n" +
		"  read  Read exactly one offline sample by opaque ID\n\n" +
		"正確なコマンドの詳細には 'cwk sample <command> --help' を実行してください。\n" +
		"1コマンドの機械可読契約には 'cwk help sample <command> --format agent' を実行してください。\n" +
		"この名前空間の機械可読索引には 'cwk help sample --format agent' を実行してください。\n" +
		"全コマンドと名前空間には 'cwk --help' を実行してください。\n"
	if stdout.String() != want {
		t.Fatalf("namespace text = %q, want %q", stdout.String(), want)
	}
}

func TestProductionRoomsNamespaceTextHelpGolden(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"rooms", "--help"}); code != ExitOK {
		t.Fatalf("Run(rooms --help) code = %d, stderr = %q", code, stderr.String())
	}
	want := "Chatwork CLI\n\n" +
		"使い方:\n" +
		"  cwk rooms <command> [arguments]\n\n" +
		"コマンド:\n" +
		"  list    参加中のChatworkルームを検索する\n" +
		"  create  メンバーを正確に指定してグループチャットを作成する\n" +
		"  show    完全一致するルームを一つ表示する\n" +
		"  update  完全一致するルームの説明情報を更新する\n" +
		"  leave   完全一致するグループチャットから退席する\n" +
		"  delete  完全一致するグループチャットを完全に削除する\n\n" +
		"正確なコマンドの詳細には 'cwk rooms <command> --help' を実行してください。\n" +
		"1コマンドの機械可読契約には 'cwk help rooms <command> --format agent' を実行してください。\n" +
		"この名前空間の機械可読索引には 'cwk help rooms --format agent' を実行してください。\n" +
		"全コマンドと名前空間には 'cwk --help' を実行してください。\n"
	if stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q, want stdout %q", stdout.String(), stderr.String(), want)
	}
}

func TestMessageListHumanHelpPublishesBoundedSelection(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"messages", "list", "--help"}); code != ExitOK {
		t.Fatalf("Run(messages list --help) code = %d, stderr = %q", code, stderr.String())
	}
	for _, want := range []string{
		"--room               必須 flag, reference=chatwork-room",
		"--window             任意 flag, values=recent|changes",
		"最新の上限付き範囲（recent、既定値）",
		"プロバイダーの差分範囲（changes）",
		"--start-index        任意 flag",
		"1始まりの順位（1〜100）",
		"--count              任意 flag",
		"終了順位ではありません",
		"--start-index 11 --count 20 は順位11〜30を選びます",
		"直接の返信コンテキストにより、表示件数はこの値を超えることがあります",
		"--sender             任意・繰り返し可 flag, reference=chatwork-account",
		"列挙した送信者のいずれかに一致させる（OR）には繰り返し指定し、完全一致参照は最大100件",
		"--context            任意 flag, values=none|replies",
		"上限付き範囲内にある明示的な返信元・返信先を1ホップだけ含める（replies）",
		"--resolve-relations  任意 flag",
		"正規 message_ref による追加の一件取得で再帰的に補う最大件数（0〜100）",
		"既定値は5、0は追加取得を無効化します",
		"機械可読契約には 'cwk help messages list --format agent' を実行してください。",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("messages list help lacks %q\n%s", want, stdout.String())
		}
	}
}

func TestMessageListScopedAgentHelpPublishesSelectionDefaults(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help", "messages", "list", "--format=agent"}); code != ExitOK {
		t.Fatalf("Run(messages list agent help) code = %d, stderr = %q", code, stderr.String())
	}
	var document agentDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatalf("agent help is not JSON: %v\n%s", err, stdout.String())
	}
	if len(document.Commands) != 1 || document.Commands[0].Path != "messages list" {
		t.Fatalf("agent commands = %+v", document.Commands)
	}
	var startIndex, count, window *CommandInput
	for index := range document.Commands[0].Contract.Inputs {
		input := &document.Commands[0].Contract.Inputs[index]
		switch input.Name {
		case "--start-index":
			startIndex = input
		case "--count":
			count = input
		case "--window":
			window = input
		}
	}
	if startIndex == nil || startIndex.Required || startIndex.Repeatable || startIndex.Source != InputSourceFlag ||
		!strings.Contains(startIndex.Description, "1始まり") || !strings.Contains(startIndex.Description, "100") ||
		!strings.Contains(startIndex.Description, "省略時は1") {
		t.Fatalf("agent start-index contract = %+v", startIndex)
	}
	if count == nil || count.Required || count.Repeatable || count.Source != InputSourceFlag ||
		!strings.Contains(count.Description, "1〜100") || !strings.Contains(count.Description, "終了順位ではありません") ||
		!strings.Contains(count.Description, "順位11〜30") || !strings.Contains(count.Description, "返信コンテキスト") {
		t.Fatalf("agent count contract = %+v", count)
	}
	if window == nil || window.Required || window.Repeatable || window.Source != InputSourceFlag ||
		!reflect.DeepEqual(window.AllowedValues, []string{"recent", "changes"}) ||
		!strings.Contains(window.Description, "recent、既定値") ||
		!strings.Contains(window.Description, "差分") {
		t.Fatalf("agent window contract = %+v", window)
	}
}

func TestEveryCatalogInputIsProjectedIntoExactHumanHelp(t *testing.T) {
	commands := DefaultCatalog().Commands()
	for _, command := range commands {
		output := string(renderCommandHelp(command, commands))
		if len(command.Agent.Inputs) == 0 {
			if strings.Contains(output, "\n入力:\n") {
				t.Errorf("input-free command %q rendered an Inputs section\n%s", command.Path, output)
			}
			continue
		}
		if !strings.Contains(output, "\n入力:\n") {
			t.Errorf("command %q lacks Inputs section\n%s", command.Path, output)
			continue
		}
		width := 0
		for _, input := range command.Agent.Inputs {
			if len(input.Name) > width {
				width = len(input.Name)
			}
		}
		for _, input := range command.Agent.Inputs {
			requirement := "任意"
			if input.Required {
				requirement = "必須"
			}
			if input.Repeatable {
				requirement += "・繰り返し可"
			}
			attributes := []string{requirement + " " + string(input.Source)}
			if len(input.AllowedValues) > 0 {
				attributes = append(attributes, "values="+strings.Join(input.AllowedValues, "|"))
			}
			if input.ReferenceKind != "" {
				attributes = append(attributes, "reference="+input.ReferenceKind)
			}
			line := fmt.Sprintf("  %-*s  %s\n    %s\n", width, input.Name, strings.Join(attributes, ", "), input.Description)
			if strings.Count(output, line) != 1 {
				t.Errorf("command %q input %q projection count != 1\nwant:\n%soutput:\n%s", command.Path, input.Name, line, output)
			}
		}
	}
}

func TestNestedNamespaceHelpUsesWordBoundariesAndRelativePaths(t *testing.T) {
	help, found := DefaultCatalog().Lookup("help")
	if !found {
		t.Fatal("default catalog lacks help")
	}
	catalog := NewCatalog(
		help,
		utilitySpec("area group first"),
		utilitySpec("area group second"),
		utilitySpec("area other inspect"),
	)
	run := func(args ...string) (int, string, string) {
		var stdout, stderr bytes.Buffer
		command := newCLI(strings.NewReader(""), &stdout, &stderr, catalog, passingInspector("unused"))
		code := runCLI(command, args)
		return code, stdout.String(), stderr.String()
	}

	aliasCode, aliasOut, aliasErr := run("area", "group", "--help")
	canonicalCode, canonicalOut, canonicalErr := run("help", "area", "group")
	if aliasCode != ExitOK || canonicalCode != ExitOK || aliasOut != canonicalOut || aliasErr != canonicalErr {
		t.Fatalf("nested namespace alias/canonical = %d %q %q / %d %q %q", aliasCode, aliasOut, aliasErr, canonicalCode, canonicalOut, canonicalErr)
	}
	for _, want := range []string{
		"  cwk area group <command> [arguments]",
		"  first   Complete a test outcome",
		"  second  Complete a test outcome",
	} {
		if !strings.Contains(aliasOut, want) {
			t.Errorf("nested namespace help lacks %q\n%s", want, aliasOut)
		}
	}
	for _, forbidden := range []string{"area group first", "area group second", "area other inspect"} {
		if strings.Contains(aliasOut, forbidden) {
			t.Errorf("nested namespace help leaked %q\n%s", forbidden, aliasOut)
		}
	}

	exactAliasCode, exactAliasOut, exactAliasErr := run("area", "group", "first", "--help")
	exactCode, exactOut, exactErr := run("help", "area", "group", "first")
	if exactAliasCode != ExitOK || exactCode != ExitOK || exactAliasOut != exactOut || exactAliasErr != exactErr {
		t.Fatalf("nested exact alias/canonical = %d %q %q / %d %q %q", exactAliasCode, exactAliasOut, exactAliasErr, exactCode, exactOut, exactErr)
	}

	partialCode, partialOut, partialErr := run("area", "grou", "--help")
	if partialCode != ExitUsage || partialOut != "" || !strings.Contains(partialErr, "code: unknown_command") {
		t.Fatalf("partial namespace = %d %q %q", partialCode, partialOut, partialErr)
	}
}

func TestHumanRootHelpGrowthDependsOnNamespacesNotLeafCommands(t *testing.T) {
	base := utilitySpec("base")
	commands := make([]CommandSpec, 0, 100)
	for index := 0; index < 100; index++ {
		spec := cloneCommandSpec(base)
		spec.Path = fmt.Sprintf("area inspect%03d", index)
		spec.Summary = fmt.Sprintf("Inspect synthetic area %03d", index)
		commands = append(commands, spec)
	}
	output := string((&CLI{catalog: NewCatalog(commands...)}).renderRootHelp())
	if strings.Count(output, "area") != 1 || !strings.Contains(output, "  area  100 コマンド\n") {
		t.Fatalf("one namespace with 100 leaves was not collapsed\n%s", output)
	}
	for _, command := range commands {
		if strings.Contains(output, command.Path) || strings.Contains(output, command.Summary) {
			t.Fatalf("root help leaked leaf %q\n%s", command.Path, output)
		}
	}
}

func TestExactCommandHelpOmitsRecipesWithUnavailableSteps(t *testing.T) {
	commands := DefaultCatalog().Commands()
	withoutFind := make([]CommandSpec, 0, len(commands)-1)
	for _, command := range commands {
		if command.Path != "members find" {
			withoutFind = append(withoutFind, command)
		}
	}
	catalog := NewCatalog(withoutFind...)
	messages, exists := catalog.Lookup("messages list")
	if !exists {
		t.Fatal("messages list is absent")
	}
	output := string(renderCommandHelp(messages, catalog.Commands()))
	if strings.Contains(output, "人物名からその人の投稿を探す") || strings.Contains(output, "cwk members find") {
		t.Fatalf("exact help retained an unavailable recipe:\n%s", output)
	}
	if !strings.Contains(output, "今日または昨日の会話を確認する") {
		t.Fatalf("independent available recipe disappeared:\n%s", output)
	}
}

func TestOnlyExactHumanHelpShowsRecipesRelatedToItsCommand(t *testing.T) {
	tests := []struct {
		args      []string
		want      []string
		forbidden []string
	}{
		{
			args:      []string{"help"},
			forbidden: []string{"よくある使い方", "人物名からその人の投稿を探す", "今日または昨日の会話を確認する"},
		},
		{
			args:      []string{"help", "messages"},
			forbidden: []string{"よくある使い方", "人物名からその人の投稿を探す", "今日または昨日の会話を確認する"},
		},
		{
			args:      []string{"help", "members"},
			forbidden: []string{"よくある使い方", "人物名からその人の投稿を探す", "今日または昨日の会話を確認する"},
		},
		{
			args: []string{"help", "messages", "list"},
			want: []string{"人物名からその人の投稿を探す", "今日または昨日の会話を確認する"},
		},
		{
			args:      []string{"help", "members", "find"},
			want:      []string{"人物名からその人の投稿を探す"},
			forbidden: []string{"今日または昨日の会話を確認する"},
		},
		{
			args:      []string{"help", "messages", "show"},
			forbidden: []string{"よくある使い方", "人物名からその人の投稿を探す", "今日または昨日の会話を確認する"},
		},
	}
	for _, test := range tests {
		t.Run(strings.Join(test.args[1:], " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			command := New(strings.NewReader(""), &stdout, &stderr)
			if code := runCLI(command, test.args); code != ExitOK {
				t.Fatalf("help code=%d stderr=%q", code, stderr.String())
			}
			for _, want := range test.want {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("related help lacks %q:\n%s", want, stdout.String())
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(stdout.String(), forbidden) {
					t.Errorf("related help contains unrelated recipe %q:\n%s", forbidden, stdout.String())
				}
			}
		})
	}
}

func TestHumanRecipesDoNotEnterAnyAgentHelpShape(t *testing.T) {
	for _, args := range [][]string{
		{"help", "--format", "agent"},
		{"help", "messages", "list", "--format", "agent"},
	} {
		var stdout, stderr bytes.Buffer
		command := New(strings.NewReader(""), &stdout, &stderr)
		if code := runCLI(command, args); code != ExitOK {
			t.Fatalf("agent help %v code=%d stderr=%q", args, code, stderr.String())
		}
		for _, forbidden := range []string{"よくある使い方", "人物名からその人の投稿を探す", "候補が複数なら account-ref", "昨日は today を yesterday"} {
			if strings.Contains(stdout.String(), forbidden) {
				t.Fatalf("agent help %v contains human recipe %q", args, forbidden)
			}
		}
		if len(args) == 3 && (!strings.Contains(stdout.String(), `"path":"members find"`) ||
			!strings.Contains(stdout.String(), `"summary":"ルーム内の表示名からメンバー候補を探す"`)) {
			t.Fatalf("root agent index cannot select members find: %s", stdout.String())
		}
	}
}

func TestDefaultHumanRecipeStepsResolveToExactCatalogCommands(t *testing.T) {
	catalog := DefaultCatalog()
	for _, command := range catalog.Commands() {
		for _, recipe := range command.HumanRecipes {
			for _, step := range recipe.Steps {
				if _, exists := catalog.Lookup(step.Path); !exists {
					t.Errorf("recipe %q step %q is not an exact catalog command", recipe.Title, step.Path)
				}
			}
		}
	}
}

func TestMessagesListExactAgentHelpIncludesMemberFindSenderWorkflow(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help", "messages", "list", "--format", "agent"}); code != ExitOK {
		t.Fatalf("messages list agent help code=%d stderr=%q", code, stderr.String())
	}
	var document agentDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, workflow := range document.Workflows {
		if workflow.ReferenceKind == "chatwork-account" && workflow.Producer.Path == "members find" &&
			workflow.Producer.Field == "account_ref" && workflow.Consumer.Path == "messages list" &&
			workflow.Consumer.Input == "--sender" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("messages list exact help lacks members find -> --sender workflow: %+v", document.Workflows)
	}
}

func TestHumanRootHelpGroupsKindsAndPreservesSectionRelativeCatalogOrder(t *testing.T) {
	paths := []string{"area first", "local", "zone first", "other", "area second"}
	commands := make([]CommandSpec, 0, len(paths))
	for _, path := range paths {
		spec := utilitySpec("base")
		spec.Path = path
		spec.Summary = "Summary for " + path
		commands = append(commands, spec)
	}
	output := string((&CLI{catalog: NewCatalog(commands...)}).renderRootHelp())
	positions := []int{
		strings.Index(output, "  local  Summary for local\n"),
		strings.Index(output, "  other  Summary for other\n"),
		strings.Index(output, "  area  2 コマンド\n"),
		strings.Index(output, "  zone  1 コマンド\n"),
	}
	for index, position := range positions {
		if position < 0 || index > 0 && position <= positions[index-1] {
			t.Fatalf("root entries are missing or out of section-relative order: positions=%v\n%s", positions, output)
		}
	}
}

func TestAgentHelpPreservesTopLevelCompatibilityFields(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := newInternalSampleHelpCLI(&stdout, &stderr)
	if code := runCLI(command, []string{"help", "sample", "list", "--format=agent"}); code != ExitOK {
		t.Fatalf("Run(selected agent help) code = %d, stderr = %q", code, stderr.String())
	}
	var raw struct {
		Commands []map[string]json.RawMessage `json:"commands"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"path", "summary", "usage", "effect", "role", "produces_refs", "consumes_refs"} {
		if _, exists := raw.Commands[0][field]; !exists {
			t.Errorf("scoped agent command lacks compatibility field %q", field)
		}
	}
	if _, exists := raw.Commands[0]["contract"]; !exists {
		t.Error("scoped agent command lacks structured contract")
	}
}

func TestAgentHelpCanSelectOneCatalogCommandWithItsWorkflow(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := newInternalSampleHelpCLI(&stdout, &stderr)
	if code := runCLI(command, []string{"help", "sample", "read", "--format=agent"}); code != ExitOK {
		t.Fatalf("Run(selected agent help) code = %d, stderr = %q", code, stderr.String())
	}
	var document agentDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Commands) != 1 || document.Commands[0].Path != "sample read" ||
		document.Commands[0].Effect != "read" || document.Commands[0].Role != "act" {
		t.Fatalf("commands = %+v", document.Commands)
	}
	if len(document.Workflows) != 1 || document.Workflows[0].Producer.Path != "sample list" ||
		document.Workflows[0].Consumer.Path != "sample read" {
		t.Fatalf("selected command workflows = %+v", document.Workflows)
	}
}

func TestAgentHelpPublishesDiscoverToActReferenceFlow(t *testing.T) {
	discoverDocument := runSampleExactAgentHelp(t, "sample list")
	actDocument := runSampleExactAgentHelp(t, "sample read")
	discover := discoverDocument.Commands[0]
	if discover.Role != "discover" || discover.Effect != "read" ||
		!reflect.DeepEqual(discover.ProducesRefs, []ProducedRef{{Kind: "sample", Field: "id"}}) ||
		len(discover.ConsumesRefs) != 0 || len(discover.NextActions) != 1 {
		t.Fatalf("sample list agent contract = %+v", discover)
	}
	act := actDocument.Commands[0]
	if act.Role != "act" || act.Effect != "read" ||
		!reflect.DeepEqual(act.ConsumesRefs, []ConsumedRef{{Kind: "sample", Argument: "--id"}}) ||
		len(act.ProducesRefs) != 0 {
		t.Fatalf("sample read agent contract = %+v", act)
	}
	action := discover.NextActions[0]
	if action.Path != "sample read" || action.ReferenceKind != "sample" ||
		action.FromField != "id" || action.ToInput != "--id" {
		t.Fatalf("derived next action = %+v", action)
	}
}

func TestAgentRoundTripContractCoversDiscoveryActionAndRecovery(t *testing.T) {
	discoverDocument := runSampleExactAgentHelp(t, "sample list")
	actDocument := runSampleExactAgentHelp(t, "sample read")
	discover := discoverDocument.Commands[0]
	act := actDocument.Commands[0]
	if discover.Contract.Output.Completeness != OutputCompletenessComplete ||
		len(discover.ProducesRefs) != 1 || discover.ProducesRefs[0] != (ProducedRef{Kind: "sample", Field: "id"}) {
		t.Fatalf("discovery contract = %+v", discover)
	}
	if len(act.Contract.Inputs) < 1 || act.Contract.Inputs[0].Name != "--id" ||
		act.Contract.Inputs[0].Source != InputSourceFlag || act.Contract.Inputs[0].ReferenceKind != "sample" ||
		act.Contract.Inputs[0].Description == "" || act.Contract.Inputs[0].AllowedValues == nil {
		t.Fatalf("action input contract = %+v", act.Contract.Inputs)
	}
	if len(actDocument.Workflows) != 1 || actDocument.Workflows[0].Producer.Path != discover.Path ||
		actDocument.Workflows[0].Consumer.Path != act.Path || actDocument.Workflows[0].Consumer.Input != "--id" {
		t.Fatalf("round-trip workflow = %+v", actDocument.Workflows)
	}
	foundRecovery := false
	for _, declared := range act.Contract.Errors {
		if declared.Code == "sample_not_found" && declared.Kind == fault.KindNotFound &&
			len(declared.NextActions) == 1 && declared.NextActions[0].Command == discover.Path {
			foundRecovery = true
		}
	}
	if !foundRecovery {
		t.Fatalf("action errors lack discover recovery: %+v", act.Contract.Errors)
	}
}

func TestHelpRejectsUnknownSelectorsAndFormats(t *testing.T) {
	for _, args := range [][]string{
		{"help", "missing"},
		{"help", "--format", "yaml"},
		{"help", "--unknown"},
	} {
		var stdout, stderr bytes.Buffer
		command := New(strings.NewReader(""), &stdout, &stderr)
		if code := runCLI(command, args); code != ExitUsage {
			t.Errorf("Run(%v) code = %d, want %d", args, code, ExitUsage)
		}
		if stdout.Len() != 0 || !strings.Contains(stderr.String(), "error:") {
			t.Errorf("Run(%v) stdout = %q, stderr = %q", args, stdout.String(), stderr.String())
		}
	}
}

func TestUnknownHelpSelectorRecoveryNamesPathsAndNamespaces(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help", "does-not-exist"}); code != ExitUsage {
		t.Fatalf("Run(help does-not-exist) code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "next_action: cwk help — text または agent 形式と、ルートヘルプにあるコマンドパスまたは名前空間を指定してください。") {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}
