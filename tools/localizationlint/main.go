// Command localizationlint verifies the repository's active Japanese entry
// documents. It intentionally ignores historical work packets and machine
// contracts, whose source language and stable identifiers must be preserved.
package main

import (
	"fmt"
	"os"
	"unicode"
	"unicode/utf8"
)

var japaneseEntryDocuments = []string{
	"README.md",
	"CONTRIBUTING.md",
	"SUPPORT.md",
	"SECURITY.md",
	"CODE_OF_CONDUCT.md",
	"docs/README.md",
	"Taskfile.yml",
	".harness/project.json",
	".github/PULL_REQUEST_TEMPLATE.md",
	".github/ISSUE_TEMPLATE/bug.yml",
	".github/ISSUE_TEMPLATE/feature.yml",
	".github/ISSUE_TEMPLATE/config.yml",
	".github/workflows/ci.yml",
	".github/workflows/security.yml",
	".github/workflows/release.yml",
}

func main() {
	root, err := os.OpenRoot(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "localizationlint: open repository root: %v\n", err)
		os.Exit(1)
	}
	defer root.Close()

	failed := false
	for _, path := range japaneseEntryDocuments {
		content, err := root.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "localizationlint: %s: read: %v\n", path, err)
			failed = true
			continue
		}
		if !utf8.Valid(content) {
			fmt.Fprintf(os.Stderr, "localizationlint: %s: active Japanese documentation must be valid UTF-8\n", path)
			failed = true
			continue
		}
		if !containsJapanese(string(content)) {
			fmt.Fprintf(os.Stderr, "localizationlint: %s: active entry documentation must contain Japanese user-facing prose\n", path)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func containsJapanese(value string) bool {
	for _, r := range value {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return true
		}
	}
	return false
}
