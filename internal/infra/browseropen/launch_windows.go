//go:build windows

package browseropen

import (
	"context"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var shellExecute = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")

func platformLaunch(ctx context.Context, raw string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return fmt.Errorf("browser activation verb is invalid")
	}
	target, err := windows.UTF16PtrFromString(raw)
	if err != nil {
		return fmt.Errorf("browser activation target is invalid")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// #nosec G103 -- ShellExecuteW requires stable pointers to validated,
	// NUL-terminated UTF-16 values at this audited Windows activation boundary.
	verbAddress := uintptr(unsafe.Pointer(verb))
	// #nosec G103 -- same audited ShellExecuteW boundary as verbAddress.
	targetAddress := uintptr(unsafe.Pointer(target))
	result, _, _ := shellExecute.Call(
		0,
		verbAddress,
		targetAddress,
		0,
		0,
		1,
	)
	runtime.KeepAlive(verb)
	runtime.KeepAlive(target)
	if result <= 32 {
		return fmt.Errorf("browser activation failed")
	}
	return nil
}
