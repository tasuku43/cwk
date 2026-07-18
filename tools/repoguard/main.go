// Command repoguard detects repository content that is unsafe to publish.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/tasuku43/cwk/tools/internal/projectconfig"
)

type issue struct {
	Path    string
	Line    int
	Message string
}

var (
	bootstrapPlaceholder  = regexp.MustCompile(`__CLI_[A-Z0-9_]+__|TODO_TEMPLATE|CHANGEME|<your[-_ ][^>]+>`)
	japaneseText          = regexp.MustCompile(`[\x{3040}-\x{30ff}\x{3400}-\x{9fff}]`)
	absoluteHome          = regexp.MustCompile(`(?:/Users/[^/\s]+|/home/[^/\s]+|[A-Za-z]:\\Users\\[^\\\s]+)`)
	privateNetwork        = regexp.MustCompile(`(?i)https?://(?:[^/]*\.(?:internal|corp|local)|10\.[0-9.]+|192\.168\.[0-9.]+|172\.(?:1[6-9]|2[0-9]|3[01])\.[0-9.]+)`)
	formulaPlaceholder    = regexp.MustCompile(`@@([A-Z0-9_]+)@@`)
	inlineMarkdownLink    = regexp.MustCompile(`!?\[[^]\n]*\]\(([^)\n]*)\)`)
	referenceMarkdownLink = regexp.MustCompile(`^\s*\[[^]\n]+\]:\s*(\S+)`)
	uriScheme             = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
	authorizationSecret   = regexp.MustCompile(`(?i)authorization\s*:\s*(?:bearer|basic)\s+([A-Za-z0-9+/=_-]{8,})`)
	assignmentSecret      = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9_])["']?(?:api[_-]?key|client[_-]?secret|password|passwd|access[_-]?token|refresh[_-]?token|private[_-]?key)["']?\s*[:=]\s*(?:"([^"\r\n]*)"|'([^'\r\n]*)'|([^# ,}\]\t\r\n]+))`)
	exampleSecret         = regexp.MustCompile(`^(?:example|dummy|fake|test|redacted|placeholder)(?:[-_.][a-z0-9][a-z0-9._-]*)?$`)
	environmentSecret     = regexp.MustCompile(`^(?:\$\{[A-Z][A-Z0-9_]*\}|env\.[A-Z][A-Z0-9_]*)$`)
	secretPatterns        = []struct {
		name string
		re   *regexp.Regexp
	}{
		{"private key", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`)},
		{"AWS access key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
		{"GitHub token", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{30,}`)},
		{"Slack token", regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{20,}`)},
		{"Google API key", regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`)},
		{"credential-bearing URL", regexp.MustCompile(`(?i)https?://[^/@\s:]+:[^/@\s]+@`)},
	}
	allowedFormulaPlaceholders = map[string]bool{
		"FORMULA_CLASS": true, "DESCRIPTION": true, "REPOSITORY_URL": true,
		"VERSION": true, "MACOS_ARM64_URL": true, "MACOS_AMD64_URL": true,
		"MACOS_ARM64_SHA256": true, "MACOS_AMD64_SHA256": true, "BINARY_NAME": true,
		"LICENSE_SPDX": true,
	}
)

func main() {
	scope := flag.String("scope", "hygiene", "hygiene, security, or public")
	rootFlag := flag.String("root", ".", "repository root")
	flag.Parse()
	if *scope != "hygiene" && *scope != "security" && *scope != "public" {
		fmt.Fprintf(os.Stderr, "repoguard: invalid scope %q\n", *scope)
		os.Exit(2)
	}
	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		fatal(err)
	}
	issues, err := inspect(root, *scope)
	if err != nil {
		fatal(err)
	}
	if len(issues) != 0 {
		for _, item := range issues {
			position := item.Path
			if item.Line > 0 {
				position = fmt.Sprintf("%s:%d", position, item.Line)
			}
			fmt.Fprintf(os.Stderr, "%s: %s\n", position, item.Message)
		}
		os.Exit(1)
	}
	fmt.Printf("repoguard (%s): OK\n", *scope)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "repoguard: %v\n", err)
	os.Exit(1)
}

