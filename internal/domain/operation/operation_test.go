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
}
