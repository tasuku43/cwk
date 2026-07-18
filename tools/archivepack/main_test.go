package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackProducesDeterministicCanonicalArchives(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "tool")
	contents := bytes.Repeat([]byte("deterministic release bytes\n"), 4096)
	if err := os.WriteFile(input, contents, archiveMode); err != nil {
		t.Fatal(err)
	}

	for _, archiveFormat := range []string{"tar.gz", "zip"} {
		t.Run(archiveFormat, func(t *testing.T) {
			first := filepath.Join(root, "first."+archiveFormat)
			second := filepath.Join(root, "second."+archiveFormat)
			if err := pack(archiveFormat, input, "tool", first); err != nil {
				t.Fatal(err)
			}
			if err := pack(archiveFormat, input, "tool", second); err != nil {
				t.Fatal(err)
			}
			firstBytes, err := os.ReadFile(first)
			if err != nil {
				t.Fatal(err)
			}
			secondBytes, err := os.ReadFile(second)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(firstBytes, secondBytes) {
				t.Fatal("same input produced different archive bytes")
			}
			info, err := os.Stat(first)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != archiveOutputMode {
				t.Fatalf("archive mode = %04o, want %04o", got, archiveOutputMode)
			}

			switch archiveFormat {
			case "tar.gz":
				assertCanonicalTarGzip(t, first, contents)
			case "zip":
				assertCanonicalZip(t, first, contents)
			}
		})
	}
}

func TestPackRefusesExistingOutput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "tool")
	output := filepath.Join(root, "tool.tar.gz")
	if err := os.WriteFile(input, []byte("binary"), archiveMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := pack("tar.gz", input, "tool", output); err == nil || !strings.Contains(err.Error(), "without overwrite") {
		t.Fatalf("pack() error = %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("existing output changed: %q", data)
	}
}

func TestPackRefusesSymbolicLinkOutputWithoutChangingTarget(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "tool")
	external := filepath.Join(root, "external")
	output := filepath.Join(root, "tool.zip")
	if err := os.WriteFile(input, []byte("binary"), archiveMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(external, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, output); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if err := pack("zip", input, "tool", output); err == nil || !strings.Contains(err.Error(), "without overwrite") {
		t.Fatalf("pack() error = %v", err)
	}
	data, err := os.ReadFile(external)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("symbolic-link target changed: %q", data)
	}
}

func TestCreateArchiveRemovesPartialOutputOnFailure(t *testing.T) {
	for _, archiveFormat := range []string{"tar.gz", "zip"} {
		t.Run(archiveFormat, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "partial."+archiveFormat)
			err := createArchive(archiveFormat, strings.NewReader("short"), "tool", 100, output)
			if err == nil || !strings.Contains(err.Error(), "input size changed") {
				t.Fatalf("createArchive() error = %v", err)
			}
			if _, err := os.Lstat(output); !os.IsNotExist(err) {
				t.Fatalf("partial output remains: %v", err)
			}
		})
	}
}

func TestPackRejectsUnsafeInputs(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "tool")
	if err := os.WriteFile(input, []byte("binary"), archiveMode); err != nil {
		t.Fatal(err)
	}
	for formatIndex, archiveFormat := range []string{"tar.gz", "zip"} {
		for entryIndex, entry := range []string{"", ".", "..", "../tool", "/absolute/tool", "dir/tool", `dir\tool`, "bad\x00name", "bad\nname"} {
			output := filepath.Join(root, fmt.Sprintf("unsafe-%d-%d.archive", formatIndex, entryIndex))
			if err := pack(archiveFormat, input, entry, output); err == nil {
				t.Fatalf("pack(%q) accepted entry %q", archiveFormat, entry)
			}
			if _, err := os.Lstat(output); !os.IsNotExist(err) {
				t.Fatalf("pack(%q) created output for entry %q: %v", archiveFormat, entry, err)
			}
		}
	}
	if err := pack("rar", input, "tool", filepath.Join(root, "tool.rar")); err == nil {
		t.Fatal("pack() accepted an unsupported format")
	}
}

func TestPackRejectsDirectoryInput(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "tool.tar.gz")
	if err := pack("tar.gz", root, "tool", output); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("pack() error = %v", err)
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("pack() created output for a directory: %v", err)
	}
}

func TestPackRejectsSymbolicLinkInput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "tool")
	if err := os.WriteFile(input, []byte("binary"), archiveMode); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked-tool")
	if err := os.Symlink(input, link); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	output := filepath.Join(root, "tool.zip")
	if err := pack("zip", link, "tool", output); err == nil || !strings.Contains(err.Error(), "without a symbolic link") {
		t.Fatalf("pack() error = %v", err)
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("pack() created output for a symbolic link: %v", err)
	}
}

func assertCanonicalTarGzip(t *testing.T, path string, want []byte) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()
	if !gzipReader.ModTime.Equal(canonicalArchiveTime) || gzipReader.Name != "" || gzipReader.Comment != "" || gzipReader.OS != 255 {
		t.Fatalf("non-canonical gzip header: %+v", gzipReader.Header)
	}
	tarReader := tar.NewReader(gzipReader)
	header, err := tarReader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header.Name != "tool" || header.Mode != archiveMode || header.Uid != 0 || header.Gid != 0 ||
		header.Uname != "" || header.Gname != "" || !header.ModTime.Equal(canonicalArchiveTime) ||
		header.Typeflag != tar.TypeReg || header.Format != tar.FormatUSTAR {
		t.Fatalf("non-canonical tar header: %+v", header)
	}
	data, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, want) {
		t.Fatal("tar entry content changed")
	}
	if _, err := tarReader.Next(); err != io.EOF {
		t.Fatalf("tar contains another entry: %v", err)
	}
}

func assertCanonicalZip(t *testing.T, path string, want []byte) {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 1 {
		t.Fatalf("zip entries = %d, want 1", len(reader.File))
	}
	entry := reader.File[0]
	if entry.Name != "tool" || entry.Mode().Perm() != archiveMode || !entry.Modified.Equal(canonicalArchiveTime) {
		t.Fatalf("non-canonical zip header: %+v", entry.FileHeader)
	}
	file, err := entry.Open()
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if !bytes.Equal(data, want) {
		t.Fatal("zip entry content changed")
	}
}
