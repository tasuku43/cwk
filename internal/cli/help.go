package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/tasuku43/cwk/internal/domain/operation"
)

const (
	maxExactHelpBytes = 2048
	maxHelpExamples   = 2
)

// helpFormat remains as the parse result shape used by command-selection
// routing. Text is now the only public help representation.
type helpFormat uint8

const helpFormatText helpFormat = 0

func runHelp(ctx context.Context, c *CLI, _ CommandSpec, _ operation.Intent, args []string) int {
	_, selector, err := parseHelpArgs(args)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error(), "help", "ルートヘルプにあるコマンドパスまたは名前空間を指定し、廃止済みの agent 形式ではなく --help を使用してください。")
	}

	commands := c.catalog.Commands()
	exact := false
	if selector != "" {
		selector = canonicalHelpSelector(c.catalog, selector)
		commands, exact = c.catalog.Select(selector)
		if len(commands) == 0 {
			return c.failUsage(ctx, "invalid_arguments", fmt.Sprintf("help セレクター %q は不明です。", selector), "help", "ルート help にある正確なコマンドパスまたは名前空間を指定してください。")
		}
	}

	if selector == "" {
		output := c.renderRootHelp()
		if staticHelpFallback(ctx) {
			output = prependStaticHelpNotice(output)
		}
		return c.emit(ctx, output)
	}
	if exact {
		return c.emit(ctx, renderCommandHelp(commands[0], c.catalog.Commands()))
	}
	output := renderNamespaceHelp(selector, commands)
	if staticHelpFallback(ctx) {
		output = prependStaticHelpNotice(output)
	}
	return c.emit(ctx, output)
}

func canonicalHelpSelector(catalog Catalog, selector string) string {
	for _, command := range catalog.Commands() {
		for _, alias := range command.DeprecatedAliases {
			if selector == alias {
				return command.Path
			}
		}
	}
	return selector
}

func prependStaticHelpNotice(output []byte) []byte {
	const notice = "注意: コマンド選択を読み取れないため静的カタログを表示しています。タスク実行前に 'cwk doctor' を実行してください。\n\n"
	result := make([]byte, 0, len(notice)+len(output))
	result = append(result, notice...)
	return append(result, output...)
}

func parseHelpArgs(args []string) (helpFormat, string, error) {
	selectorWords := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--format":
			value := ""
			if index+1 < len(args) {
				value = args[index+1]
			}
			return helpFormatText, "", retiredHelpFormatError(value)
		case strings.HasPrefix(arg, "--format="):
			return helpFormatText, "", retiredHelpFormatError(strings.TrimPrefix(arg, "--format="))
		case strings.HasPrefix(arg, "-"):
			return helpFormatText, "", unknownHelpFlagError(arg)
		default:
			selectorWords = append(selectorWords, strings.Split(arg, ".")...)
		}
	}
	return helpFormatText, strings.Join(selectorWords, " "), nil
}

