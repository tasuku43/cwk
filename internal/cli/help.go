package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

type helpFormat uint8

const (
	helpFormatText helpFormat = iota
	helpFormatAgent
	agentHelpSchemaVersion = 3
)

type agentIndexDocument struct {
	SchemaVersion int                 `json:"schema_version"`
	View          string              `json:"view"`
	Program       string              `json:"program"`
	ScopeRequest  agentScopeRequest   `json:"scope_request"`
	Commands      []agentIndexCommand `json:"commands"`
}

type agentScopeRequest struct {
	InvocationTemplate           string   `json:"invocation_template"`
	SelectorFields               []string `json:"selector_fields"`
	UnknownOutcomeMaxInvocations int      `json:"unknown_outcome_max_invocations"`
	KnownPathMaxInvocations      int      `json:"known_path_max_invocations"`
}

type agentIndexCommand struct {
	Path         string `json:"path"`
	Namespace    string `json:"namespace"`
	Summary      string `json:"summary"`
	CapabilityID string `json:"capability_id"`
	Outcome      string `json:"outcome"`
	Effect       string `json:"effect"`
	Role         string `json:"role"`
}

type agentDocument struct {
	SchemaVersion int                `json:"schema_version"`
	View          string             `json:"view"`
	Program       string             `json:"program"`
	Scope         agentScope         `json:"scope"`
	GlobalInputs  []CommandInput     `json:"global_inputs"`
	IOContract    agentIOContract    `json:"io_contract"`
	ErrorContract agentErrorContract `json:"error_contract"`
	Commands      []agentCommand     `json:"commands"`
	Workflows     []agentWorkflow    `json:"workflows"`
}

type agentScope struct {
	Selector string `json:"selector"`
	Kind     string `json:"kind"`
}

type agentIOContract struct {
	SuccessStream                      string `json:"success_stream"`
	ErrorStream                        string `json:"error_stream"`
	SuccessStatusRequiresCompleteWrite bool   `json:"success_status_requires_complete_write"`
	PartialOutputIsSuccess             bool   `json:"partial_output_is_success"`
	ExternalTextTrust                  string `json:"external_text_trust"`
	ExternalTextProjection             string `json:"external_text_projection"`
	OpaqueReferencePolicy              string `json:"opaque_reference_policy"`
}

type agentErrorContract struct {
	Formats            []string          `json:"formats"`
	DefaultFormat      string            `json:"default_format"`
	JSONSchemaVersion  int               `json:"json_schema_version"`
	Fields             []agentErrorField `json:"fields"`
	ExitCodes          []agentExitCode   `json:"exit_codes"`
	GlobalErrors       []CommandError    `json:"global_errors"`
	CommandErrorsField string            `json:"command_errors_field"`
}

type agentErrorField struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type agentExitCode struct {
	Kind fault.Kind `json:"kind"`
	Code int        `json:"code"`
}

type agentCommand struct {
	Path         string            `json:"path"`
	Summary      string            `json:"summary"`
	Usage        string            `json:"usage"`
	Args         string            `json:"args,omitempty"`
	Effect       string            `json:"effect"`
	Role         string            `json:"role"`
	Contract     AgentContract     `json:"contract"`
	ProducesRefs []ProducedRef     `json:"produces_refs"`
	ConsumesRefs []ConsumedRef     `json:"consumes_refs"`
	NextActions  []agentNextAction `json:"next_actions"`
}

type agentWorkflow struct {
	ReferenceKind string                `json:"reference_kind"`
	Producer      agentWorkflowProducer `json:"producer"`
	Consumer      agentWorkflowConsumer `json:"consumer"`
}

type agentWorkflowProducer struct {
	Path  string `json:"path"`
	Usage string `json:"usage"`
	Field string `json:"field"`
}

type agentWorkflowConsumer struct {
	Path  string `json:"path"`
	Usage string `json:"usage"`
	Input string `json:"input"`
}

type agentNextAction struct {
	Path          string `json:"path"`
	Usage         string `json:"usage"`
	ReferenceKind string `json:"reference_kind"`
	FromField     string `json:"from_field"`
	ToInput       string `json:"to_input"`
}

