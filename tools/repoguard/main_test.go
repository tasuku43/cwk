package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/tools/internal/projectconfig"
)

func TestRepositoryPathsSkipsUnstagedTrackedDeletions(t *testing.T) {
	root := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	runGit("init", "--quiet")
	writeRepositoryFixture(t, root, "old.txt", "tracked before rename\n")
	runGit("add", "old.txt")
	if err := os.Remove(filepath.Join(root, "old.txt")); err != nil {
		t.Fatal(err)
	}
	writeRepositoryFixture(t, root, "new.txt", "untracked rename target\n")

	paths, err := repositoryPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "new.txt" {
		t.Fatalf("repository paths = %v", paths)
	}
}

func TestCheckTextDetectsPublicLeaksAndUnsafeSecrets(t *testing.T) {
	config := projectconfig.Config{Profile: "ready"}
	text := strings.Join([]string{
		"home=/Users" + "/alice/private",
		"docs=https://service." + "corp/runbook",
		"api_" + "key=real-production-value",
		"TODO_" + "TEMPLATE",
		"internal-ticket-123",
	}, "\n")
	issues := checkText("README.md", text, config, []string{"internal-ticket"}, "public")
	if len(issues) != 5 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestCheckTextAllowsJapaneseDocumentation(t *testing.T) {
	t.Parallel()

	config := projectconfig.Config{Profile: "template"}
	if issues := checkText("README.md", "日本の利用者向けドキュメントです。", config, nil, "public"); len(issues) != 0 {
		t.Fatalf("Japanese documentation issues = %#v", issues)
	}
}

func TestCheckTextDetectsQuotedJSONSecretsAndMarkerSubstrings(t *testing.T) {
	config := projectconfig.Config{Profile: "template"}
	unsafe := jsonSecretAssignment("client_"+"secret", "prod-contest-value")
	issues := checkText("config.json", unsafe, config, nil, "security")
	if len(issues) != 1 || issues[0].Message != "secret-like value assigned in source" {
		t.Fatalf("unsafe JSON issues = %#v", issues)
	}

	for _, value := range []string{
		jsonSecretAssignment("client_"+"secret", "dummy-value"),
		jsonSecretAssignment("access_"+"token", "${ACCESS_TOKEN}"),
		jsonSecretAssignment("pass"+"word", "env.PASSWORD"),
		jsonSecretAssignment("pass"+"word", "[redacted]"),
	} {
		if issues := checkText("config.json", value, config, nil, "security"); len(issues) != 0 {
			t.Errorf("safe example %q issues = %#v", value, issues)
		}
	}

	for _, value := range []string{
		jsonSecretAssignment("client_"+"secret", "production-dummy"),
		jsonSecretAssignment("client_"+"secret", "contest-token"),
		jsonSecretAssignment("client_"+"secret", "env.PASSWORD-extra"),
	} {
		issues := checkText("config.json", value, config, nil, "security")
		if len(issues) != 1 || issues[0].Message != "secret-like value assigned in source" {
			t.Errorf("embedded marker %q issues = %#v", value, issues)
		}
	}
}

func TestCheckSecretsDistinguishesGoCredentialFlowFromHardcodedValues(t *testing.T) {
	for _, safe := range []string{
		"stored.AccessToken = refreshed.AccessToken",
		"if token.RefreshToken == other.RefreshToken {",
		"AccessToken: token.AccessToken,",
	} {
		if issues := checkSecrets("adapter.go", safe, 1); len(issues) != 0 {
			t.Errorf("Go credential flow %q issues = %#v", safe, issues)
		}
	}
	for _, unsafe := range []string{
		"access_" + `token := "production-credential-value"`,
		"refresh_" + "token := `production-credential-value`",
	} {
		issues := checkSecrets("adapter.go", unsafe, 1)
		if len(issues) != 1 || issues[0].Message != "secret-like value assigned in source" {
			t.Errorf("hardcoded Go value %q issues = %#v", unsafe, issues)
		}
	}
}

func TestCheckSecretsDetectsAuthorizationHeadersAndCredentialURLs(t *testing.T) {
	header := "Authorization: Bearer " + "liveToken123"
	issues := checkSecrets("fixture.txt", header, 1)
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "authorization header") {
		t.Fatalf("header issues = %#v", issues)
	}
	if issues := checkSecrets("fixture.txt", "Authorization: Bearer dummy-token", 1); len(issues) != 0 {
		t.Fatalf("example header issues = %#v", issues)
	}
	credentialURL := "https://user:" + "live-password@example.com/resource"
	issues = checkSecrets("fixture.txt", credentialURL, 1)
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "credential-bearing URL") {
		t.Fatalf("URL issues = %#v", issues)
	}
}