func retiredHelpFormatError(value string) error {
	if value == "agent" {
		return fmt.Errorf("help の agent 形式は廃止済みです。'cwk <command> --help' または 'cwk help <command>' を使ってください")
	}
	return fmt.Errorf("help の --format は廃止済みです。'cwk <command> --help' または 'cwk help <command>' を使ってください")
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
	commands := c.catalog.Commands()
	directCommands, namespaces := rootTextHelpEntries(commands)
	unconfigured := c.selectionLoaded && !c.selectionSaved
	var output bytes.Buffer
	fmt.Fprintln(&output, "Chatwork CLI")
	if unconfigured {
		fmt.Fprintln(&output)
		fmt.Fprintln(&output, "初回設定:")
		fmt.Fprintln(&output, "  config が未設定のため、現在は制御コマンドだけを表示しています。")
		fmt.Fprintf(&output, "  '%s config' で使用するChatworkコマンドを選んでください。\n", ProgramName)
		fmt.Fprintln(&output, "  必要なコマンドだけを表示することで、エージェントの選択ミスとトークン消費を抑えられます。")
	}
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "使い方:")
	fmt.Fprintf(&output, "  %s [--error-format text|json] <command> [arguments]\n", ProgramName)
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "グローバルオプション:")
	fmt.Fprintln(&output, "  --error-format text|json  構造化エラーの表示形式を選択します（既定: text）")
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "出力:")
	fmt.Fprintln(&output, "  Chatwork タスクの成功結果は text です。JSON は --error-format json のエラーと doctor --format json で利用できます。")
	fmt.Fprintln(&output, "  Chatwork API に全体全文検索とリアクション集計はありません。代替は 'cwk messages list --help' を確認してください。")
	if len(directCommands) > 0 {
		fmt.Fprintln(&output)
		output.Write(renderTextHelpIndex("コマンド:", directCommands))
	}
	if len(namespaces) > 0 {
		fmt.Fprintln(&output)
		output.Write(renderNamespaceIndex("名前空間:", namespaces))
	}
	fmt.Fprintln(&output)
	if unconfigured {
		fmt.Fprintf(&output, "初回設定を始めるには '%s config' を実行してください。\n", ProgramName)
		fmt.Fprintf(&output, "設定方法には '%s config --help' を実行してください。\n", ProgramName)
		return output.Bytes()
	}
	fmt.Fprintf(&output, "コマンドを選ぶには '%s <namespace> --help' を実行してください。\n", ProgramName)
	fmt.Fprintln(&output, "詳細を確認するには、正確なコマンドの末尾に '--help' を付けてください。")
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
	var output bytes.Buffer
	fmt.Fprintln(&output, "Chatwork CLI")
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "名前空間: %s\n", selector)
	fmt.Fprintln(&output, "コマンド:")
	for _, command := range commands {
		fmt.Fprintf(&output, "  %s\n", command.Usage())
		fmt.Fprintf(&output, "    %s\n", command.Summary)
		if required := requiredCommandLineInputs(command); len(required) > 0 {
			fmt.Fprintf(&output, "    必須: %s\n", strings.Join(required, ", "))
		}
		for _, group := range command.ExclusiveInputGroups {
			fmt.Fprintf(&output, "    必須(いずれか1つ): %s\n", strings.Join(group, " | "))
		}
	}
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "詳細には、正確なコマンドの末尾に '--help' を付けてください。全体には '%s --help' を実行してください。\n", ProgramName)
	return output.Bytes()
}

func requiredCommandLineInputs(command CommandSpec) []string {
	required := make([]string, 0)
	for _, input := range command.Agent.Inputs {
		if !input.Required || input.Source != InputSourceFlag {
			continue
		}
		required = append(required, input.Name)
	}
	return required
}

type commandHelpExample struct {
	Title      string
	Invocation string
}

func renderCommandHelp(command CommandSpec, allCommands []CommandSpec) []byte {
	examples := commandHelpExamples(command, allCommands)
	minimum := renderExactCommandHelp(command, examples, 12)
	if len(minimum) > maxExactHelpBytes {
		return minimum
	}

	low, high := 12, 1024
	best := minimum
	for low <= high {
		middle := low + (high-low)/2
		candidate := renderExactCommandHelp(command, examples, middle)
		if len(candidate) <= maxExactHelpBytes {
			best = candidate
			low = middle + 1
			continue
		}
		high = middle - 1
	}
	return best
}

func renderExactCommandHelp(command CommandSpec, examples []commandHelpExample, descriptionLimit int) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, "使い方:")
	fmt.Fprintln(&output, "  "+command.Usage())
	if inputs := commandLineHelpInputs(command.Agent.Inputs); len(inputs) > 0 {
		fmt.Fprintln(&output)
		fmt.Fprintln(&output, "入力:")
		for _, input := range inputs {
			requirement := "任意"
			if input.Required {
				requirement = "必須"
			} else if exclusiveInputGroupMember(command, input.Name) {
				requirement = "選択必須"
			}
			if input.Repeatable {
				requirement += "・繰り返し可"
			}
			fmt.Fprintf(&output, "  %s (%s, %s)  %s\n", input.Name, commandInputType(command, input), requirement, truncateHelpText(input.Description, descriptionLimit))
		}
	}
	if len(command.HelpConstraints) > 0 {
		fmt.Fprintln(&output)
		fmt.Fprintln(&output, "併用制約:")
		for _, constraint := range command.HelpConstraints {
			fmt.Fprintf(&output, "  %s\n", constraint)
		}
	}
	if len(examples) > 0 {
		fmt.Fprintln(&output)
		fmt.Fprintln(&output, "例:")
		for _, example := range examples {
			fmt.Fprintf(&output, "  %s: %s\n", example.Title, example.Invocation)
		}
	}
	return output.Bytes()
}

