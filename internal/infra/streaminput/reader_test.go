package streaminput

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"testing/iotest"
	"time"
)

type emptyReader struct{ calls int }

func (r *emptyReader) Read([]byte) (int, error) {
	r.calls++
	return 0, nil
}

type partialEOFReader struct{}

func (partialEOFReader) Read(p []byte) (int, error) {
	return copy(p, "part"), io.EOF
}

type contextualReader struct {
	ctx context.Context
}

func (r *contextualReader) Read([]byte) (int, error) { panic("Read must not be used") }

func (r *contextualReader) ReadContext(ctx context.Context, p []byte) (int, error) {
	r.ctx = ctx
	return copy(p, "ok"), io.EOF
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

type signaledConn struct {
	net.Conn
	started chan struct{}
	once    sync.Once
}

func (c *signaledConn) Read(p []byte) (int, error) {
	c.once.Do(func() { close(c.started) })
	return c.Conn.Read(p)
}

func TestReadBoundedLimitsAndEOF(t *testing.T) {
	for _, test := range []struct {
		name   string
		reader io.Reader
		limit  int
		want   string
	}{
		{"short input", bytes.NewBufferString("abc"), 10, "abc"},
		{"multiple partial reads", iotest.OneByteReader(bytes.NewBufferString("abc")), 10, "abc"},
		{"exact limit", bytes.NewBufferString("abc"), 3, "abc"},
		{"long input", bytes.NewBufferString("abcdef"), 3, "abc"},
		{"partial read with EOF", partialEOFReader{}, 10, "part"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ReadBounded(context.Background(), test.reader, test.limit)
			if err != nil || string(got) != test.want {
				t.Fatalf("ReadBounded = %q, %v; want %q, nil", got, err, test.want)
			}
		})
	}
}

func TestReadBoundedLeavesBytesAfterLimitUnread(t *testing.T) {
	reader := bytes.NewBufferString("abcdef")
	got, err := ReadBounded(context.Background(), reader, 3)
	if err != nil || string(got) != "abc" {
		t.Fatalf("ReadBounded = %q, %v; want abc, nil", got, err)
	}
	if remaining := reader.String(); remaining != "def" {
		t.Fatalf("remaining input = %q, want def", remaining)
	}
}

func TestReadBoundedRejectsInvalidInputsAndStopsOnErrors(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	sentinel := errors.New("synthetic read failure")
	for _, test := range []struct {
		name   string
		ctx    context.Context
		reader io.Reader
		limit  int
		want   error
	}{
		{"nil context", nil, bytes.NewBufferString("x"), 1, context.Canceled},
		{"canceled", canceled, bytes.NewBufferString("x"), 1, context.Canceled},
		{"nil reader", context.Background(), nil, 1, io.ErrUnexpectedEOF},
		{"zero limit", context.Background(), bytes.NewBufferString("x"), 0, io.ErrUnexpectedEOF},
		{"negative limit", context.Background(), bytes.NewBufferString("x"), -1, io.ErrUnexpectedEOF},
		{"read error", context.Background(), failingReader{sentinel}, 1, sentinel},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ReadBounded(test.ctx, test.reader, test.limit)
			if got != nil || !errors.Is(err, test.want) {
				t.Fatalf("ReadBounded = %q, %v; want nil, %v", got, err, test.want)
			}
		})
	}
	empty := &emptyReader{}
	if got, err := ReadBounded(context.Background(), empty, 1); got != nil || !errors.Is(err, io.ErrNoProgress) || empty.calls != 100 {
		t.Fatalf("empty reader = %q, %v after %d calls; want no progress after 100", got, err, empty.calls)
	}
}

func TestReadBoundedUsesContextReader(t *testing.T) {
	reader := &contextualReader{}
	ctx := context.WithValue(context.Background(), struct{}{}, "synthetic")
	got, err := ReadBounded(ctx, reader, 10)
	if err != nil || string(got) != "ok" || reader.ctx != ctx {
		t.Fatalf("contextual read = %q, %v, context %v", got, err, reader.ctx == ctx)
	}
}

func TestReadBoundedCancellationRestoresReadDeadline(t *testing.T) {
	readEnd, writeEnd := net.Pipe()
	defer readEnd.Close()
	defer writeEnd.Close()
	reader := &signaledConn{Conn: readEnd, started: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := ReadBounded(ctx, reader, 1)
		result <- err
	}()
	select {
	case <-reader.started:
	case <-time.After(5 * time.Second):
		t.Fatal("read did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled read = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled read did not return")
	}
	writeResult := make(chan error, 1)
	go func() {
		_, err := writeEnd.Write([]byte("x"))
		writeResult <- err
	}()
	got, err := ReadBounded(context.Background(), reader, 1)
	if err != nil || string(got) != "x" {
		t.Fatalf("read after cancellation = %q, %v; deadline was not restored", got, err)
	}
	if err := <-writeResult; err != nil {
		t.Fatal(err)
	}
}
