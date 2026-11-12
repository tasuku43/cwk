//go:build readiness_eval

package main

import (
	"encoding/json"
	"os"
	"testing"
)

// The candidate schema fixes answer shape only. It must not expose expected
// values, and its shape must remain compatible with the separate answer key.
func TestReadinessSemanticSchemaMatchesKeyWithoutAnswerValues(t *testing.T) {
	data, err := os.ReadFile("testdata/semantic_answer.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	var answer any
	if err := json.Unmarshal([]byte(readinessSemanticAnswerKey), &answer); err != nil {
		t.Fatal(err)
	}
	checkReadinessSchemaNode(t, schema, answer)
}

func checkReadinessSchemaNode(t *testing.T, schema map[string]any, answer any) {
	t.Helper()
	kind, ok := schema["type"].(string)
	if !ok {
		t.Fatal("schema has no type")
	}
	if kind != "object" {
		if len(schema) != 1 {
			t.Fatalf("leaf schema includes constraints or expected values: %#v", schema)
		}
		switch kind {
		case "string":
			if _, ok := answer.(string); !ok {
				t.Fatal("answer is not string")
			}
		case "boolean":
			if _, ok := answer.(bool); !ok {
				t.Fatal("answer is not boolean")
			}
		case "integer":
			if value, ok := answer.(float64); !ok || value != float64(int64(value)) {
				t.Fatal("answer is not integer")
			}
		default:
			t.Fatalf("unsupported schema type %q", kind)
		}
		return
	}
	if len(schema) != 4 || schema["additionalProperties"] != false {
		t.Fatalf("unexpected object schema fields: %#v", schema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("missing properties")
	}
	values, ok := answer.(map[string]any)
	if !ok || len(properties) != len(values) {
		t.Fatal("schema and answer field counts differ")
	}
	required, ok := schema["required"].([]any)
	if !ok || len(required) != len(properties) {
		t.Fatal("all fields must be required")
	}
	seen := make(map[string]bool)
	for _, entry := range required {
		name, ok := entry.(string)
		if !ok || seen[name] {
			t.Fatal("invalid or duplicate required field")
		}
		seen[name] = true
	}
	for name, raw := range properties {
		child, ok := raw.(map[string]any)
		if !ok || !seen[name] {
			t.Fatalf("invalid schema field %q", name)
		}
		value, exists := values[name]
		if !exists {
			t.Fatalf("schema field %q absent from answer", name)
		}
		checkReadinessSchemaNode(t, child, value)
	}
}
