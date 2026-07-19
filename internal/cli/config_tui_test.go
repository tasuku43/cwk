package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/operation"
)

var configTUITestAlwaysPaths = []string{"doctor", "help", "version", "config"}

func TestConfigTUIModelPreservesCatalogOrderStopsAtEdgesAndTogglesCurrentItem(t *testing.T) {
	model := newConfigTUIModel([]configTUIItem{
		{Path: "rooms list", Summary: "rooms", Effect: operation.EffectRead, Enabled: true},
		{Path: "account show", Summary: "account", Effect: operation.EffectRead},
		{Path: "messages list", Summary: "messages", Effect: operation.EffectRead, Enabled: true},
	}, configTUITestAlwaysPaths)
	if got, want := []string{model.items[0].Path, model.items[1].Path, model.items[2].Path}, []string{"rooms list", "account show", "messages list"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog-order paths = %v, want %v", got, want)
	}

	model.apply(configTUIKeyUp)
	if model.cursor != 0 {
		t.Fatalf("cursor moved above first item: %d", model.cursor)
	}
	for range 5 {
		model.apply(configTUIKeyDown)
	}
	if model.cursor != 2 {
		t.Fatalf("cursor moved below last item: %d", model.cursor)
	}
	model.apply(configTUIKeyToggle)
	if model.items[2].Enabled {
		t.Fatal("Space did not toggle only the current item")
	}
	if got, want := model.enabledPaths(), []string{"rooms list"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("enabled paths = %v, want %v", got, want)
	}
	if decision := model.apply(configTUIKeySave); decision != configTUISave {
		t.Fatalf("save decision = %v", decision)
	}
	if decision := model.apply(configTUIKeyCancel); decision != configTUICancel {
		t.Fatalf("cancel decision = %v", decision)
	}
	if decision := model.apply(configTUIKeyInterrupt); decision != configTUIInterrupted {
		t.Fatalf("interrupt decision = %v", decision)
	}
}

func TestConfigTUIModelCopiesCallerItemsAndHandlesEmptySelection(t *testing.T) {
	items := []configTUIItem{{Path: "rooms list", Effect: operation.EffectRead, Enabled: true}}
	always := append([]string(nil), configTUITestAlwaysPaths...)
	model := newConfigTUIModel(items, always)
	items[0].Path = "mutated"
	always[0] = "mutated"
	model.apply(configTUIKeyToggle)
	if model.items[0].Path != "rooms list" || model.items[0].Enabled || model.alwaysPaths[0] != "doctor" {
		t.Fatalf("model aliases caller state: items=%+v always=%v", model.items, model.alwaysPaths)
	}

	empty := newConfigTUIModel(nil, nil)
	empty.apply(configTUIKeyUp)
	empty.apply(configTUIKeyDown)
	empty.apply(configTUIKeyToggle)
	if empty.cursor != 0 || len(empty.enabledPaths()) != 0 {
		t.Fatalf("empty model changed: %+v", empty)
	}
}

func TestConfigTUIKeyParserHandlesFragmentedCSIAndSS3Arrows(t *testing.T) {
	var parser configTUIKeyParser
	if keys := parser.feed([]byte{0x1b}); len(keys) != 0 {
		t.Fatalf("fragmented Escape yielded keys: %v", keys)
	}
	if keys := parser.feed([]byte{'['}); len(keys) != 0 {
		t.Fatalf("fragmented CSI yielded keys: %v", keys)
	}
	if keys := parser.feed([]byte{'A'}); !reflect.DeepEqual(keys, []configTUIKey{configTUIKeyUp}) {
		t.Fatalf("CSI Up = %v", keys)
	}

	if keys := parser.feed([]byte{0x1b, 'O'}); len(keys) != 0 {
		t.Fatalf("fragmented SS3 yielded keys: %v", keys)
	}
	if keys := parser.feed([]byte{'B'}); !reflect.DeepEqual(keys, []configTUIKey{configTUIKeyDown}) {
		t.Fatalf("SS3 Down = %v", keys)
	}

	if keys := parser.feed([]byte{0x1b, '[', '1', ';', '5', 'A'}); !reflect.DeepEqual(keys, []configTUIKey{configTUIKeyUp}) {
		t.Fatalf("parameterized fragmented CSI Up = %v", keys)
	}
}

