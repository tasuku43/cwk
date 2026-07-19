//go:build windows

package terminalui

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"golang.org/x/sys/windows"
)

func TestReadWindowsTerminalReturnsReadyInputAndClosesThreadHandle(t *testing.T) {
	var closeCalls atomic.Int32
	operations := windowsReadOperations{
		duplicateThread: func() (windows.Handle, error) { return 41, nil },
		read:            func(_ windows.Handle, buffer []byte) (int, error) { return copy(buffer, "ready"), nil },
		cancel:          func(windows.Handle) error { return errors.New("unexpected cancellation") },
		close: func(handle windows.Handle) error {
			if handle != 41 {
				return errors.New("wrong thread handle")
			}
			closeCalls.Add(1)
			return nil
		},
	}
	buffer := make([]byte, 16)
	count, err := readWindowsTerminal(context.Background(), 1, buffer, operations)
	if err != nil {
		t.Fatalf("readWindowsTerminal(): %v", err)
	}
	if got := string(buffer[:count]); got != "ready" {
		t.Fatalf("readWindowsTerminal() = %q, want ready", got)
	}
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("thread handle close calls = %d, want 1", got)
	}
}

func TestReadWindowsTerminalMapsZeroReadToEOF(t *testing.T) {
	var closeCalls atomic.Int32
	operations := windowsReadOperations{
		duplicateThread: func() (windows.Handle, error) { return 42, nil },
		read:            func(windows.Handle, []byte) (int, error) { return 0, nil },
		cancel:          func(windows.Handle) error { return errors.New("unexpected cancellation") },
		close: func(windows.Handle) error {
			closeCalls.Add(1)
			return nil
		},
	}
	if count, err := readWindowsTerminal(context.Background(), 1, make([]byte, 8), operations); count != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("readWindowsTerminal() = (%d, %v), want (0, io.EOF)", count, err)
	}
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("thread handle close calls = %d, want 1", got)
	}
}

func TestReadWindowsTerminalCancellationJoinsReaderBeforeFutureRead(t *testing.T) {
	ctx, cancelContext := context.WithCancel(context.Background())
	readStarted := make(chan struct{})
	abortRead := make(chan struct{})
	var abortOnce sync.Once
	var readCalls atomic.Int32
	var closeCalls atomic.Int32
	var firstReadFinished atomic.Bool
	operations := windowsReadOperations{
		duplicateThread: func() (windows.Handle, error) { return 43, nil },
		read: func(_ windows.Handle, buffer []byte) (int, error) {
			if readCalls.Add(1) == 1 {
				close(readStarted)
				<-abortRead
				firstReadFinished.Store(true)
				return 0, windows.ERROR_OPERATION_ABORTED
			}
			return copy(buffer, "later"), nil
		},
		cancel: func(windows.Handle) error {
			abortOnce.Do(func() { close(abortRead) })
			return nil
		},
		close: func(windows.Handle) error {
			closeCalls.Add(1)
			return nil
		},
	}
	go func() {
		<-readStarted
		cancelContext()
	}()

	if count, err := readWindowsTerminal(ctx, 1, make([]byte, 8), operations); count != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled readWindowsTerminal() = (%d, %v), want (0, context.Canceled)", count, err)
	}
	if !firstReadFinished.Load() {
		t.Fatal("canceled read returned before its worker was joined")
	}

	buffer := make([]byte, 8)
	count, err := readWindowsTerminal(context.Background(), 1, buffer, operations)
	if err != nil {
		t.Fatalf("future readWindowsTerminal(): %v", err)
	}
	if got := string(buffer[:count]); got != "later" {
		t.Fatalf("future input = %q, want later", got)
	}
	if got := readCalls.Load(); got != 2 {
		t.Fatalf("read calls = %d, want 2", got)
	}
	if got := closeCalls.Load(); got != 2 {
		t.Fatalf("thread handle close calls = %d, want 2", got)
	}
}