func inspect(root, scope string) ([]issue, error) {
	paths, err := repositoryPaths(root)
	if err != nil {
		return nil, err
	}
	if err := validateRepositoryPaths(root, paths); err != nil {
		return nil, err
	}
	config, err := projectconfig.Load(root)
	if err != nil {
		return nil, err
	}
	denylist, err := readDenylist(root, config.PublicGuard.DenylistFile)
	if err != nil {
		return nil, err
	}
	var issues []issue
	shapeIssues, err := checkFilesystemShape(root, config)
	if err != nil {
		return nil, err
	}
	issues = append(issues, shapeIssues...)
	issues = append(issues, checkRequired(root, config, scope)...)
	issues = append(issues, checkLicense(root, config, scope)...)
	issues = append(issues, checkAgentHarness(root)...)
	linkIssues, err := checkMarkdownLinks(root, paths)
	if err != nil {
		return nil, err
	}
	issues = append(issues, linkIssues...)
	for _, relative := range paths {
		issues = append(issues, checkPath(relative)...)
		if config.Profile == "ready" && relative != "tools/internal/projectconfig/defaults.go" {
			if identity := remainingTemplateIdentity(relative, config.Project); identity != "" {
				issues = append(issues, issue{Path: relative, Message: fmt.Sprintf("template identity %q remains in path after bootstrap", identity)})
			}
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative))) // #nosec G304 -- git and fallback paths are validated as local repository paths.
		if err != nil {
			return nil, err
		}
		if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
			continue
		}
		issues = append(issues, checkText(relative, string(data), config, denylist, scope)...)
	}
	if scope == "public" && config.Profile == "ready" {
		for _, problem := range projectconfig.ReadyProblems(config.Project) {
			issues = append(issues, issue{Path: ".harness/project.json", Message: problem})
		}
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		return issues[i].Message < issues[j].Message
	})
	return issues, nil
}

func checkLicense(root string, config projectconfig.Config, scope string) []issue {
	if scope != "public" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(root, "LICENSE")) // #nosec G304 -- LICENSE is a fixed file below the selected repository root.
	if err != nil {
		return nil
	}
	text := string(data)
	valid := false
	switch config.Project.LicenseSPDX {
	case "MIT":
		valid = strings.Contains(text, "MIT License") && strings.Contains(text, "Permission is hereby granted")
	case "Apache-2.0":
		valid = strings.Contains(text, "Apache License") && strings.Contains(text, "Version 2.0")
	default:
		valid = strings.Contains(text, config.Project.LicenseSPDX)
	}
	if !valid {
		return []issue{{Path: "LICENSE", Message: "content does not match project.license_spdx; choose and update the license deliberately"}}
	}
	return nil
}

func repositoryPaths(root string) ([]string, error) {
	command := exec.Command("git", "ls-files", "-co", "--exclude-standard", "-z")
	command.Dir = root
	output, err := command.Output()
	if err == nil {
		var paths []string
		for _, raw := range bytes.Split(output, []byte{0}) {
			if len(raw) != 0 {
				relative := string(raw)
				if !filepath.IsLocal(relative) {
					return nil, fmt.Errorf("git returned a non-local path %q", relative)
				}
				if _, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(relative))); os.IsNotExist(statErr) {
					continue
				} else if statErr != nil {
					return nil, fmt.Errorf("inspect git path %q: %w", relative, statErr)
				}
				paths = append(paths, filepath.ToSlash(relative))
			}
		}
		sort.Strings(paths)
		return paths, nil
	}
	var paths []string
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "bin" || entry.Name() == "dist") {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

// validateRepositoryPaths rejects links and special files before repoguard
// reads any repository-controlled content. A link in any path component is
// rejected so an apparently local path cannot redirect a read outside root.
func validateRepositoryPaths(root string, paths []string) error {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("repository root is a symbolic link: %s", root)
	}
	if !rootInfo.IsDir() {
		return fmt.Errorf("repository root is not a directory: %s", root)
	}
	for _, relative := range paths {
		if !filepath.IsLocal(relative) || filepath.IsAbs(relative) {
			return fmt.Errorf("repository path is not local: %q", relative)
		}
		parts := strings.Split(filepath.Clean(filepath.FromSlash(relative)), string(filepath.Separator))
		current := root
		for index, part := range parts {
			current = filepath.Join(current, part)
			info, err := os.Lstat(current)
			if err != nil {
				return fmt.Errorf("inspect repository path %q: %w", relative, err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("repository path is a symbolic link: %s", relative)
			}
			if index < len(parts)-1 && !info.IsDir() {
				return fmt.Errorf("repository path component is not a directory: %s", relative)
			}
			if index == len(parts)-1 && !info.Mode().IsRegular() {
				return fmt.Errorf("repository path is not a regular file: %s", relative)
			}
		}
	}
	return nil
}

func checkRequired(root string, config projectconfig.Config, scope string) []issue {
	if scope != "public" {
		return nil
	}
	var issues []issue
	for _, path := range config.PublicGuard.Required {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			issues = append(issues, issue{Path: path, Message: "required public repository path is missing"})
		}
	}
	return issues
}