func runHelp(ctx context.Context, c *CLI, _ CommandSpec, _ operation.Intent, args []string) int {
	format, selector, err := parseHelpArgs(args)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error(), "help", "Use a supported format and canonical selector.")
	}

	commands := c.catalog.Commands()
	exact := false
	if selector != "" {
		commands, exact = c.catalog.Select(selector)
		if len(commands) == 0 {
			return c.failUsage(ctx, "invalid_arguments", fmt.Sprintf("Unknown help selector %q.", selector), "help", "Use an exact command path or namespace from the root help.")
		}
	}

	if format == helpFormatAgent {
		var output []byte
		if selector == "" {
			output, err = c.renderAgentIndex(commands)
		} else {
			output, err = c.renderAgentHelp(selector, exact, commands)
		}
		if err != nil {
			return c.fail(ctx, err)
		}
		return c.emit(ctx, output)
	}
	if selector == "" {
		return c.emit(ctx, c.renderRootHelp())
	}
	if exact {
		return c.emit(ctx, renderCommandHelp(commands[0]))
	}
	return c.emit(ctx, renderNamespaceHelp(selector, commands))
}

func parseHelpArgs(args []string) (helpFormat, string, error) {
	format := helpFormatText
	selectorWords := make([]string, 0, len(args))
	seenFormat := false
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--format":
			if seenFormat {
				return format, "", fmt.Errorf("--format may be specified only once")
			}
			if index+1 >= len(args) {
				return format, "", fmt.Errorf("--format requires text or agent")
			}
			index++
			parsed, err := parseHelpFormat(args[index])
			if err != nil {
				return format, "", err
			}
			format = parsed
			seenFormat = true
		case strings.HasPrefix(arg, "--format="):
			if seenFormat {
				return format, "", fmt.Errorf("--format may be specified only once")
			}
			parsed, err := parseHelpFormat(strings.TrimPrefix(arg, "--format="))
			if err != nil {
				return format, "", err
			}
			format = parsed
			seenFormat = true
		case strings.HasPrefix(arg, "-"):
			return format, "", fmt.Errorf("unknown help flag %q", arg)
		default:
			selectorWords = append(selectorWords, arg)
		}
	}
	return format, strings.Join(selectorWords, " "), nil
}

func parseHelpFormat(value string) (helpFormat, error) {
	switch value {
	case "text":
		return helpFormatText, nil
	case "agent":
		return helpFormatAgent, nil
	default:
		return helpFormatText, fmt.Errorf("--format must be text or agent")
	}
}

// Select returns an exact command or every command beneath a canonical word
// boundary namespace. Catalog order remains the stable presentation order.
func (c Catalog) Select(selector string) ([]CommandSpec, bool) {
	if err := operation.ValidateCommandPath(selector); err != nil {
		return []CommandSpec{}, false
	}
	if command, found := c.Lookup(selector); found {
		return []CommandSpec{command}, true
	}
	commands := make([]CommandSpec, 0)
	for _, command := range c.commands {
		if strings.HasPrefix(command.Path, selector+" ") {
			commands = append(commands, cloneCommandSpec(command))
		}
	}
	return commands, false
}

func (c *CLI) renderRootHelp() []byte {
	directCommands, namespaces := rootTextHelpEntries(c.catalog.Commands())
	var output bytes.Buffer
	fmt.Fprintln(&output, "Chatwork CLI")
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "Usage:")
	fmt.Fprintf(&output, "  %s [--error-format text|json] <command> [arguments]\n", ProgramName)
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "Global options:")
	fmt.Fprintln(&output, "  --error-format text|json  Select structured failure presentation (default: text)")
	if len(directCommands) > 0 {
		fmt.Fprintln(&output)
		output.Write(renderTextHelpIndex("Commands:", directCommands))
	}
	if len(namespaces) > 0 {
		fmt.Fprintln(&output)
		output.Write(renderNamespaceIndex("Namespaces:", namespaces))
	}
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "Run '%s <namespace> --help' to choose a command.\n", ProgramName)
	fmt.Fprintln(&output, "Append '--help' to any exact command for details.")
	fmt.Fprintf(&output, "Run '%s help <command-or-namespace> --format agent' for a scoped machine contract.\n", ProgramName)
	return output.Bytes()
}

type textHelpEntry struct {
	Name    string
	Summary string
}

type textHelpNamespace struct {
	Name         string
	CommandCount int
}