func TestReadWindowsTerminalRetriesNotFoundRaceUntilReadIsCancelable(t *testing.T) {
	ctx, cancelContext := context.WithCancel(context.Background())
	allowRead := make(chan struct{})
	readStarted := make(chan struct{})
	abortRead := make(chan struct{})
	var allowOnce sync.Once
	var abortOnce sync.Once
	var cancelCalls atomic.Int32
	var closeCalls atomic.Int32
	operations := windowsReadOperations{
		duplicateThread: func() (windows.Handle, error) {
			cancelContext()
			return 44, nil
		},
		read: func(windows.Handle, []byte) (int, error) {
			<-allowRead
			close(readStarted)
			<-abortRead
			return 0, windows.ERROR_OPERATION_ABORTED
		},
		cancel: func(windows.Handle) error {
			if cancelCalls.Add(1) == 1 {
				allowOnce.Do(func() { close(allowRead) })
				return windows.ERROR_NOT_FOUND
			}
			<-readStarted
			abortOnce.Do(func() { close(abortRead) })
			return nil
		},
		close: func(windows.Handle) error {
			closeCalls.Add(1)
			return nil
		},
	}

	if count, err := readWindowsTerminal(ctx, 1, make([]byte, 8), operations); count != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled readWindowsTerminal() = (%d, %v), want (0, context.Canceled)", count, err)
	}
	if got := cancelCalls.Load(); got < 2 {
		t.Fatalf("cancellation calls = %d, want ERROR_NOT_FOUND retry", got)
	}
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("thread handle close calls = %d, want 1", got)
	}
}

func TestReadWindowsTerminalJoinsAfterCancellationFailure(t *testing.T) {
	ctx, cancelContext := context.WithCancel(context.Background())
	readStarted := make(chan struct{})
	releaseRead := make(chan struct{})
	cancelFailure := errors.New("cancel failed")
	var readFinished atomic.Bool
	var closeCalls atomic.Int32
	operations := windowsReadOperations{
		duplicateThread: func() (windows.Handle, error) { return 45, nil },
		read: func(windows.Handle, []byte) (int, error) {
			close(readStarted)
			<-releaseRead
			readFinished.Store(true)
			return 0, windows.ERROR_OPERATION_ABORTED
		},
		cancel: func(windows.Handle) error {
			close(releaseRead)
			return cancelFailure
		},
		close: func(windows.Handle) error {
			closeCalls.Add(1)
			return nil
		},
	}
	go func() {
		<-readStarted
		cancelContext()
	}()

	count, err := readWindowsTerminal(ctx, 1, make([]byte, 8), operations)
	if count != 0 || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelFailure) {
		t.Fatalf("readWindowsTerminal() = (%d, %v), want context and cancellation failures", count, err)
	}
	if !readFinished.Load() {
		t.Fatal("cancellation failure returned before the read worker was joined")
	}
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("thread handle close calls = %d, want 1", got)
	}
}

func TestReadWindowsTerminalSetupAndHandleCloseFailuresAreReported(t *testing.T) {
	setupFailure := errors.New("duplicate failed")
	readCalls := 0
	setupOperations := windowsReadOperations{
		duplicateThread: func() (windows.Handle, error) { return 0, setupFailure },
		read: func(windows.Handle, []byte) (int, error) {
			readCalls++
			return 0, nil
		},
		cancel: func(windows.Handle) error { return nil },
		close:  func(windows.Handle) error { return errors.New("must not close an absent handle") },
	}
	if _, err := readWindowsTerminal(context.Background(), 1, make([]byte, 8), setupOperations); !errors.Is(err, setupFailure) {
		t.Fatalf("setup error = %v, want %v", err, setupFailure)
	}
	if readCalls != 0 {
		t.Fatalf("setup failure read calls = %d, want 0", readCalls)
	}

	closeFailure := errors.New("close failed")
	closeOperations := windowsReadOperations{
		duplicateThread: func() (windows.Handle, error) { return 46, nil },
		read:            func(_ windows.Handle, buffer []byte) (int, error) { return copy(buffer, "ignored"), nil },
		cancel:          func(windows.Handle) error { return nil },
		close:           func(windows.Handle) error { return closeFailure },
	}
	if count, err := readWindowsTerminal(context.Background(), 1, make([]byte, 8), closeOperations); count != 0 || !errors.Is(err, closeFailure) {
		t.Fatalf("handle close failure = (%d, %v), want (0, %v)", count, err, closeFailure)
	}
}
