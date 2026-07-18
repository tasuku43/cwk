//go:build !windows

package chatworkconfig

import (
	"errors"
	"os"
	"path/filepath"
)

var errConfigDirectoryResolution = errors.New("user configuration directory could not be resolved")

// resolveUserConfigDir deliberately applies XDG semantics on macOS as well as
// other Unix-like systems so cwk has one portable configuration location.
func resolveUserConfigDir() (string, error) {
	if configured := os.Getenv("XDG_CONFIG_HOME"); configured != "" {
		if !filepath.IsAbs(configured) {
			return "", errConfigDirectoryResolution
		}
		return filepath.Clean(configured), nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" || !filepath.IsAbs(home) {
		return "", errConfigDirectoryResolution
	}
	return filepath.Join(filepath.Clean(home), ".config"), nil
}
