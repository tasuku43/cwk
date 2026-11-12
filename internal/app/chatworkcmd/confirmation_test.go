package chatworkcmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/fault"
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
	for name, test := range map[string]struct {
		policy   ConfirmationPolicy
		expected string
		private  string
	}{
		"missing":                {policy: ConfirmationPolicy{Required: "destructive"}, expected: "--confirm=destructive"},
		"wrong":                  {policy: ConfirmationPolicy{Required: "access-change", Provided: "destructive"}, expected: "--confirm=access-change"},
		"hostile provided value": {policy: ConfirmationPolicy{Required: "destructive", Provided: "private-confirmation-canary"}, expected: "--confirm=destructive", private: "private-confirmation-canary"},
	} {
		t.Run(name, func(t *testing.T) {
			err := test.policy.Check(context.Background(), confirmationIntent())
			if err == nil {
				t.Fatal("Check() succeeded")
			}
			var structured *fault.Error
			if !errors.As(err, &structured) || structured.Kind != fault.KindRejected || structured.Code != "mutation_rejected" ||
				structured.Retryable || !strings.Contains(structured.Message, test.expected) ||
				(test.private != "" && strings.Contains(structured.Message, test.private)) {
				t.Fatalf("Check() error = %#v, want safe exact confirmation refusal %q", err, test.expected)
			}
		})
	}
	if err := (ConfirmationPolicy{Required: "destructive", Provided: "destructive"}).Check(context.Background(), confirmationIntent()); err != nil {
		t.Fatal(err)
	}
}

func TestConfirmationPolicyDoesNotReflectUnknownRequiredValue(t *testing.T) {
	const privateCanary = "private-required-confirmation-canary"
	err := (ConfirmationPolicy{Required: privateCanary, Provided: "wrong"}).Check(context.Background(), confirmationIntent())
	if err == nil || strings.Contains(err.Error(), privateCanary) {
		t.Fatalf("Check() error = %v, want generic non-reflecting refusal", err)
	}
	var structured *fault.Error
	if errors.As(err, &structured) {
		t.Fatalf("unknown required value produced a public fault: %+v", structured)
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
