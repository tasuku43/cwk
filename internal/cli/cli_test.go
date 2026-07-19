package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tasuku43/cwk/internal/domain/doctor"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

type cliInspector struct {
	report doctor.Report
	err    error
	calls  int
	ctx    context.Context
}

func (i *cliInspector) Inspect(ctx context.Context) (doctor.Report, error) {
	i.calls++
	i.ctx = ctx
	return i.report, i.err
}

func newTestCLI(inspector *cliInspector) (*CLI, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := newCLI(strings.NewReader(""), stdout, stderr, DefaultCatalog(), inspector)
	return command, stdout, stderr
}

func passingInspector(detail string) *cliInspector {
	return &cliInspector{report: doctor.Report{Checks: []doctor.Check{
		{Name: "runtime", Status: doctor.CheckStatusPass, Detail: detail},
	}}}
}

func runCLI(command *CLI, args []string) int {
	return command.RunContext(context.Background(), args)
}

func TestExitCodesAreStable(t *testing.T) {
	want := []int{0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
	got := []int{ExitOK, ExitUsage, ExitInternal, ExitAuthentication, ExitPermission, ExitNotFound, ExitAmbiguous, ExitRateLimited, ExitUnavailable, ExitRejected, ExitCanceled, ExitUnsupported, ExitContract}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("exit code %d = %d, want %d", index, got[index], want[index])
		}
	}
}

