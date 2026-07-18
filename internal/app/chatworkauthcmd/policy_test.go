package chatworkauthcmd

import (
	"context"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/operation"
)

func TestExactProfilePolicyAcceptsOnlyMatchingCreateAndWriteTarget(t *testing.T) {
	ref := exactRef(t)
	policy := ExactProfilePolicy{Profile: ref}
	create := operation.Intent{
		Command: "auth login", Effect: operation.EffectCreate,
		Target: operation.TargetRef{Kind: "chatwork-oauth-credential", ParentID: ref.Value()},
		Impact: operation.Impact{Cardinality: operation.CardinalityOne, Notification: operation.DeclarationNo, AccessChange: operation.DeclarationYes, Destructive: operation.DeclarationNo},
	}
	if err := policy.Check(context.Background(), create); err != nil {
		t.Fatalf("create rejected: %v", err)
	}
	write := create
	write.Command = "auth logout"
	write.Effect = operation.EffectWrite
	write.Target = operation.TargetRef{Kind: "chatwork-oauth-profile", ID: ref.Value()}
	write.Impact.Destructive = operation.DeclarationYes
	if err := policy.Check(context.Background(), write); err != nil {
		t.Fatalf("write rejected: %v", err)
	}

	write.Target.ID = "other"
	if err := policy.Check(context.Background(), write); err == nil {
		t.Fatal("mismatched write target accepted")
	}
}
