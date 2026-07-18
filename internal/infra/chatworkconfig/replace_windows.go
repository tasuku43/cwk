//go:build windows

package chatworkconfig

import (
	"syscall"
	"unsafe"
)

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

var moveFileExW = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func atomicReplace(source, destination string) error {
	sourcePointer, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destinationPointer, err := syscall.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	// #nosec G103 -- MoveFileExW requires stable pointers to NUL-terminated
	// UTF-16 path buffers for the duration of this synchronous call.
	sourceAddress := uintptr(unsafe.Pointer(sourcePointer))
	// #nosec G103 -- same audited Windows syscall boundary as sourceAddress.
	destinationAddress := uintptr(unsafe.Pointer(destinationPointer))
	result, _, callErr := moveFileExW.Call(
		sourceAddress,
		destinationAddress,
		moveFileReplaceExisting|moveFileWriteThrough,
	)
	if result == 0 {
		return callErr
	}
	return nil
}
