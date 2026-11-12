//go:build readiness_eval

package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// The schema and answer key are private to this synthetic fixture. This
// validator is not connected to the disabled runner: trace provenance, process
// isolation and observation of writes still need independent verification.
//
//go:embed testdata/semantic_answer.json
var readinessSemanticAnswerKey string

const readinessSemanticAnswerLimit = 1 << 20
const readinessSemanticDepthLimit = 16

func validateReadinessSemanticAnswer(answer []byte) error {
	if len(answer) > readinessSemanticAnswerLimit {
		return errors.New("semantic answer exceeds byte limit")
	}
	actual, err := parseReadinessSemanticJSON(answer)
	if err != nil {
		return fmt.Errorf("invalid semantic answer: %w", err)
	}
	expected, err := parseReadinessSemanticJSON([]byte(readinessSemanticAnswerKey))
	if err != nil {
		return errors.New("invalid fixture answer key")
	}
	if !reflect.DeepEqual(actual, expected) {
		return errors.New("semantic answer does not match the fixture")
	}
	return nil
}

func parseReadinessSemanticJSON(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	value, err := readReadinessSemanticJSON(decoder, 0)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errors.New("semantic answer has trailing data")
	}
	return value, nil
}

func readReadinessSemanticJSON(decoder *json.Decoder, depth int) (any, error) {
	if depth > readinessSemanticDepthLimit {
		return nil, errors.New("semantic answer nesting limit exceeded")
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return token, nil
	}
	switch delim {
	case '{':
		object := make(map[string]any)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, errors.New("semantic answer object key is invalid")
			}
			if _, duplicate := object[key]; duplicate {
				return nil, errors.New("semantic answer has duplicate object keys")
			}
			value, err := readReadinessSemanticJSON(decoder, depth+1)
			if err != nil {
				return nil, err
			}
			object[key] = value
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return nil, errors.New("semantic answer object is incomplete")
		}
		return object, nil
	case '[':
		values := []any{}
		for decoder.More() {
			value, err := readReadinessSemanticJSON(decoder, depth+1)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return nil, errors.New("semantic answer array is incomplete")
		}
		return values, nil
	default:
		return nil, errors.New("semantic answer has unexpected delimiter")
	}
}
