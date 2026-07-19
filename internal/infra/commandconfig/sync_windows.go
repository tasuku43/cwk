//go:build windows

package commandconfig

import "os"

// os.Root.Rename requests replace-existing behavior on Windows. The portable
// API does not expose directory Sync and makes no cross-platform atomicity or
// durability guarantee, so there is no additional operation available here.
func syncDirectory(*os.Root) error {
	return nil
}
