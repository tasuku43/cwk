package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

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
		return c.failUsage(ctx, "invalid_arguments", err.Error(), "help", "対応している形式と正規セレクターを指定してください。")
	}

	commands := c.catalog.Commands()
	exact := false
	if selector != "" {
		commands, exact = c.catalog.Select(selector)
		if len(commands) == 0 {
			return c.failUsage(ctx, "invalid_arguments", fmt.Sprintf("help セレクター %q は不明です。", selector), "help", "ルート help にある正確なコマンドパスまたは名前空間を指定してください。")
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
				return format, "", fmt.Errorf("--format は1回だけ指定できます")
			}
			if index+1 >= len(args) {
				return format, "", fmt.Errorf("--format には text または agent が必要です")
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
				return format, "", fmt.Errorf("--format は1回だけ指定できます")
			}
			parsed, err := parseHelpFormat(strings.TrimPrefix(arg, "--format="))
			if err != nil {
				return format, "", err
			}
			format = parsed
			seenFormat = true
		case strings.HasPrefix(arg, "-"):
			return format, "", fmt.Errorf("help フラグ %q は不明です", arg)
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
		return helpFormatText, fmt.Errorf("--format は text または agent で指定してください")
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
	fmt.Fprintln(&output, "使い方:")
	fmt.Fprintf(&output, "  %s [--error-format text|json] <command> [arguments]\n", ProgramName)
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "グローバルオプション:")
	fmt.Fprintln(&output, "  --error-format text|json  構造化エラーの表示形式を選択します（既定: text）")
	if len(directCommands) > 0 {
		fmt.Fprintln(&output)
		output.Write(renderTextHelpIndex("コマンド:", directCommands))
	}
	if len(namespaces) > 0 {
		fmt.Fprintln(&output)
		output.Write(renderNamespaceIndex("名前空間:", namespaces))
	}
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "コマンドを選ぶには '%s <namespace> --help' を実行してください。\n", ProgramName)
	fmt.Fprintln(&output, "詳細を確認するには、正確なコマンドの末尾に '--help' を付けてください。")
	fmt.Fprintf(&output, "範囲を限定した機械可読契約を確認するには '%s help <command-or-namespace> --format agent' を実行してください。\n", ProgramName)
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
		entries = append(entries, textHelpEntry{
			Name:    namespace.Name,
			Summary: fmt.Sprintf("%d コマンド", namespace.CommandCount),
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
	fmt.Fprintln(&output, "使い方:")
	fmt.Fprintf(&output, "  %s %s <command> [arguments]\n", ProgramName, selector)
	fmt.Fprintln(&output)
	output.Write(renderTextHelpIndex("コマンド:", entries))
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "正確なコマンドの詳細には '%s %s <command> --help' を実行してください。\n", ProgramName, selector)
	fmt.Fprintf(&output, "1コマンドの機械可読契約には '%s help %s <command> --format agent' を実行してください。\n", ProgramName, selector)
	fmt.Fprintf(&output, "この名前空間にある全機械可読契約には '%s help %s --format agent' を実行してください。\n", ProgramName, selector)
	fmt.Fprintf(&output, "全コマンドと名前空間には '%s --help' を実行してください。\n", ProgramName)
	return output.Bytes()
}

func renderCommandHelp(command CommandSpec) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, "使い方:")
	fmt.Fprintln(&output, "  "+command.Usage())
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, localizedSentence(command.Summary))
	if len(command.Agent.Inputs) > 0 {
		fmt.Fprintln(&output)
		output.Write(renderCommandInputs(command.Agent.Inputs))
	}
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "機能ID: "+command.Agent.CapabilityID)
	fmt.Fprintln(&output, "結果: "+command.Agent.Outcome)
	fmt.Fprintln(&output, "効果: "+command.Effect.String())
	fmt.Fprintln(&output, "役割: "+command.Role.String())
	for _, reference := range command.ProducedRefs() {
		fmt.Fprintf(&output, "生成する参照: %s（フィールド %s）\n", reference.Kind, reference.Field)
	}
	for _, reference := range command.ConsumedRefs() {
		fmt.Fprintf(&output, "使用する参照: %s（入力 %s）\n", reference.Kind, reference.Argument)
	}
	fmt.Fprintln(&output)
	namespace := commandNamespace(command.Path)
	if namespace == command.Path {
		fmt.Fprintf(&output, "全コマンドと名前空間には '%s --help' を実行してください。\n", ProgramName)
	} else {
		fmt.Fprintf(&output, "この名前空間の他のコマンドには '%s %s --help' を実行してください。\n", ProgramName, namespace)
	}
	fmt.Fprintf(&output, "機械可読契約には '%s help %s --format agent' を実行してください。\n", ProgramName, command.Path)
	return output.Bytes()
}

func localizedSentence(value string) string {
	for _, r := range value {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return value + "。"
		}
	}
	return value + "."
}

func renderCommandInputs(inputs []CommandInput) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, "入力:")
	width := 0
	for _, input := range inputs {
		if len(input.Name) > width {
			width = len(input.Name)
		}
	}
	for _, input := range inputs {
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
		return nil, fault.Wrap(fault.KindContract, "output_encoding_failed", "agent help の索引をエンコードできませんでした。", false, err)
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
			Description:   "stderr を text または安定した JSON で表示します。このグローバルオプションはコマンドより前に置いてください。",
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
		return nil, fault.Wrap(fault.KindContract, "output_encoding_failed", "agent help 文書をエンコードできませんでした。", false, err)
	}
	return append(output, '\n'), nil
}

func defaultAgentErrorContract() agentErrorContract {
	return agentErrorContract{
		Formats:           []string{"text", "json"},
		DefaultFormat:     "text",
		JSONSchemaVersion: 1,
		Fields: []agentErrorField{
			{Name: "kind", Description: "コマンド間で共通の復旧分類。"},
			{Name: "code", Description: "安定したコマンド固有のエラーコード。"},
			{Name: "message", Description: "上流の原因を含まない、安全な人向け説明。"},
			{Name: "retryable", Description: "コマンドの意図を変えずに再試行して成功し得るか。"},
			{Name: "retry_after", Description: "任意の安定した待機時間、または null。"},
			{Name: "next_actions", Description: "復旧用の構造化されたコマンドと理由。"},
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
			declaredCommandError(fault.KindInvalidInput, "invalid_root_options", false, "help", "グローバルオプションを修正してください。"),
			declaredCommandError(fault.KindInvalidInput, "missing_command", false, "help", "利用できるコマンドの結果を確認してください。"),
			declaredCommandError(fault.KindInvalidInput, "unknown_command", false, "help", "正確なコマンドパスまたは名前空間を確認してください。"),
			declaredCommandError(fault.KindInvalidInput, "command_selection_invalid", false, "config", "無効なコマンド選択を明示的に置き換えてください。"),
			declaredCommandError(fault.KindUnavailable, "command_selection_unsafe", false, "doctor", "ローカル設定パスを復旧してから、コマンド選択の診断を確認してください。"),
			declaredCommandError(fault.KindUnavailable, "command_selection_unavailable", true, "doctor", "ローカル設定パスを復旧してから、コマンド選択の診断を確認してください。"),
			declaredCommandError(fault.KindContract, "missing_context", false, "help", "コンテキスト対応の CLI エントリーポイントから再試行してください。"),
			declaredCommandError(fault.KindContract, "invalid_catalog", false, "help", "ディスパッチ前にカタログを修復してください。"),
			declaredCommandError(fault.KindCanceled, "operation_canceled", true, "help", "呼び出し元の準備ができたら再試行してください。"),
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
