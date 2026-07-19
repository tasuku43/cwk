package terminalui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
)

type descriptorReader struct {
	io.Reader
	fd uintptr
}

func (r descriptorReader) Fd() uintptr { return r.fd }

type writeStep struct {
	n   int
	err error
}

type descriptorWriter struct {
	fd     uintptr
	steps  []writeStep
	writes bytes.Buffer
	calls  int
}

func (w *descriptorWriter) Fd() uintptr { return w.fd }

func (w *descriptorWriter) Write(contents []byte) (int, error) {
	step := writeStep{n: -1}
	if w.calls < len(w.steps) {
		step = w.steps[w.calls]
	}
	w.calls++
	written := step.n
	if written < 0 || written > len(contents) {
		written = len(contents)
	}
	_, _ = w.writes.Write(contents[:written])
	return written, step.err
}

type fakeTerminalDriver struct {
	terminals map[int]bool
	events    []string

	rawState    any
	outputState any
	width       int
	height      int

	rawErr           error
	enableOutputErr  error
	restoreOutputErr error
	restoreRawErr    error
	sizeErr          error
}

func (d *fakeTerminalDriver) isTerminal(fd int) bool {
	d.events = append(d.events, "terminal:"+strconv.Itoa(fd))
	return d.terminals[fd]
}

func (d *fakeTerminalDriver) makeRaw(fd int) (any, error) {
	d.events = append(d.events, "raw:"+strconv.Itoa(fd))
	return d.rawState, d.rawErr
}

func (d *fakeTerminalDriver) restoreRaw(fd int, state any) error {
	d.events = append(d.events, "restore-raw:"+strconv.Itoa(fd)+":"+stateName(state))
	return d.restoreRawErr
}

func (d *fakeTerminalDriver) size(fd int) (int, int, error) {
	d.events = append(d.events, "size:"+strconv.Itoa(fd))
	return d.width, d.height, d.sizeErr
}

func (d *fakeTerminalDriver) enableOutput(fd int) (any, error) {
	d.events = append(d.events, "enable-output:"+strconv.Itoa(fd))
	return d.outputState, d.enableOutputErr
}

func (d *fakeTerminalDriver) restoreOutput(fd int, state any) error {
	d.events = append(d.events, "restore-output:"+strconv.Itoa(fd)+":"+stateName(state))
	return d.restoreOutputErr
}

func stateName(value any) string {
	text, _ := value.(string)
	return text
}

func testOpener(driver *fakeTerminalDriver) *opener {
	return &opener{driver: driver}
}

func testStreams() (descriptorReader, *descriptorWriter) {
	return descriptorReader{Reader: strings.NewReader(""), fd: 11}, &descriptorWriter{fd: 12}
}

func TestSessionEntersSizesAndRestoresTerminalExactlyOnce(t *testing.T) {
	driver := &fakeTerminalDriver{
		terminals:   map[int]bool{11: true, 12: true},
		rawState:    "raw-state",
		outputState: "output-state",
		width:       100,
		height:      30,
	}
	input, output := testStreams()
	session, err := testOpener(driver).Open(input, output)
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	if got := output.writes.String(); got != enterScreen {
		t.Fatalf("enter output = %q, want %q", got, enterScreen)
	}
	width, height, err := session.Size()
	if err != nil || width != 100 || height != 30 {
		t.Fatalf("Size() = (%d, %d, %v), want (100, 30, nil)", width, height, err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("second Close(): %v", err)
	}
	if got := output.writes.String(); got != enterScreen+leaveScreen {
		t.Fatalf("complete output = %q, want enter and leave controls", got)
	}
	wantEvents := []string{
		"terminal:11", "terminal:12", "raw:11", "enable-output:12", "size:12",
		"restore-output:12:output-state", "restore-raw:11:raw-state",
	}
	if got := driver.events; !equalStrings(got, wantEvents) {
		t.Fatalf("driver events = %v, want %v", got, wantEvents)
	}
	if _, _, err := session.Size(); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Size() after Close error = %v, want ErrSessionClosed", err)
	}
}

