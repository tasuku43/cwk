package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/app/configcmd"
	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type diagnosticSelectionStore struct {
	profile    commandselection.Profile
	configured bool
	err        error
}

func (s diagnosticSelectionStore) Load(context.Context) (commandselection.Profile, bool, error) {
	return s.profile, s.configured, s.err
}

func (diagnosticSelectionStore) Save(context.Context, commandselection.Profile) error { return nil }

func TestCommandSelectionFingerprintIsOrderedAndUnambiguous(t *testing.T) {
	first := commandSelectionFingerprint([]string{"a", "bc"})
	if first != commandSelectionFingerprint([]string{"a", "bc"}) {
		t.Fatal("fingerprint is not deterministic")
	}
	if first == commandSelectionFingerprint([]string{"ab", "c"}) || first == commandSelectionFingerprint([]string{"bc", "a"}) {
		t.Fatal("fingerprint lost path boundaries or order")
	}
	if !strings.HasPrefix(first, "sha256:") || len(first) != len("sha256:")+64 {
		t.Fatalf("fingerprint = %q", first)
	}
}

func TestCommandSelectionDoctorCheckNormalizesLegacyLocalCommands(t *testing.T) {
	profile, err := commandselection.New([]string{"doctor", "rooms list", "version"})
	if err != nil {
		t.Fatal(err)
	}
	command := newCLI(strings.NewReader(""), &strings.Builder{}, &strings.Builder{}, DefaultCatalog(), passingInspector("unused"))
	command.commandSelection = configcmd.New(diagnosticSelectionStore{profile: profile, configured: true})
	check, present, err := command.commandSelectionDoctorCheck(context.Background())
	if err != nil || !present {
		t.Fatalf("check present=%v err=%v", present, err)
	}
	if check.Status != "warn" || !strings.Contains(check.Detail, "enabled=1") || !strings.Contains(check.Detail, "legacy=2") || !strings.Contains(check.Detail, "fingerprint=sha256:") {
		t.Fatalf("check = %+v", check)
	}
}

func TestCommandSelectionDoctorCheckFailureKeepsTheFixedDetailGrammar(t *testing.T) {
	tests := []struct {
		name       string
		loadErr    error
		wantDetail string
	}{
		{
			name:       "invalid saved content",
			loadErr:    fault.New(fault.KindInvalidInput, "command_selection_invalid", "invalid", false),
			wantDetail: "state=invalid source=saved enabled=unknown disabled=unknown stale=unknown legacy=unknown fingerprint=unavailable",
		},
		{
			name:       "unsafe storage",
			loadErr:    fault.New(fault.KindUnavailable, "command_selection_unsafe", "unsafe", false),
			wantDetail: "state=unsafe source=unknown enabled=unknown disabled=unknown stale=unknown legacy=unknown fingerprint=unavailable",
		},
		{
			name:       "unavailable storage",
			loadErr:    fault.New(fault.KindUnavailable, "command_selection_unavailable", "unavailable", true),
			wantDetail: "state=unavailable source=unknown enabled=unknown disabled=unknown stale=unknown legacy=unknown fingerprint=unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := newCLI(strings.NewReader(""), &strings.Builder{}, &strings.Builder{}, DefaultCatalog(), passingInspector("unused"))
			command.commandSelection = configcmd.New(diagnosticSelectionStore{err: test.loadErr})
			check, present, err := command.commandSelectionDoctorCheck(context.Background())
			if err != nil || !present {
				t.Fatalf("check present=%v err=%v", present, err)
			}
			if check.Name != "command-selection" || check.Status != "fail" || check.Detail != test.wantDetail {
				t.Fatalf("check = %+v, want detail %q", check, test.wantDetail)
			}
		})
	}
}
