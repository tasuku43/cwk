//go:build windows

package terminalui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"

	"golang.org/x/sys/windows"
)

var cancelSynchronousIOProc = windows.NewLazySystemDLL("kernel32.dll").NewProc("CancelSynchronousIo")

func readTerminal(ctx context.Context, fd int, buffer []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if len(buffer) == 0 {
		return 0, nil
	}
	// Resolve the cancellation procedure before a synchronous ReadFile can
	// begin. Starting the worker without a usable cancellation path could leave
	// it blocked after the caller's context is done.
	if err := cancelSynchronousIOProc.Find(); err != nil {
		return 0, fmt.Errorf("resolve CancelSynchronousIo: %w", err)
	}
	return readWindowsTerminal(ctx, windows.Handle(fd), buffer, systemWindowsReadOperations)
}

type windowsReadFunc func(windows.Handle, []byte) (int, error)
type windowsThreadFunc func() (windows.Handle, error)
type windowsCancelFunc func(windows.Handle) error
type windowsCloseFunc func(windows.Handle) error

type windowsReadOperations struct {
	duplicateThread windowsThreadFunc
	read            windowsReadFunc
	cancel          windowsCancelFunc
	close           windowsCloseFunc
}

var systemWindowsReadOperations = windowsReadOperations{
	duplicateThread: duplicateCurrentWindowsThread,
	read:            windows.Read,
	cancel:          cancelSynchronousIO,
	close:           windows.CloseHandle,
}

type windowsReadWorkerSetup struct {
	thread windows.Handle
	err    error
}

type windowsReadResult struct {
	count int
	err   error
}

func readWindowsTerminal(ctx context.Context, input windows.Handle, buffer []byte, operations windowsReadOperations) (int, error) {
	if ctx == nil {
		return 0, errNilReadContext
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if len(buffer) == 0 {
		return 0, nil
	}
	if err := operations.validate(); err != nil {
		return 0, err
	}

	setup := make(chan windowsReadWorkerSetup, 1)
	completed := make(chan windowsReadResult, 1)
	joined := make(chan struct{})
	// A console handle is signaled for every input record, while ReadFile may
	// discard a non-character record and continue blocking. Keep that synchronous
	// call on one locked OS thread so cancellation can target the exact reader.
	go func() {
		runtime.LockOSThread()
		// Signal joined only after the goroutine has released the dedicated OS
		// thread. The parent never returns while a reader could still consume a
		// later invocation's input.
		defer close(joined)
		defer runtime.UnlockOSThread()

		thread, err := operations.duplicateThread()
		setup <- windowsReadWorkerSetup{thread: thread, err: err}
		if err != nil {
			return
		}
		count, readErr := operations.read(input, buffer)
		completed <- windowsReadResult{count: count, err: readErr}
	}()

	worker := <-setup
	if worker.err != nil {
		<-joined
		return 0, fmt.Errorf("duplicate terminal read thread handle: %w", worker.err)
	}

	result, cancelErr := awaitWindowsRead(ctx, worker.thread, completed, operations.cancel)
	<-joined
	closeErr := operations.close(worker.thread)

	// Cancellation owns the result even if ReadFile won the race and completed
	// normally. Joining first makes that decision without abandoning a reader or
	// accessing its buffer while the synchronous operation is still in flight.
	if contextErr := ctx.Err(); contextErr != nil {
		return 0, errors.Join(
			contextErr,
			wrapWindowsReadError("cancel synchronous terminal input", cancelErr),
			wrapWindowsReadError("close terminal read thread handle", closeErr),
		)
	}
	if cancelErr != nil {
		return 0, fmt.Errorf("cancel synchronous terminal input: %w", cancelErr)
	}
	if closeErr != nil {
		return 0, fmt.Errorf("close terminal read thread handle: %w", closeErr)
	}
	if result.err != nil {
		return result.count, result.err
	}
	if result.count == 0 {
		return 0, io.EOF
	}
	return result.count, nil
}

func (operations windowsReadOperations) validate() error {
	switch {
	case operations.duplicateThread == nil:
		return errors.New("terminalui: Windows read thread duplication is unavailable")
	case operations.read == nil:
		return errors.New("terminalui: Windows terminal read is unavailable")
	case operations.cancel == nil:
		return errors.New("terminalui: Windows terminal read cancellation is unavailable")
	case operations.close == nil:
		return errors.New("terminalui: Windows read thread cleanup is unavailable")
	default:
		return nil
	}
}

func awaitWindowsRead(ctx context.Context, thread windows.Handle, completed <-chan windowsReadResult, cancel windowsCancelFunc) (windowsReadResult, error) {
	select {
	case result := <-completed:
		return result, nil
	case <-ctx.Done():
	}

	for {
		err := cancel(thread)
		if err == nil {
			return <-completed, nil
		}
		if !errors.Is(err, windows.ERROR_NOT_FOUND) {
			// A failed cancellation cannot make it safe to abandon the buffer or
			// reader. Join the synchronous call before reporting both failures.
			return <-completed, err
		}

		// ERROR_NOT_FOUND is the documented race when cancellation reaches the
		// worker immediately before ReadFile is issued or immediately after it
		// completes. If the result is not available yet, yield and retry until the
		// one synchronous call is either observable or cancellable.
		select {
		case result := <-completed:
			return result, nil
		default:
			runtime.Gosched()
		}
	}
}

func duplicateCurrentWindowsThread() (windows.Handle, error) {
	var thread windows.Handle
	process := windows.CurrentProcess()
	if err := windows.DuplicateHandle(
		process,
		windows.CurrentThread(),
		process,
		&thread,
		windows.THREAD_TERMINATE,
		false,
		0,
	); err != nil {
		return 0, err
	}
	return thread, nil
}

func cancelSynchronousIO(thread windows.Handle) error {
	result, _, callErr := cancelSynchronousIOProc.Call(uintptr(thread))
	if result != 0 {
		return nil
	}
	if callErr == nil || errors.Is(callErr, windows.ERROR_SUCCESS) {
		return errors.New("CancelSynchronousIo failed without a Windows error code")
	}
	return callErr
}

func wrapWindowsReadError(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}
