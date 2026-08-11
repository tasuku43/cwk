package cli

import (
	"strings"
	"testing"
)

func TestUnknownFlagErrorSuggestsOneNearbyDeclaredFlag(t *testing.T) {
	err := unknownFlagError("--coutn", []string{"--room", "--count", "--context"})
	if err == nil || !strings.Contains(err.Error(), `もしかして "--count"`) {
		t.Fatalf("unknownFlagError() = %v", err)
	}
}

func TestClosestFlagDeclinesDistantOrNonFlagInputs(t *testing.T) {
	for _, name := range []string{"query", "--unrelated"} {
		if got := closestFlag(name, []string{"--room", "--count"}); got != "" {
			t.Errorf("closestFlag(%q) = %q, want no suggestion", name, got)
		}
	}
}

func TestClosestFlagIsDeterministicAcrossCandidateOrderAndDuplicates(t *testing.T) {
	first := closestFlag("--cont", []string{"--context", "--count", "--count"})
	second := closestFlag("--cont", []string{"--count", "--context"})
	if first != "--count" || second != first {
		t.Fatalf("closestFlag() = %q / %q, want deterministic --count", first, second)
	}
}
