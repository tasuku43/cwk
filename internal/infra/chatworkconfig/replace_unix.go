//go:build !windows

package chatworkconfig

import "os"

func atomicReplace(source, destination string) error {
	return os.Rename(source, destination)
}
