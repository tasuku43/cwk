package cli

import (
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

const commandSelectionCapability = "cli.command-selection"

func configCommandSpecs() []CommandSpec {
	return []CommandSpec{
		{
			Path:    "config show",
			Summary: "Show the active command selection",
			Effect:  operation.EffectRead,
			Role:    RoleUtility,
			Agent: AgentContract{
				CapabilityID: commandSelectionCapability,
				Outcome:      "Inspect the tool-local attention filter without treating it as an authorization boundary",
				Inputs:       []CommandInput{},
				Output: CommandOutput{
					Formats:       []OutputFormat{OutputFormatText},
					DefaultFormat: OutputFormatText,
					Fields: []OutputField{
						{Name: "source", Type: OutputFieldTypeString, Description: "Whether the active selection comes from the all-enabled default or a saved profile."},
						{Name: "security_boundary", Type: OutputFieldTypeBoolean, Description: "Always false; this setting reduces attention surface and grants no authority."},
						{Name: "always_on", Type: OutputFieldTypeArray, Description: "Exact control-plane command paths that cannot be switched off."},
						{Name: "enabled", Type: OutputFieldTypeArray, Description: "Selected configurable exact command paths in catalog order."},
						{Name: "disabled", Type: OutputFieldTypeArray, Description: "Unselected configurable exact command paths in catalog order."},
						{Name: "stale", Type: OutputFieldTypeArray, Description: "Saved canonical paths that no longer exist in this executable."},
					},
					Completeness: OutputCompletenessComplete,
				},
				Prerequisites: []string{},
				Errors: []CommandError{
					declaredCommandError(fault.KindInvalidInput, "invalid_arguments", false, "help config show", "Run the read-only inspection without arguments."),
					declaredCommandError(fault.KindInvalidInput, "command_selection_invalid", false, "config edit", "Repair the saved command selection explicitly."),
					declaredCommandError(fault.KindUnavailable, "command_selection_unavailable", true, "config show", "Restore access to the user configuration directory, then retry."),
					declaredCommandError(fault.KindInternal, "output_write_failed", true, "config show", "Retry with a writable output stream."),
					declaredCommandError(fault.KindCanceled, "operation_canceled", true, "config show", "Retry when the caller is ready."),
				},
			},
			handler: runConfigShow,
		},
		{
			Path:    "config edit",
			Summary: "Select the commands visible to agents",
			Effect:  operation.EffectWrite,
			Role:    RoleAct,
			Agent: AgentContract{
				CapabilityID: commandSelectionCapability,
				Outcome:      "Persist one curated command view through an explicit line-oriented selection without changing authority",
				Inputs: []CommandInput{
					{Name: "selection", Source: InputSourceStdin, Required: true, Description: "Use catalog-local numbers to toggle choices, or one of all, none, save, and cancel; save is the only token that writes.", AllowedValues: []string{}},
				},
				Output: CommandOutput{
					Formats:       []OutputFormat{OutputFormatText},
					DefaultFormat: OutputFormatText,
					Fields: []OutputField{
						{Name: "security_boundary", Type: OutputFieldTypeBoolean, Description: "Always false; selector input is not proof of authority."},
						{Name: "always_on", Type: OutputFieldTypeArray, Description: "Exact control-plane paths displayed as non-selectable."},
						{Name: "choices", Type: OutputFieldTypeArray, Description: "Catalog-order exact paths with document-local numbers, current selection state, and summaries."},
						{Name: "enabled", Type: OutputFieldTypeInteger, Description: "Number of configurable commands saved as enabled."},
						{Name: "disabled", Type: OutputFieldTypeInteger, Description: "Number of configurable commands saved as disabled."},
						{Name: "changed", Type: OutputFieldTypeInteger, Description: "Number of current catalog choices changed from the loaded selection."},
						{Name: "stale_removed", Type: OutputFieldTypeInteger, Description: "Number of no-longer-known saved paths omitted from the replacement."},
					},
					Completeness: OutputCompletenessComplete,
				},
				FixedTarget: &FixedTargetContract{
					Scope:       FixedTargetScopeToolLocal,
					Kind:        "command-selection",
					StableID:    "default",
					Description: "The single user-local set of exact cwk command paths presented to agents.",
				},
				Prerequisites: []string{"Readable line-oriented stdin and a writable user configuration directory."},
				Errors: []CommandError{
					declaredCommandError(fault.KindInvalidInput, "invalid_arguments", false, "help config edit", "Run the selector without command arguments."),
					declaredCommandError(fault.KindInvalidInput, "command_selection_invalid", false, "config edit", "Repair the saved command selection explicitly."),
					declaredCommandError(fault.KindUnavailable, "command_selection_unavailable", true, "config edit", "Restore access to the user configuration directory, then retry."),
					declaredCommandError(fault.KindCanceled, "configuration_canceled", false, "config show", "Inspect the unchanged command selection."),
					declaredCommandError(fault.KindInternal, "configuration_input_failed", false, "config edit", "Retry with a readable bounded stdin stream."),
					declaredCommandError(fault.KindContract, "invalid_mutation_contract", false, "help config edit", "Repair the fixed command-selection target and impact declaration."),
					declaredCommandError(fault.KindContract, "missing_mutation_action", false, "help config edit", "Repair command-selection mutation composition."),
					declaredCommandError(fault.KindRejected, "missing_mutation_policy", false, "help config edit", "Restore the explicit-save mutation policy."),
					declaredCommandError(fault.KindRejected, "mutation_rejected", false, "config edit", "Enter save only after reviewing the exact selection."),
					declaredCommandError(fault.KindContract, "unclassified_mutation_outcome", false, "config show", "Reconcile the saved selection before editing again."),
					declaredCommandError(fault.KindInternal, "output_write_failed", true, "config show", "Inspect the saved selection using a writable output stream."),
					declaredCommandError(fault.KindCanceled, "operation_canceled", true, "config show", "Inspect the unchanged or uncertain selection before retrying."),
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
			handler: runConfigEdit,
		},
	}
}
