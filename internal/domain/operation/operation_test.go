package operation

import (
	"encoding/json"
	"strings"
	"testing"
)

func declaredSingleImpact() Impact {
	return Impact{
		Cardinality:  CardinalityOne,
		Notification: DeclarationNo,
		AccessChange: DeclarationNo,
		Destructive:  DeclarationNo,
	}
}

func TestEffectZeroValueFailsClosed(t *testing.T) {
	if err := EffectUnknown.Validate(); err == nil {
		t.Fatal("EffectUnknown.Validate() succeeded")
	}
	if err := Effect(99).Validate(); err == nil {
		t.Fatal("unknown Effect.Validate() succeeded")
	}
}

func TestIntentValidatesEffectSpecificTargets(t *testing.T) {
	tests := []struct {
		name    string
		intent  Intent
		wantErr bool
	}{
		{
			name:   "read without target",
			intent: Intent{Command: "doctor", Effect: EffectRead},
		},
		{
			name: "create with parent scope",
			intent: Intent{
				Command: "items create",
				Effect:  EffectCreate,
				Target:  TargetRef{Kind: "item", ParentID: "collection-1"},
				Impact:  declaredSingleImpact(),
			},
		},
		{
			name: "write with object ID",
			intent: Intent{
				Command: "items update",
				Effect:  EffectWrite,
				Target:  TargetRef{Kind: "item", ParentID: "collection-1", ID: "item-1"},
				Impact:  declaredSingleImpact(),
			},
		},
		{
			name:    "unknown effect",
			intent:  Intent{Command: "doctor"},
			wantErr: true,
		},
		{
			name:    "read with target",
			intent:  Intent{Command: "doctor", Effect: EffectRead, Target: TargetRef{Kind: "system", ID: "local"}},
			wantErr: true,
		},
		{
			name:    "create without parent",
			intent:  Intent{Command: "items create", Effect: EffectCreate, Target: TargetRef{Kind: "item"}},
			wantErr: true,
		},
		{
			name:    "write without object ID",
			intent:  Intent{Command: "items update", Effect: EffectWrite, Target: TargetRef{Kind: "item"}},
			wantErr: true,
		},
		{
			name: "write with line separator in opaque ID",
			intent: Intent{
				Command: "items update", Effect: EffectWrite,
				Target: TargetRef{Kind: "item", ID: "item\u2028unsafe"}, Impact: declaredSingleImpact(),
			},
			wantErr: true,
		},
		{
			name: "create with paragraph separator in parent",
			intent: Intent{
				Command: "items create", Effect: EffectCreate,
				Target: TargetRef{Kind: "item", ParentID: "parent\u2029unsafe"}, Impact: declaredSingleImpact(),
			},
			wantErr: true,
		},
		{
			name: "mutation without impact declaration",
			intent: Intent{
				Command: "items update",
				Effect:  EffectWrite,
				Target:  TargetRef{Kind: "item", ID: "item-1"},
			},
			wantErr: true,
		},
		{
			name: "mutation with one omitted impact dimension",
			intent: Intent{
				Command: "items update",
				Effect:  EffectWrite,
				Target:  TargetRef{Kind: "item", ID: "item-1"},
				Impact: Impact{
					Cardinality:  CardinalityOne,
					Notification: DeclarationNo,
					AccessChange: DeclarationNo,
				},
			},
			wantErr: true,
		},
		{
			name: "read with mutation impact",
			intent: Intent{
				Command: "items list",
				Effect:  EffectRead,
				Impact:  declaredSingleImpact(),
			},
			wantErr: true,
		},
		{
			name:    "invalid command path",
			intent:  Intent{Command: "Items Update", Effect: EffectRead},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.intent.Validate()
			if test.wantErr && err == nil {
				t.Fatal("Validate() succeeded")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestIntentJSONUsesStableSemanticEnumNames(t *testing.T) {
	intent := Intent{
		Command: "items update",
		Effect:  EffectWrite,
		Target:  TargetRef{Kind: "item", ID: "item-1"},
		Impact:  declaredSingleImpact(),
	}
	data, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{
		`"effect":"write"`,
		`"cardinality":"one"`,
		`"notification":"no"`,
		`"access_change":"no"`,
		`"destructive":"no"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("JSON = %s, want %s", got, want)
		}
	}

	if _, err := json.Marshal(Intent{Command: "items list", Effect: EffectUnknown}); err == nil {
		t.Fatal("JSON marshal accepted an unknown effect")
	}

	var decoded Intent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON did not round trip: %v", err)
	}
	if decoded.Impact != intent.Impact {
		t.Fatalf("decoded impact = %+v, want %+v", decoded.Impact, intent.Impact)
	}
	for _, malformed := range []string{
		`{"cardinality":"unknown","notification":"no","access_change":"no","destructive":"no"}`,
		`{"cardinality":"one","notification":"maybe","access_change":"no","destructive":"no"}`,
	} {
		var impact Impact
		if err := json.Unmarshal([]byte(malformed), &impact); err == nil {
			t.Fatalf("JSON unmarshal accepted %s", malformed)
		}
	}
}

func TestIntentPreservesOrderedPreflightReferencesWithoutChangingMutationTarget(t *testing.T) {
	intent := Intent{
		Command: "messages send",
		Effect:  EffectCreate,
		Target:  TargetRef{Kind: "chatwork-message", ParentID: "42"},
		Impact:  declaredSingleImpact(),
		PreflightRefs: []PreflightRef{
			{Input: "--to", Kind: "account", Value: "9"},
			{Input: "--to", Kind: "account", Value: "3"},
			{Input: "--reply-to", Kind: "message", Value: "101"},
		},
	}
	if err := intent.Validate(); err != nil {
		t.Fatalf("valid typed intent: %v", err)
	}
	if got := intent.Target; got != (TargetRef{Kind: "chatwork-message", ParentID: "42"}) {
		t.Fatalf("preflight refs changed mutation parent target: %+v", got)
	}

	cloned := intent
	cloned.PreflightRefs = append([]PreflightRef(nil), intent.PreflightRefs...)
	if !intent.Equal(cloned) {
		t.Fatal("equal ordered intents did not compare equal")
	}
	cloned.PreflightRefs[0], cloned.PreflightRefs[1] = cloned.PreflightRefs[1], cloned.PreflightRefs[0]
	if intent.Equal(cloned) {
		t.Fatal("preflight reference order was ignored")
	}

	for name, ref := range map[string]PreflightRef{
		"missing input": {Kind: "account", Value: "9"},
		"invalid input": {Input: "to", Kind: "account", Value: "9"},
		"missing kind":  {Input: "--to", Value: "9"},
		"invalid value": {Input: "--to", Kind: "account", Value: "9\n"},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := intent
			invalid.PreflightRefs = []PreflightRef{ref}
			if err := invalid.Validate(); err == nil {
				t.Fatal("invalid preflight reference passed intent validation")
			}
		})
	}
	duplicate := intent
	duplicate.PreflightRefs = []PreflightRef{{Input: "--to", Kind: "account", Value: "9"}, {Input: "--to", Kind: "account", Value: "9"}}
	if err := duplicate.Validate(); err == nil {
		t.Fatal("duplicate preflight reference passed intent validation")
	}
	read := Intent{Command: "messages list", Effect: EffectRead, PreflightRefs: intent.PreflightRefs}
	if err := read.Validate(); err == nil {
		t.Fatal("read intent with mutation preflight refs passed validation")
	}
}
