//go:build readiness_eval && (darwin || linux)

package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func readinessHelperProcess(path string) *exec.Cmd {
	return exec.Command(os.Args[0], "-test.run=^TestReadinessHelper$", "--", readinessEvalAuditFlag, path, "--help")
}

func TestReadinessReservationSerializesProcessesAndEnforcesBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var successes atomic.Int64
	start := make(chan struct{})
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if readinessHelperProcess(path).Run() == nil {
				successes.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	count, err := countReadinessJournalEvents(path)
	if err != nil || count != int(successes.Load()) || count < 1 || count > readinessEvalCallLimit {
		t.Fatalf("successes=%d count=%d err=%v", successes.Load(), count, err)
	}
	for count < readinessEvalCallLimit {
		if out, err := readinessHelperProcess(path).CombinedOutput(); err != nil {
			t.Fatalf("sequential helper: %v %s", err, out)
		}
		count++
	}
	if out, err := readinessHelperProcess(path).CombinedOutput(); err == nil {
		t.Fatalf("budget exceeded without failure: %s", out)
	}
	if got, err := countReadinessJournalEvents(path); err != nil || got != readinessEvalCallLimit {
		t.Fatalf("count=%d err=%v", got, err)
	}
}

func TestReadinessIncompleteReservationBlocksNextInvocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal")
	start, err := json.Marshal(readinessInvocationEvent{Type: "cwk_invocation_started", Argv: []string{"--help"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(start, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readinessHelperProcess(path).CombinedOutput(); err == nil {
		t.Fatal("incomplete prior reservation allowed another invocation")
	}
	if _, err := countReadinessJournalEvents(path); err == nil {
		t.Fatal("incomplete reservation accepted")
	}
}

func TestReadinessReservationSurvivesInterruptedHelper(t *testing.T) {
	if len(os.Args) >= 3 && os.Args[len(os.Args)-2] == "--cwk-reservation-test-child" {
		path := os.Args[len(os.Args)-1]
		os.Args = []string{os.Args[0], "--", readinessEvalAuditFlag, path, "--help"}
		readinessReservationTestHook = func() { fmt.Fprintln(os.Stdout, "reserved"); time.Sleep(time.Hour) }
		TestReadinessHelper(t)
		return
	}
	path := filepath.Join(t.TempDir(), "journal")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestReadinessReservationSurvivesInterruptedHelper$", "--", "--cwk-reservation-test-child", path)
	stdout, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if line != "reserved\n" {
			t.Fatalf("helper did not reserve: %q", line)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("helper did not reach reservation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	contender := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestReadinessHelper$", "--", readinessEvalAuditFlag, path, "--help")
	if output, err := contender.CombinedOutput(); err == nil || ctx.Err() != nil || !strings.Contains(string(output), "readiness journal lock is unavailable") {
		t.Fatalf("second helper did not promptly reject held lock: error=%v context=%v output=%q", err, ctx.Err(), output)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	if output, err := readinessHelperProcess(path).CombinedOutput(); err == nil || !strings.Contains(string(output), "incomplete invocation reservation") {
		t.Fatalf("next helper did not reject interrupted reservation: error=%v output=%q", err, output)
	}
	if _, err := countReadinessJournalEvents(path); err == nil {
		t.Fatal("interrupted reservation was accepted")
	}
}
