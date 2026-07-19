package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type expectedEntry struct {
	name     string
	mode     fs.FileMode
	contents []byte
}

func TestPackProducesDeterministicCanonicalMultiEntryArchives(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "tool")
	license := filepath.Join(root, "license")
	notice := filepath.Join(root, "notices")
	executableContents := bytes.Repeat([]byte("deterministic release bytes\n"), 4096)
	licenseContents := []byte("project license\n")
	noticeContents := []byte("dependency notices\n")
	if err := os.WriteFile(executable, executableContents, executableMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(license, licenseContents, noticeMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(notice, noticeContents, noticeMode); err != nil {
		t.Fatal(err)
	}

	want := []expectedEntry{
		{name: "LICENSE", mode: noticeMode, contents: licenseContents},
		{name: "THIRD_PARTY_NOTICES", mode: noticeMode, contents: noticeContents},
		{name: "tool", mode: executableMode, contents: executableContents},
	}
	forward := []inputSpec{
		{path: executable, name: "tool", mode: executableMode},
		{path: notice, name: "THIRD_PARTY_NOTICES", mode: noticeMode},
		{path: license, name: "LICENSE", mode: noticeMode},
	}
	reverse := []inputSpec{forward[2], forward[1], forward[0]}

	for _, archiveFormat := range []string{"tar.gz", "zip"} {
		t.Run(archiveFormat, func(t *testing.T) {
			first := filepath.Join(root, "first."+archiveFormat)
			second := filepath.Join(root, "second."+archiveFormat)
			if err := pack(archiveFormat, first, forward); err != nil {
				t.Fatal(err)
			}
			if err := pack(archiveFormat, second, reverse); err != nil {
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
				t.Fatal("equivalent inputs in different orders produced different archive bytes")
			}
			info, err := os.Stat(first)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != archiveOutputMode {
				t.Fatalf("archive mode = %04o, want %04o", got, archiveOutputMode)
			}
			if err := verifyArchive(archiveFormat, first, reverse); err != nil {
				t.Fatalf("verifyArchive() rejected canonical archive: %v", err)
			}

			switch archiveFormat {
			case "tar.gz":
				assertCanonicalTarGzip(t, first, want)
			case "zip":
				assertCanonicalZip(t, first, want)
			}
		})
	}
}

func TestVerifyArchiveRejectsWrongHeaderMode(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "tool")
	if err := os.WriteFile(executable, []byte("binary"), executableMode); err != nil {
		t.Fatal(err)
	}
	for _, archiveFormat := range []string{"tar.gz", "zip"} {
		t.Run(archiveFormat, func(t *testing.T) {
			archive := filepath.Join(root, "wrong-mode."+archiveFormat)
			if err := pack(archiveFormat, archive, oneInput(executable, "tool", noticeMode)); err != nil {
				t.Fatal(err)
			}
			err := verifyArchive(archiveFormat, archive, oneInput(executable, "tool", executableMode))
			if err == nil || !strings.Contains(err.Error(), "mode") {
				t.Fatalf("verifyArchive() error = %v, want mode mismatch", err)
			}
		})
	}
}

func TestVerifyArchiveRejectsChangedReviewedInput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "notices")
	if err := os.WriteFile(input, []byte("reviewed\n"), noticeMode); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "notices.tar.gz")
	if err := pack("tar.gz", archive, oneInput(input, "THIRD_PARTY_NOTICES", noticeMode)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("changed!\n"), noticeMode); err != nil {
		t.Fatal(err)
	}
	err := verifyArchive("tar.gz", archive, oneInput(input, "THIRD_PARTY_NOTICES", noticeMode))
	if err == nil || (!strings.Contains(err.Error(), "size") && !strings.Contains(err.Error(), "content")) {
		t.Fatalf("verifyArchive() error = %v, want content mismatch", err)
	}
}

