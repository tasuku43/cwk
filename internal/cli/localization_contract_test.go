package cli

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/tasuku43/cwk/internal/domain/fault"
)

func TestPublicCatalogNaturalLanguageIsJapanese(t *testing.T) {
	t.Parallel()

	for _, command := range DefaultCatalog().Commands() {
		assertJapaneseProse(t, command.Path+" summary", command.Summary)
		assertJapaneseProse(t, command.Path+" outcome", command.Agent.Outcome)
		for _, input := range command.Agent.Inputs {
			assertJapaneseProse(t, command.Path+" input "+input.Name, input.Description)
		}
		for _, field := range command.Agent.Output.Fields {
			assertJapaneseProse(t, command.Path+" output "+field.Name, field.Description)
		}
		for index, prerequisite := range command.Agent.Prerequisites {
			assertJapaneseProse(t, command.Path+" prerequisite "+string(rune(index+'0')), prerequisite)
		}
		if command.Agent.FixedTarget != nil {
			assertJapaneseProse(t, command.Path+" fixed target", command.Agent.FixedTarget.Description)
		}
		for _, commandError := range command.Agent.Errors {
			if commandError.MessageGrammar != "" {
				assertJapaneseProse(t, command.Path+" error grammar "+commandError.Code, commandError.MessageGrammar)
			}
			for _, action := range commandError.NextActions {
				assertJapaneseProse(t, command.Path+" recovery "+commandError.Code, action.Reason)
			}
		}
	}
}

func TestHumanHelpAndFaultMessagesDefaultToJapanese(t *testing.T) {
	t.Parallel()

	command, stdout, stderr := newTestCLI(passingInspector("unused"))
	if code := runCLI(command, []string{"--help"}); code != ExitOK {
		t.Fatalf("root help code = %d, stderr = %q", code, stderr.String())
	}
	assertJapaneseProse(t, "root human help", stdout.String())
	if strings.Contains(stdout.String(), "Usage:") || strings.Contains(stdout.String(), "Global options:") {
		t.Fatalf("root help retained legacy English headings: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCLI(command, []string{"not-a-command"}); code != ExitUsage {
		t.Fatalf("unknown command code = %d, stderr = %q", code, stderr.String())
	}
	assertJapaneseProse(t, "public fault", stderr.String())
	for _, stable := range []string{"kind: invalid_input", "code: unknown_command", "next_action: cwk help"} {
		if !strings.Contains(stderr.String(), stable) {
			t.Fatalf("fault lost stable machine token %q: %q", stable, stderr.String())
		}
	}
}

func TestREADMEExitCodeMappingMatchesRuntime(t *testing.T) {
	content, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	const heading = "## 終了コードとエラー分類"
	start := strings.Index(string(content), heading)
	if start < 0 {
		t.Fatalf("README.md is missing %q", heading)
	}
	section := string(content[start+len(heading):])
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}

	expected := map[string]int{
		"成功":                             ExitOK,
		string(fault.KindInvalidInput):   exitCodeForKind(fault.KindInvalidInput),
		string(fault.KindInternal):       exitCodeForKind(fault.KindInternal),
		string(fault.KindAuthentication): exitCodeForKind(fault.KindAuthentication),
		string(fault.KindPermission):     exitCodeForKind(fault.KindPermission),
		string(fault.KindNotFound):       exitCodeForKind(fault.KindNotFound),
		string(fault.KindAmbiguous):      exitCodeForKind(fault.KindAmbiguous),
		string(fault.KindRateLimited):    exitCodeForKind(fault.KindRateLimited),
		string(fault.KindUnavailable):    exitCodeForKind(fault.KindUnavailable),
		string(fault.KindRejected):       exitCodeForKind(fault.KindRejected),
		string(fault.KindCanceled):       exitCodeForKind(fault.KindCanceled),
		string(fault.KindUnsupported):    exitCodeForKind(fault.KindUnsupported),
		string(fault.KindContract):       exitCodeForKind(fault.KindContract),
	}
	actual := make(map[string]int)
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 3 {
			continue
		}
		kind := strings.Trim(strings.TrimSpace(cells[1]), "`")
		if kind == "kind` / 状態" || strings.Trim(kind, "-: ") == "" {
			continue
		}
		codeText := strings.Trim(strings.TrimSpace(cells[2]), "`")
		code, err := strconv.Atoi(codeText)
		if err != nil {
			t.Fatalf("README.md exit code for %q is not an integer: %q", kind, codeText)
		}
		actual[kind] = code
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("README.md exit-code mapping = %v, want runtime mapping %v", actual, expected)
	}
}

func assertJapaneseProse(t *testing.T, name, value string) {
	t.Helper()
	for _, r := range value {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return
		}
	}
	t.Errorf("%s must contain Japanese user-facing prose: %q", name, value)
}
