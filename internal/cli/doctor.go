package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/tasuku43/cwk/internal/domain/doctor"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

const (
	maxDoctorChecks      = 100
	maxDoctorNameBytes   = 256
	maxDoctorDetailBytes = 64 * 1024
)

func runDoctor(ctx context.Context, c *CLI, command CommandSpec, intent operation.Intent, args []string) int {
	format, err := parseFormatOnlyArgs(args)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", err.Error()+"; 使い方: "+command.Usage(), "help doctor", "コマンド引数を修正してください。")
	}
	report, err := c.doctor.Run(ctx, intent)
	if err != nil {
		return c.fail(ctx, err)
	}
	if check, present, err := c.commandSelectionDoctorCheck(ctx); err != nil {
		return c.fail(ctx, err)
	} else if present {
		report.Checks = append(report.Checks, check)
		if err := report.Validate(); err != nil {
			return c.fail(ctx, fault.Wrap(fault.KindInternal, "internal_error", "統合した診断レポートは無効です。", false, err))
		}
	}
	if err := validateDoctorProjection(report); err != nil {
		return c.fail(ctx, err)
	}
	output, err := renderDoctorReport(report, format)
	if err != nil {
		return c.fail(ctx, err)
	}
	if code := c.emit(ctx, output); code != ExitOK {
		return code
	}
	if !report.Healthy() {
		return c.fail(ctx, fault.New(
			fault.KindRejected,
			"diagnostic_failed",
			"1件以上の診断が失敗しました。",
			false,
			fault.NextAction{Command: "doctor", Reason: "レポートを確認して失敗した前提条件を修正し、診断を再実行してください。"},
		))
	}
	return ExitOK
}

func validateDoctorProjection(report doctor.Report) error {
	if len(report.Checks) > maxDoctorChecks {
		return outputContractExceeded("診断レポートが宣言済みのチェック数上限を超えています。", "doctor")
	}
	for _, check := range report.Checks {
		if len(check.Name) > maxDoctorNameBytes || len(check.Detail) > maxDoctorDetailBytes {
			return outputContractExceeded("診断フィールドが宣言済みのバイト数上限を超えています。", "doctor")
		}
	}
	return nil
}

type doctorJSONDocument struct {
	SchemaVersion int               `json:"schema_version"`
	Report        []doctorJSONCheck `json:"report"`
}

type doctorJSONCheck struct {
	Check  string `json:"check"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func renderDoctorReport(report doctor.Report, format successFormat) ([]byte, error) {
	if format == successFormatJSON {
		document := doctorJSONDocument{SchemaVersion: 1, Report: make([]doctorJSONCheck, 0, len(report.Checks))}
		for _, check := range report.Checks {
			document.Report = append(document.Report, doctorJSONCheck{
				Check:  safeExternalText(check.Name),
				Status: string(check.Status),
				Detail: safeExternalText(check.Detail),
			})
		}
		output, err := json.Marshal(document)
		if err != nil {
			return nil, fault.Wrap(fault.KindContract, "output_encoding_failed", "診断 JSON をエンコードできませんでした。", false, err)
		}
		return append(output, '\n'), nil
	}

	var output bytes.Buffer
	fmt.Fprintln(&output, "チェック\t状態\t詳細")
	for _, check := range report.Checks {
		fmt.Fprintf(&output, "%s\t%s\t%s\n", escapeTSVCell(check.Name), check.Status, escapeTSVCell(check.Detail))
	}
	return output.Bytes(), nil
}

func outputContractExceeded(message, command string) *fault.Error {
	return fault.New(
		fault.KindContract,
		"output_contract_exceeded",
		message,
		false,
		fault.NextAction{Command: command, Reason: "上限付き出力契約と上流レスポンスを確認してください。"},
	)
}
