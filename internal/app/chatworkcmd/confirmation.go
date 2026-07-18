package chatworkcmd

import (
	"context"
	"fmt"

	"github.com/tasuku43/cwk/internal/domain/operation"
)

// ConfirmationPolicy is the project-selected automation policy. An exact
// command and typed intent authorize ordinary mutations; access-changing and
// destructive tasks additionally require their one literal confirmation.
type ConfirmationPolicy struct {
	Required string
	Provided string
}

func (p ConfirmationPolicy) Check(ctx context.Context, intent operation.Intent) error {
	if ctx == nil {
		return fmt.Errorf("confirmation context is missing")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := intent.Validate(); err != nil {
		return err
	}
	if p.Required == "" {
		if p.Provided != "" {
			return fmt.Errorf("confirmation is not accepted for this ordinary mutation")
		}
		return nil
	}
	if p.Provided != p.Required {
		return fmt.Errorf("exact %s confirmation is required", p.Required)
	}
	return nil
}
