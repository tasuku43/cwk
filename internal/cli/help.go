package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/tasuku43/cwk/internal/domain/operation"
)

const (
	maxExactHelpBytes               = 2048
	maxHelpExamples                 = 2
	maxRootHelpOutcomesPerNamespace = 3
	maxRootHelpSummaryBytes         = 192
)

const staticHelpNotice = "注意: コマンド選択を読み取れないため静的カタログを表示しています。タスク実行前に 'cwk doctor' を実行してください。\n\n"

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
	disabledCommands := c.disabledCommandSpecs()
	exact := false
	if selector != "" {
		selector = canonicalHelpSelector(c.catalog, selector)
		commands, exact = c.catalog.Select(selector)
		if len(commands) == 0 {
			if selector == "tasks" {
				message := taskHelpSelectorMessage(c.catalog)
				if staticHelpFallback(ctx) {
					message = genericHelpSelectorMessage(selector)
				}
				return c.failUsage(ctx, "invalid_arguments", message, "help", "ルートヘルプから現在使える名前空間を確認してください。")
			}
			known, knownExact := c.completeCatalog().Select(selector)
			if knownExact || len(known) == 0 || len(disabledCommandsForNamespace(disabledCommands, selector)) == 0 {
				return c.failUsage(ctx, "invalid_arguments", fmt.Sprintf("help セレクター %q は不明です。", selector), "help", "ルート help にある正確なコマンドパスまたは名前空間を指定してください。")
			}
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
	output := renderNamespaceHelpWithDisabled(selector, commands, disabledCommands)
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

const (
	roomTasksHelpNamespace     = "room-tasks"
	personalTasksHelpNamespace = "personal-tasks"
)

func taskNamespaceHelpCandidates() []string {
	return []string{roomTasksHelpNamespace, personalTasksHelpNamespace}
}

func taskHelpSelectorMessage(active Catalog) string {
	namespaces := activeTaskHelpNamespaces(active)
	switch len(namespaces) {
	case 1:
		return fmt.Sprintf("help セレクター \"tasks\" は名前空間ではありません。利用できる候補の詳細は 'cwk help %s' を確認してください。", namespaces[0])
	case 2:
		return "help セレクター \"tasks\" は名前空間ではありません。複数のタスク名前空間が使えるため、'cwk help' から選んでください。"
	default:
		return genericHelpSelectorMessage("tasks")
	}
}

func genericHelpSelectorMessage(selector string) string {
	return fmt.Sprintf("help セレクター %q は不明です。利用できるコマンドは 'cwk --help' で確認してください。", selector)
}

func activeTaskHelpNamespaces(catalog Catalog) []string {
	candidates := taskNamespaceHelpCandidates()
	activeNamespaces := make([]string, 0, len(candidates))
	for _, namespace := range candidates {
		for _, command := range catalog.commands {
			if strings.HasPrefix(command.Path, namespace+" ") {
				activeNamespaces = append(activeNamespaces, namespace)
				break
			}
		}
	}
	return activeNamespaces
}

func hasConfigurableTaskHelpNamespace(catalog Catalog) bool {
	for _, namespace := range taskNamespaceHelpCandidates() {
		for _, command := range catalog.commands {
			if command.Configurable && strings.HasPrefix(command.Path, namespace+" ") {
				return true
			}
		}
	}
	return false
}

func prependStaticHelpNotice(output []byte) []byte {
	result := make([]byte, 0, len(staticHelpNotice)+len(output))
	result = append(result, staticHelpNotice...)
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
	disabled := c.disabledCommandSpecs()
	complete := c.completeCatalog().Commands()
	if len(disabled) == 0 {
		complete = commands
	}
	directCommands, namespaces := rootTextHelpEntriesForView(complete, commands, disabled)
	unconfigured := c.selectionLoaded && !c.selectionSaved
	var output bytes.Buffer
	fmt.Fprintln(&output, "Chatwork CLI")
	if unconfigured {
		fmt.Fprintln(&output)
		fmt.Fprintln(&output, "初回設定:")
		fmt.Fprintln(&output, "  config が未設定のため、このcwkで実行できるのは制御コマンドだけです。")
		fmt.Fprintf(&output, "  '%s config' で実行できるChatworkコマンドを選んでください。\n", ProgramName)
		fmt.Fprintln(&output, "  必要なコマンドだけを有効にすると、エージェントの選択ミスとトークン消費も抑えられます。")
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
	fmt.Fprintln(&output, "  Chatwork API に全体全文検索とリアクション集計はありません。")
	if command, found := c.catalog.Lookup("messages list"); found {
		fmt.Fprintf(&output, "  代替は '%s %s --help' を確認してください。\n", ProgramName, command.Path)
	}
	if len(directCommands) > 0 {
		fmt.Fprintln(&output)
		output.Write(renderTextHelpIndex("コマンド:", directCommands))
	}
	if len(namespaces) > 0 {
		fmt.Fprintln(&output)
		output.Write(renderNamespaceIndex("名前空間:", namespaces))
	}
	resolver, hasResolver := c.catalog.Lookup("messages resolve-link")
	show, hasShow := c.catalog.Lookup("messages show")
	if hasResolver {
		fmt.Fprintln(&output)
		fmt.Fprintf(&output, "リンクから参照を得る: '%s %s --help'\n", ProgramName, resolver.Path)
		if !hasShow {
			fmt.Fprintln(&output, "発言を読むには、人に config で messages show の有効化を依頼してください。")
		}
	}
	if hasShow {
		fmt.Fprintln(&output)
		fmt.Fprintf(&output, "参照から発言を読む: '%s %s --help'\n", ProgramName, show.Path)
	}
	fmt.Fprintln(&output)
	if unconfigured {
		fmt.Fprintf(&output, "実行範囲の設定を始めるには '%s config' を実行してください。\n", ProgramName)
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
	Name          string
	CommandCount  int
	DisabledCount int
	OutcomeCount  int
	Outcomes      []textHelpOutcome
}

type textHelpOutcome struct {
	Summary string
	Path    string
}

func rootTextHelpEntries(commands []CommandSpec) ([]textHelpEntry, []textHelpNamespace) {
	return rootTextHelpEntriesForView(commands, commands, nil)
}

func rootTextHelpEntriesForView(complete, active, disabled []CommandSpec) ([]textHelpEntry, []textHelpNamespace) {
	directCommands := make([]textHelpEntry, 0)
	namespaces := make([]textHelpNamespace, 0)
	namespaceIndexes := make(map[string]int)
	activeCommands := make(map[string]CommandSpec, len(active))
	seenOutcomes := make(map[string]map[string]struct{})
	for _, command := range active {
		activeCommands[command.Path] = command
		namespace := commandNamespace(command.Path)
		if namespace == command.Path {
			directCommands = append(directCommands, textHelpEntry{Name: command.Path, Summary: command.Summary})
		}
	}
	disabledPaths := make(map[string]struct{}, len(disabled))
	for _, command := range disabled {
		disabledPaths[command.Path] = struct{}{}
	}
	for _, command := range complete {
		namespace := commandNamespace(command.Path)
		if namespace == command.Path {
			continue
		}
		index, exists := namespaceIndexes[namespace]
		if !exists {
			index = len(namespaces)
			namespaceIndexes[namespace] = index
			namespaces = append(namespaces, textHelpNamespace{Name: namespace})
		}
		if activeCommand, visible := activeCommands[command.Path]; visible {
			namespaces[index].CommandCount++
			if seenOutcomes[namespace] == nil {
				seenOutcomes[namespace] = make(map[string]struct{})
			}
			if _, duplicate := seenOutcomes[namespace][activeCommand.Summary]; duplicate {
				continue
			}
			seenOutcomes[namespace][activeCommand.Summary] = struct{}{}
			namespaces[index].OutcomeCount++
			if len(namespaces[index].Outcomes) < maxRootHelpOutcomesPerNamespace {
				namespaces[index].Outcomes = append(namespaces[index].Outcomes, textHelpOutcome{
					Summary: activeCommand.Summary,
					Path:    activeCommand.Path,
				})
			}
		} else if _, hidden := disabledPaths[command.Path]; hidden {
			namespaces[index].DisabledCount++
		}
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
	var output bytes.Buffer
	fmt.Fprintln(&output, title)
	width := 0
	for _, namespace := range namespaces {
		if len(namespace.Name) > width {
			width = len(namespace.Name)
		}
	}
	for _, namespace := range namespaces {
		summary := fmt.Sprintf("%d コマンド", namespace.CommandCount)
		switch {
		case namespace.CommandCount == 0 && namespace.DisabledCount > 0:
			summary = "無効なコマンドのみ"
		case namespace.CommandCount == 0:
			summary = "利用できるコマンドなし"
		}
		fmt.Fprintf(&output, "  %-*s  %s\n", width, namespace.Name, summary)
		for _, outcome := range namespace.Outcomes {
			fmt.Fprintf(&output, "    - %s: %s %s --help\n", outcome.Summary, ProgramName, outcome.Path)
		}
		if remaining := namespace.OutcomeCount - len(namespace.Outcomes); remaining > 0 {
			fmt.Fprintf(&output, "    - ほか%d件: %s %s --help\n", remaining, ProgramName, namespace.Name)
		}
	}
	return output.Bytes()
}

func renderNamespaceHelp(selector string, commands []CommandSpec) []byte {
	return renderNamespaceHelpWithDisabled(selector, commands, nil)
}

func renderNamespaceHelpWithDisabled(selector string, commands, disabled []CommandSpec) []byte {
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
	disabledCommands := disabledCommandsForNamespace(disabled, selector)
	if len(disabledCommands) > 0 {
		fmt.Fprintln(&output)
		fmt.Fprintln(&output, "無効なコマンド:")
		for _, command := range disabledCommands {
			fmt.Fprintf(&output, "  %s（無効）\n", command.Path)
		}
	}
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "詳細には、正確なコマンドの末尾に '--help' を付けてください。全体には '%s --help' を実行してください。\n", ProgramName)
	return output.Bytes()
}

func (c *CLI) completeCatalog() Catalog {
	if c != nil && len(c.baseCatalog.commands) > 0 {
		return c.baseCatalog
	}
	if c != nil {
		return c.catalog
	}
	return Catalog{}
}

func (c *CLI) disabledCommandSpecs() []CommandSpec {
	if c == nil || !c.selectionLoaded || !c.selectionSaved {
		return nil
	}
	complete := c.completeCatalog()
	active := make(map[string]struct{}, len(c.catalog.commands))
	for _, command := range c.catalog.commands {
		active[command.Path] = struct{}{}
	}
	disabled := make([]CommandSpec, 0)
	for _, command := range complete.ConfigurableCommands() {
		if _, enabled := active[command.Path]; !enabled {
			disabled = append(disabled, command)
		}
	}
	return disabled
}

func disabledCommandsForNamespace(commands []CommandSpec, namespace string) []CommandSpec {
	disabled := make([]CommandSpec, 0)
	for _, command := range commands {
		if strings.HasPrefix(command.Path, namespace+" ") {
			disabled = append(disabled, command)
		}
	}
	return disabled
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
	for count := len(examples); count >= 0; count-- {
		candidate := renderExactCommandHelp(command, examples[:count])
		if len(candidate) <= maxExactHelpBytes {
			return candidate
		}
	}
	// Return the complete minimum projection when it does not fit so catalog
	// validation can reject it instead of silently dropping invocation facts.
	return renderExactCommandHelp(command, nil)
}

func renderExactCommandHelp(command CommandSpec, examples []commandHelpExample) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, "使い方:")
	fmt.Fprintln(&output, " "+command.Usage())
	if inputs := commandLineHelpInputs(command.Agent.Inputs); len(inputs) > 0 {
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
			fmt.Fprintf(&output, "%s %s %s: %s\n", input.Name, commandInputType(command, input), requirement, input.Description)
		}
	}
	if len(command.HelpConstraints) > 0 {
		fmt.Fprintln(&output)
		fmt.Fprintln(&output, "制約:")
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

func commandHelpExamples(command CommandSpec, allCommands []CommandSpec) []commandHelpExample {
	recipes := relevantHumanHelpRecipes(allCommands, []CommandSpec{command})
	examples := make([]commandHelpExample, 0, maxHelpExamples)
	seen := make(map[string]struct{})
	for _, recipe := range recipes {
		for stepIndex, step := range recipe.Steps {
			if step.Path != command.Path {
				continue
			}
			invocation := ProgramName + " " + step.Path
			if step.Args != "" {
				invocation += " " + step.Args
			}
			title := recipe.Title
			if len(recipe.Steps) > 1 {
				title = fmt.Sprintf("%s（手順 %d/%d）", title, stepIndex+1, len(recipe.Steps))
			}
			key := title + "\x00" + invocation
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			examples = append(examples, commandHelpExample{Title: title, Invocation: invocation})
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
