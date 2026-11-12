//go:build readiness_eval

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadinessSemanticAnswerRejectsContradictionsAndMissingFacts(t *testing.T) {
	if err := validateReadinessSemanticAnswer([]byte(readinessSemanticAnswerKey)); err != nil {
		t.Fatal(err)
	}
	for name, answer := range map[string]string{
		"contradictory extra claim": strings.TrimSuffix(strings.TrimSpace(readinessSemanticAnswerKey), "}") + `,"conclusion":"全件確認済み"}`,
		"duplicate key":             strings.Replace(readinessSemanticAnswerKey, `"complete":false`, `"complete":false,"complete":true`, 1),
		"wrong file completeness":   strings.Replace(readinessSemanticAnswerKey, `"complete":false`, `"complete":true`, 1),
		"missing task absence":      strings.Replace(readinessSemanticAnswerKey, `,"absence":"unknown"`, "", 1),
		"changed ref":               strings.Replace(readinessSemanticAnswerKey, `"44501"`, `"044501"`, 1),
		"wrong task body":           strings.Replace(readinessSemanticAnswerKey, `staging 確認後にリリース候補を公開する`, `公開しない`, 1),
		"null count":                strings.Replace(readinessSemanticAnswerKey, `"source_count":2`, `"source_count":null`, 1),
		"trailing prose":            readinessSemanticAnswerKey + " 全件確認済み",
		"second object":             readinessSemanticAnswerKey + `{}`,
		"oversized":                 strings.Repeat(" ", readinessSemanticAnswerLimit+1),
		"deep nesting":              strings.Repeat("[", readinessSemanticDepthLimit+2) + "0" + strings.Repeat("]", readinessSemanticDepthLimit+2),
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateReadinessSemanticAnswer([]byte(answer)); err == nil {
				t.Fatal("invalid answer accepted")
			}
		})
	}
}

func TestReadinessSemanticAnswerBindsEveryCategoryFact(t *testing.T) {
	for _, mutation := range []struct {
		category, field string
		replacement     any
	}{
		{"files", "source_count", 3}, {"files", "limit", 99}, {"files", "complete", true}, {"files", "absence", "absent"}, {"files", "matched", 2}, {"files", "ref", "44502"}, {"files", "name", "別資料"},
		{"tasks", "source_count", 3}, {"tasks", "limit", 99}, {"tasks", "complete", true}, {"tasks", "absence", "absent"}, {"tasks", "matched", 2}, {"tasks", "ref", "66302"}, {"tasks", "body", "別タスク"}, {"tasks", "status", "done"},
	} {
		t.Run(mutation.category+"/"+mutation.field, func(t *testing.T) {
			var answer map[string]any
			if err := json.Unmarshal([]byte(readinessSemanticAnswerKey), &answer); err != nil {
				t.Fatal(err)
			}
			answer[mutation.category].(map[string]any)[mutation.field] = mutation.replacement
			encoded, err := json.Marshal(answer)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateReadinessSemanticAnswer(encoded); err == nil {
				t.Fatal("accepted altered category fact")
			}
		})
	}
}

func TestReadinessSemanticAnswerBindsRoomMessageAndKind(t *testing.T) {
	for field, value := range map[string]string{"room_ref": "8174", "message_ref": "921005", "kind": "files"} {
		t.Run(field, func(t *testing.T) {
			var answer map[string]any
			if err := json.Unmarshal([]byte(readinessSemanticAnswerKey), &answer); err != nil {
				t.Fatal(err)
			}
			answer[field] = value
			encoded, err := json.Marshal(answer)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateReadinessSemanticAnswer(encoded); err == nil {
				t.Fatal("accepted altered top-level fact")
			}
		})
	}
}
