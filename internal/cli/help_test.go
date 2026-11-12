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
		"Discover test items: cwk items list --help",
		"Read test items: cwk items read --help",
		"Chatwork タスクの成功結果は text",
		"全体全文検索とリアクション集計はありません",
		"cwk <namespace> --help",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("root help lacks %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"--format agent", "機械可読契約"} {
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
	for _, want := range []string{"cwk messages send", "cwk messages reconcile-send", "cwk messages update"} {
		if !strings.Contains(output, want) {
			t.Errorf("messages namespace lacks conditional requirement %q:\n%s", want, output)
		}
	}
	if got := strings.Count(output, "必須(いずれか1つ): --body | --body-file"); got != 3 {
		t.Errorf("messages namespace body-source requirement count=%d:\n%s", got, output)
	}
	if len(output) > 8192 {
		t.Fatalf("messages namespace help is unexpectedly large: %d bytes", len(output))
	}
}

func TestProductionHelpRecipeCoversMeasuredRoomNameEntrypoint(t *testing.T) {
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
}

func TestMultiStepHelpExamplesIdentifyTheirStepAndHideUnavailableSteps(t *testing.T) {
	catalog := helpTestCatalog(t)
	discover, found := catalog.Lookup("items list")
	if !found {
		t.Fatal("catalog lacks items list")
	}
	read, found := catalog.Lookup("items read")
	if !found {
		t.Fatal("catalog lacks items read")
	}

	discoverHelp := string(renderCommandHelp(discover, catalog.Commands()))
	if !strings.Contains(discoverHelp, "項目を見つけて確認する（手順 1/2）: cwk items list --format json") ||
		strings.Contains(discoverHelp, "cwk items read") {
		t.Fatalf("first-step example title or invocation is misleading:\n%s", discoverHelp)
	}
	readHelp := string(renderCommandHelp(read, catalog.Commands()))
	if !strings.Contains(readHelp, "項目を見つけて確認する（手順 2/2）: cwk items read --id <item-id>") ||
		strings.Contains(readHelp, "cwk items list") {
		t.Fatalf("second-step example title or invocation is misleading:\n%s", readHelp)
	}
	for path, output := range map[string]string{"items list": discoverHelp, "items read": readHelp} {
		if len(output) > maxExactHelpBytes {
			t.Errorf("%s exact help exceeds %d bytes: %d", path, maxExactHelpBytes, len(output))
		}
	}

	partial := string(renderCommandHelp(discover, []CommandSpec{discover}))
	if strings.Contains(partial, "項目を見つけて確認する") || strings.Contains(partial, "cwk items read") {
		t.Fatalf("partial active view exposed a recipe with an unavailable step:\n%s", partial)
	}
}

func TestProductionMemberSearchHelpLabelsOnlyItsActiveRecipeStep(t *testing.T) {
	catalog := DefaultCatalog()
	find, found := catalog.Lookup("members find")
	if !found {
		t.Fatal("members find is absent")
	}
	findHelp := string(renderCommandHelp(find, catalog.Commands()))
	if !strings.Contains(findHelp, "人物の発言を期間で追う（手順 1/2）: cwk members find --room <room-id> --query <name>") ||
		strings.Contains(findHelp, "cwk messages list") {
		t.Fatalf("members find help does not accurately identify its recipe step:\n%s", findHelp)
	}
	if len(findHelp) > maxExactHelpBytes {
		t.Fatalf("members find exact help exceeds %d bytes: %d", maxExactHelpBytes, len(findHelp))
	}
	active := make([]CommandSpec, 0, len(catalog.Commands())-1)
	for _, command := range catalog.Commands() {
		if command.Path != "messages list" {
			active = append(active, command)
		}
	}
	partial := string(renderCommandHelp(find, active))
	if strings.Contains(partial, "人物の発言を期間で追う") || strings.Contains(partial, "cwk messages list") {
		t.Fatalf("members find help exposed a recipe with a disabled step:\n%s", partial)
	}
}

