package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/tasuku43/agentic-cli-foundry/internal/domain/authn"
	"github.com/tasuku43/agentic-cli-foundry/internal/domain/fault"
	"github.com/tasuku43/agentic-cli-foundry/internal/domain/operation"
)

func noOpHandler(context.Context, *CLI, CommandSpec, operation.Intent, []string) int { return ExitOK }

func utilitySpec(path string) CommandSpec {
	return CommandSpec{
		Path:    path,
		Summary: "Complete a test outcome",
		Effect:  operation.EffectRead,
		Role:    RoleUtility,
		Agent: AgentContract{
			CapabilityID: "test.complete",
			Outcome:      "Complete a bounded test outcome",
			Inputs:       []CommandInput{},
			Output: CommandOutput{
				Formats:       []OutputFormat{OutputFormatText},
				DefaultFormat: OutputFormatText,
				Fields:        []OutputField{{Name: "result", Type: OutputFieldTypeString, Description: "Stable test result."}},
				Completeness:  OutputCompletenessComplete,
			},
			Prerequisites: []string{},
			Errors: []CommandError{
				{
					Code:        "test_failed",
					Kind:        fault.KindInternal,
					Retryable:   false,
					NextActions: []fault.NextAction{{Command: path, Reason: "Inspect the test fixture and correct it."}},
				},
				declaredCommandError(fault.KindInternal, "output_write_failed", true, path, "Retry with a writable output stream."),
				declaredCommandError(fault.KindCanceled, "operation_canceled", true, path, "Retry when the caller is ready."),
			},
		},
		handler: noOpHandler,
	}
}

func mutationRuntimeErrors(path string) []CommandError {
	namespace := strings.Fields(path)[0]
	return []CommandError{
		declaredCommandError(fault.KindContract, "invalid_mutation_contract", false, path, "Repair the mutation declaration."),
		declaredCommandError(fault.KindContract, "missing_mutation_action", false, path, "Configure the mutation action."),
		declaredCommandError(fault.KindRejected, "missing_mutation_policy", false, path, "Configure the project mutation policy."),
		declaredCommandError(fault.KindRejected, "mutation_rejected", false, path, "Review the project mutation policy."),
		declaredCommandError(fault.KindContract, "unclassified_mutation_outcome", false, namespace+" list", "Reconcile the target before deciding whether another mutation is safe."),
	}
}

func authenticationGateRuntimeErrors(path string) []CommandError {
	return []CommandError{
		declaredCommandError(fault.KindContract, "missing_authentication_context", false, path, "Repair the authenticated use case wiring."),
		declaredCommandError(fault.KindContract, "missing_authenticated_action", false, path, "Configure the authenticated action."),
		declaredCommandError(fault.KindContract, "invalid_authentication_requirement", false, path, "Repair the authentication requirement."),
		declaredCommandError(fault.KindAuthentication, "missing_authenticator", false, path, "Configure a supported authentication method."),
		declaredCommandError(fault.KindContract, "missing_authentication_clock", false, path, "Configure the authentication clock."),
		declaredCommandError(fault.KindAuthentication, "invalid_authentication_session", false, path, "Repair the authentication adapter contract."),
		declaredCommandError(fault.KindContract, "authentication_evaluation_failed", false, path, "Repair the authentication evaluation contract."),
		declaredCommandError(fault.KindPermission, "insufficient_authentication_capability", false, path, "Obtain the required capability."),
		declaredCommandError(fault.KindAuthentication, "authentication_expired", false, path, "Reacquire authentication according to project policy."),
		declaredCommandError(fault.KindAuthentication, "authentication_context_mismatch", false, path, "Select the required account and authentication context."),
		declaredCommandError(fault.KindAuthentication, "authentication_failed", false, path, "Establish authentication with a supported method."),
		declaredCommandError(fault.KindCanceled, "authentication_canceled", false, path, "Start a new attempt when the caller is ready."),
		declaredCommandError(fault.KindInternal, "unclassified_authenticated_action_error", false, path, "Repair the adapter fault classification."),
	}
}

