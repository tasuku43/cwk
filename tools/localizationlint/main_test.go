package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalizationLintEntryDocuments(t *testing.T) {
	if os.Getenv("CWK_LOCALIZATIONLINT_HELPER") == "1" {
		main()
		return
	}
	for _, tc := range []struct {
		name   string
		change func(string)
		want   []string
	}{
		{name: "all Japanese documents"},
		{name: "reports all invalid documents", change: func(root string) {
			if err := os.Remove(filepath.Join(root, japaneseEntryDocuments[0])); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, japaneseEntryDocuments[1]), []byte{0xff}, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, japaneseEntryDocuments[2]), []byte("English only"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, want: []string{"README.md: read:", "CONTRIBUTING.md: active Japanese documentation must be valid UTF-8", "SUPPORT.md: active entry documentation must contain Japanese"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, path := range japaneseEntryDocuments {
				full := filepath.Join(root, path)
				if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte("利用方法"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.change != nil {
				tc.change(root)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestLocalizationLintEntryDocuments$")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "CWK_LOCALIZATIONLINT_HELPER=1")
			output, err := cmd.CombinedOutput()
			if (err != nil) != (len(tc.want) != 0) {
				t.Fatalf("exit error = %v, output = %q", err, output)
			}
			for _, want := range tc.want {
				if !strings.Contains(string(output), want) {
					t.Errorf("output %q missing %q", output, want)
				}
			}
		})
	}
}

func TestContainsJapanese(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "hiragana", value: "コマンドを実行してください", want: true},
		{name: "katakana", value: "エラー", want: true},
		{name: "han", value: "日本", want: true},
		{name: "machine identifiers", value: "cwk rooms list --format json", want: false},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := containsJapanese(test.value); got != test.want {
				t.Fatalf("containsJapanese(%q) = %t, want %t", test.value, got, test.want)
			}
		})
	}
}