func TestConfigTUIKeyParserFlushesLoneOrIncompleteEscapeAsCancel(t *testing.T) {
	tests := [][]byte{{0x1b}, {0x1b, '['}, {0x1b, 'O'}}
	for _, input := range tests {
		var parser configTUIKeyParser
		if keys := parser.feed(input); len(keys) != 0 {
			t.Fatalf("feed(%v) = %v, want pending", input, keys)
		}
		if keys := parser.flushEscape(); !reflect.DeepEqual(keys, []configTUIKey{configTUIKeyCancel}) {
			t.Fatalf("flushEscape(%v) = %v", input, keys)
		}
		if keys := parser.flushEscape(); len(keys) != 0 {
			t.Fatalf("second flushEscape(%v) = %v", input, keys)
		}
	}
}

func TestConfigTUIKeyParserNormalizesCRLFAndRecognizesSelectorKeys(t *testing.T) {
	var parser configTUIKeyParser
	keys := parser.feed([]byte{' ', '\r', '\n', 'q', 0x03, 'x'})
	want := []configTUIKey{configTUIKeyToggle, configTUIKeySave, configTUIKeyCancel, configTUIKeyInterrupt}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}

	var fragmented configTUIKeyParser
	if keys := fragmented.feed([]byte{'\r'}); !reflect.DeepEqual(keys, []configTUIKey{configTUIKeySave}) {
		t.Fatalf("CR = %v", keys)
	}
	if keys := fragmented.feed([]byte{'\n', ' '}); !reflect.DeepEqual(keys, []configTUIKey{configTUIKeyToggle}) {
		t.Fatalf("fragmented LF + Space = %v", keys)
	}
}

func TestConfigTUIKeyParserRecognizesFragmentedFullWidthSpace(t *testing.T) {
	var parser configTUIKeyParser
	encoded := []byte("　")
	for index, value := range encoded {
		keys := parser.feed([]byte{value})
		if index+1 < len(encoded) {
			if len(keys) != 0 {
				t.Fatalf("fragment %d yielded keys: %v", index, keys)
			}
			continue
		}
		if !reflect.DeepEqual(keys, []configTUIKey{configTUIKeyToggle}) {
			t.Fatalf("full-width Space = %v, want toggle", keys)
		}
	}

	// A different three-byte rune must not consume the following ASCII Space
	// or fabricate an additional toggle.
	if keys := parser.feed([]byte("あ ")); !reflect.DeepEqual(keys, []configTUIKey{configTUIKeyToggle}) {
		t.Fatalf("unrelated UTF-8 plus Space = %v, want one toggle", keys)
	}
}

func TestConfigTUIKeyParserReprocessesByteAfterNonSequenceEscape(t *testing.T) {
	var parser configTUIKeyParser
	keys := parser.feed([]byte{0x1b, ' '})
	want := []configTUIKey{configTUIKeyCancel, configTUIKeyToggle}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("Escape + Space = %v, want %v", keys, want)
	}
}

func TestRenderConfigTUIScreenIsBoundedAndKeepsCursorInViewport(t *testing.T) {
	items := make([]configTUIItem, 10)
	for index := range items {
		items[index] = configTUIItem{Path: "command " + string(rune('a'+index)), Summary: "summary", Effect: operation.EffectRead, Enabled: index%2 == 0}
	}
	model := newConfigTUIModel(items, configTUITestAlwaysPaths)
	model.cursor = 8
	const width, height = 80, 7
	output := renderConfigTUIScreen(model, width, height)
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != height {
		t.Fatalf("rendered lines = %d, want %d:\n%s", len(lines), height, output)
	}
	for index, line := range lines {
		if got := configTUIDisplayWidth(line); got > width {
			t.Fatalf("line %d width = %d, want <= %d: %q", index, got, width, line)
		}
	}
	if !strings.Contains(output, "> [x] \x1b[36m[read]\x1b[0m   command i") {
		t.Fatalf("cursor is outside viewport:\n%s", output)
	}
	if strings.Count(output, "Always on: doctor, help, version, config") != 1 {
		t.Fatalf("always-on line missing or repeated:\n%s", output)
	}
	if strings.Count(output, configTUIFooter) != 1 {
		t.Fatalf("footer missing or repeated:\n%s", output)
	}
	for _, forbidden := range []string{"purpose=", "security-boundary=", "source="} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("legacy header %q leaked:\n%s", forbidden, output)
		}
	}
}

