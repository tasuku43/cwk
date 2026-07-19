package main

import "testing"

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
