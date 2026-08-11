//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package cli

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestReadContextBoundedCancelsAnInheritedBlockingPipeWithoutConsumingFutureInput(t *testing.T) {
	descriptors := []int{0, 0}
	if err := syscall.Pipe(descriptors); err != nil {
		t.Fatal(err)
	}
	original := os.NewFile(uintptr(descriptors[0]), "inherited-stdin")
	writer := os.NewFile(uintptr(descriptors[1]), "inherited-writer")
	defer original.Close()
	defer writer.Close()
	if err := original.SetReadDeadline(time.Time{}); !errors.Is(err, os.ErrNoDeadline) {
		t.Fatalf("inherited pipe unexpectedly supports deadlines: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, readErr := readContextBounded(ctx, original, 16)
		result <- readErr
	}()
	select {
	case readErr := <-result:
		t.Fatalf("inherited blocking pipe returned before cancellation: %v", readErr)
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	select {
	case readErr := <-result:
		if !errors.Is(readErr, context.Canceled) {
			t.Fatalf("read error = %v", readErr)
		}
	case <-time.After(time.Second):
		t.Fatal("inherited blocking pipe did not cancel")
	}
	if _, err := writer.Write([]byte("later")); err != nil {
		t.Fatalf("write after cancellation: %v", err)
	}
	buffer := make([]byte, 5)
	if _, err := original.Read(buffer); err != nil {
		t.Fatalf("read after cancellation: %v", err)
	}
	if string(buffer) != "later" {
		t.Fatalf("future input = %q, want later", buffer)
	}
}