func TestOpenRequiresBothInjectedStreamsToBeTerminals(t *testing.T) {
	tests := []struct {
		name   string
		input  io.Reader
		output io.Writer
		driver *fakeTerminalDriver
	}{
		{
			name:   "input has no descriptor",
			input:  strings.NewReader(""),
			output: &descriptorWriter{fd: 12},
			driver: &fakeTerminalDriver{terminals: map[int]bool{12: true}},
		},
		{
			name:   "output has no descriptor",
			input:  descriptorReader{Reader: strings.NewReader(""), fd: 11},
			output: &bytes.Buffer{},
			driver: &fakeTerminalDriver{terminals: map[int]bool{11: true}},
		},
		{
			name:   "input is not terminal",
			input:  descriptorReader{Reader: strings.NewReader(""), fd: 11},
			output: &descriptorWriter{fd: 12},
			driver: &fakeTerminalDriver{terminals: map[int]bool{11: false, 12: true}},
		},
		{
			name:   "output is not terminal",
			input:  descriptorReader{Reader: strings.NewReader(""), fd: 11},
			output: &descriptorWriter{fd: 12},
			driver: &fakeTerminalDriver{terminals: map[int]bool{11: true, 12: false}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session, err := testOpener(test.driver).Open(test.input, test.output)
			if session != nil || !errors.Is(err, ErrNotTerminal) {
				t.Fatalf("Open() = (%v, %v), want (nil, ErrNotTerminal)", session, err)
			}
			for _, event := range test.driver.events {
				if strings.HasPrefix(event, "raw:") || strings.HasPrefix(event, "enable-output:") {
					t.Fatalf("non-terminal setup changed mode: events=%v", test.driver.events)
				}
			}
		})
	}
}

func TestOpenRollsBackEveryCompletedModeChange(t *testing.T) {
	rawFailure := errors.New("raw failed")
	outputFailure := errors.New("output failed")
	writeFailure := errors.New("write failed")
	tests := []struct {
		name       string
		driver     *fakeTerminalDriver
		steps      []writeStep
		wantError  error
		wantEvents []string
	}{
		{
			name: "raw setup fails before change",
			driver: &fakeTerminalDriver{
				terminals: map[int]bool{11: true, 12: true}, rawErr: rawFailure,
			},
			wantError:  rawFailure,
			wantEvents: []string{"terminal:11", "terminal:12", "raw:11"},
		},
		{
			name: "output setup restores raw input",
			driver: &fakeTerminalDriver{
				terminals: map[int]bool{11: true, 12: true}, rawState: "raw-state", enableOutputErr: outputFailure,
			},
			wantError: outputFailure,
			wantEvents: []string{
				"terminal:11", "terminal:12", "raw:11", "enable-output:12", "restore-raw:11:raw-state",
			},
		},
		{
			name: "screen setup restores output then input",
			driver: &fakeTerminalDriver{
				terminals: map[int]bool{11: true, 12: true}, rawState: "raw-state", outputState: "output-state",
			},
			steps:     []writeStep{{n: 0, err: writeFailure}},
			wantError: writeFailure,
			wantEvents: []string{
				"terminal:11", "terminal:12", "raw:11", "enable-output:12",
				"restore-output:12:output-state", "restore-raw:11:raw-state",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, output := testStreams()
			output.steps = test.steps
			session, err := testOpener(test.driver).Open(input, output)
			if session != nil || !errors.Is(err, test.wantError) {
				t.Fatalf("Open() = (%v, %v), want error %v", session, err, test.wantError)
			}
			if errors.Is(err, ErrRestoreFailed) {
				t.Fatalf("successful setup rollback was marked ErrRestoreFailed: %v", err)
			}
			if got := test.driver.events; !equalStrings(got, test.wantEvents) {
				t.Fatalf("driver events = %v, want %v", got, test.wantEvents)
			}
		})
	}
}

func TestOpenReportsRollbackFailureAlongsideSetupFailure(t *testing.T) {
	setupFailure := errors.New("output setup failed")
	restoreFailure := errors.New("raw restore failed")
	driver := &fakeTerminalDriver{
		terminals:       map[int]bool{11: true, 12: true},
		rawState:        "raw-state",
		enableOutputErr: setupFailure,
		restoreRawErr:   restoreFailure,
	}
	input, output := testStreams()
	session, err := testOpener(driver).Open(input, output)
	if session != nil || !errors.Is(err, setupFailure) || !errors.Is(err, restoreFailure) {
		t.Fatalf("Open() = (%v, %v), want both setup and rollback failures", session, err)
	}
	if !errors.Is(err, ErrRestoreFailed) {
		t.Fatalf("Open() error = %v, want ErrRestoreFailed", err)
	}
}