func TestVerifyArchiveRejectsSymbolicLinks(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "notices")
	if err := os.WriteFile(input, []byte("reviewed\n"), noticeMode); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "notices.zip")
	if err := pack("zip", archive, oneInput(input, "THIRD_PARTY_NOTICES", noticeMode)); err != nil {
		t.Fatal(err)
	}
	archiveLink := filepath.Join(root, "linked.zip")
	inputLink := filepath.Join(root, "linked-notices")
	if err := os.Symlink(archive, archiveLink); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if err := os.Symlink(input, inputLink); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if err := verifyArchive("zip", archiveLink, oneInput(input, "THIRD_PARTY_NOTICES", noticeMode)); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("verifyArchive() accepted symbolic-link archive: %v", err)
	}
	if err := verifyArchive("zip", archive, oneInput(inputLink, "THIRD_PARTY_NOTICES", noticeMode)); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("verifyArchive() accepted symbolic-link reviewed input: %v", err)
	}
}

func TestPackRefusesExistingOutput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "tool")
	output := filepath.Join(root, "tool.tar.gz")
	if err := os.WriteFile(input, []byte("binary"), executableMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := pack("tar.gz", output, oneInput(input, "tool", executableMode)); err == nil || !strings.Contains(err.Error(), "without overwrite") {
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
	if err := os.WriteFile(input, []byte("binary"), executableMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(external, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, output); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if err := pack("zip", output, oneInput(input, "tool", executableMode)); err == nil || !strings.Contains(err.Error(), "without overwrite") {
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

func TestCreateArchiveRemovesPartialOutputOnLaterEntryFailure(t *testing.T) {
	entries := []archiveEntry{
		{input: strings.NewReader("complete"), name: "a", size: 8, mode: noticeMode},
		{input: strings.NewReader("short"), name: "b", size: 100, mode: executableMode},
	}
	for _, archiveFormat := range []string{"tar.gz", "zip"} {
		t.Run(archiveFormat, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "partial."+archiveFormat)
			err := createArchive(archiveFormat, entries, output)
			if err == nil || !strings.Contains(err.Error(), "size changed") {
				t.Fatalf("createArchive() error = %v", err)
			}
			if _, err := os.Lstat(output); !os.IsNotExist(err) {
				t.Fatalf("partial output remains: %v", err)
			}
		})
	}
}

func TestPackRejectsUnsafeEntryNamesBeforeCreatingOutput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "tool")
	if err := os.WriteFile(input, []byte("binary"), executableMode); err != nil {
		t.Fatal(err)
	}
	for formatIndex, archiveFormat := range []string{"tar.gz", "zip"} {
		for entryIndex, entry := range []string{"", ".", "..", "../tool", "/absolute/tool", "dir/tool", `dir\tool`, "bad\x00name", "bad\nname"} {
			output := filepath.Join(root, fmt.Sprintf("unsafe-%d-%d.archive", formatIndex, entryIndex))
			if err := pack(archiveFormat, output, oneInput(input, entry, executableMode)); err == nil {
				t.Fatalf("pack(%q) accepted entry %q", archiveFormat, entry)
			}
			if _, err := os.Lstat(output); !os.IsNotExist(err) {
				t.Fatalf("pack(%q) created output for entry %q: %v", archiveFormat, entry, err)
			}
		}
	}
	output := filepath.Join(root, "tool.rar")
	if err := pack("rar", output, oneInput(input, "tool", executableMode)); err == nil {
		t.Fatal("pack() accepted an unsupported format")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("pack() created output for an unsupported format: %v", err)
	}
}

func TestPackRejectsDuplicateNamesAndUnsupportedModesBeforeCreatingOutput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "tool")
	if err := os.WriteFile(input, []byte("binary"), executableMode); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		specs []inputSpec
		want  string
	}{
		{
			name: "duplicate",
			specs: []inputSpec{
				{path: input, name: "tool", mode: executableMode},
				{path: input, name: "tool", mode: noticeMode},
			},
			want: "duplicate",
		},
		{
			name:  "unsupported mode",
			specs: oneInput(input, "tool", 0o600),
			want:  "unsupported archive entry mode",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := filepath.Join(root, strings.ReplaceAll(test.name, " ", "-")+".zip")
			err := pack("zip", output, test.specs)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("pack() error = %v, want text %q", err, test.want)
			}
			if _, err := os.Lstat(output); !os.IsNotExist(err) {
				t.Fatalf("pack() created output: %v", err)
			}
		})
	}
}