func TestExactHelpUsesCompleteCatalogInputFactsAndConstraints(t *testing.T) {
	command, found := DefaultCatalog().Lookup("messages list")
	if !found {
		t.Fatal("default catalog lacks messages list")
	}
	output := string(renderCommandHelp(command, DefaultCatalog().Commands()))
	for _, want := range []string{
		command.Usage(),
		"--room room-id 必須",
		"--room room-id 必須: rooms list の room-ref をそのまま（正の10進数、1〜32桁、先頭0なし）。",
		"--window recent|changes 任意",
		"--start-index index 任意",
		"--sender account-ref 任意・繰り返し可: members find の account-ref をそのまま。完全一致、最大100回、反復OR。",
		"--context none|replies 任意",
		"--resolve-relations count 任意",
		"--jst boolean 任意: +09:00日時を併記。",
		"既定5",
		"制約:",
		"--onと--since/--untilは排他",
		"--containsは取得済み範囲でローカル照合（API検索なし）。",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("exact help lacks %q:\n%s", want, output)
		}
	}
	if len(output) > maxExactHelpBytes {
		t.Fatalf("messages list help exceeds the exact-help budget: %d bytes:\n%s", len(output), output)
	}
	if !strings.Contains(output, "完全一致") || !strings.Contains(output, "最大100回") || !strings.Contains(output, "反復OR") {
		t.Fatalf("messages list help weakened the exact sender predicate:\n%s", output)
	}
	for _, forbidden := range []string{"機能ID:", "生成する参照:", "使用する参照:", "機械可読契約", "--format agent"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("exact help retained %q:\n%s", forbidden, output)
		}
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
			for _, implementationTerm := range []string{
				"完全一致するルーム", "完全一致のルーム", "完全一致ルーム", "正規参照", "正規の room_ref", "上限付き",
			} {
				if strings.Contains(command.Summary, implementationTerm) || strings.Contains(string(output), implementationTerm) {
					t.Errorf("%s public help uses implementation-oriented wording %q:\nsummary: %s\n%s", command.Path, implementationTerm, command.Summary, output)
				}
			}
			for _, input := range command.Agent.Inputs {
				if input.Source != InputSourceFlag {
					continue
				}
				needle := "\n" + input.Name + " "
				if !bytes.Contains(output, []byte(needle)) {
					t.Errorf("exact help lacks input %q:\n%s", input.Name, output)
				}
				if !helpHasCompleteInputDescription(string(output), input) {
					t.Errorf("exact help does not preserve the complete description for %q: %q\n%s", input.Name, input.Description, output)
				}
			}
		})
	}
}

func TestExactHelpDropsOptionalExamplesBeforeDescriptions(t *testing.T) {
	target := utilitySpec("items inspect")
	target.Args = "[--detail <text>]"
	target.Agent.Inputs = []CommandInput{{
		Name: "--detail", Source: InputSourceFlag, Required: false,
		Description: strings.Repeat("d", 1800), AllowedValues: []string{},
	}}
	first := utilitySpec("items first")
	first.HumanRecipes = []HumanHelpRecipe{{
		Title: "first example",
		Steps: []HumanHelpRecipeStep{{Path: target.Path, Args: strings.Repeat("x", 100)}},
	}, {
		Title: "second example",
		Steps: []HumanHelpRecipeStep{{Path: target.Path, Args: strings.Repeat("y", 100)}},
	}}
	output := string(renderCommandHelp(target, []CommandSpec{first, target}))
	if len(output) > maxExactHelpBytes {
		t.Fatalf("exact help is %d bytes, maximum is %d", len(output), maxExactHelpBytes)
	}
	if !helpHasCompleteInputDescription(output, target.Agent.Inputs[0]) {
		t.Fatalf("exact help cut the input description:\n%s", output)
	}
	if !strings.Contains(output, "first example: cwk items inspect "+strings.Repeat("x", 100)) {
		t.Fatalf("exact help dropped the first example before the second:\n%s", output)
	}
	if strings.Contains(output, "second example") {
		t.Fatalf("exact help retained the second example after the byte budget was exceeded:\n%s", output)
	}
}