func discoverSpec(path, kind string) CommandSpec {
	spec := utilitySpec(path)
	spec.Summary = "Discover test items"
	spec.Role = RoleDiscover
	spec.Agent.Outcome = "Discover stable test item references"
	spec.Agent.Output.Formats = []OutputFormat{OutputFormatTSV, OutputFormatJSON}
	spec.Agent.Output.DefaultFormat = OutputFormatTSV
	spec.Agent.Output.Fields = []OutputField{
		{Name: "id", Type: OutputFieldTypeString, Description: "Opaque test item ID.", ReferenceKind: kind},
		{Name: "name", Type: OutputFieldTypeString, Description: "Test item name."},
	}
	spec.Agent.Output.JSONEnvelope = "items"
	spec.Agent.Output.JSONSchemaVersion = 1
	return spec
}

func actSpec(path, kind string, inputs ...string) CommandSpec {
	spec := utilitySpec(path)
	spec.Summary = "Read test items"
	spec.Role = RoleAct
	spec.Agent.Outcome = "Read the selected test items"
	spec.Agent.Inputs = make([]CommandInput, 0, len(inputs))
	parts := make([]string, 0, len(inputs)*2)
	for _, input := range inputs {
		spec.Agent.Inputs = append(spec.Agent.Inputs, CommandInput{
			Name: input, Source: InputSourceFlag, Required: true,
			Description: "Opaque test item ID.", AllowedValues: []string{}, ReferenceKind: kind,
		})
		parts = append(parts, input, "<"+kind+"-id>")
	}
	spec.Args = strings.Join(parts, " ")
	return spec
}

func TestDefaultCatalogIsValidAndUnique(t *testing.T) {
	catalog := DefaultCatalog()
	if err := catalog.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	seen := map[string]bool{}
	for _, command := range catalog.Commands() {
		if seen[command.Path] {
			t.Fatalf("duplicate command path %q", command.Path)
		}
		seen[command.Path] = true
		if err := command.Effect.Validate(); err != nil {
			t.Errorf("%s effect: %v", command.Path, err)
		}
		if err := command.Role.validate(); err != nil {
			t.Errorf("%s role: %v", command.Path, err)
		}
	}
	for _, required := range []string{"doctor", "help", "version"} {
		if !seen[required] {
			t.Errorf("catalog is missing %q", required)
		}
	}
}

func TestCatalogRejectsIncompleteDeclarations(t *testing.T) {
	valid := utilitySpec("valid")
	missingEffect := utilitySpec("missing-effect")
	missingEffect.Effect = operation.EffectUnknown
	missingRole := utilitySpec("missing-role")
	missingRole.Role = RoleUnknown
	badPath := utilitySpec("Bad Path")
	missingSummary := utilitySpec("missing-summary")
	missingSummary.Summary = ""
	missingHandler := utilitySpec("missing-handler")
	missingHandler.handler = nil

	tests := []Catalog{
		{},
		NewCatalog(missingEffect),
		NewCatalog(missingRole),
		NewCatalog(badPath),
		NewCatalog(missingSummary),
		NewCatalog(missingHandler),
		NewCatalog(valid, valid),
	}
	for index, catalog := range tests {
		if err := catalog.Validate(); err == nil {
			t.Errorf("invalid catalog %d passed validation", index)
		}
	}
}

func TestCatalogRejectsCommandPathNamespaceCollision(t *testing.T) {
	catalog := NewCatalog(utilitySpec("foo"), utilitySpec("foo bar"))
	if err := catalog.Validate(); err == nil || !strings.Contains(err.Error(), "command/namespace boundary") {
		t.Fatalf("Validate() error = %v, want command/namespace collision", err)
	}
}

func TestCatalogRejectsStructuralLineSeparators(t *testing.T) {
	for _, separator := range []rune{'\u2028', '\u2029'} {
		t.Run(strings.ToUpper(strconv.FormatInt(int64(separator), 16)), func(t *testing.T) {
			if err := validateContractText("test value", "before"+string(separator)+"after"); err == nil {
				t.Fatal("structural separator passed contract text validation")
			}

			spec := utilitySpec("test")
			spec.Args = "[label" + string(separator) + "]"
			if err := NewCatalog(spec).Validate(); err == nil || !strings.Contains(err.Error(), "invalid argument syntax") {
				t.Fatalf("Validate() error = %v, want invalid argument syntax", err)
			}
		})
	}
}

