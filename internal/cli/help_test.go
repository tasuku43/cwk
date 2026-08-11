package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func helpTestCatalog(t *testing.T) Catalog {
	t.Helper()
	help, found := DefaultCatalog().Lookup("help")
	if !found {
		t.Fatal("default catalog lacks help")
	}
	discover := discoverSpec("items list", "item")
	discover.Args = "[--format tsv|json]"
	discover.Agent.Inputs = []CommandInput{{
		Name: "--format", Source: InputSourceFlag, Required: false,
		Description: "Select the result format.", AllowedValues: []string{"tsv", "json"},
	}}
	read := actSpec("items read", "item", "--id")
	read.HumanRecipes = []HumanHelpRecipe{{
		Title: "項目を見つけて確認する",
		Steps: []HumanHelpRecipeStep{
			{Path: "items list", Args: "--format json"},
			{Path: "items read", Args: "--id <item-id>"},
		},
	}}
	read.HelpConstraints = []string{"--id は items list が返した値を変更せず指定します。"}
	catalog := NewCatalog(help, discover, read)
	if err := catalog.Validate(); err != nil {
		t.Fatalf("test catalog is invalid: %v", err)
	}
	return catalog
}

func runHelpWithCatalog(t *testing.T, catalog Catalog, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader(""), &stdout, &stderr, catalog, nil)
	code := runCLI(command, args)
	return code, stdout.String(), stderr.String()
}

func TestRootTextHelpIsACatalogDerivedNamespaceIndex(t *testing.T) {
	catalog := helpTestCatalog(t)
	code, output, stderr := runHelpWithCatalog(t, catalog, "help")
	if code != ExitOK || stderr != "" {
		t.Fatalf("help exit=%d stderr=%q", code, stderr)
	}
	for _, want := range []string{
		"Chatwork CLI\n",
		"cwk [--error-format text|json] <command> [arguments]",
		"コマンド:\n  help",
		"名前空間:\n  items  2 コマンド",
		"Chatwork タスクの成功結果は text",
		"全体全文検索とリアクション集計はありません",
		"cwk <namespace> --help",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("root help lacks %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"items list", "items read", "--format agent", "機械可読契約"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("root help retained %q:\n%s", forbidden, output)
		}
	}
}

func TestHelpAgentFormatIsExplicitlyRetired(t *testing.T) {
	catalog := helpTestCatalog(t)
	for _, args := range [][]string{
		{"help", "--format", "agent"},
		{"help", "--format=agent"},
		{"help", "items", "--format", "agent"},
		{"help", "items", "read", "--format=agent"},
	} {
		code, stdout, stderr := runHelpWithCatalog(t, catalog, args...)
		if code != ExitUsage {
			t.Errorf("Run(%v) exit=%d, want %d", args, code, ExitUsage)
		}
		if stdout != "" || !strings.Contains(stderr, "code: invalid_arguments") ||
			!strings.Contains(stderr, "agent 形式は廃止済み") || !strings.Contains(stderr, "--help") {
			t.Errorf("Run(%v) stdout=%q stderr=%q", args, stdout, stderr)
		}
	}
}

func TestHelpCatalogIsTextOnly(t *testing.T) {
	help, found := DefaultCatalog().Lookup("help")
	if !found {
		t.Fatal("default catalog lacks help")
	}
	if help.Args != "[command]" || !reflect.DeepEqual(help.Agent.Output.Formats, []OutputFormat{OutputFormatText}) ||
		help.Agent.Output.DefaultFormat != OutputFormatText || help.Agent.Output.JSONEnvelope != "" || help.Agent.Output.JSONSchemaVersion != 0 {
		t.Fatalf("help contract is not text-only: args=%q output=%+v", help.Args, help.Agent.Output)
	}
	for _, input := range help.Agent.Inputs {
		if input.Name == "--format" {
			t.Fatalf("help retained public format input: %+v", input)
		}
	}
}