func rootTextHelpEntries(commands []CommandSpec) ([]textHelpEntry, []textHelpNamespace) {
	directCommands := make([]textHelpEntry, 0)
	namespaces := make([]textHelpNamespace, 0)
	namespaceIndexes := make(map[string]int)
	for _, command := range commands {
		namespace := commandNamespace(command.Path)
		if namespace == command.Path {
			directCommands = append(directCommands, textHelpEntry{Name: command.Path, Summary: command.Summary})
			continue
		}
		index, exists := namespaceIndexes[namespace]
		if !exists {
			index = len(namespaces)
			namespaceIndexes[namespace] = index
			namespaces = append(namespaces, textHelpNamespace{Name: namespace})
		}
		namespaces[index].CommandCount++
	}
	return directCommands, namespaces
}

func renderTextHelpIndex(title string, entries []textHelpEntry) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, title)
	width := 0
	for _, entry := range entries {
		if len(entry.Name) > width {
			width = len(entry.Name)
		}
	}
	for _, entry := range entries {
		fmt.Fprintf(&output, "  %-*s  %s\n", width, entry.Name, entry.Summary)
	}
	return output.Bytes()
}

func renderNamespaceIndex(title string, namespaces []textHelpNamespace) []byte {
	entries := make([]textHelpEntry, 0, len(namespaces))
	for _, namespace := range namespaces {
		unit := "commands"
		if namespace.CommandCount == 1 {
			unit = "command"
		}
		entries = append(entries, textHelpEntry{
			Name:    namespace.Name,
			Summary: fmt.Sprintf("%d %s", namespace.CommandCount, unit),
		})
	}
	return renderTextHelpIndex(title, entries)
}

func renderNamespaceHelp(selector string, commands []CommandSpec) []byte {
	entries := make([]textHelpEntry, 0, len(commands))
	prefix := selector + " "
	for _, command := range commands {
		entries = append(entries, textHelpEntry{
			Name:    strings.TrimPrefix(command.Path, prefix),
			Summary: command.Summary,
		})
	}

	var output bytes.Buffer
	fmt.Fprintln(&output, "Chatwork CLI")
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "Usage:")
	fmt.Fprintf(&output, "  %s %s <command> [arguments]\n", ProgramName, selector)
	fmt.Fprintln(&output)
	output.Write(renderTextHelpIndex("Commands:", entries))
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "Run '%s %s <command> --help' for exact command details.\n", ProgramName, selector)
	fmt.Fprintf(&output, "Run '%s help %s <command> --format agent' for one command's machine contract.\n", ProgramName, selector)
	fmt.Fprintf(&output, "Run '%s help %s --format agent' for all machine contracts in this namespace.\n", ProgramName, selector)
	fmt.Fprintf(&output, "Run '%s --help' for all commands and namespaces.\n", ProgramName)
	return output.Bytes()
}

func renderCommandHelp(command CommandSpec) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, "Usage:")
	fmt.Fprintln(&output, "  "+command.Usage())
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, command.Summary+".")
	if len(command.Agent.Inputs) > 0 {
		fmt.Fprintln(&output)
		output.Write(renderCommandInputs(command.Agent.Inputs))
	}
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "Capability: "+command.Agent.CapabilityID)
	fmt.Fprintln(&output, "Outcome: "+command.Agent.Outcome)
	fmt.Fprintln(&output, "Effect: "+command.Effect.String())
	fmt.Fprintln(&output, "Role: "+command.Role.String())
	for _, reference := range command.ProducedRefs() {
		fmt.Fprintf(&output, "Produces reference: %s in field %s\n", reference.Kind, reference.Field)
	}
	for _, reference := range command.ConsumedRefs() {
		fmt.Fprintf(&output, "Consumes reference: %s from input %s\n", reference.Kind, reference.Argument)
	}
	fmt.Fprintln(&output)
	namespace := commandNamespace(command.Path)
	if namespace == command.Path {
		fmt.Fprintf(&output, "Run '%s --help' for all commands and namespaces.\n", ProgramName)
	} else {
		fmt.Fprintf(&output, "Run '%s %s --help' for other commands in this namespace.\n", ProgramName, namespace)
	}
	fmt.Fprintf(&output, "Run '%s help %s --format agent' for the machine contract.\n", ProgramName, command.Path)
	return output.Bytes()
}