func helpHasCompleteInputDescription(output string, input CommandInput) bool {
	prefix := input.Name + " "
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.HasSuffix(line, ": "+input.Description)
		}
	}
	return false
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
	if len(direct) != 2 || len(namespaces) != 1 || namespaces[0].Name != "items" || namespaces[0].CommandCount != 100 || namespaces[0].OutcomeCount != 1 {
		t.Fatalf("root entries direct=%+v namespaces=%+v", direct, namespaces)
	}
	var command CLI
	command.catalog = NewCatalog(base...)
	output := string(command.renderRootHelp())
	if strings.Count(output, "cwk items command") != 1 || strings.Contains(output, "ほか") {
		t.Fatalf("duplicate leaf outcomes were not collapsed:\n%s", output)
	}
}

func TestRootHelpIndexesThreeUniqueOutcomesAndRoutesRemainingToNamespace(t *testing.T) {
	help, found := DefaultCatalog().Lookup("help")
	if !found {
		t.Fatal("default catalog lacks help")
	}
	commands := []CommandSpec{help}
	for _, item := range []struct{ path, summary string }{
		{"items first", "重複する成果"},
		{"items second", "重複する成果"},
		{"items third", "三番目の成果"},
		{"items fourth", "四番目の成果"},
		{"items fifth", "五番目の成果"},
		{"items sixth", "六番目の成果"},
	} {
		command := utilitySpec(item.path)
		command.Summary = item.summary
		commands = append(commands, command)
	}
	catalog := NewCatalog(commands...)
	if err := catalog.Validate(); err != nil {
		t.Fatalf("synthetic outcome catalog should remain valid: %v", err)
	}
	output := string((&CLI{catalog: catalog}).renderRootHelp())
	for _, want := range []string{
		"items  6 コマンド",
		"重複する成果: cwk items first --help",
		"三番目の成果: cwk items third --help",
		"四番目の成果: cwk items fourth --help",
		"ほか2件: cwk items --help",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("root outcome index lacks %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"items second --help", "items fifth --help", "items sixth --help", "五番目の成果", "六番目の成果"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("root outcome index exceeded its bounded projection with %q:\n%s", forbidden, output)
		}
	}
	if count := strings.Count(output, "重複する成果"); count != 1 {
		t.Fatalf("duplicate outcome was rendered %d times:\n%s", count, output)
	}
}

func TestRootHelpLinkReadingGuideUsesOnlyAnActiveExactCatalogPath(t *testing.T) {
	help, found := DefaultCatalog().Lookup("help")
	if !found {
		t.Fatal("default catalog lacks help")
	}
	list := utilitySpec("messages list")
	list.Configurable = true
	show := utilitySpec("messages show")
	show.Configurable = true
	complete := NewCatalog(help, list, show)
	active := NewCatalog(help, list)
	configuredOutput := string((&CLI{
		baseCatalog: complete, catalog: active, selectionLoaded: true, selectionSaved: true,
	}).renderRootHelp())
	if strings.Contains(configuredOutput, "参照から発言を読む") || strings.Contains(configuredOutput, "messages show") {
		t.Fatalf("disabled messages show leaked into the active root index:\n%s", configuredOutput)
	}

	showOnly := NewCatalog(help, show)
	activeOutput := string((&CLI{catalog: showOnly}).renderRootHelp())
	if !strings.Contains(activeOutput, "参照から発言を読む: 'cwk messages show --help'") {
		t.Fatalf("root help lacks the catalog-backed link-reading guide:\n%s", activeOutput)
	}
	if strings.Contains(activeOutput, "--context replies") || strings.Contains(activeOutput, "--around") {
		t.Fatalf("link-reading guide appended undeclared invocation arguments:\n%s", activeOutput)
	}
}

func TestRootHelpReportsFullAndStaticFallbackByteCounts(t *testing.T) {
	catalog := DefaultCatalog()
	output := (&CLI{catalog: catalog}).renderRootHelp()
	withNotice := prependStaticHelpNotice(output)
	if len(output) == 0 || len(withNotice) != len(staticHelpNotice)+len(output) {
		t.Fatalf("unexpected root-help byte counts: normal=%d fallback=%d notice=%d", len(output), len(withNotice), len(staticHelpNotice))
	}
	for _, route := range []string{"cwk messages list --help", "cwk messages show --help"} {
		if !strings.Contains(string(output), route) || !strings.Contains(string(withNotice), route) {
			t.Errorf("complete catalog root/fallback help lacks exact route %q", route)
		}
	}
	for _, fullHelp := range []string{string(output), string(withNotice)} {
		if !strings.Contains(fullHelp, "全体全文検索とリアクション集計はありません。") ||
			!strings.Contains(fullHelp, "代替は 'cwk messages list --help' を確認してください。") {
			t.Errorf("active full-catalog help lacks the search limitation or its catalog-backed alternative:\n%s", fullHelp)
		}
	}
	t.Logf("root help bytes: full=%d static-fallback=%d notice=%d", len(output), len(withNotice), len(staticHelpNotice))
}