func checkAgentHarness(root string) []issue {
	paths := []string{
		".agents/skills/bootstrap-derived-cli/SKILL.md",
		".agents/skills/bootstrap-derived-cli/agents/openai.yaml",
		".agents/skills/add-capability/SKILL.md",
		".codex/hooks.json",
		".codex/hooks/check.sh",
	}
	var issues []issue
	for _, path := range paths {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			issues = append(issues, issue{Path: path, Message: "required Codex harness file is missing"})
		}
	}
	if len(issues) != 0 {
		return issues
	}
	issues = append(issues, checkCodexStopHook(root)...)
	return issues
}

type codexHookDocument struct {
	Hooks map[string][]codexHookGroup `json:"hooks"`
}

type codexHookGroup struct {
	Hooks []codexCommandHook `json:"hooks"`
}

type codexCommandHook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

func checkCodexStopHook(root string) []issue {
	hooksPath := filepath.Join(root, ".codex", "hooks.json")
	data, err := os.ReadFile(hooksPath) // #nosec G304 -- path is fixed below the selected repository root.
	if err != nil {
		return []issue{{Path: ".codex/hooks.json", Message: "cannot read the Codex hook contract"}}
	}
	var document codexHookDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return []issue{{Path: ".codex/hooks.json", Message: "Codex hook configuration is not valid JSON"}}
	}
	validStop := false
	for _, group := range document.Hooks["Stop"] {
		for _, hook := range group.Hooks {
			if hook.Type == "command" && hook.Timeout > 0 &&
				strings.Contains(hook.Command, "git rev-parse --show-toplevel") &&
				strings.Contains(hook.Command, "/.codex/hooks/check.sh") {
				validStop = true
			}
		}
	}
	if !validStop {
		return []issue{{Path: ".codex/hooks.json", Message: "Stop hook must resolve the canonical check script from the Git root with a finite timeout"}}
	}

	scriptPath := filepath.Join(root, ".codex", "hooks", "check.sh")
	script, err := os.ReadFile(scriptPath) // #nosec G304 -- path is fixed below the selected repository root.
	if err != nil {
		return []issue{{Path: ".codex/hooks/check.sh", Message: "cannot read the Codex check hook"}}
	}
	text := string(script)
	if !strings.Contains(text, "./scripts/check.sh fast") ||
		!strings.Contains(text, `"continue":false`) {
		return []issue{{Path: ".codex/hooks/check.sh", Message: "Stop hook must delegate to the canonical fast gate and return structured continuation on failure"}}
	}
	return nil
}

func checkFilesystemShape(root string, config projectconfig.Config) ([]issue, error) {
	var issues []issue
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "bin" || entry.Name() == "dist") {
			return filepath.SkipDir
		}
		if strings.EqualFold(entry.Name(), ".claude") {
			issues = append(issues, issue{Path: relative, Message: "Claude-specific harness paths are outside this Codex-only template"})
			if entry.IsDir() {
				return filepath.SkipDir
			}
		}
		if entry.Type()&os.ModeSymlink != 0 {
			issues = append(issues, issue{Path: relative, Message: "symbolic links are not allowed in the public working tree"})
			return nil
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			issues = append(issues, issue{Path: relative, Message: "special files are not allowed in the public working tree"})
			return nil
		}
		if !entry.IsDir() {
			issues = append(issues, checkWorkingTreeArtifact(relative, config)...)
		}
		return nil
	})
	return issues, err
}

func checkWorkingTreeArtifact(path string, config projectconfig.Config) []issue {
	lower := strings.ToLower(path)
	base := strings.ToLower(filepath.Base(path))
	if strings.EqualFold(base, "claude.md") {
		return []issue{{Path: path, Message: "Claude-specific instructions are outside this Codex-only template"}}
	}
	if strings.HasSuffix(lower, ".bootstrap.tmp") || strings.HasSuffix(lower, ".bootstrap.orig") {
		return []issue{{Path: path, Message: "interrupted bootstrap residue must not be published"}}
	}
	if filepath.ToSlash(path) == config.Project.BinaryName || filepath.ToSlash(path) == config.Project.BinaryName+".exe" {
		return []issue{{Path: path, Message: "root build artifact must not be published; use bin/ or a temporary release directory"}}
	}
	return nil
}

func checkPath(path string) []issue {
	lower := strings.ToLower(path)
	base := strings.ToLower(filepath.Base(path))
	forbiddenBase := map[string]bool{
		".env": true, ".ds_store": true, "credentials.json": true, "secrets.json": true,
		"id_rsa": true, "id_ed25519": true,
	}
	if forbiddenBase[base] || strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key") || strings.HasSuffix(lower, ".p12") || strings.HasSuffix(lower, ".pfx") {
		return []issue{{Path: path, Message: "sensitive or local-only path must not be published"}}
	}
	return nil
}