func TestRenderConfigTUIAlwaysUsesProvidedOrderAndEscapesStructure(t *testing.T) {
	model := newConfigTUIModel([]configTUIItem{{Path: "rooms list", Effect: operation.EffectRead}}, []string{"version", "doctor\nunsafe", "config"})
	output := renderConfigTUIScreen(model, 100, 4)
	if !strings.Contains(output, `Always on: version, doctor\nunsafe, config`) {
		t.Fatalf("always-on projection did not preserve the supplied catalog order safely:\n%s", output)
	}
	if strings.Contains(output, "doctor\nunsafe") {
		t.Fatalf("always-on projection retained structural input:\n%s", output)
	}
}

func TestRenderConfigTUIScreenEscapesHostileTextAndKeepsOnePhysicalLinePerItem(t *testing.T) {
	model := newConfigTUIModel([]configTUIItem{{
		Path:    "rooms\nlist\x1b[2J",
		Summary: "summary\r\n\t\\\u202e",
		Effect:  operation.EffectWrite,
		Enabled: true,
	}}, configTUITestAlwaysPaths)
	output := renderConfigTUIScreen(model, 120, 5)
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("hostile item created physical lines: %d\n%s", len(lines), output)
	}
	item := lines[2]
	for _, want := range []string{`rooms\nlist\u001B[2J`, `summary\r\n\t\\\u202E`} {
		if !strings.Contains(item, want) {
			t.Fatalf("item lacks escaped %q: %q", want, item)
		}
	}
	if strings.Contains(item, "\x1b[2J") || strings.ContainsRune(item, '\r') || strings.ContainsRune(item, '\t') {
		t.Fatalf("item retained hostile terminal structure: %q", item)
	}
	if strings.Count(item, configTUIColorMagenta) != 1 || strings.Count(item, configTUIColorReset) != 1 {
		t.Fatalf("item lacks one CLI-authored write color span: %q", item)
	}
}

func TestRenderConfigTUIScreenTruncatesEveryLineAndHandlesTinyFrames(t *testing.T) {
	model := newConfigTUIModel([]configTUIItem{{
		Path:    strings.Repeat("path", 30),
		Summary: strings.Repeat("サ", 30),
		Effect:  operation.EffectCreate,
	}}, configTUITestAlwaysPaths)
	for _, test := range []struct {
		width  int
		height int
	}{
		{width: 1, height: 1},
		{width: 12, height: 2},
		{width: 20, height: 4},
		{width: 0, height: 4},
		{width: 20, height: 0},
	} {
		output := renderConfigTUIScreen(model, test.width, test.height)
		if test.width <= 0 || test.height <= 0 {
			if output != "" {
				t.Fatalf("render(%d,%d) = %q, want empty", test.width, test.height, output)
			}
			continue
		}
		lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
		if len(lines) > test.height {
			t.Fatalf("render(%d,%d) lines = %d", test.width, test.height, len(lines))
		}
		for _, line := range lines {
			if got := configTUIDisplayWidth(line); got > test.width {
				t.Fatalf("render(%d,%d) line width = %d: %q", test.width, test.height, got, line)
			}
		}
	}
}

func TestRenderConfigTUIScreenNeverTruncatesAnEffectBadge(t *testing.T) {
	model := newConfigTUIModel([]configTUIItem{{
		Path: "rooms create", Effect: operation.EffectCreate, Enabled: true,
	}}, configTUITestAlwaysPaths)
	prefix, badge, _ := configTUIItemPrefix(model.items[0], true)
	identityWidth := configTUIDisplayWidth(prefix + badge + " " + model.items[0].Path)

	narrow := renderConfigTUIScreen(model, identityWidth-1, 5)
	for _, forbidden := range []string{"[cre", "[create]", "rooms create"} {
		if strings.Contains(narrow, forbidden) {
			t.Fatalf("narrow screen exposed a partial or unusable item %q:\n%s", forbidden, narrow)
		}
	}
	if !strings.Contains(narrow, "Resize terminal") {
		t.Fatalf("narrow screen lacks resize guidance:\n%s", narrow)
	}

	minimum := renderConfigTUIScreen(model, identityWidth, 5)
	if !strings.Contains(minimum, configTUIColorYellow+"[create]"+configTUIColorReset) || !strings.Contains(minimum, "rooms create") {
		t.Fatalf("minimum usable width lacks the complete effect and exact command identity:\n%s", minimum)
	}
}

