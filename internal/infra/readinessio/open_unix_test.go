//go:build readiness_eval && (aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos)

package readinessio

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestOpenRegularRejectsLinksAndFIFOsWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	regular := filepath.Join(root, "regular")
	if err := os.WriteFile(regular, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := OpenRegular(regular)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(root, "fifo")
	if err := unix.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{root, link, fifo} {
		for name, open := range map[string]func(string) (*os.File, error){"read": OpenRegular, "append": OpenAppendRegular} {
			t.Run(filepath.Base(path)+"/"+name, func(t *testing.T) {
				done := make(chan error, 1)
				go func() {
					opened, err := open(path)
					if opened != nil {
						_ = opened.Close()
					}
					done <- err
				}()
				select {
				case err := <-done:
					if err == nil {
						t.Fatal("accepted non-regular or linked input")
					}
				case <-time.After(time.Second):
					t.Fatal("opening input blocked")
				}
			})
		}
	}
}
