package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
)

func verifyArchive(archiveFormat, archivePath string, specs []inputSpec) error {
	if err := validateArchiveFormat(archiveFormat); err != nil {
		return err
	}
	if err := validateInputSpecs(specs); err != nil {
		return err
	}
	orderedSpecs := append([]inputSpec(nil), specs...)
	sort.Slice(orderedSpecs, func(left, right int) bool {
		return orderedSpecs[left].name < orderedSpecs[right].name
	})

	archive, archiveInfo, err := openVerifiedRegularFile(archivePath, "archive")
	if err != nil {
		return err
	}
	defer archive.Close()

	switch archiveFormat {
	case "tar.gz":
		return verifyTarGzipArchive(archive, orderedSpecs)
	case "zip":
		return verifyZipArchive(archive, archiveInfo.Size(), orderedSpecs)
	default:
		return fmt.Errorf("unsupported archive format %q", archiveFormat)
	}
}

func openVerifiedRegularFile(path, description string) (*os.File, os.FileInfo, error) {
	info, err := os.Lstat(path) // #nosec G703 -- release verification intentionally accepts an explicit local path.
	if err != nil {
		return nil, nil, fmt.Errorf("inspect %s: %w", description, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s must be a regular file reached without a symbolic link", description)
	}
	file, err := os.Open(path) // #nosec G304 G703 -- the explicit file was type-checked immediately above.
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", description, err)
	}
	openedInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, fmt.Errorf("inspect opened %s: %w", description, err)
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		_ = file.Close()
		return nil, nil, fmt.Errorf("%s changed while it was being opened", description)
	}
	return file, openedInfo, nil
}

func verifyTarGzipArchive(archive *os.File, specs []inputSpec) error {
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("open gzip archive: %w", err)
	}
	defer gzipReader.Close()
	if !gzipReader.ModTime.Equal(canonicalArchiveTime) || gzipReader.OS != 255 ||
		gzipReader.Name != "" || gzipReader.Comment != "" || len(gzipReader.Extra) != 0 {
		return errors.New("gzip header is not canonical")
	}

	tarReader := tar.NewReader(gzipReader)
	for _, spec := range specs {
		header, err := tarReader.Next()
		if err != nil {
			return fmt.Errorf("read tar header for %q: %w", spec.name, err)
		}
		if header.Name != spec.name {
			return fmt.Errorf("tar entry order or name = %q, want %q", header.Name, spec.name)
		}
		if header.Mode != int64(spec.mode) {
			return fmt.Errorf("tar entry %q mode = %04o, want %04o", spec.name, header.Mode, spec.mode)
		}
		if header.Typeflag != tar.TypeReg || header.Uid != 0 || header.Gid != 0 ||
			header.Uname != "" || header.Gname != "" || !header.ModTime.Equal(canonicalArchiveTime) ||
			header.Format != tar.FormatUSTAR {
			return fmt.Errorf("tar entry %q header is not canonical", spec.name)
		}
		if err := verifyEntryContent(tarReader, header.Size, spec); err != nil {
			return err
		}
	}
	if header, err := tarReader.Next(); err != io.EOF {
		if err != nil {
			return fmt.Errorf("read trailing tar entry: %w", err)
		}
		return fmt.Errorf("tar archive contains unexpected entry %q", header.Name)
	}
	return nil
}

func verifyZipArchive(archive *os.File, archiveSize int64, specs []inputSpec) error {
	zipReader, err := zip.NewReader(archive, archiveSize)
	if err != nil {
		return fmt.Errorf("open zip archive: %w", err)
	}
	if len(zipReader.File) != len(specs) {
		return fmt.Errorf("zip entry count = %d, want %d", len(zipReader.File), len(specs))
	}
	for index, spec := range specs {
		entry := zipReader.File[index]
		if entry.Name != spec.name {
			return fmt.Errorf("zip entry order or name = %q, want %q", entry.Name, spec.name)
		}
		if !entry.Mode().IsRegular() || entry.Mode().Perm() != spec.mode {
			return fmt.Errorf("zip entry %q mode = %04o, want regular %04o", spec.name, entry.Mode().Perm(), spec.mode)
		}
		if entry.Method != zip.Deflate || !entry.Modified.Equal(canonicalArchiveTime) {
			return fmt.Errorf("zip entry %q header is not canonical", spec.name)
		}
		content, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %q: %w", spec.name, err)
		}
		if entry.UncompressedSize64 > math.MaxInt64 {
			_ = content.Close()
			return fmt.Errorf("zip entry %q is too large to verify", spec.name)
		}
		// #nosec G115 -- the explicit MaxInt64 bound above makes this conversion exact.
		verifyErr := verifyEntryContent(content, int64(entry.UncompressedSize64), spec)
		closeErr := content.Close()
		if verifyErr != nil {
			return verifyErr
		}
		if closeErr != nil {
			return fmt.Errorf("close zip entry %q: %w", spec.name, closeErr)
		}
	}
	return nil
}

func verifyEntryContent(actual io.Reader, actualSize int64, spec inputSpec) error {
	expected, expectedInfo, err := openVerifiedRegularFile(spec.path, fmt.Sprintf("expected input %q", spec.name))
	if err != nil {
		return err
	}
	defer expected.Close()
	if actualSize != expectedInfo.Size() {
		return fmt.Errorf("archive entry %q size = %d, want %d", spec.name, actualSize, expectedInfo.Size())
	}

	actualBuffer := make([]byte, deterministicBuffer)
	expectedBuffer := make([]byte, deterministicBuffer)
	for {
		actualRead, actualErr := io.ReadFull(actual, actualBuffer)
		expectedRead, expectedErr := io.ReadFull(expected, expectedBuffer)
		if actualRead != expectedRead || !bytes.Equal(actualBuffer[:actualRead], expectedBuffer[:expectedRead]) {
			return fmt.Errorf("archive entry %q content differs from its reviewed input", spec.name)
		}
		actualDone := actualErr == io.EOF || actualErr == io.ErrUnexpectedEOF
		expectedDone := expectedErr == io.EOF || expectedErr == io.ErrUnexpectedEOF
		if actualDone || expectedDone {
			if actualDone && expectedDone {
				return nil
			}
			return fmt.Errorf("archive entry %q content length changed during verification", spec.name)
		}
		if actualErr != nil {
			return fmt.Errorf("read archive entry %q: %w", spec.name, actualErr)
		}
		if expectedErr != nil {
			return fmt.Errorf("read expected input %q: %w", spec.name, expectedErr)
		}
	}
}