func TestRenderConfigTUIScreenWrapsNoticeWithoutBreakingHeightOrBadges(t *testing.T) {
	model := newConfigTUIModel([]configTUIItem{{
		Path: "messages mark-read", Effect: operation.EffectWrite, Enabled: true,
	}}, configTUITestAlwaysPaths)
	model.notice = "dependency recovery remains actionable after review"
	const width, height = 48, 6
	output := renderConfigTUIScreen(model, width, height)
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != height {
		t.Fatalf("notice frame lines=%d want=%d:\n%s", len(lines), height, output)
	}
	if strings.Count(output, "Notice:") != 1 || !strings.Contains(output, configTUIColorMagenta+"[write]"+configTUIColorReset) {
		t.Fatalf("notice displaced structure or badge:\n%s", output)
	}
	for index, line := range lines {
		if got := configTUIDisplayWidth(line); got > width {
			t.Fatalf("line %d width=%d want<=%d: %q", index, got, width, line)
		}
	}
}

func TestRenderConfigTUIScreenRequiresTheCompleteNoticeAndExactIdentity(t *testing.T) {
	model := newConfigTUIModel([]configTUIItem{{
		Path: "messages mark-read", Effect: operation.EffectWrite, Enabled: true,
	}}, configTUITestAlwaysPaths)
	model.notice = "Cannot save: enable producer rooms list before retry\nreview"
	escapedNotice := "Notice: " + safeExternalText(model.notice)
	prefix, badge, _ := configTUIItemPrefix(model.items[0], true)
	identity := prefix + badge + " " + model.items[0].Path

	for _, test := range []struct {
		name          string
		width, height int
	}{
		{name: "no notice row", width: 120, height: 4},
		{name: "notice width incomplete", width: configTUIDisplayWidth(identity), height: 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := renderConfigTUIScreen(model, test.width, test.height)
			if !strings.Contains(output, "Resize terminal") || strings.Contains(output, "Notice:") || strings.Contains(output, identity) {
				t.Fatalf("incomplete notice or identity did not produce only resize guidance:\n%s", output)
			}
		})
	}

	width := max(configTUIDisplayWidth(escapedNotice), configTUIDisplayWidth(identity))
	output := renderConfigTUIScreen(model, width, 5)
	if !strings.Contains(output, escapedNotice) || !strings.Contains(output, model.items[0].Path) || strings.Contains(output, "…") {
		t.Fatalf("usable frame lacks the complete escaped notice and identity:\n%s", output)
	}
}

func TestRenderConfigTUIScreenUsesTextEffectBadgesAndNonRedANSIColors(t *testing.T) {
	model := newConfigTUIModel([]configTUIItem{
		{Path: "read task", Effect: operation.EffectRead},
		{Path: "create task", Effect: operation.EffectCreate},
		{Path: "write task", Effect: operation.EffectWrite},
	}, configTUITestAlwaysPaths)
	output := renderConfigTUIScreen(model, 100, 6)
	for _, want := range []string{
		configTUIColorCyan + "[read]" + configTUIColorReset,
		configTUIColorYellow + "[create]" + configTUIColorReset,
		configTUIColorMagenta + "[write]" + configTUIColorReset,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("screen lacks effect badge %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "\x1b[31m") || strings.Contains(output, "\x1b[91m") {
		t.Fatalf("screen uses red for an effect badge:\n%s", output)
	}
	plain := strings.NewReplacer(
		configTUIColorCyan, "",
		configTUIColorYellow, "",
		configTUIColorMagenta, "",
		configTUIColorReset, "",
	).Replace(output)
	for _, badge := range []string{"[read]", "[create]", "[write]"} {
		if !strings.Contains(plain, badge) {
			t.Fatalf("color was the only effect signal for %q:\n%s", badge, output)
		}
	}
	lines := strings.Split(strings.TrimSuffix(plain, "\n"), "\n")
	wantColumn := -1
	for index, path := range []string{"read task", "create task", "write task"} {
		column := strings.Index(lines[index+2], path)
		if column < 0 {
			t.Fatalf("screen lacks path %q: %q", path, lines[index+2])
		}
		if wantColumn < 0 {
			wantColumn = column
			continue
		}
		if column != wantColumn {
			t.Fatalf("path %q starts at column %d, want %d:\n%s", path, column, wantColumn, plain)
		}
	}
}
