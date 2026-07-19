//go:build windows

package commandconfig

import "os"

// resolveUserConfigDir uses Windows' roaming AppData location. XDG and HOME
// are intentionally not consulted on Windows.
func resolveUserConfigDir() (string, error) {
	return os.UserConfigDir()
}