func TestCloseAttemptsAllRestorationAfterControlWriteFailure(t *testing.T) {
	writeFailure := errors.New("leave write failed")
	outputFailure := errors.New("output restore failed")
	rawFailure := errors.New("raw restore failed")
	driver := &fakeTerminalDriver{
		terminals:        map[int]bool{11: true, 12: true},
		rawState:         "raw-state",
		outputState:      "output-state",
		restoreOutputErr: outputFailure,
		restoreRawErr:    rawFailure,
	}
	input, output := testStreams()
	output.steps = []writeStep{{n: -1}, {n: 0, err: writeFailure}}
	session, err := testOpener(driver).Open(input, output)
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	err = session.Close()
	if !errors.Is(err, writeFailure) || !errors.Is(err, outputFailure) || !errors.Is(err, rawFailure) {
		t.Fatalf("Close() error = %v, want write and both restore failures", err)
	}
	if !errors.Is(err, ErrRestoreFailed) {
		t.Fatalf("Close() error = %v, want ErrRestoreFailed", err)
	}
	wantSuffix := []string{"restore-output:12:output-state", "restore-raw:11:raw-state"}
	if got := driver.events[len(driver.events)-2:]; !equalStrings(got, wantSuffix) {
		t.Fatalf("restoration suffix = %v, want %v", got, wantSuffix)
	}
	if second := session.Close(); !errors.Is(second, writeFailure) || len(driver.events) != 6 {
		t.Fatalf("second Close() = %v, events=%v; restoration was not idempotent", second, driver.events)
	}
}

func TestControlWriteCompletesShortWritesAndRejectsNoProgress(t *testing.T) {
	t.Run("short write", func(t *testing.T) {
		driver := &fakeTerminalDriver{
			terminals: map[int]bool{11: true, 12: true}, rawState: "raw-state", outputState: "output-state",
		}
		input, output := testStreams()
		output.steps = []writeStep{{n: 2}}
		session, err := testOpener(driver).Open(input, output)
		if err != nil {
			t.Fatalf("Open(): %v", err)
		}
		if got := output.writes.String(); got != enterScreen {
			t.Fatalf("short-write output = %q, want %q", got, enterScreen)
		}
		if err := session.Close(); err != nil {
			t.Fatalf("Close(): %v", err)
		}
	})

	t.Run("no progress", func(t *testing.T) {
		driver := &fakeTerminalDriver{
			terminals: map[int]bool{11: true, 12: true}, rawState: "raw-state", outputState: "output-state",
		}
		input, output := testStreams()
		output.steps = []writeStep{{n: 0}}
		session, err := testOpener(driver).Open(input, output)
		if session != nil || !errors.Is(err, io.ErrNoProgress) {
			t.Fatalf("Open() = (%v, %v), want io.ErrNoProgress", session, err)
		}
		if got := driver.events[len(driver.events)-2:]; !equalStrings(got, []string{"restore-output:12:output-state", "restore-raw:11:raw-state"}) {
			t.Fatalf("no-progress rollback = %v", driver.events)
		}
	})
}

func TestSessionSizePreservesDriverFailure(t *testing.T) {
	sizeFailure := errors.New("size failed")
	driver := &fakeTerminalDriver{
		terminals: map[int]bool{11: true, 12: true}, rawState: "raw-state", outputState: "output-state", sizeErr: sizeFailure,
	}
	input, output := testStreams()
	session, err := testOpener(driver).Open(input, output)
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer func() { _ = session.Close() }()
	if _, _, err := session.Size(); !errors.Is(err, sizeFailure) {
		t.Fatalf("Size() error = %v, want size failure", err)
	}
}

func TestProductionAdapterRejectsOrdinaryStreamsWithoutWriting(t *testing.T) {
	var output bytes.Buffer
	session, err := New().Open(strings.NewReader(""), &output)
	if session != nil || !errors.Is(err, ErrNotTerminal) {
		t.Fatalf("Open() = (%v, %v), want (nil, ErrNotTerminal)", session, err)
	}
	if output.Len() != 0 {
		t.Fatalf("non-terminal output = %q, want empty", output.String())
	}
}

func TestSessionReadRejectsClosedSessionAndNilContextWithoutWaiting(t *testing.T) {
	session := &terminalSession{}
	if _, err := session.Read(nil, make([]byte, 1)); !errors.Is(err, errNilReadContext) {
		t.Fatalf("nil-context Read() error = %v", err)
	}
	session.closed.Store(true)
	if _, err := session.Read(context.Background(), make([]byte, 1)); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("closed Read() error = %v", err)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