func renderCommandInputs(inputs []CommandInput) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, "Inputs:")
	width := 0
	for _, input := range inputs {
		if len(input.Name) > width {
			width = len(input.Name)
		}
	}
	for _, input := range inputs {
		requirement := "optional"
		if input.Required {
			requirement = "required"
		}
		if input.Repeatable {
			requirement += " repeatable"
		}
		attributes := []string{requirement + " " + string(input.Source)}
		if len(input.AllowedValues) > 0 {
			attributes = append(attributes, "values="+strings.Join(input.AllowedValues, "|"))
		}
		if input.ReferenceKind != "" {
			attributes = append(attributes, "reference="+input.ReferenceKind)
		}
		fmt.Fprintf(&output, "  %-*s  %s\n", width, input.Name, strings.Join(attributes, ", "))
		fmt.Fprintf(&output, "    %s\n", input.Description)
	}
	return output.Bytes()
}

func (c *CLI) renderAgentIndex(commands []CommandSpec) ([]byte, error) {
	document := agentIndexDocument{
		SchemaVersion: agentHelpSchemaVersion,
		View:          "index",
		Program:       ProgramName,
		ScopeRequest: agentScopeRequest{
			InvocationTemplate:           ProgramName + " help <command-or-namespace> --format agent",
			SelectorFields:               []string{"commands[].path", "commands[].namespace"},
			UnknownOutcomeMaxInvocations: 2,
			KnownPathMaxInvocations:      1,
		},
		Commands: make([]agentIndexCommand, 0, len(commands)),
	}
	for _, command := range commands {
		document.Commands = append(document.Commands, projectAgentIndexCommand(command))
	}
	output, err := json.Marshal(document)
	if err != nil {
		return nil, fault.Wrap(fault.KindContract, "output_encoding_failed", "The agent help index could not be encoded.", false, err)
	}
	return append(output, '\n'), nil
}

func projectAgentIndexCommand(command CommandSpec) agentIndexCommand {
	return agentIndexCommand{
		Path:         command.Path,
		Namespace:    commandNamespace(command.Path),
		Summary:      command.Summary,
		CapabilityID: command.Agent.CapabilityID,
		Outcome:      command.Agent.Outcome,
		Effect:       command.Effect.String(),
		Role:         command.Role.String(),
	}
}

func commandNamespace(path string) string {
	if boundary := strings.IndexByte(path, ' '); boundary >= 0 {
		return path[:boundary]
	}
	return path
}

func (c *CLI) renderAgentHelp(selector string, exact bool, commands []CommandSpec) ([]byte, error) {
	workflows := c.catalog.referenceWorkflows()
	scopeKind := "namespace"
	if exact {
		scopeKind = "command"
	}
	document := agentDocument{
		SchemaVersion: agentHelpSchemaVersion,
		View:          "scope",
		Program:       ProgramName,
		Scope:         agentScope{Selector: selector, Kind: scopeKind},
		GlobalInputs: []CommandInput{{
			Name: "--error-format", Source: InputSourceFlag, Required: false,
			Description:   "Select text or stable JSON stderr; place this global option before the command.",
			AllowedValues: []string{"text", "json"},
		}},
		ErrorContract: defaultAgentErrorContract(),
		IOContract: agentIOContract{
			SuccessStream: "stdout", ErrorStream: "stderr",
			SuccessStatusRequiresCompleteWrite: true,
			PartialOutputIsSuccess:             false,
			ExternalTextTrust:                  "untrusted_data",
			ExternalTextProjection:             "visible_escape",
			OpaqueReferencePolicy:              "validated_exact_bytes",
		},
		Commands:  make([]agentCommand, 0, len(commands)),
		Workflows: workflowsForCommands(workflows, commands),
	}
	for _, command := range commands {
		document.Commands = append(document.Commands, agentCommand{
			Path:         command.Path,
			Summary:      command.Summary,
			Usage:        command.Usage(),
			Args:         command.Args,
			Effect:       command.Effect.String(),
			Role:         command.Role.String(),
			Contract:     cloneAgentContract(command.Agent),
			ProducesRefs: command.ProducedRefs(),
			ConsumesRefs: command.ConsumedRefs(),
			NextActions:  nextActionsForCommand(workflows, command.Path),
		})
	}
	output, err := json.Marshal(document)
	if err != nil {
		return nil, fault.Wrap(fault.KindContract, "output_encoding_failed", "The agent help document could not be encoded.", false, err)
	}
	return append(output, '\n'), nil
}

