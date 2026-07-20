package cli

import (
	"context"
	"os"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/infra/commandconfig"
)

// TestMain keeps production-constructor tests isolated from the developer's
// real command-selection profile. Individual tests may override either
// platform variable with t.Setenv.
func TestMain(m *testing.M) {
	base, err := os.MkdirTemp("", "cwk-cli-test-config-*")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_CONFIG_HOME", base); err != nil {
		panic(err)
	}
	if err := os.Setenv("AppData", base); err != nil {
		panic(err)
	}
	profile, err := commandselection.New(configurableCommandPaths(DefaultCatalog()))
	if err != nil {
		panic(err)
	}
	if err := commandconfig.NewFileStoreAt(base).Save(context.Background(), profile); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(base)
	os.Exit(code)
}
