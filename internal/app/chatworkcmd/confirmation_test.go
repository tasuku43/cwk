package chatworkcmd

import (
	"context"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/operation"
)

func confirmationIntent() operation.Intent {
	return operation.Intent{
		Command: "messages delete", Effect: operation.EffectWrite,
		Target: operation.TargetRef{Kind: "chatwork-message", ID: "12"},
		Impact: operation.Impact{Cardinality: operation.CardinalityOne, Notification: operation.DeclarationNo, AccessChange: operation.DeclarationNo, Destructive: operation.DeclarationYes},
	}
}

func TestConfirmationPolicyRequiresExactLiteral(t *testing.T) {
	for name, policy := range map[string]ConfirmationPolicy{
		"missing": {Required: "destructive"},
		"wrong":   {Required: "destructive", Provided: "access-change"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := policy.Check(context.Background(), confirmationIntent()); err == nil {
				t.Fatal("Check() succeeded")
			}
		})
	}
	if err := (ConfirmationPolicy{Required: "destructive", Provided: "destructive"}).Check(context.Background(), confirmationIntent()); err != nil {
		t.Fatal(err)
	}
}

func TestConfirmationPolicyAllowsOnlyEmptyConfirmationForOrdinaryMutation(t *testing.T) {
	if err := (ConfirmationPolicy{}).Check(context.Background(), confirmationIntent()); err != nil {
		t.Fatal(err)
	}
	if err := (ConfirmationPolicy{Provided: "destructive"}).Check(context.Background(), confirmationIntent()); err == nil {
		t.Fatal("unexpected confirmation succeeded")
	}
}
