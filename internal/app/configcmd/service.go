// Package configcmd owns the local command-selection use cases.
package configcmd

import (
	"context"
	"errors"

	"github.com/tasuku43/cwk/internal/app/execution"
	"github.com/tasuku43/cwk/internal/app/portcheck"
	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

const (
	// EditCommand is the exact catalog path that owns the fixed local profile.
	EditCommand = "config edit"
	// FixedTargetKind and FixedTargetID identify the command-owned singleton.
	FixedTargetKind = "command-selection"
	FixedTargetID   = "default"
)

var editImpact = operation.Impact{
	Cardinality:  operation.CardinalityOne,
	Notification: operation.DeclarationNo,
	AccessChange: operation.DeclarationNo,
	Destructive:  operation.DeclarationNo,
}

// StorePort is the smallest persistence capability needed by command
// selection. Configured distinguishes an absent preference from a deliberately
// saved empty allowlist.
type StorePort interface {
	Load(context.Context) (profile commandselection.Profile, configured bool, err error)
	Save(context.Context, commandselection.Profile) error
}

// Service validates profile values and keeps filesystem concerns behind the
// application-owned port.
type Service struct {
	store StorePort
}

// New constructs a command-selection service.
func New(store StorePort) *Service {
	return &Service{store: store}
}

// Load returns the persisted profile and whether the user has saved one.
func (s *Service) Load(ctx context.Context) (commandselection.Profile, bool, error) {
	if ctx == nil {
		return commandselection.Profile{}, false, fault.New(fault.KindContract, "missing_context", "command selection context is not configured", false)
	}
	if err := ctx.Err(); err != nil {
		return commandselection.Profile{}, false, canceled(err)
	}
	if s == nil || portcheck.IsNil(s.store) {
		return commandselection.Profile{}, false, unavailable(nil)
	}

	profile, configured, err := s.store.Load(ctx)
	if err != nil {
		if structured, ok := fault.PublicCopy(err); ok {
			return commandselection.Profile{}, false, structured
		}
		if contextErr := ctx.Err(); contextErr != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			if contextErr == nil {
				contextErr = err
			}
			return commandselection.Profile{}, false, canceled(contextErr)
		}
		return commandselection.Profile{}, false, fault.Wrap(
			fault.KindUnavailable,
			"command_selection_unavailable",
			"command selection is unavailable",
			true,
			err,
		)
	}
	if err := profile.Validate(); err != nil || (!configured && len(profile.EnabledCommands()) != 0) {
		return commandselection.Profile{}, false, fault.New(
			fault.KindInvalidInput,
			"command_selection_invalid",
			"command selection is invalid",
			false,
		)
	}
	if contextErr := ctx.Err(); contextErr != nil {
		return commandselection.Profile{}, false, canceled(contextErr)
	}
	return profile, configured, nil
}

// Save validates and delegates one explicit profile replacement. Callers must
// put this method behind execution.Invoker with ExplicitSavePolicy. Errors from
// the store are intentionally not wrapped: an adapter can return a raw error
// after an uncertain replacement attempt so Invoker classifies the outcome as
// unclassified instead of claiming the previous file is intact.
func (s *Service) Save(ctx context.Context, profile commandselection.Profile) error {
	if ctx == nil {
		return fault.New(fault.KindContract, "missing_context", "command selection context is not configured", false)
	}
	if err := ctx.Err(); err != nil {
		return canceled(err)
	}
	if err := profile.Validate(); err != nil {
		return fault.Wrap(fault.KindInvalidInput, "command_selection_invalid", "command selection is invalid", false, err)
	}
	if s == nil || portcheck.IsNil(s.store) {
		return unavailable(nil)
	}
	return s.store.Save(ctx, profile)
}

// EditIntent returns the exact fixed-target write intent owned by config edit.
func EditIntent() operation.Intent {
	return operation.Intent{
		Command: EditCommand,
		Effect:  operation.EffectWrite,
		Target: operation.TargetRef{
			Kind: FixedTargetKind,
			ID:   FixedTargetID,
		},
		Impact: editImpact,
	}
}

// EditRequest binds EditIntent to the execution boundary's expected catalog
// declaration.
func EditRequest() execution.Request {
	intent := EditIntent()
	return execution.Request{
		Intent:          intent,
		ExpectedCommand: intent.Command,
		ExpectedEffect:  intent.Effect,
		ExpectedTarget:  intent.Target,
		ExpectedImpact:  intent.Impact,
	}
}

// ExplicitSavePolicy admits only the exact config-edit singleton mutation and
// only when the selector observed its literal save token. It is an execution
// invariant, not a security or human-authority claim.
type ExplicitSavePolicy struct {
	Confirmed bool
}

// Check implements execution.Policy.
func (p ExplicitSavePolicy) Check(ctx context.Context, intent operation.Intent) error {
	if ctx == nil {
		return errors.New("command selection confirmation context is missing")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.Confirmed {
		return errors.New("literal save confirmation is required")
	}
	if intent != EditIntent() {
		return errors.New("command selection confirmation requires the exact config edit intent")
	}
	return nil
}

func canceled(cause error) error {
	return fault.Wrap(
		fault.KindCanceled,
		"operation_canceled",
		"command selection operation was canceled",
		true,
		cause,
	)
}

func unavailable(cause error) error {
	return fault.Wrap(
		fault.KindUnavailable,
		"command_selection_unavailable",
		"command selection is unavailable",
		true,
		cause,
	)
}

var _ execution.Policy = ExplicitSavePolicy{}
