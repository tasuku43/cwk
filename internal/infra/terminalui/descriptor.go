package terminalui

import (
	"context"
	"fmt"
)

// ReadDescriptor performs one cancellation-aware read from an inherited
// stream descriptor. Unix readiness polling and Windows synchronous-I/O
// cancellation remain infrastructure-owned and are shared by terminal and
// non-interactive stdin consumers.
func ReadDescriptor(ctx context.Context, descriptor uintptr, buffer []byte) (int, error) {
	if ctx == nil {
		return 0, errNilReadContext
	}
	if len(buffer) == 0 {
		return 0, nil
	}
	value := int(descriptor)
	if uintptr(value) != descriptor {
		return 0, fmt.Errorf("terminalui: stream descriptor is out of range")
	}
	return readTerminal(ctx, value, buffer)
}
