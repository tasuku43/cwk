package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"testing"
	"time"
)

type notifyingDeadlineReader struct {
	file    *os.File
	started chan struct{}
}

func (r *notifyingDeadlineReader) Read(value []byte) (int, error) {
	select {
	case <-r.started:
	default:
		close(r.started)
	}
	return r.file.Read(value)
}

func (r *notifyingDeadlineReader) SetReadDeadline(deadline time.Time) error {
	return r.file.SetReadDeadline(deadline)
}

func TestReadContextBoundedCancelsAWaitingProductionPipeWithoutLeakingAReader(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		_, readErr := readContextBounded(ctx, &notifyingDeadlineReader{file: reader, started: started}, 16)
		result <- readErr
	}()
	<-started
	cancel()
	select {
	case readErr := <-result:
		if !errors.Is(readErr, context.Canceled) {
			t.Fatalf("read error = %v", readErr)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled pipe read did not return")
	}
}

type blockingNoDeadlineReader struct {
	started  chan struct{}
	released chan struct{}
	closed   chan struct{}
	once     sync.Once
}

func (r *blockingNoDeadlineReader) Read([]byte) (int, error) {
	select {
	case <-r.started:
	default:
		close(r.started)
	}
	<-r.released
	return 0, io.ErrClosedPipe
}

func (r *blockingNoDeadlineReader) SetReadDeadline(time.Time) error {
	return os.ErrNoDeadline
}

func (r *blockingNoDeadlineReader) Close() error {
	r.once.Do(func() {
		close(r.released)
		close(r.closed)
	})
	return nil
}

func TestReadContextBoundedClosesAndJoinsANonPollableInheritedInput(t *testing.T) {
	reader := &blockingNoDeadlineReader{
		started: make(chan struct{}), released: make(chan struct{}), closed: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, readErr := readContextBounded(ctx, reader, 16)
		result <- readErr
	}()
	<-reader.started
	cancel()
	select {
	case readErr := <-result:
		if !errors.Is(readErr, context.Canceled) {
			t.Fatalf("read error = %v", readErr)
		}
	case <-time.After(time.Second):
		t.Fatal("non-pollable canceled read did not return")
	}
	select {
	case <-reader.closed:
	default:
		t.Fatal("non-pollable canceled reader was not closed")
	}
}