func TestPackRejectsDirectoryInput(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "tool.tar.gz")
	if err := pack("tar.gz", output, oneInput(root, "tool", executableMode)); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("pack() error = %v", err)
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("pack() created output for a directory: %v", err)
	}
}

func TestPackRejectsSymbolicLinkInAnyInput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "tool")
	if err := os.WriteFile(input, []byte("binary"), executableMode); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked-notices")
	if err := os.Symlink(input, link); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	output := filepath.Join(root, "tool.zip")
	specs := []inputSpec{
		{path: input, name: "tool", mode: executableMode},
		{path: link, name: "THIRD_PARTY_NOTICES", mode: noticeMode},
	}
	if err := pack("zip", output, specs); err == nil || !strings.Contains(err.Error(), "without a symbolic link") {
		t.Fatalf("pack() error = %v", err)
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("pack() created output for a symbolic-link input: %v", err)
	}
}

func TestCreateArchiveRejectsEmptyEntrySetBeforeCreatingOutput(t *testing.T) {
	output := filepath.Join(t.TempDir(), "empty.tar.gz")
	if err := createArchive("tar.gz", nil, output); err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("createArchive() error = %v", err)
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("createArchive() created empty archive output: %v", err)
	}
}

func TestParseArchiveModeAcceptsOnlyCanonicalReleaseModes(t *testing.T) {
	for value, want := range map[string]fs.FileMode{"0755": executableMode, "0644": noticeMode} {
		got, err := parseArchiveMode(value)
		if err != nil {
			t.Fatalf("parseArchiveMode(%q): %v", value, err)
		}
		if got != want {
			t.Fatalf("parseArchiveMode(%q) = %04o, want %04o", value, got, want)
		}
	}
	for _, value := range []string{"755", "644", "0600", "0777", ""} {
		if _, err := parseArchiveMode(value); err == nil {
			t.Fatalf("parseArchiveMode(%q) succeeded", value)
		}
	}
}

func oneInput(path, name string, mode fs.FileMode) []inputSpec {
	return []inputSpec{{path: path, name: name, mode: mode}}
}

func assertCanonicalTarGzip(t *testing.T, path string, want []expectedEntry) {
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
	for _, expected := range want {
		header, err := tarReader.Next()
		if err != nil {
			t.Fatal(err)
		}
		if header.Name != expected.name || header.Mode != int64(expected.mode) || header.Uid != 0 || header.Gid != 0 ||
			header.Uname != "" || header.Gname != "" || !header.ModTime.Equal(canonicalArchiveTime) ||
			header.Typeflag != tar.TypeReg || header.Format != tar.FormatUSTAR {
			t.Fatalf("non-canonical tar header for %q: %+v", expected.name, header)
		}
		data, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, expected.contents) {
			t.Fatalf("tar entry %q content changed", expected.name)
		}
	}
	if _, err := tarReader.Next(); err != io.EOF {
		t.Fatalf("tar contains another entry: %v", err)
	}
}

func assertCanonicalZip(t *testing.T, path string, want []expectedEntry) {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != len(want) {
		t.Fatalf("zip entries = %d, want %d", len(reader.File), len(want))
	}
	for index, expected := range want {
		entry := reader.File[index]
		if entry.Name != expected.name || entry.Mode().Perm() != expected.mode || !entry.Mode().IsRegular() || !entry.Modified.Equal(canonicalArchiveTime) {
			t.Fatalf("non-canonical zip header for %q: %+v", expected.name, entry.FileHeader)
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
		if !bytes.Equal(data, expected.contents) {
			t.Fatalf("zip entry %q content changed", expected.name)
		}
	}
}
