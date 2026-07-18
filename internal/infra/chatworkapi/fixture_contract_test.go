package chatworkapi

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

type responseFixtureCorpus struct {
	Schema     string                   `json:"schema"`
	Provenance string                   `json:"provenance"`
	Success    []responseSuccessFixture `json:"success"`
	Invalid    []responseInvalidFixture `json:"invalid"`
	Error      json.RawMessage          `json:"error_envelope"`
}

type responseSuccessFixture struct {
	Task   chatwork.Task   `json:"task"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

type responseInvalidFixture struct {
	Name        string        `json:"name"`
	Task        chatwork.Task `json:"task"`
	EncodedBody string        `json:"encoded_body"`
}

func TestSyntheticResponseCorpusCoversEveryTaskThroughCandidateC(t *testing.T) {
	corpus := readResponseFixtureCorpus(t)
	if corpus.Schema != "cwk-chatwork-api-v2-response-fixtures/1" || corpus.Provenance == "" {
		t.Fatalf("fixture metadata is incomplete: %+v", corpus)
	}
	if len(corpus.Success) != 33 {
		t.Fatalf("success fixtures = %d, want 33 task outcomes over 32 operations", len(corpus.Success))
	}

	seen := make(map[chatwork.Task]struct{}, len(corpus.Success))
	for _, fixture := range corpus.Success {
		fixture := fixture
		t.Run(string(fixture.Task), func(t *testing.T) {
			if !fixture.Task.Valid() {
				t.Fatalf("fixture task %q is invalid", fixture.Task)
			}
			if _, duplicate := seen[fixture.Task]; duplicate {
				t.Fatalf("duplicate fixture task %q", fixture.Task)
			}
			seen[fixture.Task] = struct{}{}

			request := completeRequest(fixture.Task)
			var result chatwork.Result
			var err error
			switch fixture.Status {
			case 200:
				result, err = mapResponse(request, fixture.Body)
			case 204:
				if !allowsNoContent(fixture.Task) {
					t.Fatalf("task %q does not declare a 204 response", fixture.Task)
				}
				result = emptyResult(request)
			default:
				t.Fatalf("fixture status = %d", fixture.Status)
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Task != fixture.Task {
				t.Fatalf("result task = %q, want %q", result.Task, fixture.Task)
			}
			if err := result.Validate(); err != nil {
				t.Fatalf("task-specific semantic result is invalid: %v", err)
			}
		})
	}
}

func TestSyntheticResponseCorpusPinsRejectedWireShapes(t *testing.T) {
	corpus := readResponseFixtureCorpus(t)
	for _, fixture := range corpus.Invalid {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			if _, err := mapResponse(completeRequest(fixture.Task), []byte(fixture.EncodedBody)); err == nil {
				t.Fatal("invalid wire response was accepted")
			}
		})
	}
	if !bytes.Contains(corpus.Error, []byte("Synthetic upstream detail")) {
		t.Fatal("representative provider error envelope is missing")
	}
}

func readResponseFixtureCorpus(t *testing.T) responseFixtureCorpus {
	t.Helper()
	data, err := os.ReadFile("testdata/chatwork-api-v2.responses.json")
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var corpus responseFixtureCorpus
	if err := decoder.Decode(&corpus); err != nil {
		t.Fatal(err)
	}
	if decoder.Decode(&struct{}{}) == nil {
		t.Fatal("fixture corpus contains a trailing JSON value")
	}
	return corpus
}