func TestNamespaceHelpPublishesEveryLeafUsageAndRequiredFlags(t *testing.T) {
	catalog := helpTestCatalog(t)
	code, output, stderr := runHelpWithCatalog(t, catalog, "items", "--help")
	if code != ExitOK || stderr != "" {
		t.Fatalf("namespace help exit=%d stderr=%q", code, stderr)
	}
	for _, want := range []string{
		"名前空間: items",
		"cwk items list [--format tsv|json]",
		"cwk items read --id <item-id>",
		"必須: --id",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("namespace help lacks %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "--format agent") {
		t.Fatalf("namespace help retained agent guidance:\n%s", output)
	}

	code2, canonical, canonicalErr := runHelpWithCatalog(t, catalog, "help", "items")
	if code2 != code || canonical != output || canonicalErr != stderr {
		t.Fatalf("trailing namespace help differs: trailing=(%d,%q,%q) canonical=(%d,%q,%q)", code, output, stderr, code2, canonical, canonicalErr)
	}
}

func TestProductionMessagesNamespaceHelpIsOneStepActionable(t *testing.T) {
	commands, exact := DefaultCatalog().Select("messages")
	if exact || len(commands) == 0 {
		t.Fatalf("messages selection exact=%t commands=%d", exact, len(commands))
	}
	output := string(renderNamespaceHelp("messages", commands))
	for _, command := range commands {
		if !strings.Contains(output, command.Usage()) {
			t.Errorf("messages namespace lacks usage %q:\n%s", command.Usage(), output)
		}
		for _, input := range command.Agent.Inputs {
			if input.Required && input.Source == InputSourceFlag && !strings.Contains(output, input.Name) {
				t.Errorf("messages namespace lacks required flag %q for %q", input.Name, command.Path)
			}
		}
	}
	for _, want := range []string{"cwk messages send", "cwk messages update"} {
		if !strings.Contains(output, want) {
			t.Errorf("messages namespace lacks conditional requirement %q:\n%s", want, output)
		}
	}
	if got := strings.Count(output, "必須(いずれか1つ): --body | --body-file"); got != 2 {
		t.Errorf("messages namespace body-source requirement count=%d:\n%s", got, output)
	}
	if len(output) > 8192 {
		t.Fatalf("messages namespace help is unexpectedly large: %d bytes", len(output))
	}
}

func TestProductionHelpRecipesCoverMeasuredInvestigationEntrypoints(t *testing.T) {
	catalog := DefaultCatalog()
	rooms, found := catalog.Lookup("rooms list")
	if !found {
		t.Fatal("rooms list is absent")
	}
	roomsHelp := string(renderCommandHelp(rooms, catalog.Commands()))
	for _, want := range []string{"ルーム名しか分からない状態から入る", "cwk rooms list --query <room-name>"} {
		if !strings.Contains(roomsHelp, want) {
			t.Errorf("rooms list help lacks %q:\n%s", want, roomsHelp)
		}
	}

	messages, found := catalog.Lookup("messages list")
	if !found {
		t.Fatal("messages list is absent")
	}
	messagesHelp := string(renderCommandHelp(messages, catalog.Commands()))
	for _, want := range []string{
		"今日または昨日の会話を確認する", "--on today",
		"人物の発言を期間で追う", "--sender <account-ref>", "--since 2026-08-01T00:00:00+09:00", "--until 2026-08-08T00:00:00+09:00",
	} {
		if !strings.Contains(messagesHelp, want) {
			t.Errorf("messages list help lacks %q:\n%s", want, messagesHelp)
		}
	}
	if len(messagesHelp) > maxExactHelpBytes {
		t.Fatalf("messages list help bytes=%d", len(messagesHelp))
	}
}

func TestExactHelpUsesCompactCatalogInputFactsConstraintsAndExamples(t *testing.T) {
	command, found := DefaultCatalog().Lookup("messages list")
	if !found {
		t.Fatal("default catalog lacks messages list")
	}
	output := string(renderCommandHelp(command, DefaultCatalog().Commands()))
	for _, want := range []string{
		command.Usage(),
		"--room (room-id, 必須)",
		"--window (recent|changes, 任意)",
		"--start-index (index, 任意)",
		"--sender (account-ref, 任意・繰り返し可)",
		"--context (none|replies, 任意)  選択時の既定値は replies",
		"--resolve-relations (count, 任意)",
		"既定値は5",
		"併用制約:",
		"--on は --since または --until と併用できません",
		"例:",
		"人物の発言を期間で追う: cwk messages list --room <room-id> --sender <account-ref> --since 2026-08-01T00:00:00+09:00 --until 2026-08-08T00:00:00+09:00",
		"今日または昨日の会話を確認する: cwk messages list --room <room-id> --on today",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("exact help lacks %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"機能ID:", "生成する参照:", "使用する参照:", "機械可読契約", "--format agent"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("exact help retained %q:\n%s", forbidden, output)
		}
	}
	if got := strings.Count(output, "cwk messages list --room <room-id>"); got != 3 {
		// One usage plus exactly two examples.
		t.Fatalf("messages list invocation count=%d, want 3:\n%s", got, output)
	}
}

func TestEveryDefaultExactHelpFitsTwoKiBAndProjectsEveryFlag(t *testing.T) {
	catalog := DefaultCatalog()
	if err := catalog.Validate(); err != nil {
		t.Fatalf("default catalog is invalid: %v", err)
	}
	for _, command := range catalog.Commands() {
		command := command
		t.Run(strings.ReplaceAll(command.Path, " ", "_"), func(t *testing.T) {
			output := renderCommandHelp(command, catalog.Commands())
			if len(output) > maxExactHelpBytes {
				t.Fatalf("exact help is %d bytes, maximum is %d:\n%s", len(output), maxExactHelpBytes, output)
			}
			if !utf8Lines(output) {
				t.Fatalf("exact help is not valid one-line UTF-8 input projection: %q", output)
			}
			if !bytes.Contains(output, []byte(command.Usage())) {
				t.Fatalf("exact help lacks usage %q:\n%s", command.Usage(), output)
			}
			for _, input := range command.Agent.Inputs {
				if input.Source != InputSourceFlag {
					continue
				}
				needle := "  " + input.Name + " ("
				if !bytes.Contains(output, []byte(needle)) {
					t.Errorf("exact help lacks input %q:\n%s", input.Name, output)
				}
			}
		})
	}
}

func utf8Lines(output []byte) bool {
	if !utf8.Valid(output) {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.ContainsAny(line, "\r\u2028\u2029") {
			return false
		}
	}
	return true
}

func TestExactHelpOmitsRecipesWithUnavailableSteps(t *testing.T) {
	catalog := helpTestCatalog(t)
	read, found := catalog.Lookup("items read")
	if !found {
		t.Fatal("catalog lacks items read")
	}
	full := string(renderCommandHelp(read, catalog.Commands()))
	if !strings.Contains(full, "項目を見つけて確認する") {
		t.Fatalf("complete view lacks recipe:\n%s", full)
	}

	partial := NewCatalog(read)
	withoutProducer := string(renderCommandHelp(read, partial.Commands()))
	if strings.Contains(withoutProducer, "項目を見つけて確認する") {
		t.Fatalf("partial view retained unavailable recipe:\n%s", withoutProducer)
	}
}

func TestCatalogSelectReturnsDeepCopiesForHelpMetadata(t *testing.T) {
	catalog := helpTestCatalog(t)
	commands, exact := catalog.Select("items")
	if exact || len(commands) != 2 {
		t.Fatalf("Select(items) exact=%t commands=%d", exact, len(commands))
	}
	commands[1].Agent.Inputs[0].AllowedValues = append(commands[1].Agent.Inputs[0].AllowedValues, "mutated")
	commands[1].HumanRecipes[0].Steps[0].Args = "mutated"
	commands[1].HelpConstraints[0] = "mutated"
	original, found := catalog.Lookup("items read")
	if !found {
		t.Fatal("catalog lacks items read")
	}
	if len(original.Agent.Inputs[0].AllowedValues) != 0 || original.HumanRecipes[0].Steps[0].Args == "mutated" || original.HelpConstraints[0] == "mutated" {
		t.Fatalf("scoped projection mutated catalog: %+v", original)
	}
}

func TestNestedNamespaceHelpUsesWordBoundariesAndFullUsages(t *testing.T) {
	help, _ := DefaultCatalog().Lookup("help")
	first := utilitySpec("area group first")
	second := utilitySpec("area group second")
	sibling := utilitySpec("area other")
	catalog := NewCatalog(help, first, second, sibling)
	if err := catalog.Validate(); err != nil {
		t.Fatal(err)
	}
	code, output, stderr := runHelpWithCatalog(t, catalog, "area", "group", "--help")
	if code != ExitOK || stderr != "" {
		t.Fatalf("nested namespace help exit=%d stderr=%q", code, stderr)
	}
	for _, want := range []string{"cwk area group first", "cwk area group second"} {
		if !strings.Contains(output, want) {
			t.Errorf("nested namespace help lacks %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "area other") {
		t.Fatalf("nested namespace leaked sibling:\n%s", output)
	}
}

func TestHumanRootHelpGrowthDependsOnNamespacesNotLeafCommands(t *testing.T) {
	base := []CommandSpec{utilitySpec("help"), utilitySpec("doctor")}
	for index := 0; index < 100; index++ {
		base = append(base, utilitySpec("items command"+strings.Repeat("x", index+1)))
	}
	direct, namespaces := rootTextHelpEntries(base)
	if len(direct) != 2 || len(namespaces) != 1 || namespaces[0].Name != "items" || namespaces[0].CommandCount != 100 {
		t.Fatalf("root entries direct=%+v namespaces=%+v", direct, namespaces)
	}
	var command CLI
	command.catalog = NewCatalog(base...)
	output := string(command.renderRootHelp())
	if strings.Contains(output, "items command") || strings.Count(output, "items") != 1 {
		t.Fatalf("root help grew with leaves:\n%s", output)
	}
}

func TestDefaultHumanRecipeStepsResolveToExactCatalogCommands(t *testing.T) {
	catalog := DefaultCatalog()
	for _, command := range catalog.Commands() {
		for _, recipe := range command.HumanRecipes {
			if len(recipe.Steps) == 0 {
				t.Fatalf("recipe %q has no steps", recipe.Title)
			}
			for _, step := range recipe.Steps {
				if _, found := catalog.Lookup(step.Path); !found {
					t.Errorf("recipe %q step %q is not an exact command", recipe.Title, step.Path)
				}
			}
		}
	}
}

func TestHumanRootHelpGroupsKindsAndPreservesSectionRelativeCatalogOrder(t *testing.T) {
	commands := []CommandSpec{
		utilitySpec("zeta first"),
		utilitySpec("doctor"),
		utilitySpec("alpha one"),
		utilitySpec("version"),
		utilitySpec("zeta second"),
		utilitySpec("alpha two"),
	}
	direct, namespaces := rootTextHelpEntries(commands)
	if got, want := []string{direct[0].Name, direct[1].Name}, []string{"doctor", "version"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("direct order=%v want=%v", got, want)
	}
	if got, want := []string{namespaces[0].Name, namespaces[1].Name}, []string{"zeta", "alpha"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("namespace order=%v want=%v", got, want)
	}
}

func TestHelpRejectsUnknownSelectorsAndFormats(t *testing.T) {
	catalog := helpTestCatalog(t)
	for _, args := range [][]string{
		{"help", "missing"},
		{"help", "--format", "yaml"},
		{"help", "--format=text"},
		{"help", "--unknown"},
	} {
		code, stdout, stderr := runHelpWithCatalog(t, catalog, args...)
		if code != ExitUsage || stdout != "" || !strings.Contains(stderr, "error:") {
			t.Errorf("Run(%v) exit=%d stdout=%q stderr=%q", args, code, stdout, stderr)
		}
	}
}

func TestUnknownHelpSelectorRecoveryNamesPathsAndNamespaces(t *testing.T) {
	catalog := helpTestCatalog(t)
	code, stdout, stderr := runHelpWithCatalog(t, catalog, "help", "does-not-exist")
	if code != ExitUsage || stdout != "" ||
		!strings.Contains(stderr, "next_action: cwk help — ルートヘルプにあるコマンドパスまたは名前空間を指定し") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
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
