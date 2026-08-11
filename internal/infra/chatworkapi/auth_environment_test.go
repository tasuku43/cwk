package chatworkapi

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/fault"
)

func TestInspectTokenEnvironmentReturnsSecretFreeState(t *testing.T) {
	const canary = "synthetic-private-token-canary"
	tests := []struct {
		name  string
		set   bool
		token string
		want  TokenEnvironmentState
	}{
		{name: "unset", want: TokenEnvironmentMissing},
		{name: "empty", set: true, token: "", want: TokenEnvironmentMissing},
		{name: "invalid", set: true, token: canary + "\n", want: TokenEnvironmentInvalid},
		{name: "valid", set: true, token: canary, want: TokenEnvironmentValid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setTokenEnvironment(t, test.set, test.token)
			inspection := InspectTokenEnvironment()
			if inspection.State != test.want || inspection.Source != "environment" ||
				inspection.Name != TokenEnvironment || inspection.Config != "unsupported" ||
				inspection.Keychain != "unsupported" {
				t.Fatalf("inspection = %+v", inspection)
			}
			if rendered := fmt.Sprintf("%+v", inspection); strings.Contains(rendered, canary) {
				t.Fatalf("inspection retained token: %q", rendered)
			}
		})
	}
}

func TestTokenEnvironmentFaultsDescribeOnlySupportedSearchSource(t *testing.T) {
	const canary = "private-token\ncanary"
	tests := []struct {
		name     string
		set      bool
		token    string
		wantCode string
	}{
		{name: "missing", wantCode: tokenMissingCode},
		{name: "invalid", set: true, token: canary, wantCode: tokenInvalidCode},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setTokenEnvironment(t, test.set, test.token)
			client, err := NewFromEnvironment()
			if client != nil || err == nil {
				t.Fatalf("client = %#v, err = %v", client, err)
			}
			var structured *fault.Error
			if !errors.As(err, &structured) || structured.Code != test.wantCode || structured.Retryable {
				t.Fatalf("error = %#v", err)
			}
			message := err.Error()
			for _, wanted := range []string{"環境変数 " + TokenEnvironment, "設定ファイル", "キーチェーン", "未対応"} {
				if !strings.Contains(message, wanted) {
					t.Fatalf("error = %q, want %q", message, wanted)
				}
			}
			if strings.Contains(message, canary) {
				t.Fatalf("error exposed token: %q", message)
			}
		})
	}
}

func setTokenEnvironment(t *testing.T, set bool, value string) {
	t.Helper()
	previous, existed := os.LookupEnv(TokenEnvironment)
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(TokenEnvironment, previous)
			return
		}
		_ = os.Unsetenv(TokenEnvironment)
	})
	if set {
		if err := os.Setenv(TokenEnvironment, value); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := os.Unsetenv(TokenEnvironment); err != nil {
		t.Fatal(err)
	}
}
