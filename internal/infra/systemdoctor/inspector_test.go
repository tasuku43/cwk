package systemdoctor

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/doctor"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
)

func TestInspectReturnsValidOfflineRuntimeCheck(t *testing.T) {
	inspector := &Inspector{inspectToken: func() chatworkapi.TokenEnvironmentInspection {
		return tokenInspection(chatworkapi.TokenEnvironmentValid)
	}}
	report, err := inspector.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if err := report.Validate(); err != nil {
		t.Fatalf("report.Validate() error = %v", err)
	}
	if len(report.Checks) != 2 || report.Checks[0].Name != "runtime" || report.Checks[1].Name != "chatwork-token" {
		t.Fatalf("checks = %+v", report.Checks)
	}
	detail := report.Checks[0].Detail
	if !strings.Contains(detail, runtime.Version()) || !strings.Contains(detail, runtime.GOOS+"/"+runtime.GOARCH) {
		t.Fatalf("runtime detail = %q", detail)
	}
	if report.Checks[1].Status != doctor.CheckStatusPass || report.Checks[1].Detail != "state=valid source=environment name=CWK_API_TOKEN config=unsupported keychain=unsupported" {
		t.Fatalf("token check = %+v", report.Checks[1])
	}
}

func TestInspectReportsMissingAndInvalidTokenAsFailuresWithoutSecrets(t *testing.T) {
	const canary = "private-token-canary"
	for _, state := range []chatworkapi.TokenEnvironmentState{chatworkapi.TokenEnvironmentMissing, chatworkapi.TokenEnvironmentInvalid} {
		t.Run(string(state), func(t *testing.T) {
			calls := 0
			inspector := &Inspector{inspectToken: func() chatworkapi.TokenEnvironmentInspection {
				calls++
				return tokenInspection(state)
			}}
			report, err := inspector.Inspect(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if calls != 1 || len(report.Checks) != 2 {
				t.Fatalf("calls = %d, checks = %+v", calls, report.Checks)
			}
			check := report.Checks[1]
			if check.Name != "chatwork-token" || check.Status != doctor.CheckStatusFail || !strings.Contains(check.Detail, "state="+string(state)) {
				t.Fatalf("token check = %+v", check)
			}
			if strings.Contains(check.Detail, canary) {
				t.Fatalf("token check exposed canary: %q", check.Detail)
			}
		})
	}
}

func TestProductionInspectorReadsTokenEnvironmentWithoutRetainingValue(t *testing.T) {
	const canary = "synthetic-doctor-token-canary"
	t.Setenv(chatworkapi.TokenEnvironment, canary)
	report, err := New().Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Checks) != 2 || report.Checks[1].Status != doctor.CheckStatusPass {
		t.Fatalf("checks = %+v", report.Checks)
	}
	if rendered := report.Checks[1].Detail; strings.Contains(rendered, canary) {
		t.Fatalf("token check exposed token: %q", rendered)
	}
}

func TestInspectHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().Inspect(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Inspect() error = %v, want context.Canceled", err)
	}
}

func tokenInspection(state chatworkapi.TokenEnvironmentState) chatworkapi.TokenEnvironmentInspection {
	return chatworkapi.TokenEnvironmentInspection{
		State:    state,
		Source:   "environment",
		Name:     chatworkapi.TokenEnvironment,
		Config:   "unsupported",
		Keychain: "unsupported",
	}
}
