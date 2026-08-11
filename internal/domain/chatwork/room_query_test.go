package chatwork

import (
	"strings"
	"testing"
)

func TestContainsFoldUsesUnicodeSimpleFoldingWithoutNormalization(t *testing.T) {
	for _, test := range []struct {
		name  string
		text  string
		query string
		want  bool
	}{
		{name: "ASCII case", text: "Platform COST Review", query: "cost", want: true},
		{name: "Japanese", text: "コスト管理定例", query: "管理", want: true},
		{name: "Unicode simple fold", text: "ΟΣ", query: "ος", want: true},
		{name: "no normalization", text: "Cafe\u0301 Review", query: "fé", want: false},
		{name: "no whitespace folding", text: "Cost  Review", query: "cost review", want: false},
		{name: "not found", text: "Security Review", query: "cost", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ContainsFold(test.text, test.query); got != test.want {
				t.Fatalf("ContainsFold(%q, %q) = %t, want %t", test.text, test.query, got, test.want)
			}
		})
	}
}

func TestRoomQueryRequestValidation(t *testing.T) {
	for _, query := range []string{"x", "コスト", strings.Repeat("x", 255)} {
		if err := (Request{Task: TaskRoomsList, RoomQuery: query}).Validate(); err != nil {
			t.Fatalf("valid query length %d: %v", len(query), err)
		}
	}

	for name, request := range map[string]Request{
		"too long":     {Task: TaskRoomsList, RoomQuery: strings.Repeat("x", 256)},
		"invalid UTF8": {Task: TaskRoomsList, RoomQuery: string([]byte{0xff})},
		"NUL":          {Task: TaskRoomsList, RoomQuery: "cost\x00"},
		"wrong task":   {Task: TaskRoomsShow, RoomQuery: "cost"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := request.Validate(); err == nil {
				t.Fatal("request unexpectedly validated")
			}
		})
	}
}

func TestQueriedRoomResultAcceptsZeroOneAndManyCandidates(t *testing.T) {
	source := []Room{
		{Ref: roomQueryReference(t, "1"), Name: "Cost Review"},
		{Ref: roomQueryReference(t, "2"), Name: "COST Planning"},
		{Ref: roomQueryReference(t, "3"), Name: "コスト管理"},
	}
	for name, test := range map[string]struct {
		query      string
		candidates []Room
	}{
		"zero": {query: "security", candidates: []Room{}},
		"one":  {query: "コスト", candidates: []Room{source[2]}},
		"many": {query: "cost", candidates: source[:2]},
	} {
		t.Run(name, func(t *testing.T) {
			result := Result{
				Task:     TaskRoomsList,
				Coverage: Coverage{Kind: "provider_collection", Complete: true},
				Rooms:    test.candidates,
				RoomSelection: &RoomSelection{
					Query: test.query, SourceCount: len(source), CandidateCount: len(test.candidates),
				},
			}
			if err := result.ValidateFor(Request{Task: TaskRoomsList, RoomQuery: test.query}); err != nil {
				t.Fatalf("queried room result: %v", err)
			}
		})
	}
}

func TestQueriedRoomResultRejectsInvalidProvenance(t *testing.T) {
	room := Room{Ref: roomQueryReference(t, "1"), Name: "Cost Review"}
	base := Result{
		Task:          TaskRoomsList,
		Coverage:      Coverage{Kind: "provider_collection", Complete: true},
		Rooms:         []Room{room},
		RoomSelection: &RoomSelection{Query: "cost", SourceCount: 2, CandidateCount: 1},
	}
	for name, mutate := range map[string]func(*Result){
		"candidate count": func(result *Result) { result.RoomSelection.CandidateCount = 0 },
		"source count":    func(result *Result) { result.RoomSelection.SourceCount = 0 },
		"incomplete":      func(result *Result) { result.Coverage.Complete = false },
		"nonmatch":        func(result *Result) { result.Rooms[0].Name = "Security" },
		"query mismatch":  func(result *Result) { result.RoomSelection.Query = "security" },
	} {
		t.Run(name, func(t *testing.T) {
			result := base
			result.Rooms = append([]Room(nil), base.Rooms...)
			selection := *base.RoomSelection
			result.RoomSelection = &selection
			mutate(&result)
			if err := result.ValidateFor(Request{Task: TaskRoomsList, RoomQuery: "cost"}); err == nil {
				t.Fatal("invalid queried room result unexpectedly validated")
			}
		})
	}

	if err := base.ValidateFor(Request{Task: TaskRoomsList}); err == nil {
		t.Fatal("unqueried request accepted queried provenance")
	}
}

func roomQueryReference(t *testing.T, value string) Reference {
	t.Helper()
	ref, err := NewReference(ReferenceRoom, value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