func checkMarkdownLinks(root string, repositoryPaths []string) ([]issue, error) {
	publishable := make(map[string]bool, len(repositoryPaths))
	for _, relative := range repositoryPaths {
		publishable[relative] = true
	}
	var issues []issue
	for _, source := range repositoryPaths {
		if !strings.HasSuffix(strings.ToLower(source), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(source))) // #nosec G304 -- source was validated as a local regular repository path.
		if err != nil {
			return nil, err
		}
		inFence := false
		for index, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}
			var destinations []string
			for _, match := range inlineMarkdownLink.FindAllStringSubmatch(line, -1) {
				destinations = append(destinations, match[1])
			}
			if match := referenceMarkdownLink.FindStringSubmatch(line); match != nil {
				destinations = append(destinations, match[1])
			}
			for _, raw := range destinations {
				destination := markdownDestination(raw)
				if destination == "" || strings.HasPrefix(destination, "#") || strings.HasPrefix(destination, "//") || uriScheme.MatchString(destination) {
					continue
				}
				local := destination
				if delimiter := strings.IndexAny(local, "?#"); delimiter >= 0 {
					local = local[:delimiter]
				}
				if local == "" {
					continue
				}
				if problem := validateMarkdownTarget(root, source, local, publishable); problem != "" {
					issues = append(issues, issue{Path: source, Line: index + 1, Message: problem})
				}
			}
		}
	}
	return issues, nil
}

func markdownDestination(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "<") {
		if end := strings.Index(trimmed, ">"); end > 0 {
			return trimmed[1:end]
		}
		return trimmed
	}
	if fields := strings.Fields(trimmed); len(fields) != 0 {
		return fields[0]
	}
	return ""
}

