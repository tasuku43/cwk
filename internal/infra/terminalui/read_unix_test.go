//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos

package terminalui

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func TestReadTerminalReportsEOFWhenInputCloses(t *testing.T) {
	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	n, err := readTerminal(ctx, int(input.Fd()), make([]byte, 8))
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("closed input = (%d, %v), want EOF", n, err)
	}
}

func TestReadTerminalRejectsInvalidDescriptor(t *testing.T) {
	if _, err := readTerminal(context.Background(), -1, make([]byte, 1)); err == nil {
		t.Fatal("negative descriptor accepted")
	}
	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	fd := int(input.Fd())
	if err := input.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := readTerminal(ctx, fd, make([]byte, 1)); err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("closed descriptor error = %v, want immediate invalid descriptor", err)
	}
}

func TestSessionReadReturnsAlreadyReadyInput(t *testing.T) {
	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer output.Close()
	if _, err := output.Write([]byte("ready")); err != nil {
		t.Fatal(err)
	}

	session := &terminalSession{inputFD: int(input.Fd())}
	buffer := make([]byte, 16)
	count, err := session.Read(context.Background(), buffer)
	if err != nil {
		t.Fatalf("Read(): %v", err)
	}
	if got := string(buffer[:count]); got != "ready" {
		t.Fatalf("Read() = %q, want ready", got)
	}
}

func TestSessionReadCancellationDoesNotConsumeFutureInput(t *testing.T) {
	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer output.Close()

	session := &terminalSession{inputFD: int(input.Fd())}
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		buffer := make([]byte, 8)
		_, err := session.Read(ctx, buffer)
		done <- err
	}()
	<-started
	// Let the implementation enter its bounded OS readiness wait before the
	// cancellation; the assertion does not rely on cancellation preceding Read.
	time.Sleep(2 * terminalReadWaitMilliseconds * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled Read() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled Read() remained blocked")
	}

	if _, err := output.Write([]byte("later")); err != nil {
		t.Fatal(err)
	}
	readCtx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	buffer := make([]byte, 8)
	count, err := session.Read(readCtx, buffer)
	if err != nil {
		t.Fatalf("Read() after cancellation: %v", err)
	}
	if got := string(buffer[:count]); got != "later" {
		t.Fatalf("future input = %q, want later", got)
	}
}