func exclusiveInputGroupMember(command CommandSpec, name string) bool {
	for _, group := range command.ExclusiveInputGroups {
		for _, member := range group {
			if member == name {
				return true
			}
		}
	}
	return false
}

func commandLineHelpInputs(inputs []CommandInput) []CommandInput {
	result := make([]CommandInput, 0, len(inputs))
	for _, input := range inputs {
		if input.Source == InputSourceFlag {
			result = append(result, input)
		}
	}
	return result
}

func commandInputType(command CommandSpec, input CommandInput) string {
	syntaxInputs, err := parseArgumentSyntaxInputs(command.Args)
	if err == nil {
		if syntax, exists := syntaxInputs[input.Name]; exists {
			if strings.HasPrefix(syntax.ValueSyntax, "<") && strings.HasSuffix(syntax.ValueSyntax, ">") {
				return strings.Trim(syntax.ValueSyntax, "<>")
			}
			if len(syntax.AllowedValues) > 0 {
				return strings.Join(syntax.AllowedValues, "|")
			}
			if syntax.ValueSyntax == "" && input.Source == InputSourceFlag {
				return "boolean"
			}
		}
	}
	if len(input.AllowedValues) > 0 {
		return strings.Join(input.AllowedValues, "|")
	}
	if input.ReferenceKind != "" {
		return input.ReferenceKind + "-ref"
	}
	if input.Source == InputSourceArgument {
		return "selector"
	}
	return "text"
}

func truncateHelpText(value string, maximum int) string {
	value = strings.Join(strings.Fields(value), " ")
	if maximum <= 0 || len(value) <= maximum {
		return value
	}
	if maximum <= len("…") {
		return "…"
	}
	end := maximum - len("…")
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return strings.TrimSpace(value[:end]) + "…"
}

func commandHelpExamples(command CommandSpec, allCommands []CommandSpec) []commandHelpExample {
	recipes := relevantHumanHelpRecipes(allCommands, []CommandSpec{command})
	examples := make([]commandHelpExample, 0, maxHelpExamples)
	seen := make(map[string]struct{})
	for _, recipe := range recipes {
		for _, step := range recipe.Steps {
			if step.Path != command.Path {
				continue
			}
			invocation := ProgramName + " " + step.Path
			if step.Args != "" {
				invocation += " " + step.Args
			}
			key := recipe.Title + "\x00" + invocation
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			examples = append(examples, commandHelpExample{Title: recipe.Title, Invocation: invocation})
			if len(examples) == maxHelpExamples {
				return examples
			}
		}
	}
	return examples
}

func relevantHumanHelpRecipes(commands, relevant []CommandSpec) []HumanHelpRecipe {
	available := make(map[string]struct{}, len(commands))
	for _, command := range commands {
		available[command.Path] = struct{}{}
	}
	relevantPaths := make(map[string]struct{}, len(relevant))
	for _, command := range relevant {
		relevantPaths[command.Path] = struct{}{}
	}
	recipes := make([]HumanHelpRecipe, 0)
	seen := make(map[string]struct{})
	for _, command := range commands {
		for _, recipe := range command.HumanRecipes {
			if _, exists := seen[recipe.Title]; exists {
				continue
			}
			complete := true
			touchesSelection := false
			for _, step := range recipe.Steps {
				if _, exists := available[step.Path]; !exists {
					complete = false
					break
				}
				if _, exists := relevantPaths[step.Path]; exists {
					touchesSelection = true
				}
			}
			if !complete || !touchesSelection {
				continue
			}
			seen[recipe.Title] = struct{}{}
			recipes = append(recipes, recipe)
		}
	}
	return recipes
}

func commandNamespace(path string) string {
	if boundary := strings.IndexByte(path, ' '); boundary >= 0 {
		return path[:boundary]
	}
	return path
}