func TestArgumentSyntaxRequiredAndAllowedValuesMatchAgentInputs(t *testing.T) {
	valid := utilitySpec("configure")
	valid.Args = "[--mode fast|safe] <target> [label]"
	valid.Agent.Inputs = []CommandInput{
		{Name: "--mode", Source: InputSourceFlag, Required: false, Description: "Select the operating mode.", AllowedValues: []string{"fast", "safe"}},
		{Name: "target", Source: InputSourceArgument, Required: true, Description: "Target value.", AllowedValues: []string{}},
		{Name: "label", Source: InputSourceArgument, Required: false, Description: "Optional display label.", AllowedValues: []string{}},
		{Name: "CLI_PROFILE", Source: InputSourceEnvironment, Required: false, Description: "Optional environment profile.", AllowedValues: []string{}},
	}
	if err := NewCatalog(valid).Validate(); err != nil {
		t.Fatalf("valid small argument grammar: %v", err)
	}

	tests := map[string]func(*CommandSpec){
		"optional flag declared required":       func(spec *CommandSpec) { spec.Agent.Inputs[0].Required = true },
		"required positional declared optional": func(spec *CommandSpec) { spec.Agent.Inputs[1].Required = false },
		"optional positional declared required": func(spec *CommandSpec) { spec.Agent.Inputs[2].Required = true },
		"enum order differs":                    func(spec *CommandSpec) { spec.Agent.Inputs[0].AllowedValues = []string{"safe", "fast"} },
		"enum set differs":                      func(spec *CommandSpec) { spec.Agent.Inputs[0].AllowedValues = []string{"fast"} },
		"free form claims enumeration":          func(spec *CommandSpec) { spec.Agent.Inputs[1].AllowedValues = []string{"fixed"} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := cloneCommandSpec(valid)
			mutate(&spec)
			if err := NewCatalog(spec).Validate(); err == nil {
				t.Fatal("argument syntax mismatch passed validation")
			}
		})
	}
}

func TestCatalogRequiresCommonRuntimeFailures(t *testing.T) {
	removeError := func(spec *CommandSpec, code string) {
		filtered := make([]CommandError, 0, len(spec.Agent.Errors))
		for _, declared := range spec.Agent.Errors {
			if declared.Code != code {
				filtered = append(filtered, declared)
			}
		}
		spec.Agent.Errors = filtered
	}
	for _, code := range []string{"operation_canceled", "output_write_failed"} {
		t.Run("missing_"+code, func(t *testing.T) {
			spec := utilitySpec("test")
			removeError(&spec, code)
			if err := NewCatalog(spec).Validate(); err == nil {
				t.Fatalf("catalog without %q passed validation", code)
			}
		})
	}

	wrong := utilitySpec("test")
	for index := range wrong.Agent.Errors {
		if wrong.Agent.Errors[index].Code == "operation_canceled" {
			wrong.Agent.Errors[index].Retryable = false
		}
	}
	if err := NewCatalog(wrong).Validate(); err == nil {
		t.Fatal("catalog with inconsistent common runtime failure passed")
	}

	noOutput := utilitySpec("test")
	noOutput.Agent.Output.Formats = []OutputFormat{OutputFormatNone}
	noOutput.Agent.Output.DefaultFormat = OutputFormatNone
	noOutput.Agent.Output.Fields = []OutputField{}
	removeError(&noOutput, "output_write_failed")
	if err := NewCatalog(noOutput).Validate(); err != nil {
		t.Fatalf("no-output command unnecessarily requires output_write_failed: %v", err)
	}
}

