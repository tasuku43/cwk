// Package streaminput owns bounded, cancellation-aware reads from an injected
// process input stream. Product policy chooses the bound; this adapter owns the
// descriptor I/O and platform cancellation mechanism.
package streaminput

import (
	"context"
	"io"
	"time"

	"github.com/tasuku43/cwk/internal/infra/terminalui"
)

type contextReader interface {
	ReadContext(context.Context, []byte) (int, error)
}

type fileDescriptor interface {
	Fd() uintptr
}

type readDeadlineSetter interface {
	SetReadDeadline(time.Time) error
}

// ReadBounded reads at most limit bytes. Production file descriptors use the
// platform adapter so cancellation never abandons a reader that could consume
// a later invocation's stdin.
func ReadBounded(ctx context.Context, reader io.Reader, limit int) ([]byte, error) {
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if reader == nil || limit < 1 {
		return nil, io.ErrUnexpectedEOF
	}

	result := make([]byte, 0, limit)
	buffer := make([]byte, 32*1024)
	emptyReads := 0
	for len(result) < limit {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		remaining := limit - len(result)
		chunk := buffer
		if remaining < len(chunk) {
			chunk = chunk[:remaining]
		}
		read, err := readContext(ctx, reader, chunk)
		if read > 0 {
			result = append(result, chunk[:read]...)
			emptyReads = 0
		} else {
			emptyReads++
		}
		if err != nil {
			if contextErr := ctx.Err(); contextErr != nil {
				return nil, contextErr
			}
			if err == io.EOF {
				return result, nil
			}
			return nil, err
		}
		if emptyReads >= 100 {
			return nil, io.ErrNoProgress
		}
	}
	return result, nil
}

func readContext(ctx context.Context, reader io.Reader, buffer []byte) (int, error) {
	if contextual, ok := reader.(contextReader); ok {
		return contextual.ReadContext(ctx, buffer)
	}
	if descriptor, ok := reader.(fileDescriptor); ok {
		return terminalui.ReadDescriptor(ctx, descriptor.Fd(), buffer)
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	cleanup := armCancellation(ctx, reader)
	read, err := reader.Read(buffer)
	cleanup()
	if contextErr := ctx.Err(); contextErr != nil {
		return 0, contextErr
	}
	return read, err
}

func armCancellation(ctx context.Context, reader io.Reader) func() {
	setter, ok := reader.(readDeadlineSetter)
	if ok && setter.SetReadDeadline(time.Time{}) == nil {
		return armWatcher(ctx, func() {
			_ = setter.SetReadDeadline(time.Now())
		}, func() {
			_ = setter.SetReadDeadline(time.Time{})
		})
	}
	closer, ok := reader.(io.Closer)
	if ok {
		return armWatcher(ctx, func() {
			_ = closer.Close()
		}, nil)
	}
	return func() {}
}

func armWatcher(ctx context.Context, cancelRead, reset func()) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-ctx.Done():
			cancelRead()
		case <-stop:
		}
	}()
	return func() {
		close(stop)
		<-done
		if reset != nil {
			reset()
		}
	}
}
