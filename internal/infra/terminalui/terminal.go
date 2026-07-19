// Package terminalui adapts an attached terminal for a bounded interactive
// CLI presentation. Product selection policy and key handling remain in the
// CLI layer; this package owns only terminal detection, mode changes,
// cancellation-aware input, sizing, and restoration.
package terminalui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"golang.org/x/term"
)

const (
	enterScreen                  = "\x1b[?1049h\x1b[?25l"
	leaveScreen                  = "\x1b[?25h\x1b[?1049l"
	terminalReadWaitMilliseconds = 25
)

var (
	// ErrNotTerminal reports that both injected streams are not attached to
	// terminals. Callers can map it to their public non-interactive contract.
	ErrNotTerminal = errors.New("terminalui: stdin and stdout must both be terminals")

	// ErrSessionClosed reports an attempt to inspect a restored session.
	ErrSessionClosed = errors.New("terminalui: session is closed")

	// ErrRestoreFailed marks a setup rollback or session close that could not
	// fully restore terminal state. Callers use this sentinel to distinguish a
	// retryable setup failure from a terminal whose prior mode is uncertain.
	ErrRestoreFailed = errors.New("terminalui: terminal restoration failed")

	errInvalidOutputState = errors.New("terminalui: invalid output state")
	errNilReadContext     = errors.New("terminalui: read context is nil")
)

// Opener creates a terminal session. The interface is intentionally small so
// CLI tests can inject a fake without constructing an operating-system TTY.
type Opener interface {
	Open(input io.Reader, output io.Writer) (Session, error)
}

// Session exposes terminal size, cancellation-aware input, and idempotent
// restoration. Presentation output continues through the stream owned by CLI.
type Session interface {
	Size() (width, height int, err error)
	Read(ctx context.Context, buffer []byte) (int, error)
	Close() error
}

// New returns the production terminal adapter.
func New() Opener {
	return &opener{driver: systemDriver{}}
}

type fileDescriptor interface {
	Fd() uintptr
}

type opener struct {
	driver terminalDriver
}

func (o *opener) Open(input io.Reader, output io.Writer) (Session, error) {
	if o == nil || o.driver == nil {
		return nil, errors.New("terminalui: terminal driver is unavailable")
	}
	inputFile, inputOK := input.(fileDescriptor)
	outputFile, outputOK := output.(fileDescriptor)
	if !inputOK || !outputOK {
		return nil, ErrNotTerminal
	}
	inputFD := int(inputFile.Fd())
	outputFD := int(outputFile.Fd())
	if !o.driver.isTerminal(inputFD) || !o.driver.isTerminal(outputFD) {
		return nil, ErrNotTerminal
	}

	rawState, err := o.driver.makeRaw(inputFD)
	if err != nil {
		return nil, fmt.Errorf("terminalui: enter raw input mode: %w", err)
	}
	outputState, err := o.driver.enableOutput(outputFD)
	if err != nil {
		restoreErr := o.driver.restoreRaw(inputFD, rawState)
		return nil, errors.Join(
			fmt.Errorf("terminalui: enable terminal output mode: %w", err),
			wrapRestoreError("restore input mode after setup failure", restoreErr),
		)
	}

	session := &terminalSession{
		driver:      o.driver,
		output:      output,
		inputFD:     inputFD,
		outputFD:    outputFD,
		rawState:    rawState,
		outputState: outputState,
	}
	if err := writeControl(output, enterScreen); err != nil {
		return nil, errors.Join(
			fmt.Errorf("terminalui: enter alternate screen: %w", err),
			session.Close(),
		)
	}
	return session, nil
}

type terminalSession struct {
	driver      terminalDriver
	output      io.Writer
	inputFD     int
	outputFD    int
	rawState    any
	outputState any

	closed    atomic.Bool
	closeOnce sync.Once
	closeErr  error
}

func (s *terminalSession) Size() (int, int, error) {
	if s == nil || s.closed.Load() {
		return 0, 0, ErrSessionClosed
	}
	width, height, err := s.driver.size(s.outputFD)
	if err != nil {
		return 0, 0, fmt.Errorf("terminalui: read terminal size: %w", err)
	}
	return width, height, nil
}

// Read waits for terminal input without leaving an uninterruptible background
// reader behind. Unix adapters use bounded readiness waits; Windows confines
// its synchronous read to one cancelable OS thread and joins it before return.
// Cancellation therefore cannot leave a reader that later steals input from a
// restored terminal or a subsequent CLI invocation.
func (s *terminalSession) Read(ctx context.Context, buffer []byte) (int, error) {
	if s == nil || s.closed.Load() {
		return 0, ErrSessionClosed
	}
	if ctx == nil {
		return 0, errNilReadContext
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if len(buffer) == 0 {
		return 0, nil
	}
	count, err := readTerminal(ctx, s.inputFD, buffer)
	if err != nil {
		return count, fmt.Errorf("terminalui: read input: %w", err)
	}
	return count, nil
}

func (s *terminalSession) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		// The leave sequence must be written while Windows VT output remains
		// enabled. Every restoration is attempted even when an earlier step
		// fails so a presentation error cannot strand the terminal in raw mode.
		s.closeErr = errors.Join(
			wrapRestoreError("leave alternate screen", writeControl(s.output, leaveScreen)),
			wrapRestoreError("restore output mode", s.driver.restoreOutput(s.outputFD, s.outputState)),
			wrapRestoreError("restore input mode", s.driver.restoreRaw(s.inputFD, s.rawState)),
		)
	})
	return s.closeErr
}

func wrapRestoreError(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrRestoreFailed, action, err)
}

func writeControl(output io.Writer, sequence string) error {
	remaining := []byte(sequence)
	for len(remaining) != 0 {
		written, err := output.Write(remaining)
		if written < 0 || written > len(remaining) {
			return fmt.Errorf("invalid write count %d", written)
		}
		remaining = remaining[written:]
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

type terminalDriver interface {
	isTerminal(fd int) bool
	makeRaw(fd int) (any, error)
	restoreRaw(fd int, state any) error
	size(fd int) (width, height int, err error)
	enableOutput(fd int) (any, error)
	restoreOutput(fd int, state any) error
}

type systemDriver struct{}

type rawState struct {
	state *term.State
}

func (systemDriver) isTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

func (systemDriver) makeRaw(fd int) (any, error) {
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return rawState{state: state}, nil
}

func (systemDriver) restoreRaw(fd int, value any) error {
	state, ok := value.(rawState)
	if !ok || state.state == nil {
		return errors.New("terminalui: invalid raw terminal state")
	}
	return term.Restore(fd, state.state)
}

func (systemDriver) size(fd int) (int, int, error) {
	return term.GetSize(fd)
}
