package cli

import (
	"fmt"
	"sort"
	"unicode/utf8"
)

// unknownFlagError keeps typo recovery deterministic and local to the
// command's declared flag surface.  It deliberately declines distant matches:
// a wrong suggestion is more expensive for an agent than one scoped-help
// lookup.
func unknownFlagError(name string, candidates []string) error {
	if suggestion := closestFlag(name, candidates); suggestion != "" {
		return fmt.Errorf("フラグ %q は不明です。もしかして %q ですか？", name, suggestion)
	}
	return fmt.Errorf("フラグ %q は不明です", name)
}

func closestFlag(name string, candidates []string) string {
	if len(candidates) == 0 || len(name) < 3 || name[0:2] != "--" {
		return ""
	}
	unique := make(map[string]struct{}, len(candidates))
	ordered := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if len(candidate) < 3 || candidate[0:2] != "--" || candidate == name {
			continue
		}
		if _, exists := unique[candidate]; exists {
			continue
		}
		unique[candidate] = struct{}{}
		ordered = append(ordered, candidate)
	}
	sort.Strings(ordered)

	best := ""
	bestDistance := int(^uint(0) >> 1)
	for _, candidate := range ordered {
		distance := editDistance([]rune(name), []rune(candidate))
		limit := utf8.RuneCountInString(candidate) / 3
		if limit < 1 {
			limit = 1
		}
		if distance > limit || distance >= bestDistance {
			continue
		}
		best = candidate
		bestDistance = distance
	}
	return best
}

func editDistance(left, right []rune) int {
	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for index := range previous {
		previous[index] = index
	}
	for leftIndex, leftRune := range left {
		current[0] = leftIndex + 1
		for rightIndex, rightRune := range right {
			cost := 1
			if leftRune == rightRune {
				cost = 0
			}
			current[rightIndex+1] = minInt(
				previous[rightIndex+1]+1,
				current[rightIndex]+1,
				previous[rightIndex]+cost,
			)
		}
		previous, current = current, previous
	}
	return previous[len(right)]
}

func minInt(values ...int) int {
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}