func TestNoArgsReturnsStructuredUsageFailure(t *testing.T) {
	command, stdout, stderr := newTestCLI(passingInspector("unused"))
	if code := runCLI(command, nil); code != ExitUsage {
		t.Fatalf("Run(nil) code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "kind: invalid_input") || !strings.Contains(stderr.String(), "code: missing_command") ||
		!strings.Contains(stderr.String(), "next_action: cwk help") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestUnknownCommandUsesUsageExitCode(t *testing.T) {
	command, stdout, stderr := newTestCLI(passingInspector("unused"))
	if code := runCLI(command, []string{"missing"}); code != ExitUsage {
		t.Fatalf("Run(missing) code = %d, want %d", code, ExitUsage)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: unknown_command") {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestVersionOutputContract(t *testing.T) {
	command, stdout, stderr := newTestCLI(passingInspector("unused"))
	command.Version = "v1.2.3"
	command.Commit = "0123456789abcdef0123456789abcdef01234567"
	if code := runCLI(command, []string{"version"}); code != ExitOK {
		t.Fatalf("Run(version) code = %d, stderr = %q", code, stderr.String())
	}
	want := "cwk v1.2.3 (0123456789abcdef0123456789abcdef01234567)\n"
	if got := stdout.String(); got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestDoctorOutputContract(t *testing.T) {
	inspector := &cliInspector{report: doctor.Report{Checks: []doctor.Check{
		{Name: "runtime", Status: doctor.CheckStatusPass, Detail: "runtime-version\ttest/test\nlocal"},
		{Name: "configuration", Status: doctor.CheckStatusWarn, Detail: "path\\value\x1b"},
	}}}
	command, stdout, stderr := newTestCLI(inspector)
	if code := runCLI(command, []string{"doctor"}); code != ExitOK {
		t.Fatalf("Run(doctor) code = %d, stderr = %q", code, stderr.String())
	}
	want := "チェック\t状態\t詳細\n" +
		"runtime\tpass\truntime-version\\ttest/test\\nlocal\n" +
		"configuration\twarn\tpath\\\\value\\u001B\n"
	if got := stdout.String(); got != want {
		t.Fatalf("doctor output = %q, want %q", got, want)
	}
	if stderr.Len() != 0 || inspector.calls != 1 {
		t.Fatalf("stderr = %q, inspector calls = %d", stderr.String(), inspector.calls)
	}
}

func TestDoctorFailureUsesRejectedExitAndStructuredRecovery(t *testing.T) {
	inspector := &cliInspector{report: doctor.Report{Checks: []doctor.Check{
		{Name: "runtime", Status: doctor.CheckStatusFail, Detail: "unsupported"},
	}}}
	command, stdout, stderr := newTestCLI(inspector)
	if code := runCLI(command, []string{"doctor"}); code != ExitRejected {
		t.Fatalf("Run(doctor) code = %d, want %d", code, ExitRejected)
	}
	if !strings.Contains(stdout.String(), "runtime\tfail\tunsupported") || !strings.Contains(stderr.String(), "code: diagnostic_failed") {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestRunContextPropagatesExactContext(t *testing.T) {
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("trace"), "value")
	inspector := passingInspector("unused")
	command, _, stderr := newTestCLI(inspector)
	if code := command.RunContext(ctx, []string{"doctor"}); code != ExitOK {
		t.Fatalf("RunContext() code = %d, stderr = %q", code, stderr.String())
	}
	if inspector.ctx == nil || inspector.ctx.Value(contextKey("trace")) != "value" {
		t.Fatalf("inspector context = %#v", inspector.ctx)
	}
}

func TestRunContextRejectsNilContextWithoutDownstreamCall(t *testing.T) {
	inspector := passingInspector("unused")
	command, stdout, stderr := newTestCLI(inspector)
	if code := command.RunContext(nil, []string{"doctor"}); code != ExitContract {
		t.Fatalf("RunContext(nil) code = %d, want %d", code, ExitContract)
	}
	if inspector.calls != 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: missing_context") {
		t.Fatalf("calls = %d, stdout = %q, stderr = %q", inspector.calls, stdout.String(), stderr.String())
	}
}

func TestCanceledContextStopsBeforeDownstreamCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	inspector := passingInspector("unused")
	command, stdout, stderr := newTestCLI(inspector)
	if code := command.RunContext(ctx, []string{"doctor"}); code != ExitCanceled {
		t.Fatalf("RunContext() code = %d, stderr = %q", code, stderr.String())
	}
	if inspector.calls != 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: operation_canceled") {
		t.Fatalf("calls = %d, stdout = %q, stderr = %q", inspector.calls, stdout.String(), stderr.String())
	}
}

func TestCanceledContextHonorsGlobalJSONErrorFormat(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command, stdout, stderr := newTestCLI(passingInspector("unused"))
	if code := command.RunContext(ctx, []string{"--error-format=json", "doctor"}); code != ExitCanceled {
		t.Fatalf("RunContext() code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 || !json.Valid(stderr.Bytes()) || !strings.Contains(stderr.String(), `"code":"operation_canceled"`) {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestEmitChecksCancellationImmediatelyBeforeStdout(t *testing.T) {
	command, stdout, stderr := newTestCLI(passingInspector("unused"))
	ctx, cancel := context.WithCancel(context.Background())
	ctx = withCommandPath(ctx, "version")
	cancel()
	if code := command.emit(ctx, []byte("must-not-be-written\n")); code != ExitCanceled {
		t.Fatalf("emit() code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: operation_canceled") {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestJSONErrorIsStableAndDoesNotExposePlainCause(t *testing.T) {
	headerName := "Authori" + "zation"
	scheme := "Bear" + "er"
	canary := "redaction" + "-canary"
	inspector := &cliInspector{err: errors.New(headerName + ": " + scheme + " " + canary + " https://redaction.invalid")}
	command, stdout, stderr := newTestCLI(inspector)
	if code := runCLI(command, []string{"--error-format", "json", "doctor"}); code != ExitInternal {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 || strings.Contains(stderr.String(), canary) || strings.Contains(stderr.String(), "redaction.invalid") {
		t.Fatalf("stdout = %q, stderr leaked cause = %q", stdout.String(), stderr.String())
	}
	var document errorDocument
	if err := json.Unmarshal(stderr.Bytes(), &document); err != nil {
		t.Fatalf("JSON error = %v, output = %q", err, stderr.String())
	}
	if document.SchemaVersion != 1 || document.Error.Kind != "internal" || document.Error.Code != "internal_error" ||
		document.Error.RetryAfter != nil || len(document.Error.NextActions) != 1 {
		t.Fatalf("error document = %+v", document)
	}
}

func TestRateLimitTimingIsExplicitAndDoesNotAuthorizeMutationRetry(t *testing.T) {
	unknown := renderTextError(errorPayload{
		Kind: fault.KindRateLimited, Code: "chatwork_rate_limited",
		Message: "Chatwork のレート上限に達しました", Retryable: true,
	})
	if !strings.Contains(string(unknown), "retry_after: unknown") {
		t.Fatalf("rate-limit text timing = %q", unknown)
	}
	nonRateLimit := renderTextError(errorPayload{
		Kind: fault.KindUnavailable, Code: "provider_unavailable",
		Message: "provider unavailable", Retryable: true,
	})
	if !strings.Contains(string(nonRateLimit), "retry_after: none") {
		t.Fatalf("non-rate-limit text timing = %q", nonRateLimit)
	}

	unknownSpec := utilitySpec("unknown-rate")
	unknownSpec.Agent.Errors = append(unknownSpec.Agent.Errors, declaredCommandError(
		fault.KindRateLimited,
		"chatwork_rate_limited",
		true,
		"unknown-rate",
		"Retry the read only after reviewing the timing evidence.",
	))
	unknownSpec.handler = func(ctx context.Context, c *CLI, _ CommandSpec, _ operation.Intent, _ []string) int {
		return c.fail(ctx, fault.New(
			fault.KindRateLimited,
			"chatwork_rate_limited",
			"Chatwork のレート上限に達しました",
			true,
		))
	}
	var unknownStdout, unknownStderr bytes.Buffer
	unknownCatalog := NewCatalog(unknownSpec)
	if err := unknownCatalog.Validate(); err != nil {
		t.Fatalf("unknown-rate catalog = %v", err)
	}
	unknownCommand := newCLI(strings.NewReader(""), &unknownStdout, &unknownStderr, unknownCatalog, passingInspector("unused"))
	if code := runCLI(unknownCommand, []string{"--error-format=json", "unknown-rate"}); code != ExitRateLimited {
		t.Fatalf("Run() code = %d, stderr = %q", code, unknownStderr.String())
	}
	var unknownDocument errorDocument
	if err := json.Unmarshal(unknownStderr.Bytes(), &unknownDocument); err != nil {
		t.Fatalf("JSON error = %v, output = %q", err, unknownStderr.String())
	}
	if unknownDocument.Error.Code != "chatwork_rate_limited" || !unknownDocument.Error.Retryable ||
		unknownDocument.Error.RetryAfter != nil {
		t.Fatalf("unknown read rate-limit error = %+v", unknownDocument.Error)
	}
	var rawUnknown struct {
		Error map[string]json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(unknownStderr.Bytes(), &rawUnknown); err != nil {
		t.Fatalf("raw JSON error = %v", err)
	}
	retryAfter, present := rawUnknown.Error["retry_after"]
	if !present || string(retryAfter) != "null" {
		t.Fatalf("unknown rate-limit retry_after = present %t, value %s", present, retryAfter)
	}

	spec := utilitySpec("test")
	spec.Agent.Errors = append(spec.Agent.Errors, declaredCommandError(
		fault.KindRateLimited,
		"chatwork_mutation_rate_limited",
		false,
		"test",
		"Inspect the mutation contract without automatically retrying.",
	))
	spec.handler = func(ctx context.Context, c *CLI, _ CommandSpec, _ operation.Intent, _ []string) int {
		err := fault.New(
			fault.KindRateLimited,
			"chatwork_mutation_rate_limited",
			"Chatwork の変更はレート上限により拒否されました。自動再試行は行いません",
			false,
		)
		err.RetryAfter = 10 * time.Second
		return c.fail(ctx, err)
	}
	var stdout, stderr bytes.Buffer
	catalog := NewCatalog(spec)
	if err := catalog.Validate(); err != nil {
		t.Fatalf("test catalog = %v", err)
	}
	command := newCLI(strings.NewReader(""), &stdout, &stderr, catalog, passingInspector("unused"))
	if code := runCLI(command, []string{"--error-format=json", "test"}); code != ExitRateLimited {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	var document errorDocument
	if err := json.Unmarshal(stderr.Bytes(), &document); err != nil {
		t.Fatalf("JSON error = %v, output = %q", err, stderr.String())
	}
	if document.Error.Retryable || document.Error.RetryAfter == nil || *document.Error.RetryAfter != "10s" {
		t.Fatalf("mutation rate-limit error = %+v", document.Error)
	}
}

func TestFaultNormalizationPreservesValidStructuredClassificationBeforeCancellation(t *testing.T) {
	const canary = "private-deadline-canary"
	ctx := withCommandPath(context.Background(), "doctor")
	providerFault := fault.Wrap(
		fault.KindUnavailable,
		"mutation_outcome_unknown",
		"The provider did not confirm the mutation outcome.",
		false,
		fmt.Errorf("%s: %w", canary, context.DeadlineExceeded),
	)
	got := normalizeUnboundFault(ctx, providerFault)
	if got.Kind != fault.KindUnavailable || got.Code != "mutation_outcome_unknown" || got.Retryable {
		t.Fatalf("normalized structured fault = %+v", got)
	}
	if errors.Unwrap(got) != nil || errors.Is(got, context.DeadlineExceeded) || strings.Contains(got.Error(), canary) {
		t.Fatalf("normalized structured fault retained private cause: %#v", got)
	}

	invalid := fault.Wrap(fault.KindUnavailable, "INVALID", "Invalid structured fault.", false, context.DeadlineExceeded)
	if got := normalizeUnboundFault(ctx, invalid); got.Kind != fault.KindContract || got.Code != "invalid_fault_contract" {
		t.Fatalf("invalid structured fault normalized as %+v", got)
	}

	if got := normalizeUnboundFault(ctx, context.DeadlineExceeded); got.Kind != fault.KindCanceled ||
		got.Code != "operation_canceled" || !got.Retryable {
		t.Fatalf("unstructured deadline normalized as %+v", got)
	}
}

func TestRuntimeFaultMustMatchCatalogAndUsesCatalogRecovery(t *testing.T) {
	tests := []struct {
		name     string
		runtime  *fault.Error
		wantCode string
		wantExit int
	}{
		{
			name:     "catalog recovery replaces runtime prose",
			runtime:  fault.New(fault.KindInternal, "test_failed", "A test failed.", false, fault.NextAction{Command: "untrusted command", Reason: "Untrusted recovery."}),
			wantCode: "test_failed", wantExit: ExitInternal,
		},
		{
			name:     "deadline cause does not replace catalog fault",
			runtime:  fault.Wrap(fault.KindInternal, "test_failed", "A test failed.", false, context.DeadlineExceeded),
			wantCode: "test_failed", wantExit: ExitInternal,
		},
		{
			name:     "undeclared code fails closed",
			runtime:  fault.New(fault.KindInternal, "unexpected_code", "An unexpected test failed.", false),
			wantCode: "undeclared_fault_contract", wantExit: ExitContract,
		},
		{
			name:     "kind mismatch fails closed",
			runtime:  fault.New(fault.KindUnavailable, "test_failed", "A test source is unavailable.", false),
			wantCode: "undeclared_fault_contract", wantExit: ExitContract,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := utilitySpec("test")
			spec.handler = func(ctx context.Context, c *CLI, _ CommandSpec, _ operation.Intent, _ []string) int {
				return c.fail(ctx, test.runtime)
			}
			var stdout, stderr bytes.Buffer
			command := newCLI(strings.NewReader(""), &stdout, &stderr, NewCatalog(spec), passingInspector("unused"))
			if code := runCLI(command, []string{"--error-format=json", "test"}); code != test.wantExit {
				t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), `"code":"`+test.wantCode+`"`) || strings.Contains(stderr.String(), "untrusted command") {
				t.Fatalf("stderr = %q", stderr.String())
			}
			if test.wantCode == "test_failed" && !strings.Contains(stderr.String(), `"command":"test"`) {
				t.Fatalf("catalog recovery was not projected: %q", stderr.String())
			}
		})
	}
}

func TestEveryFaultKindHasStableExitCode(t *testing.T) {
	tests := map[fault.Kind]int{
		fault.KindInvalidInput: ExitUsage, fault.KindAuthentication: ExitAuthentication,
		fault.KindPermission: ExitPermission, fault.KindNotFound: ExitNotFound,
		fault.KindAmbiguous: ExitAmbiguous, fault.KindRateLimited: ExitRateLimited,
		fault.KindUnavailable: ExitUnavailable, fault.KindRejected: ExitRejected,
		fault.KindCanceled: ExitCanceled, fault.KindUnsupported: ExitUnsupported,
		fault.KindContract: ExitContract, fault.KindInternal: ExitInternal,
	}
	for kind, want := range tests {
		if got := exitCodeForKind(kind); got != want {
			t.Errorf("exitCodeForKind(%q) = %d, want %d", kind, got, want)
		}
	}
}

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	return len(data) - 1, nil
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestSuccessWriterFailureIsNotReportedAsSuccess(t *testing.T) {
	var stderr bytes.Buffer
	command := New(strings.NewReader(""), shortWriter{}, &stderr)
	if code := runCLI(command, []string{"version"}); code != ExitInternal {
		t.Fatalf("short write code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "code: output_write_failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	stderr.Reset()
	command = New(strings.NewReader(""), errorWriter{err: io.ErrClosedPipe}, &stderr)
	if code := runCLI(command, []string{"version"}); code != ExitInternal {
		t.Fatalf("write error code = %d, stderr = %q", code, stderr.String())
	}
}

func TestDoctorJSONSnapshotEscapesExternalCategoryC(t *testing.T) {
	inspector := &cliInspector{report: doctor.Report{Checks: []doctor.Check{{
		Name: "runtime", Status: doctor.CheckStatusPass, Detail: "line\nESC:\x1b bidi:\u202e",
	}}}}
	command, stdout, stderr := newTestCLI(inspector)
	if code := runCLI(command, []string{"doctor", "--format", "json"}); code != ExitOK {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	want := "{\"schema_version\":1,\"report\":[{\"check\":\"runtime\",\"status\":\"pass\",\"detail\":\"line\\\\nESC:\\\\u001B bidi:\\\\u202E\"}]}\n"
	if stdout.String() != want {
		t.Fatalf("JSON output = %q, want %q", stdout.String(), want)
	}
}

func TestDoctorOversizeReturnsNoStdout(t *testing.T) {
	inspector := &cliInspector{report: doctor.Report{Checks: []doctor.Check{{
		Name: "runtime", Status: doctor.CheckStatusPass, Detail: strings.Repeat("x", maxDoctorDetailBytes+1),
	}}}}
	command, stdout, stderr := newTestCLI(inspector)
	if code := runCLI(command, []string{"doctor"}); code != ExitContract {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: output_contract_exceeded") {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestDoctorRejectsArgumentsBeforeInspection(t *testing.T) {
	inspector := passingInspector("unused")
	command, stdout, stderr := newTestCLI(inspector)
	if code := runCLI(command, []string{"doctor", "extra"}); code != ExitUsage {
		t.Fatalf("Run(doctor extra) code = %d, want %d", code, ExitUsage)
	}
	if inspector.calls != 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "使い方: cwk doctor") {
		t.Fatalf("calls = %d, stdout = %q, stderr = %q", inspector.calls, stdout.String(), stderr.String())
	}
}

func TestE2EDoctorUsesProductionOfflineAdapter(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := New(strings.NewReader(""), &stdout, &stderr)
	if code := runCLI(command, []string{"doctor"}); code != ExitOK {
		t.Fatalf("Run(doctor) code = %d, stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	if !strings.HasPrefix(output, "チェック\t状態\t詳細\nruntime\tpass\t") {
		t.Fatalf("doctor output = %q", output)
	}
	if !strings.Contains(output, runtime.Version()) || !strings.Contains(output, runtime.GOOS+"/"+runtime.GOARCH) {
		t.Fatalf("doctor output does not describe runtime: %q", output)
	}
}

func TestEveryCatalogCommandMatchesExactlyAndHasHelp(t *testing.T) {
	for _, spec := range DefaultCatalog().Commands() {
		matched, rest, found := DefaultCatalog().Match(strings.Fields(spec.Path))
		if !found || matched.Path != spec.Path || len(rest) != 0 {
			t.Errorf("Match(%q) = %q/%v/%t, want exact match", spec.Path, matched.Path, rest, found)
		}
		inspector := passingInspector("test/test")
		command, _, stderr := newTestCLI(inspector)
		args := strings.Split(spec.Path, " ")
		args = append(args, "--help")
		if code := runCLI(command, args); code != ExitOK {
			t.Errorf("Run(%q) code = %d, stderr = %q", spec.Path, code, stderr.String())
		}
	}
}

func TestTrailingHelpAliasesMatchCanonicalHelpSelectors(t *testing.T) {
	command := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	namespaces := make(map[string]struct{})
	for _, spec := range command.catalog.Commands() {
		if strings.Contains(spec.Path, " ") {
			namespaces[commandNamespace(spec.Path)] = struct{}{}
		}
		assertHelpAliasMatchesSelector(t, strings.Fields(spec.Path))
	}
	for namespace := range namespaces {
		assertHelpAliasMatchesSelector(t, []string{namespace})
	}
}

func assertHelpAliasMatchesSelector(t *testing.T, selector []string) {
	t.Helper()
	for _, flag := range []string{"--help", "-h"} {
		var aliasOut, aliasErr bytes.Buffer
		alias := New(strings.NewReader(""), &aliasOut, &aliasErr)
		aliasArgs := append(append([]string{}, selector...), flag)
		if code := runCLI(alias, aliasArgs); code != ExitOK {
			t.Fatalf("Run(%v) code = %d, stderr = %q", aliasArgs, code, aliasErr.String())
		}

		var canonicalOut, canonicalErr bytes.Buffer
		canonical := New(strings.NewReader(""), &canonicalOut, &canonicalErr)
		canonicalArgs := append([]string{"help"}, selector...)
		if code := runCLI(canonical, canonicalArgs); code != ExitOK {
			t.Fatalf("Run(%v) code = %d, stderr = %q", canonicalArgs, code, canonicalErr.String())
		}
		if aliasOut.String() != canonicalOut.String() || aliasErr.String() != canonicalErr.String() {
			t.Fatalf("Run(%v) = %q/%q, Run(%v) = %q/%q", aliasArgs, aliasOut.String(), aliasErr.String(), canonicalArgs, canonicalOut.String(), canonicalErr.String())
		}
	}
}

func TestUnknownNamespaceHelpSuffixRemainsUnknownCommand(t *testing.T) {
	for _, args := range [][]string{{"missing", "--help"}, {"missing", "-h"}, {"rooms", "missing", "--help"}} {
		command, stdout, stderr := newTestCLI(passingInspector("unused"))
		if code := runCLI(command, args); code != ExitUsage {
			t.Errorf("Run(%v) code = %d, want %d", args, code, ExitUsage)
		}
		if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: unknown_command") ||
			!strings.Contains(stderr.String(), "next_action: cwk help") {
			t.Errorf("Run(%v) stdout = %q, stderr = %q", args, stdout.String(), stderr.String())
		}
	}
}

func TestRootAliasesUseCatalogCommands(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"--help"}, want: "Chatwork CLI\n"},
		{args: []string{"-h"}, want: "Chatwork CLI\n"},
		{args: []string{"--version"}, want: "cwk dev\n"},
		{args: []string{"-v"}, want: "cwk dev\n"},
	}
	for _, test := range tests {
		command, stdout, stderr := newTestCLI(passingInspector("unused"))
		if code := runCLI(command, test.args); code != ExitOK {
			t.Errorf("Run(%v) code = %d, stderr = %q", test.args, code, stderr.String())
		}
		if !strings.HasPrefix(stdout.String(), test.want) {
			t.Errorf("Run(%v) output = %q, want prefix %q", test.args, stdout.String(), test.want)
		}
	}
}
