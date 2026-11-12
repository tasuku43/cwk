package cli

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/tasuku43/cwk/internal/domain/operation"
)

var rootOptionCandidates = []string{"--error-format", "--help", "--version"}

func unknownRootOptionError(name string) error {
	if suggestion := closestFlag(name, rootOptionCandidates); suggestion != "" {
		return fmt.Errorf("グローバルオプション %q は不明です。もしかして %q ですか？", name, suggestion)
	}
	return fmt.Errorf("グローバルオプション %q は不明です", name)
}

func unknownHelpFlagError(name string) error {
	if closestFlag(name, []string{"--help"}) == "--help" {
		return fmt.Errorf("help フラグ %q は不明です。代わりに 'cwk --help' を使ってください", name)
	}
	return fmt.Errorf("help フラグ %q は不明です", name)
}

func unknownCommandMessage(args []string, catalog Catalog) string {
	entered := strings.Join(args, " ")
	if len(args) >= 2 && args[0] == "messages" && args[1] == "search" {
		if _, active := catalog.Lookup("messages list"); active {
			return `コマンド "messages search" はありません。--contains は取得済みの最大100件をローカル照合します（既定範囲は recent、API検索なし）。詳しくは 'cwk help messages list' を確認してください。`
		}
		return `コマンド "messages search" は不明です。利用できるコマンドは 'cwk --help' で確認してください。`
	}
	if suggestion := closestCommandPath(args, catalog.Commands()); suggestion != "" {
		// Keep the nearest candidate's identity. Promoting a farther read command
		// after a create/write/unknown match could turn an unrelated typo into a
		// misleading task suggestion.
		candidate, found := catalog.Lookup(suggestion)
		if found && candidate.Effect == operation.EffectRead {
			return fmt.Sprintf("コマンド %q は不明です。もしかして %q ですか？", entered, ProgramName+" "+candidate.Path)
		}
	}
	return fmt.Sprintf("コマンド %q は不明です。", entered)
}

// closestCommandPath uses only canonical paths in the supplied catalog. The
// caller passes the active catalog so a disabled command cannot leak through a
// typo hint. Suggestions remain advisory: routing never rewrites argv.
func closestCommandPath(args []string, commands []CommandSpec) string {
	inputWords := commandSuggestionWords(args)
	if len(inputWords) == 0 {
		return ""
	}
	paths := make([]string, 0, len(commands))
	for _, command := range commands {
		paths = append(paths, command.Path)
	}
	sort.Strings(paths)

	best := ""
	bestDistance := int(^uint(0) >> 1)
	for _, path := range paths {
		candidateWords := strings.Fields(path)
		comparedWords := inputWords
		if len(comparedWords) > len(candidateWords) {
			comparedWords = comparedWords[:len(candidateWords)]
		}
		input := strings.Join(comparedWords, " ")
		distance := editDistance([]rune(input), []rune(path))
		limit := utf8.RuneCountInString(path) / 3
		if limit < 1 {
			limit = 1
		}
		if distance > limit || distance >= bestDistance {
			continue
		}
		best = path
		bestDistance = distance
	}
	return best
}

func commandSuggestionWords(args []string) []string {
	words := make([]string, 0, len(args))
	for _, argument := range args {
		if strings.HasPrefix(argument, "-") {
			break
		}
		words = append(words, argument)
	}
	return words
}
