package cli

import (
	"strings"
	"testing"
	"unicode"
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

func assertJapaneseProse(t *testing.T, name, value string) {
	t.Helper()
	for _, r := range value {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return
		}
	}
	t.Errorf("%s must contain Japanese user-facing prose: %q", name, value)
}