func TestAgentContractValidationFailsClosed(t *testing.T) {
	tests := map[string]func(*CommandSpec){
		"missing capability": func(spec *CommandSpec) { spec.Agent.CapabilityID = "" },
		"missing outcome":    func(spec *CommandSpec) { spec.Agent.Outcome = "" },
		"unknown inputs":     func(spec *CommandSpec) { spec.Agent.Inputs = nil },
		"unknown input source": func(spec *CommandSpec) {
			spec.Args = "--id <item-id>"
			spec.Agent.Inputs = []CommandInput{{Name: "--id", Required: true, Description: "Item ID.", AllowedValues: []string{}}}
		},
		"undocumented argument": func(spec *CommandSpec) {
			spec.Args = "--id <item-id>"
		},
		"input absent from syntax": func(spec *CommandSpec) {
			spec.Agent.Inputs = []CommandInput{{Name: "--id", Source: InputSourceFlag, Description: "Item ID.", AllowedValues: []string{}}}
		},
		"missing input description": func(spec *CommandSpec) {
			spec.Args = "--id <item-id>"
			spec.Agent.Inputs = []CommandInput{{Name: "--id", Source: InputSourceFlag, AllowedValues: []string{}}}
		},
		"unknown allowed values": func(spec *CommandSpec) {
			spec.Args = "--id <item-id>"
			spec.Agent.Inputs = []CommandInput{{Name: "--id", Source: InputSourceFlag, Description: "Item ID."}}
		},
		"unknown formats":        func(spec *CommandSpec) { spec.Agent.Output.Formats = nil },
		"unknown default format": func(spec *CommandSpec) { spec.Agent.Output.DefaultFormat = OutputFormatUnknown },
		"unknown fields":         func(spec *CommandSpec) { spec.Agent.Output.Fields = nil },
		"missing field description": func(spec *CommandSpec) {
			spec.Agent.Output.Fields[0].Description = ""
		},
		"unknown completeness": func(spec *CommandSpec) {
			spec.Agent.Output.Completeness = OutputCompletenessUnknown
		},
		"unknown prerequisites": func(spec *CommandSpec) { spec.Agent.Prerequisites = nil },
		"unknown errors":        func(spec *CommandSpec) { spec.Agent.Errors = nil },
		"missing next action":   func(spec *CommandSpec) { spec.Agent.Errors[0].NextActions = nil },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := utilitySpec("test")
			mutate(&spec)
			if err := NewCatalog(spec).Validate(); err == nil {
				t.Fatal("incomplete agent contract passed validation")
			}
		})
	}
}

func TestCatalogMatchUsesLongestDeclarativePath(t *testing.T) {
	catalog := NewCatalog(utilitySpec("items"), utilitySpec("items list"))
	command, rest, found := catalog.Match([]string{"items", "list", "--limit", "2"})
	if !found {
		t.Fatal("Match() did not find a command")
	}
	if command.Path != "items list" {
		t.Fatalf("Match() path = %q, want items list", command.Path)
	}
	if got := strings.Join(rest, " "); got != "--limit 2" {
		t.Fatalf("Match() rest = %q", got)
	}
}

func TestCatalogEnforcesRoleAndReferenceFlowContracts(t *testing.T) {
	discover := discoverSpec("items list", "item")
	act := actSpec("items read", "item", "--id")
	if err := NewCatalog(discover, act).Validate(); err != nil {
		t.Fatalf("valid reference flow: %v", err)
	}

	utilityWithRef := discoverSpec("utility", "item")
	utilityWithRef.Role = RoleUtility
	mutatingDiscovery := discoverSpec("items list", "item")
	mutatingDiscovery.Effect = operation.EffectWrite
	emptyDiscovery := utilitySpec("items list")
	emptyDiscovery.Role = RoleDiscover
	emptyAct := utilitySpec("items read")
	emptyAct.Role = RoleAct
	optionalAct := actSpec("items inspect", "item", "--id")
	optionalAct.Args = "[--id <item-id>]"
	optionalAct.Agent.Inputs[0].Required = false
	invalidProducer := discoverSpec("items list", "Item")
	invalidConsumer := actSpec("items read", "Item", "--id")

	invalid := []Catalog{
		NewCatalog(utilityWithRef, act),
		NewCatalog(mutatingDiscovery, act),
		NewCatalog(emptyDiscovery),
		NewCatalog(emptyAct),
		NewCatalog(discover, optionalAct),
		NewCatalog(discover),
		NewCatalog(act),
		NewCatalog(invalidProducer, act),
		NewCatalog(discover, invalidConsumer),
	}
	for index, catalog := range invalid {
		if err := catalog.Validate(); err == nil {
			t.Errorf("invalid role/reference catalog %d passed validation", index)
		}
	}
}

