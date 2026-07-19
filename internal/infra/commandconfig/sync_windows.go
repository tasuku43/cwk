//go:build windows

package commandconfig

// Directory Sync is not supported by the portable Windows filesystem API. The
// same-directory replacement itself is performed by os.Root.Rename.
func syncDirectory(string) error {
	return nil
}