func TestCheckTextRejectsReadyTemplateIdentityOutsideDefaults(t *testing.T) {
	config := projectconfig.Config{
		Profile: "ready",
		Project: projectconfig.Project{
			Name: "Acme Tool", BinaryName: "acme", GoModule: "github.com/acme/tool",
			GitHubOwner: "acme", GitHubRepository: "tool", Description: "An Acme tool.",
			FormulaClass: "Acme", SecurityContact: "security@acme.example",
		},
	}
	issues := checkText("README.md", "module "+projectconfig.Defaults.GoModule, config, nil, "public")
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "template identity") {
		t.Fatalf("identity issues = %#v", issues)
	}
	if issues := checkText("tools/internal/projectconfig/defaults.go", projectconfig.Defaults.GoModule, config, nil, "public"); len(issues) != 0 {
		t.Fatalf("protected defaults issues = %#v", issues)
	}
}

func TestCheckTextAllowsConfiguredIdentityContainingTemplateSubstring(t *testing.T) {
	config := projectconfig.Config{
		Profile: "ready",
		Project: projectconfig.Project{
			Name: "Acme Chatwork CLI", BinaryName: "acme-cwk", GoModule: "github.com/acme/acme-cwk",
			GitHubOwner: "acme", GitHubRepository: "acme-cwk", Description: "An Acme CLI template.",
			FormulaClass: "AcmeCwk", SecurityContact: "security@acme.example",
		},
	}
	line := "github.com/acme/acme-cwk acme-cwk AcmeCwk"
	if issues := checkText("README.md", line, config, nil, "public"); len(issues) != 0 {
		t.Fatalf("configured identity issues = %#v", issues)
	}
	line += " " + projectconfig.Defaults.GoModule
	if issues := checkText("README.md", line, config, nil, "public"); len(issues) != 1 {
		t.Fatalf("residual identity issues = %#v", issues)
	}
}

func TestValidateRepositoryPathsRejectsSymbolicLinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("public fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "linked.txt")); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if err := validateRepositoryPaths(root, []string{"linked.txt"}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("validateRepositoryPaths() error = %v", err)
	}
}

func TestValidateRepositoryPathsRejectsSymbolicDirectoryComponents(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "fixture.txt"), []byte("external\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "linked-dir")); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if err := validateRepositoryPaths(root, []string{"linked-dir/fixture.txt"}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("validateRepositoryPaths() error = %v", err)
	}
}

func TestCheckTextAllowsDocumentedExamplesAndReleasePlaceholders(t *testing.T) {
	config := projectconfig.Config{Profile: "template"}
	if issues := checkText("example.env", "api_key=dummy-value", config, nil, "security"); len(issues) != 0 {
		t.Fatalf("example issues = %#v", issues)
	}
	marker := "@" + "@"
	formula := "url \"" + marker + "MACOS_ARM64_URL" + marker + "\"\nsha256 \"" + marker + "MACOS_ARM64_SHA256" + marker + "\""
	if issues := checkText("Formula/cwk.rb.template", formula, config, nil, "public"); len(issues) != 0 {
		t.Fatalf("formula issues = %#v", issues)
	}
}

func TestCheckFilesystemShapeRejectsClaudePolicyAndRootBuildArtifacts(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cwk"), []byte("binary fixture\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	config := projectconfig.Config{Project: projectconfig.Project{BinaryName: "cwk"}}
	issues, err := checkFilesystemShape(root, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Fatalf("issues = %#v", issues)
	}
	messages := issues[0].Message + "\n" + issues[1].Message
	for _, expected := range []string{"Claude-specific", "root build artifact"} {
		if !strings.Contains(messages, expected) {
			t.Errorf("issues do not contain %q: %#v", expected, issues)
		}
	}
}

