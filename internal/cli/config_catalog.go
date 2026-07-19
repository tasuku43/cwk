package cli

import (
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

const (
	commandSelectionCapability              = "cli.command-selection"
	commandSelectionUncertainMessageGrammar = "Command-selection save outcome is uncertain; expected-source=saved candidate-fingerprint=<sha256:64-lowercase-hex>."
)

func commandSelectionUncertainMessage(fingerprint string) string {
	return "Command-selection save outcome is uncertain; expected-source=saved candidate-fingerprint=" + fingerprint + "."
}

func configCommandSpecs() []CommandSpec {
	return []CommandSpec{{
		Path:    "config",
		Summary: "Select the commands visible to agents",
		Effect:  operation.EffectWrite,
		Role:    RoleAct,
		Agent: AgentContract{
			CapabilityID: commandSelectionCapability,
			Outcome:      "Persist one curated command view through an interactive terminal selector without changing authority",
			Inputs: []CommandInput{
				{Name: "selection", Source: InputSourceStdin, Required: true, Description: "On an interactive terminal, use Up and Down to move, Space to toggle, Enter to save, or q to leave the last saved selection unchanged.", AllowedValues: []string{}},
			},
			Output: CommandOutput{
				Formats:       []OutputFormat{OutputFormatText},
				DefaultFormat: OutputFormatText,
				Fields: []OutputField{
					{Name: "status", Type: OutputFieldTypeString, Description: "Whether the selector saved a replacement or left the prior profile unchanged."},
					{Name: "enabled", Type: OutputFieldTypeInteger, Description: "Number of selectable Chatwork commands enabled after a confirmed save."},
					{Name: "disabled", Type: OutputFieldTypeInteger, Description: "Number of selectable Chatwork commands disabled after a confirmed save."},
					{Name: "changed", Type: OutputFieldTypeInteger, Description: "Number of current catalog choices changed from the loaded selection."},
					{Name: "stale_removed", Type: OutputFieldTypeInteger, Description: "Number of no-longer-known saved paths omitted from the replacement."},
					{Name: "legacy_removed", Type: OutputFieldTypeInteger, Description: "Number of formerly selectable local command paths normalized out of the replacement."},
					{Name: "fingerprint", Type: OutputFieldTypeString, Description: "Deterministic SHA-256 identity of the saved canonical selection; an uncertain outcome is reconciled only when doctor reports both source=saved and the candidate fingerprint."},
				},
				Completeness: OutputCompletenessComplete,
			},
			FixedTarget: &FixedTargetContract{
				Scope:       FixedTargetScopeToolLocal,
				Kind:        "command-selection",
				StableID:    "default",
				Description: "The single user-local set of exact cwk command paths presented to agents.",
			},
			Prerequisites: []string{"Interactive stdin and stdout terminals and a writable user configuration directory. This attention filter is not an authorization or security boundary."},
			Errors: []CommandError{
				declaredCommandError(fault.KindInvalidInput, "invalid_arguments", false, "help config", "Run the selector without command arguments."),
				declaredCommandError(fault.KindInvalidInput, "command_selection_invalid", false, "config", "Explicitly replace the invalid saved command selection."),
				declaredCommandError(fault.KindUnavailable, "command_selection_unsafe", false, "doctor", "Repair the local configuration file or directory, then inspect command-selection diagnostics."),
				declaredCommandError(fault.KindUnavailable, "command_selection_unavailable", true, "doctor", "Restore access to the user configuration directory, then inspect local diagnostics."),
				declaredCommandError(fault.KindUnavailable, "interactive_terminal_required", false, "help config", "Run the selector with interactive stdin and stdout terminals."),
				declaredCommandError(fault.KindInternal, "terminal_setup_failed", true, "config", "Restore a usable terminal and retry the selector."),
				declaredCommandError(fault.KindInternal, "terminal_restore_failed", false, "doctor", "Inspect local terminal and command-selection state before retrying."),
				declaredCommandError(fault.KindInternal, "configuration_input_failed", false, "config", "Retry with a readable interactive terminal."),
				declaredCommandError(fault.KindContract, "invalid_mutation_contract", false, "help config", "Repair the fixed command-selection target and impact declaration."),
				declaredCommandError(fault.KindContract, "missing_mutation_action", false, "help config", "Repair command-selection mutation composition."),
				declaredCommandError(fault.KindRejected, "missing_mutation_policy", false, "help config", "Restore the explicit-Enter mutation policy."),
				declaredCommandError(fault.KindRejected, "mutation_rejected", false, "config", "Press Enter only after reviewing the exact selection."),
				declaredCommandErrorWithMessageGrammar(fault.KindContract, "unclassified_mutation_outcome", false, commandSelectionUncertainMessageGrammar, "doctor", "Require source=saved and the candidate selection fingerprint before another mutation."),
				declaredCommandError(fault.KindInternal, "output_write_failed", true, "doctor", "Inspect the saved selection after restoring a writable output stream."),
				declaredCommandError(fault.KindCanceled, "operation_canceled", true, "config", "Retry when the caller is ready."),
			},
			Mutation: &MutationContract{
				TargetKind:   "command-selection",
				TargetInputs: []string{},
				Impact: operation.Impact{
					Cardinality:  operation.CardinalityOne,
					Notification: operation.DeclarationNo,
					AccessChange: operation.DeclarationNo,
					Destructive:  operation.DeclarationNo,
				},
			},
		},
		handler: runConfig,
	}}
}
