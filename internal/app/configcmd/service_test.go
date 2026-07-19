package configcmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/app/execution"
	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

type storeStub struct {
	profile    commandselection.Profile
	configured bool
	loadErr    error
	saveErr    error
	loadCalls  int
	saveCalls  int
	saved      []string
}

func (s *storeStub) Load(context.Context) (commandselection.Profile, bool, error) {
	s.loadCalls++
	return s.profile, s.configured, s.loadErr
}

func (s *storeStub) Save(_ context.Context, profile commandselection.Profile) error {
	s.saveCalls++
	s.saved = profile.EnabledCommands()
	return s.saveErr
}

type typedNilStore struct{}

func (*typedNilStore) Load(context.Context) (commandselection.Profile, bool, error) {
	panic("typed nil store must not be called")
}

func (*typedNilStore) Save(context.Context, commandselection.Profile) error {
	panic("typed nil store must not be called")
}

func TestServiceLoadsConfiguredAndMissingProfiles(t *testing.T) {
	configuredProfile, err := commandselection.New([]string{"rooms list"})
	if err != nil {
		t.Fatal(err)
	}
	for name, stub := range map[string]*storeStub{
		"configured": {profile: configuredProfile, configured: true},
		"missing":    {},
	} {
		t.Run(name, func(t *testing.T) {
			profile, configured, err := New(stub).Load(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if configured != stub.configured || stub.loadCalls != 1 {
				t.Fatalf("configured=%t calls=%d", configured, stub.loadCalls)
			}
			if got, want := len(profile.EnabledCommands()), len(stub.profile.EnabledCommands()); got != want {
				t.Fatalf("enabled count=%d want=%d", got, want)
			}
		})
	}
}

func TestServiceRejectsInvalidLoadContractAndMissingStore(t *testing.T) {
	profile, _ := commandselection.New([]string{"rooms list"})
	tests := map[string]*Service{
		"nil service":       nil,
		"nil store":         New(nil),
		"typed nil store":   New((*typedNilStore)(nil)),
		"unconfigured data": New(&storeStub{profile: profile, configured: false}),
	}
	for name, service := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, err := service.Load(context.Background())
			if name == "unconfigured data" {
				assertFault(t, err, fault.KindInvalidInput, "command_selection_invalid")
			} else {
				assertFault(t, err, fault.KindUnavailable, "command_selection_unavailable")
			}
		})
	}
}

func TestServiceSanitizesRawLoadErrorAndPreservesStructuredFault(t *testing.T) {
	const canary = "private-path-canary"
	_, _, err := New(&storeStub{loadErr: errors.New(canary)}).Load(context.Background())
	assertFault(t, err, fault.KindUnavailable, "command_selection_unavailable")
	if errors.Unwrap(err) == nil || err.Error() == canary {
		t.Fatalf("raw load error was not safely wrapped: %#v", err)
	}

	upstream := fault.Wrap(fault.KindInvalidInput, "command_selection_invalid", "command selection is invalid", false, errors.New(canary))
	_, _, err = New(&storeStub{loadErr: upstream}).Load(context.Background())
	assertFault(t, err, fault.KindInvalidInput, "command_selection_invalid")
	if errors.Unwrap(err) != nil {
		t.Fatalf("structured load fault retained private cause: %#v", err)
	}
}

func TestServiceSaveDelegatesAndPreservesAdapterOutcome(t *testing.T) {
	profile, _ := commandselection.New([]string{"messages list", "rooms list"})
	const canary = "post-replace-canary"
	adapterErr := errors.New(canary)
	store := &storeStub{saveErr: adapterErr}
	err := New(store).Save(context.Background(), profile)
	if !errors.Is(err, adapterErr) || store.saveCalls != 1 {
		t.Fatalf("Save error=%v calls=%d", err, store.saveCalls)
	}
	if got := len(store.saved); got != 2 {
		t.Fatalf("saved enabled count=%d", got)
	}

	invokedErr := execution.New(ExplicitSavePolicy{Confirmed: true}).Invoke(
		context.Background(),
		Request(),
		func(ctx context.Context, _ operation.Intent) error {
			return New(&storeStub{saveErr: adapterErr}).Save(ctx, profile)
		},
	)
	assertFault(t, invokedErr, fault.KindContract, "unclassified_mutation_outcome")
	if strings.Contains(invokedErr.Error(), canary) || errors.Unwrap(invokedErr) != nil {
		t.Fatalf("execution boundary leaked uncertain adapter outcome: %#v", invokedErr)
	}
}

func TestServiceCancellationMakesZeroStoreCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := &storeStub{}
	_, _, loadErr := New(store).Load(ctx)
	assertFault(t, loadErr, fault.KindCanceled, "operation_canceled")
	saveErr := New(store).Save(ctx, commandselection.Profile{})
	assertFault(t, saveErr, fault.KindCanceled, "operation_canceled")
	if store.loadCalls != 0 || store.saveCalls != 0 {
		t.Fatalf("canceled calls: load=%d save=%d", store.loadCalls, store.saveCalls)
	}
}

func TestExplicitSavePolicyAndRequestPinExactFixedIntent(t *testing.T) {
	request := Request()
	if err := request.Intent.Validate(); err != nil {
		t.Fatalf("Intent invalid: %v", err)
	}
	if request.Intent.Command != "config" || request.Intent.Effect != operation.EffectWrite ||
		request.Intent.Target != (operation.TargetRef{Kind: "command-selection", ID: "default"}) ||
		request.Intent.Impact.Cardinality != operation.CardinalityOne ||
		request.Intent.Impact.Notification != operation.DeclarationNo ||
		request.Intent.Impact.AccessChange != operation.DeclarationNo ||
		request.Intent.Impact.Destructive != operation.DeclarationNo {
		t.Fatalf("Request = %+v", request)
	}

	if err := (ExplicitSavePolicy{}).Check(context.Background(), request.Intent); err == nil {
		t.Fatal("unconfirmed save was admitted")
	}
	policy := ExplicitSavePolicy{Confirmed: true}
	if err := policy.Check(context.Background(), request.Intent); err != nil {
		t.Fatalf("confirmed exact save rejected: %v", err)
	}
	wrong := request.Intent
	wrong.Target.ID = "other"
	if err := policy.Check(context.Background(), wrong); err == nil {
		t.Fatal("wrong fixed target was admitted")
	}

	store := &storeStub{}
	profile, _ := commandselection.New([]string{"rooms list"})
	err := execution.New(policy).Invoke(context.Background(), request, func(ctx context.Context, _ operation.Intent) error {
		return New(store).Save(ctx, profile)
	})
	if err != nil || store.saveCalls != 1 {
		t.Fatalf("invoked save error=%v calls=%d", err, store.saveCalls)
	}
}

func assertFault(t *testing.T, err error, kind fault.Kind, code string) {
	t.Helper()
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Validate() != nil || structured.Kind != kind {
		t.Fatalf("error=%#v, want valid %s fault", err, kind)
	}
	if code != "" && structured.Code != code {
		t.Fatalf("fault code=%q want=%q", structured.Code, code)
	}
}