func TestReferenceGraphRejectsClosedCyclesAndAcceptsReachableChains(t *testing.T) {
	selfCycle := actSpec("items rotate", "item", "--id")
	selfCycle.Agent.Output.Fields[0] = OutputField{
		Name: "id", Type: OutputFieldTypeString, Description: "Rotated item ID.", ReferenceKind: "item",
	}
	if err := NewCatalog(selfCycle).Validate(); err == nil || !strings.Contains(err.Error(), "closed required-reference cycle") {
		t.Fatalf("self-contained reference cycle error = %v", err)
	}

	alpha := actSpec("alpha derive", "beta", "--beta-id")
	alpha.Agent.Output.Fields[0] = OutputField{
		Name: "alpha_id", Type: OutputFieldTypeString, Description: "Derived alpha ID.", ReferenceKind: "alpha",
	}
	beta := actSpec("beta derive", "alpha", "--alpha-id")
	beta.Agent.Output.Fields[0] = OutputField{
		Name: "beta_id", Type: OutputFieldTypeString, Description: "Derived beta ID.", ReferenceKind: "beta",
	}
	if err := NewCatalog(alpha, beta).Validate(); err == nil || !strings.Contains(err.Error(), "closed required-reference cycle") {
		t.Fatalf("multi-kind reference cycle error = %v", err)
	}

	workspaces := discoverSpec("workspaces list", "workspace")
	items := discoverSpec("items list", "item")
	items.Args = "--workspace-id <workspace-id>"
	items.Agent.Inputs = []CommandInput{{
		Name: "--workspace-id", Source: InputSourceFlag, Required: true,
		Description: "Opaque workspace ID.", AllowedValues: []string{}, ReferenceKind: "workspace",
	}}
	read := actSpec("items read", "item", "--id")
	if err := NewCatalog(workspaces, items, read).Validate(); err != nil {
		t.Fatalf("reachable reference chain failed validation: %v", err)
	}
}

func TestReferenceGraphAllowsMultipleInputsOfTheSameKind(t *testing.T) {
	discover := discoverSpec("items list", "item")
	act := actSpec("items compare", "item", "--left-id", "--right-id")
	catalog := NewCatalog(discover, act)
	if err := catalog.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	consumed := act.ConsumedRefs()
	if len(consumed) != 2 || consumed[0].Argument != "--left-id" || consumed[1].Argument != "--right-id" {
		t.Fatalf("ConsumedRefs() = %+v", consumed)
	}
	workflows := catalog.referenceWorkflows()
	if len(workflows) != 2 {
		t.Fatalf("reference workflows = %+v, want one per same-kind input", workflows)
	}
}

func TestInvalidCatalogFailsBeforeDispatch(t *testing.T) {
	called := false
	bad := utilitySpec("unsafe")
	bad.Effect = operation.EffectUnknown
	bad.handler = func(context.Context, *CLI, CommandSpec, operation.Intent, []string) int {
		called = true
		return ExitOK
	}
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader(""), &stdout, &stderr, NewCatalog(bad), nil)
	if code := runCLI(command, []string{"unsafe"}); code != ExitContract {
		t.Fatalf("Run() code = %d, want %d", code, ExitContract)
	}
	if called {
		t.Fatal("handler ran for an invalid catalog")
	}
	if !strings.Contains(stderr.String(), "code: invalid_catalog") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCatalogCommandsReturnsDeepCopy(t *testing.T) {
	catalog := DefaultCatalog()
	commands := catalog.Commands()
	commands[0].Path = "changed"
	commands[0].Agent.Outcome = "changed"
	commands[0].Agent.Output.Formats[0] = OutputFormatNone
	commands[0].Agent.Output.Fields[0].Name = "changed"
	commands[0].Agent.Inputs[0].AllowedValues[0] = "changed"
	commands[0].Agent.Prerequisites = append(commands[0].Agent.Prerequisites, "changed")
	commands[0].Agent.Errors[0].Code = "changed"
	commands[0].Agent.Errors[0].NextActions[0].Command = "changed"

	doctor, found := catalog.Lookup("doctor")
	if !found {
		t.Fatal("mutating Commands() changed the catalog")
	}
	want, _ := DefaultCatalog().Lookup("doctor")
	if !reflect.DeepEqual(doctor.Agent, want.Agent) {
		t.Fatalf("nested agent contract was mutated: %+v", doctor.Agent)
	}
}