func TestCheckFilesystemShapeDoesNotTreatIgnoredLocalFilesAsPublishable(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{".env", ".DS_Store"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("local-only\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	config := projectconfig.Config{Project: projectconfig.Project{BinaryName: "cwk"}}
	issues, err := checkFilesystemShape(root, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ignored local files became shape issues: %#v", issues)
	}
}

func TestCheckAgentHarnessValidatesCanonicalStopHook(t *testing.T) {
	root := t.TempDir()
	writeRepositoryFixture(t, root, ".agents/skills/bootstrap-derived-cli/SKILL.md", "# Skill\n")
	writeRepositoryFixture(t, root, ".agents/skills/bootstrap-derived-cli/agents/openai.yaml", "interface: {}\n")
	writeRepositoryFixture(t, root, ".agents/skills/add-capability/SKILL.md", "# Skill\n")
	validHooks := `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"bash \"$(git rev-parse --show-toplevel)/.codex/hooks/check.sh\"","timeout":180}]}]}}`
	validScript := "#!/usr/bin/env bash\n./scripts/check.sh fast\nprintf '%s\\n' '{\"continue\":false}'\n"
	writeRepositoryFixture(t, root, ".codex/hooks.json", validHooks)
	writeRepositoryFixture(t, root, ".codex/hooks/check.sh", validScript)

	if issues := checkAgentHarness(root); len(issues) != 0 {
		t.Fatalf("valid harness issues = %#v", issues)
	}

	writeRepositoryFixture(t, root, ".codex/hooks.json", `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"bash .codex/hooks/check.sh","timeout":180}]}]}}`)
	if issues := checkAgentHarness(root); len(issues) != 1 || !strings.Contains(issues[0].Message, "Git root") {
		t.Fatalf("relative hook issues = %#v", issues)
	}

	writeRepositoryFixture(t, root, ".codex/hooks.json", validHooks)
	writeRepositoryFixture(t, root, ".codex/hooks/check.sh", "#!/usr/bin/env bash\n./scripts/check.sh fast\n")
	if issues := checkAgentHarness(root); len(issues) != 1 || !strings.Contains(issues[0].Message, "structured continuation") {
		t.Fatalf("unstructured hook issues = %#v", issues)
	}
}

func TestCheckPathRejectsParallelAgentPolicyFiles(t *testing.T) {
	config := projectconfig.Config{Project: projectconfig.Project{BinaryName: "cwk"}}
	for _, path := range []string{"CLAUDE.md", "docs/Claude.md"} {
		issues := checkWorkingTreeArtifact(path, config)
		if len(issues) != 1 || !strings.Contains(issues[0].Message, "Claude-specific") {
			t.Errorf("checkPath(%q) = %#v", path, issues)
		}
	}
}

func TestCheckMarkdownLinksAllowsPublishableFilesAndSkipsExternalTargets(t *testing.T) {
	root := t.TempDir()
	writeRepositoryFixture(t, root, "README.md", "# Root\n")
	writeRepositoryFixture(t, root, "docs/guide.md", strings.Join([]string{
		"[root](../README.md#root)",
		"[external](https://example.com/docs)",
		"[mail](mailto:task.teckac@gmail.com)",
		"[same page](#section)",
		"[root reference]: ../README.md",
		"```text",
		"[example only](missing.md)",
		"```",
	}, "\n"))
	paths := []string{"README.md", "docs/guide.md"}
	issues, err := checkMarkdownLinks(root, paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestCheckMarkdownLinksRejectsUnsafeOrUnpublishedTargets(t *testing.T) {
	root := t.TempDir()
	writeRepositoryFixture(t, root, "README.md", "# Root\n")
	writeRepositoryFixture(t, root, "docs/bad.md", strings.Join([]string{
		"[missing](missing.md)",
		"[escape](../../outside.md)",
		"[absolute](/README.md)",
		"[noncanonical](../docs/../README.md)",
		"[directory](../fixtures/)",
		"[link](../linked.md)",
	}, "\n"))
	if err := os.Mkdir(filepath.Join(root, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "README.md"), filepath.Join(root, "linked.md")); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	paths := []string{"README.md", "docs/bad.md", "linked.md"}
	issues, err := checkMarkdownLinks(root, paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 6 {
		t.Fatalf("issues = %#v", issues)
	}
	messages := make([]string, 0, len(issues))
	for _, item := range issues {
		messages = append(messages, item.Message)
	}
	joined := strings.Join(messages, "\n")
	for _, expected := range []string{"publishable regular file", "escapes the repository", "repository-relative", "not canonical", "symbolic link"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("issues do not contain %q: %#v", expected, issues)
		}
	}
}

func writeRepositoryFixture(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func jsonSecretAssignment(name, value string) string {
	return `{"` + name + `":"` + value + `"}`
}