func validateMarkdownTarget(root, source, local string, publishable map[string]bool) string {
	if strings.HasPrefix(local, "/") || strings.Contains(local, `\`) {
		return fmt.Sprintf("local Markdown link %q must use a repository-relative slash path", local)
	}
	cleaned := pathpkg.Clean(local)
	canonical := cleaned
	if strings.HasSuffix(local, "/") && cleaned != "." {
		canonical += "/"
	}
	if local != canonical {
		return fmt.Sprintf("local Markdown link %q is not canonical", local)
	}
	resolved := pathpkg.Clean(pathpkg.Join(pathpkg.Dir(source), cleaned))
	if resolved == ".." || strings.HasPrefix(resolved, "../") || pathpkg.IsAbs(resolved) {
		return fmt.Sprintf("local Markdown link %q escapes the repository", local)
	}
	if !publishable[resolved] {
		return fmt.Sprintf("local Markdown link %q does not target a publishable regular file", local)
	}
	current := root
	for _, component := range strings.Split(resolved, "/") {
		current = filepath.Join(current, filepath.FromSlash(component))
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Sprintf("local Markdown link %q cannot be resolved: %v", local, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Sprintf("local Markdown link %q resolves through a symbolic link", local)
		}
	}
	info, err := os.Lstat(current)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Sprintf("local Markdown link %q does not target a regular file", local)
	}
	return ""
}

func checkText(path, text string, config projectconfig.Config, denylist []string, scope string) []issue {
	var issues []issue
	scannerSource := path == "tools/repoguard/main.go"
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		lineNumber := index + 1
		if config.Profile == "ready" && path != "tools/internal/projectconfig/defaults.go" {
			if identity := remainingTemplateIdentity(line, config.Project); identity != "" {
				issues = append(issues, issue{Path: path, Line: lineNumber, Message: fmt.Sprintf("template identity %q remains after bootstrap", identity)})
			}
		}
		if config.Profile == "ready" && !scannerSource && bootstrapPlaceholder.MatchString(line) {
			issues = append(issues, issue{Path: path, Line: lineNumber, Message: "unresolved bootstrap placeholder"})
		}
		if strings.HasSuffix(strings.ToLower(path), ".md") && japaneseText.MatchString(line) {
			issues = append(issues, issue{Path: path, Line: lineNumber, Message: "documentation must be written in English"})
		}
		if !scannerSource && absoluteHome.MatchString(line) {
			issues = append(issues, issue{Path: path, Line: lineNumber, Message: "machine-specific home directory path"})
		}
		if !scannerSource && privateNetwork.MatchString(line) {
			issues = append(issues, issue{Path: path, Line: lineNumber, Message: "private hostname or network address"})
		}
		for _, term := range denylist {
			if path != filepath.ToSlash(config.PublicGuard.DenylistFile) && strings.Contains(strings.ToLower(line), strings.ToLower(term)) {
				issues = append(issues, issue{Path: path, Line: lineNumber, Message: fmt.Sprintf("custom denylist term %q", term)})
			}
		}
		if !scannerSource {
			issues = append(issues, checkFormulaPlaceholder(path, line, lineNumber)...)
		}
		if !scannerSource && (scope == "security" || scope == "public") {
			issues = append(issues, checkSecrets(path, line, lineNumber)...)
		}
	}
	return issues
}

func checkFormulaPlaceholder(path, line string, lineNumber int) []issue {
	matches := formulaPlaceholder.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}
	validPath := (strings.HasPrefix(filepath.ToSlash(path), "Formula/") && strings.HasSuffix(path, ".rb.template")) ||
		path == "scripts/render-formula.sh" || path == "scripts/lint-release.sh"
	var issues []issue
	for _, match := range matches {
		if !validPath || !allowedFormulaPlaceholders[match[1]] {
			issues = append(issues, issue{Path: path, Line: lineNumber, Message: "unknown or misplaced release-time placeholder " + match[0]})
		}
	}
	return issues
}

func checkSecrets(path, line string, lineNumber int) []issue {
	var issues []issue
	for _, pattern := range secretPatterns {
		if pattern.re.MatchString(line) {
			issues = append(issues, issue{Path: path, Line: lineNumber, Message: "secret-like content: " + pattern.name})
		}
	}
	for _, match := range authorizationSecret.FindAllStringSubmatch(line, -1) {
		if !safeExampleSecret(match[1]) {
			issues = append(issues, issue{Path: path, Line: lineNumber, Message: "secret-like content: authorization header"})
		}
	}
	for _, match := range assignmentSecret.FindAllStringSubmatch(line, -1) {
		value := ""
		for _, candidate := range match[1:] {
			if candidate != "" {
				value = candidate
				break
			}
		}
		if !safeExampleSecret(value) {
			issues = append(issues, issue{Path: path, Line: lineNumber, Message: "secret-like value assigned in source"})
		}
	}
	return issues
}

func safeExampleSecret(value string) bool {
	trimmed := strings.TrimSpace(strings.Trim(value, `"'`))
	lower := strings.ToLower(trimmed)
	if lower == "" || lower == "none" || lower == "null" || lower == "[redacted]" {
		return true
	}
	return exampleSecret.MatchString(lower) || environmentSecret.MatchString(trimmed)
}

func remainingTemplateIdentity(line string, target projectconfig.Project) string {
	defaults := projectconfig.Defaults
	type identityReplacement struct {
		from string
		to   string
	}
	values := []identityReplacement{
		{"https://github.com/" + defaults.GitHubOwner + "/" + defaults.GitHubRepository, "https://github.com/" + target.GitHubOwner + "/" + target.GitHubRepository},
		{defaults.GoModule, target.GoModule},
		{defaults.GitHubOwner + "/" + defaults.GitHubRepository, target.GitHubOwner + "/" + target.GitHubRepository},
		{defaults.Description, target.Description},
		{defaults.SecurityContact, target.SecurityContact},
		{defaults.Name, target.Name},
		{defaults.FormulaClass, target.FormulaClass},
		{defaults.BinaryName, target.BinaryName},
		{defaults.GitHubRepository, target.GitHubRepository},
	}
	sort.SliceStable(values, func(i, j int) bool { return len(values[i].to) > len(values[j].to) })
	withoutTargetIdentity := line
	for _, value := range values {
		if value.to != "" && value.to != value.from {
			withoutTargetIdentity = strings.ReplaceAll(withoutTargetIdentity, value.to, "")
		}
	}
	sort.SliceStable(values, func(i, j int) bool { return len(values[i].from) > len(values[j].from) })
	for _, value := range values {
		if strings.Contains(withoutTargetIdentity, value.from) {
			return value.from
		}
	}
	return ""
}

func readDenylist(root, relative string) ([]string, error) {
	if !filepath.IsLocal(relative) {
		return nil, fmt.Errorf("denylist path is not local: %q", relative)
	}
	path := filepath.Join(root, filepath.FromSlash(relative))
	file, err := os.Open(path) // #nosec G304 -- relative is validated and scoped to the selected repository root.
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var terms []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		term := strings.TrimSpace(scanner.Text())
		if term != "" && !strings.HasPrefix(term, "#") {
			terms = append(terms, term)
		}
	}
	return terms, scanner.Err()
}