func TestRootSearchAlternativeIsHiddenWhenMessagesListIsSavedDisabled(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	saveCommandSelection(t, h.store, []string{})
	if code := runCLI(h.command, []string{"help"}); code != ExitOK {
		t.Fatalf("root help exit=%d stderr=%q", code, h.stderr.String())
	}
	output := h.stdout.String()
	if !strings.Contains(output, "全体全文検索とリアクション集計はありません。") {
		t.Fatalf("root help lost the full-search limitation:\n%s", output)
	}
	if strings.Contains(output, "代替は 'cwk messages list --help'") {
		t.Fatalf("root help suggested disabled messages list:\n%s", output)
	}
	hasDisabledMessagesNamespace := false
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "messages") && strings.Contains(line, "無効なコマンドのみ") {
			hasDisabledMessagesNamespace = true
			break
		}
	}
	if !hasDisabledMessagesNamespace || strings.Contains(output, "messages list") {
		t.Fatalf("saved disabled namespace was not name-only:\n%s", output)
	}
}

func TestRootSearchAlternativeUsesExactActiveMessagesListPath(t *testing.T) {
	help, found := DefaultCatalog().Lookup("help")
	if !found {
		t.Fatal("default catalog lacks help")
	}
	list := utilitySpec("messages list")
	list.Configurable = true
	show := utilitySpec("messages show")
	show.Configurable = true
	complete := NewCatalog(help, list, show)
	active := NewCatalog(help, show)
	command := &CLI{baseCatalog: complete, catalog: active, selectionLoaded: true, selectionSaved: true}
	output := string(command.renderRootHelp())
	if !strings.Contains(output, "参照から発言を読む: 'cwk messages show --help'") {
		t.Fatalf("active messages show guide is missing:\n%s", output)
	}
	if !strings.Contains(output, "全体全文検索とリアクション集計はありません。") ||
		strings.Contains(output, "代替は 'cwk messages list --help'") {
		t.Fatalf("root search alternative did not follow the exact active messages list path:\n%s", output)
	}
	if strings.Contains(output, "messages list") {
		t.Fatalf("disabled messages list details leaked into root help:\n%s", output)
	}
}

func TestRootHelpHidesInactiveAndDisabledOutcomeMetadata(t *testing.T) {
	help, found := DefaultCatalog().Lookup("help")
	if !found {
		t.Fatal("default catalog lacks help")
	}
	active := utilitySpec("active list")
	active.Summary = "有効な成果"
	active.Configurable = true
	disabled := utilitySpec("inactive show")
	disabled.Summary = "無効な詳細"
	disabled.Configurable = true
	disabledOnly := utilitySpec("hidden-only list")
	disabledOnly.Summary = "非表示の成果"
	disabledOnly.Configurable = true
	complete := []CommandSpec{help, active, disabled, disabledOnly}
	visible := []CommandSpec{help, active}
	hidden := []CommandSpec{disabled, disabledOnly}
	_, namespaces := rootTextHelpEntriesForView(complete, visible, hidden)
	output := string(renderNamespaceIndex("名前空間:", namespaces))
	if !strings.Contains(output, "有効な成果: cwk active list --help") || strings.Contains(output, "無効な詳細") || strings.Contains(output, "非表示の成果") {
		t.Fatalf("root namespace index exposed non-active outcome metadata:\n%s", output)
	}
	if !strings.Contains(output, "hidden-only  無効なコマンドのみ") || strings.Contains(output, "hidden-only list") {
		t.Fatalf("disabled-only namespace did not remain name-only:\n%s", output)
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