func TestMutationContractFailsClosedAndDeepCopies(t *testing.T) {
	spec := utilitySpec("items update")
	spec.Effect = operation.EffectWrite
	spec.Role = RoleAct
	spec.Agent.Errors = append(spec.Agent.Errors, mutationRuntimeErrors(spec.Path)...)
	spec.Args = "--id <item-id>"
	spec.Agent.Inputs = []CommandInput{{
		Name: "--id", Source: InputSourceFlag, Required: true,
		Description: "Target item ID.", AllowedValues: []string{}, ReferenceKind: "item",
	}}
	spec.Agent.Mutation = &MutationContract{
		TargetKind: "item", TargetInputs: []string{"--id"}, TargetIDInput: "--id",
		Impact: operation.Impact{
			Cardinality: operation.CardinalityOne, Notification: operation.DeclarationNo,
			AccessChange: operation.DeclarationNo, Destructive: operation.DeclarationNo,
		},
	}
	if err := validateAgentContract(spec); err != nil {
		t.Fatalf("valid mutation contract: %v", err)
	}
	if err := NewCatalog(discoverSpec("items list", "item"), spec).Validate(); err != nil {
		t.Fatalf("valid act mutation catalog: %v", err)
	}
	unsafeRecovery := cloneCommandSpec(spec)
	for index := range unsafeRecovery.Agent.Errors {
		if unsafeRecovery.Agent.Errors[index].Code == "unclassified_mutation_outcome" {
			unsafeRecovery.Agent.Errors[index].NextActions[0].Command = unsafeRecovery.Path
		}
	}
	if err := NewCatalog(discoverSpec("items list", "item"), unsafeRecovery).Validate(); err == nil ||
		!strings.Contains(err.Error(), "read-only reconciliation") {
		t.Fatalf("unsafe unknown-outcome recovery error = %v", err)
	}

	missing := cloneCommandSpec(spec)
	missing.Agent.Mutation = nil
	if err := validateAgentContract(missing); err == nil {
		t.Fatal("mutation without declaration passed")
	}
	wrongInput := cloneCommandSpec(spec)
	wrongInput.Agent.Mutation.TargetInputs[0] = "--missing"
	if err := validateAgentContract(wrongInput); err == nil {
		t.Fatal("mutation with unknown target input passed")
	}
	clone := cloneCommandSpec(spec)
	clone.Agent.Mutation.TargetInputs[0] = "changed"
	if spec.Agent.Mutation.TargetInputs[0] != "--id" {
		t.Fatal("mutation target inputs share storage")
	}
	missingTargetBinding := cloneCommandSpec(spec)
	missingTargetBinding.Agent.Mutation.TargetIDInput = ""
	if err := validateAgentContract(missingTargetBinding); err == nil {
		t.Fatal("write mutation without target ID binding passed")
	}
	mismatchedTargetKind := cloneCommandSpec(spec)
	mismatchedTargetKind.Agent.Mutation.TargetKind = "other"
	if err := validateAgentContract(mismatchedTargetKind); err == nil {
		t.Fatal("write mutation with mismatched target reference kind passed")
	}
	optionalTarget := cloneCommandSpec(spec)
	optionalTarget.Args = "[--id <item-id>]"
	optionalTarget.Agent.Inputs[0].Required = false
	if err := validateAgentContract(optionalTarget); err == nil || !strings.Contains(err.Error(), "must be required") {
		t.Fatalf("optional mutation target error = %v", err)
	}
	configuredTarget := cloneCommandSpec(spec)
	configuredTarget.Args = ""
	configuredTarget.Agent.Inputs[0].Source = InputSourceConfiguration
	if err := validateAgentContract(configuredTarget); err == nil || !strings.Contains(err.Error(), "command argument or flag") {
		t.Fatalf("non-CLI mutation target error = %v", err)
	}
	withParent := cloneCommandSpec(spec)
	withParent.Args += " --collection-id <collection-id>"
	withParent.Agent.Inputs = append(withParent.Agent.Inputs, CommandInput{
		Name: "--collection-id", Source: InputSourceFlag, Required: true,
		Description: "Parent collection ID.", AllowedValues: []string{}, ReferenceKind: "collection",
	})
	withParent.Agent.Mutation.ParentInput = "--collection-id"
	withParent.Agent.Mutation.TargetInputs = append(withParent.Agent.Mutation.TargetInputs, "--collection-id")
	if err := validateAgentContract(withParent); err != nil {
		t.Fatalf("write mutation with parent binding: %v", err)
	}
	ambiguousTargets := cloneCommandSpec(withParent)
	ambiguousTargets.Args += " --scope-id <scope-id>"
	ambiguousTargets.Agent.Inputs = append(ambiguousTargets.Agent.Inputs, CommandInput{
		Name: "--scope-id", Source: InputSourceFlag, Required: true,
		Description: "Unbound scope ID.", AllowedValues: []string{}, ReferenceKind: "scope",
	})
	ambiguousTargets.Agent.Mutation.TargetInputs = append(ambiguousTargets.Agent.Mutation.TargetInputs, "--scope-id")
	if err := validateAgentContract(ambiguousTargets); err == nil {
		t.Fatal("write mutation with an unbound target input passed")
	}
	encoded, err := json.Marshal(spec.Agent)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"target_id_input":"--id"`, `"cardinality":"one"`, `"notification":"no"`, `"access_change":"no"`, `"destructive":"no"`} {
		if !bytes.Contains(encoded, []byte(want)) {
			t.Errorf("mutation JSON lacks %s: %s", want, encoded)
		}
	}
}

func TestMutationContractRequiresInvokerFailureSurface(t *testing.T) {
	spec := utilitySpec("items update")
	spec.Effect = operation.EffectWrite
	spec.Role = RoleAct
	spec.Agent.Errors = append(spec.Agent.Errors, mutationRuntimeErrors(spec.Path)...)
	spec.Args = "--id <item-id>"
	spec.Agent.Inputs = []CommandInput{{
		Name: "--id", Source: InputSourceFlag, Required: true,
		Description: "Target item ID.", AllowedValues: []string{}, ReferenceKind: "item",
	}}
	spec.Agent.Mutation = &MutationContract{
		TargetKind: "item", TargetInputs: []string{"--id"}, TargetIDInput: "--id",
		Impact: operation.Impact{
			Cardinality: operation.CardinalityOne, Notification: operation.DeclarationNo,
			AccessChange: operation.DeclarationNo, Destructive: operation.DeclarationNo,
		},
	}
	if err := validateAgentContract(spec); err != nil {
		t.Fatalf("valid mutation failure surface: %v", err)
	}
	for _, missing := range []string{"invalid_mutation_contract", "missing_mutation_action", "missing_mutation_policy", "mutation_rejected", "unclassified_mutation_outcome"} {
		t.Run(missing, func(t *testing.T) {
			candidate := cloneCommandSpec(spec)
			filtered := make([]CommandError, 0, len(candidate.Agent.Errors)-1)
			for _, declared := range candidate.Agent.Errors {
				if declared.Code != missing {
					filtered = append(filtered, declared)
				}
			}
			candidate.Agent.Errors = filtered
			if err := validateAgentContract(candidate); err == nil {
				t.Fatalf("mutation without %q passed", missing)
			}
		})
	}
}

func TestCreateMutationBindsOpaqueParentOnly(t *testing.T) {
	spec := utilitySpec("items create")
	spec.Effect = operation.EffectCreate
	spec.Role = RoleAct
	spec.Agent.Errors = append(spec.Agent.Errors, mutationRuntimeErrors(spec.Path)...)
	spec.Args = "--collection-id <collection-id>"
	spec.Agent.Inputs = []CommandInput{{
		Name: "--collection-id", Source: InputSourceFlag, Required: true,
		Description: "Parent collection ID.", AllowedValues: []string{}, ReferenceKind: "collection",
	}}
	spec.Agent.Mutation = &MutationContract{
		TargetKind: "item", TargetInputs: []string{"--collection-id"}, ParentInput: "--collection-id",
		Impact: operation.Impact{
			Cardinality: operation.CardinalityOne, Notification: operation.DeclarationNo,
			AccessChange: operation.DeclarationNo, Destructive: operation.DeclarationNo,
		},
	}
	if err := validateAgentContract(spec); err != nil {
		t.Fatalf("valid create mutation: %v", err)
	}

	missingParent := cloneCommandSpec(spec)
	missingParent.Agent.Mutation.ParentInput = ""
	if err := validateAgentContract(missingParent); err == nil {
		t.Fatal("create mutation without parent binding passed")
	}
	withTargetID := cloneCommandSpec(spec)
	withTargetID.Agent.Mutation.TargetIDInput = "--collection-id"
	if err := validateAgentContract(withTargetID); err == nil {
		t.Fatal("create mutation with an existing target ID passed")
	}
	parentOutsideTargets := cloneCommandSpec(spec)
	parentOutsideTargets.Agent.Mutation.TargetInputs = []string{"--missing"}
	if err := validateAgentContract(parentOutsideTargets); err == nil {
		t.Fatal("create mutation with unbound parent passed")
	}
}

func TestAuthenticationRequirementRequiresGateFailureSurfaceAndDeepCopies(t *testing.T) {
	spec := utilitySpec("items read")
	spec.Agent.Authentication = &authn.Requirement{
		Methods: []authn.Method{authn.MethodOAuth2, authn.MethodPAT}, Authority: "example",
		Audience: "items", RequiredCapabilities: []string{"items.read"},
	}
	if err := validateAgentContract(spec); err == nil {
		t.Fatal("authenticated command without gate errors passed")
	}
	required := authenticationGateRuntimeErrors(spec.Path)
	spec.Agent.Errors = append(spec.Agent.Errors, required...)
	if err := validateAgentContract(spec); err != nil {
		t.Fatalf("valid authenticated contract: %v", err)
	}
	for _, contract := range required {
		contract := contract
		t.Run("missing_"+contract.Code, func(t *testing.T) {
			candidate := cloneCommandSpec(spec)
			filtered := make([]CommandError, 0, len(candidate.Agent.Errors)-1)
			for _, declared := range candidate.Agent.Errors {
				if declared.Code != contract.Code {
					filtered = append(filtered, declared)
				}
			}
			candidate.Agent.Errors = filtered
			if err := validateAgentContract(candidate); err == nil || !strings.Contains(err.Error(), contract.Code) {
				t.Fatalf("authenticated command without %q error = %v", contract.Code, err)
			}
		})
		t.Run("wrong_kind_"+contract.Code, func(t *testing.T) {
			candidate := cloneCommandSpec(spec)
			for index := range candidate.Agent.Errors {
				if candidate.Agent.Errors[index].Code == contract.Code {
					candidate.Agent.Errors[index].Kind = fault.KindInternal
					if contract.Kind == fault.KindInternal {
						candidate.Agent.Errors[index].Kind = fault.KindContract
					}
				}
			}
			if err := validateAgentContract(candidate); err == nil || !strings.Contains(err.Error(), contract.Code) {
				t.Fatalf("authenticated command with wrong %q kind error = %v", contract.Code, err)
			}
		})
		t.Run("wrong_retryability_"+contract.Code, func(t *testing.T) {
			candidate := cloneCommandSpec(spec)
			for index := range candidate.Agent.Errors {
				if candidate.Agent.Errors[index].Code == contract.Code {
					candidate.Agent.Errors[index].Retryable = !contract.Retryable
				}
			}
			if err := validateAgentContract(candidate); err == nil || !strings.Contains(err.Error(), contract.Code) {
				t.Fatalf("authenticated command with wrong %q retryability error = %v", contract.Code, err)
			}
		})
	}
	withProviderFault := cloneCommandSpec(spec)
	withProviderFault.Agent.Errors = append(withProviderFault.Agent.Errors,
		declaredCommandError(fault.KindRateLimited, "identity_rate_limited", true, spec.Path, "Retry after the provider delay."),
	)
	if err := validateAgentContract(withProviderFault); err != nil {
		t.Fatalf("provider-specific authentication fault: %v", err)
	}
	clone := cloneCommandSpec(spec)
	clone.Agent.Authentication.Methods[0] = authn.MethodPAT
	clone.Agent.Authentication.RequiredCapabilities[0] = "changed"
	if spec.Agent.Authentication.Methods[0] != authn.MethodOAuth2 || spec.Agent.Authentication.RequiredCapabilities[0] != "items.read" {
		t.Fatal("authentication requirement shares slice storage")
	}
}

func TestCatalogValidatesExecutableRecoveryCommandGrammar(t *testing.T) {
	help, found := DefaultCatalog().Lookup("help")
	if !found {
		t.Fatal("default catalog lacks help")
	}
	sampleList := utilitySpec("sample list")
	for _, action := range []string{"help", "sample list", "help sample", "help sample list"} {
		t.Run("valid_"+strings.ReplaceAll(action, " ", "_"), func(t *testing.T) {
			spec := utilitySpec("test")
			spec.Agent.Errors[0].NextActions[0].Command = action
			if err := NewCatalog(help, sampleList, spec).Validate(); err != nil {
				t.Fatalf("valid recovery command %q: %v", action, err)
			}
		})
	}

	for _, action := range []string{
		"missing command",
		"sample list --bogus",
		"help nonexistent",
		"help sample list extra",
		"help sample --format agent",
		"sample  list",
	} {
		t.Run("invalid_"+strings.ReplaceAll(action, " ", "_"), func(t *testing.T) {
			spec := utilitySpec("test")
			spec.Agent.Errors[0].NextActions[0].Command = action
			if err := NewCatalog(help, sampleList, spec).Validate(); err == nil {
				t.Fatalf("invalid recovery command %q passed catalog validation", action)
			}
		})
	}
}
