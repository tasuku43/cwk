//go:build !windows

package terminalui

type nonWindowsOutputState struct{}

func (systemDriver) enableOutput(int) (any, error) {
	return nonWindowsOutputState{}, nil
}

func (systemDriver) restoreOutput(_ int, state any) error {
	if _, ok := state.(nonWindowsOutputState); !ok {
		return errInvalidOutputState
	}
	return nil
}
