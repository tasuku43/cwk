//go:build readiness_eval && (darwin || linux)

package readinessio

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"golang.org/x/sys/unix"
)

func TestAppendBoundedCrossProcessLock(t *testing.T) {
	if path := os.Getenv("CWK_READINESS_LOCK_TEST"); path != "" {
		if err := AppendBounded(path, []byte("child\n"), 100); err == nil {
			t.Fatal("append succeeded while another process held the lock")
		}
		return
	}
	path := filepath.Join(t.TempDir(), "journal")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	f, err := OpenAppendRegular(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer unix.Flock(int(f.Fd()), unix.LOCK_UN)
	cmd := exec.Command(os.Args[0], "-test.run=^TestAppendBoundedCrossProcessLock$")
	cmd.Env = append(os.Environ(), "CWK_READINESS_LOCK_TEST="+path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child: %v: %s", err, out)
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) != 0 {
		t.Fatalf("journal changed: %q, %v", data, err)
	}
}

func TestAppendBoundedConcurrentLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	const limit = 64
	var wg sync.WaitGroup
	var successes atomic.Int64
	for i := 0; i < 128; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if AppendBounded(path, []byte("x\n"), limit) == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if successes.Load() == 0 || int64(len(data)) != successes.Load()*2 || len(data) > limit || !bytes.Equal(data, bytes.Repeat([]byte("x\n"), int(successes.Load()))) {
		t.Fatalf("invalid journal: %d successes, %q", successes.Load(), data)
	}
	if err := AppendBounded(path, make([]byte, limit+1), limit); err == nil {
		t.Fatal("oversized append succeeded")
	}
}