func defaultAgentErrorContract() agentErrorContract {
	return agentErrorContract{
		Formats:           []string{"text", "json"},
		DefaultFormat:     "text",
		JSONSchemaVersion: 1,
		Fields: []agentErrorField{
			{Name: "kind", Description: "Cross-command recovery class."},
			{Name: "code", Description: "Stable command-specific failure code."},
			{Name: "message", Description: "Safe human explanation that excludes upstream causes."},
			{Name: "retryable", Description: "Whether retrying without changing command intent can succeed."},
			{Name: "retry_after", Description: "Optional stable duration or null."},
			{Name: "next_actions", Description: "Structured commands and reasons for recovery."},
		},
		ExitCodes: []agentExitCode{
			{Kind: fault.KindInvalidInput, Code: ExitUsage},
			{Kind: fault.KindAuthentication, Code: ExitAuthentication},
			{Kind: fault.KindPermission, Code: ExitPermission},
			{Kind: fault.KindNotFound, Code: ExitNotFound},
			{Kind: fault.KindAmbiguous, Code: ExitAmbiguous},
			{Kind: fault.KindRateLimited, Code: ExitRateLimited},
			{Kind: fault.KindUnavailable, Code: ExitUnavailable},
			{Kind: fault.KindRejected, Code: ExitRejected},
			{Kind: fault.KindCanceled, Code: ExitCanceled},
			{Kind: fault.KindUnsupported, Code: ExitUnsupported},
			{Kind: fault.KindContract, Code: ExitContract},
			{Kind: fault.KindInternal, Code: ExitInternal},
		},
		GlobalErrors: []CommandError{
			declaredCommandError(fault.KindInvalidInput, "invalid_root_options", false, "help", "Correct the global options."),
			declaredCommandError(fault.KindInvalidInput, "missing_command", false, "help", "Discover available command outcomes."),
			declaredCommandError(fault.KindInvalidInput, "unknown_command", false, "help", "Discover an exact command path or namespace."),
			declaredCommandError(fault.KindContract, "missing_context", false, "help", "Retry through a context-aware CLI entry point."),
			declaredCommandError(fault.KindContract, "invalid_catalog", false, "help", "Repair the catalog before dispatch."),
			declaredCommandError(fault.KindCanceled, "operation_canceled", true, "help", "Retry when the caller is ready."),
		},
		CommandErrorsField: "commands[].contract.errors",
	}
}

func (c Catalog) referenceWorkflows() []agentWorkflow {
	commands := c.Commands()
	workflows := make([]agentWorkflow, 0)
	for _, producer := range commands {
		for _, produced := range producer.ProducedRefs() {
			for _, consumer := range commands {
				for _, consumed := range consumer.ConsumedRefs() {
					if consumed.Kind != produced.Kind {
						continue
					}
					workflows = append(workflows, agentWorkflow{
						ReferenceKind: produced.Kind,
						Producer: agentWorkflowProducer{
							Path: producer.Path, Usage: producer.Usage(), Field: produced.Field,
						},
						Consumer: agentWorkflowConsumer{
							Path: consumer.Path, Usage: consumer.Usage(), Input: consumed.Argument,
						},
					})
				}
			}
		}
	}
	return workflows
}

func workflowsForCommands(workflows []agentWorkflow, commands []CommandSpec) []agentWorkflow {
	selected := make(map[string]struct{}, len(commands))
	for _, command := range commands {
		selected[command.Path] = struct{}{}
	}
	filtered := make([]agentWorkflow, 0)
	for _, workflow := range workflows {
		_, produces := selected[workflow.Producer.Path]
		_, consumes := selected[workflow.Consumer.Path]
		if produces || consumes {
			filtered = append(filtered, workflow)
		}
	}
	return filtered
}

func nextActionsForCommand(workflows []agentWorkflow, path string) []agentNextAction {
	actions := make([]agentNextAction, 0)
	for _, workflow := range workflows {
		if workflow.Producer.Path != path {
			continue
		}
		actions = append(actions, agentNextAction{
			Path:          workflow.Consumer.Path,
			Usage:         workflow.Consumer.Usage,
			ReferenceKind: workflow.ReferenceKind,
			FromField:     workflow.Producer.Field,
			ToInput:       workflow.Consumer.Input,
		})
	}
	return actions
}
