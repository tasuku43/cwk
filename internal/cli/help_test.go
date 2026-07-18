package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tasuku43/agentic-cli-foundry/internal/domain/fault"
)

func TestRootHelpIsDerivedFromCatalog(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help"}); code != ExitOK {
		t.Fatalf("Run(help) code = %d, stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	for _, spec := range command.catalog.Commands() {
		if !strings.Contains(output, spec.Path) || !strings.Contains(output, spec.Summary) {
			t.Errorf("root help does not contain catalog entry %+v\n%s", spec, output)
		}
	}
}

func TestCommandHelpUsesCatalogMetadataAndDerivedReferences(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"sample", "read", "--help"}); code != ExitOK {
		t.Fatalf("Run(sample read --help) code = %d, stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"Usage:\n  agentic-cli-foundry sample read --id <sample-id> [--format tsv|json]",
		"Read exactly one offline sample by opaque ID.",
		"Effect: read",
		"Role: act",
		"Consumes reference: sample from input --id",
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
	if document.ScopeRequest.InvocationTemplate != "agentic-cli-foundry help <command-or-namespace> --format agent" ||
		!reflect.DeepEqual(document.ScopeRequest.SelectorFields, []string{"commands[].path", "commands[].namespace"}) ||
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
				len(document.ErrorContract.GlobalErrors) != 6 || document.ErrorContract.JSONSchemaVersion != 1 {
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

func TestAgentHelpRootAndScopedShapeSnapshots(t *testing.T) {
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

	scoped := runAgentHelpForTest(t, []string{"help", "sample", "--format=agent"})
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
	perCommandGrowth := (len(many) - len(one)) / 100
	if perCommandGrowth > 320 {
		t.Fatalf("root index grew by %d bytes per command, want <= 320", perCommandGrowth)
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
		if bytes.Contains(many, []byte(`"`+forbidden+`"`)) {
			t.Errorf("root index leaked detailed field %q", forbidden)
		}
	}

	oversized := cloneCommandSpec(base)
	oversized.Summary = strings.Repeat("s", maxAgentIndexEntryBytes)
	if err := NewCatalog(oversized).Validate(); err == nil || !strings.Contains(err.Error(), "agent index entry") {
		t.Fatalf("oversized root index entry error = %v", err)
	}
}

func TestCatalogSelectReturnsDeepCopiesForScopedProjection(t *testing.T) {
	catalog := DefaultCatalog()
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
	if code := runCLI(command, args); code != ExitOK {
		t.Fatalf("Run(%v) code = %d, stderr = %q", args, code, stderr.String())
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatalf("agent help is not JSON: %v\n%s", err, stdout.String())
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

func TestAgentHelpCanSelectNamespaceWithoutLoadingWholeCatalog(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help", "sample", "--format=agent"}); code != ExitOK {
		t.Fatalf("Run(namespace agent help) code = %d, stderr = %q", code, stderr.String())
	}
	var document agentDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Commands) != 2 || document.Commands[0].Path != "sample list" || document.Commands[1].Path != "sample read" {
		t.Fatalf("namespace commands = %+v", document.Commands)
	}
	if len(document.Workflows) != 1 {
		t.Fatalf("namespace workflows = %+v", document.Workflows)
	}
	for _, entry := range document.Commands {
		if !strings.HasPrefix(entry.Path, "sample ") {
			t.Fatalf("unscoped command leaked into namespace help: %+v", entry)
		}
	}
}

func TestTextHelpCanSelectNamespace(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help", "sample"}); code != ExitOK {
		t.Fatalf("Run(namespace help) code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "sample list") || !strings.Contains(stdout.String(), "sample read") || strings.Contains(stdout.String(), "doctor") {
		t.Fatalf("namespace text = %q", stdout.String())
	}
}

func TestAgentHelpPreservesTopLevelCompatibilityFields(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
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
	command := New(strings.NewReader(""), &stdout, &stderr)
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
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help", "sample", "--format", "agent"}); code != ExitOK {
		t.Fatalf("Run(agent help) code = %d, stderr = %q", code, stderr.String())
	}
	var document agentDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	commands := make(map[string]agentCommand, len(document.Commands))
	for _, entry := range document.Commands {
		commands[entry.Path] = entry
	}
	discover := commands["sample list"]
	if discover.Role != "discover" || discover.Effect != "read" ||
		!reflect.DeepEqual(discover.ProducesRefs, []ProducedRef{{Kind: "sample", Field: "id"}}) ||
		len(discover.ConsumesRefs) != 0 || len(discover.NextActions) != 1 {
		t.Fatalf("sample list agent contract = %+v", discover)
	}
	act := commands["sample read"]
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
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"help", "sample", "--format=agent"}); code != ExitOK {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	var document agentDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	commands := make(map[string]agentCommand, len(document.Commands))
	for _, entry := range document.Commands {
		commands[entry.Path] = entry
	}
	discover := commands["sample list"]
	act := commands["sample read"]
	if discover.Contract.Output.Completeness != OutputCompletenessComplete ||
		len(discover.ProducesRefs) != 1 || discover.ProducesRefs[0] != (ProducedRef{Kind: "sample", Field: "id"}) {
		t.Fatalf("discovery contract = %+v", discover)
	}
	if len(act.Contract.Inputs) < 1 || act.Contract.Inputs[0].Name != "--id" ||
		act.Contract.Inputs[0].Source != InputSourceFlag || act.Contract.Inputs[0].ReferenceKind != "sample" ||
		act.Contract.Inputs[0].Description == "" || act.Contract.Inputs[0].AllowedValues == nil {
		t.Fatalf("action input contract = %+v", act.Contract.Inputs)
	}
	if len(document.Workflows) != 1 || document.Workflows[0].Producer.Path != discover.Path ||
		document.Workflows[0].Consumer.Path != act.Path || document.Workflows[0].Consumer.Input != "--id" {
		t.Fatalf("round-trip workflow = %+v", document.Workflows)
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
