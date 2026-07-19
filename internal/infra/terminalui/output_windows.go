//go:build windows

package terminalui

import "golang.org/x/sys/windows"

type windowsOutputState struct {
	mode    uint32
	changed bool
}

func (systemDriver) enableOutput(fd int) (any, error) {
	handle := windows.Handle(fd)
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return nil, err
	}
	wanted := mode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if wanted == mode {
		return windowsOutputState{mode: mode}, nil
	}
	if err := windows.SetConsoleMode(handle, wanted); err != nil {
		return nil, err
	}
	return windowsOutputState{mode: mode, changed: true}, nil
}

func (systemDriver) restoreOutput(fd int, value any) error {
	state, ok := value.(windowsOutputState)
	if !ok {
		return errInvalidOutputState
	}
	if !state.changed {
		return nil
	}
	return windows.SetConsoleMode(windows.Handle(fd), state.mode)
}
