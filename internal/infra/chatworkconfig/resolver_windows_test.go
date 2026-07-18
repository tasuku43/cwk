//go:build windows

package chatworkconfig

import (
	"path/filepath"
	"testing"
)

func TestResolveUserConfigDirUsesWindowsAppData(t *testing.T) {
	appData := filepath.Join(t.TempDir(), "AppData", "Roaming")
	t.Setenv("AppData", appData)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg-must-not-be-used"))
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home-must-not-be-used"))

	got, err := resolveUserConfigDir()
	if err != nil {
		t.Fatalf("resolveUserConfigDir: %v", err)
	}
	if got != appData {
		t.Fatalf("resolveUserConfigDir() = %q, want %q", got, appData)
	}
}
