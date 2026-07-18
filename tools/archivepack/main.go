// Command archivepack creates one byte-for-byte reproducible release archive.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	archiveMode         = 0o755
	archiveOutputMode   = 0o644
	privateOutputMode   = 0o600
	deterministicBuffer = 32 * 1024
)

var canonicalArchiveTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

func main() {
	if len(os.Args) != 5 {
		fatal(errors.New("usage: archivepack <tar.gz|zip> <input-file> <entry-name> <output-file>"))
	}
	if err := pack(os.Args[1], os.Args[2], os.Args[3], os.Args[4]); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "archivepack: %v\n", err)
	os.Exit(1)
}

func pack(archiveFormat, inputPath, entryName, outputPath string) (returnErr error) {
	if archiveFormat != "tar.gz" && archiveFormat != "zip" {
		return fmt.Errorf("unsupported archive format %q", archiveFormat)
	}
	if err := validateEntryName(entryName); err != nil {
		return err
	}

	info, err := os.Lstat(inputPath) // #nosec G703 -- archivepack intentionally accepts an explicit local release-staging path, not a path relative to a trusted root.
	if err != nil {
		return fmt.Errorf("inspect input: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("input must be a regular file reached without a symbolic link")
	}
	input, err := os.Open(inputPath) // #nosec G304 G703 -- the explicit release-staging input was type-checked above; no trusted base-directory boundary is claimed.
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer input.Close()
	return createArchive(archiveFormat, input, entryName, info.Size(), outputPath)
}

func createArchive(archiveFormat string, input io.Reader, entryName string, size int64, outputPath string) (returnErr error) {
	output, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, privateOutputMode) // #nosec G304 G703 -- the caller supplies an explicit staging output and O_EXCL prevents overwrite or symlink traversal.
	if err != nil {
		return fmt.Errorf("create output without overwrite: %w", err)
	}
	success := false
	defer func() {
		if closeErr := output.Close(); returnErr == nil && closeErr != nil {
			returnErr = fmt.Errorf("close output: %w", closeErr)
		}
		if returnErr != nil || !success {
			_ = os.Remove(outputPath) // #nosec G703 -- cleanup targets only the exact staging path this call created with O_EXCL.
		}
	}()

	switch archiveFormat {
	case "tar.gz":
		returnErr = writeTarGzip(output, input, entryName, size)
	case "zip":
		returnErr = writeZip(output, input, entryName, size)
	default:
		returnErr = fmt.Errorf("unsupported archive format %q", archiveFormat)
	}
	if returnErr != nil {
		return returnErr
	}
	if err := output.Chmod(archiveOutputMode); err != nil {
		return fmt.Errorf("set completed archive permissions: %w", err)
	}
	success = true
	return nil
}

func validateEntryName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
		return fmt.Errorf("entry name %q is not a safe archive basename", name)
	}
	for _, character := range name {
		if character < 0x20 || character == 0x7f {
			return fmt.Errorf("entry name %q contains a control character", name)
		}
	}
	return nil
}

func writeTarGzip(output io.Writer, input io.Reader, entryName string, size int64) error {
	gzipWriter, err := gzip.NewWriterLevel(output, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("create gzip writer: %w", err)
	}
	gzipWriter.Header.ModTime = canonicalArchiveTime
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)
	tarClosed := false
	gzipClosed := false
	defer func() {
		if !tarClosed {
			_ = tarWriter.Close()
		}
		if !gzipClosed {
			_ = gzipWriter.Close()
		}
	}()

	header := &tar.Header{
		Name:     entryName,
		Mode:     archiveMode,
		Uid:      0,
		Gid:      0,
		Size:     size,
		ModTime:  canonicalArchiveTime,
		Typeflag: tar.TypeReg,
		Format:   tar.FormatUSTAR,
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("write tar header: %w", err)
	}
	written, err := copyDeterministic(tarWriter, input)
	if err != nil {
		return fmt.Errorf("write tar entry: %w", err)
	}
	if written != size {
		return fmt.Errorf("input size changed while archiving: wrote %d bytes, expected %d", written, size)
	}
	tarClosed = true
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("close tar writer: %w", err)
	}
	gzipClosed = true
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}
	return nil
}

func writeZip(output io.Writer, input io.Reader, entryName string, size int64) error {
	zipWriter := zip.NewWriter(output)
	zipWriter.RegisterCompressor(zip.Deflate, func(destination io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(destination, flate.BestCompression)
	})
	closed := false
	defer func() {
		if !closed {
			_ = zipWriter.Close()
		}
	}()

	header := &zip.FileHeader{Name: entryName, Method: zip.Deflate}
	header.SetMode(archiveMode)
	header.SetModTime(canonicalArchiveTime)
	entry, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("write zip header: %w", err)
	}
	written, err := copyDeterministic(entry, input)
	if err != nil {
		return fmt.Errorf("write zip entry: %w", err)
	}
	if written != size {
		return fmt.Errorf("input size changed while archiving: wrote %d bytes, expected %d", written, size)
	}
	closed = true
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("close zip writer: %w", err)
	}
	return nil
}

func copyDeterministic(destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, deterministicBuffer)
	var total int64
	for {
		read, readErr := io.ReadFull(source, buffer)
		if read != 0 {
			written, writeErr := destination.Write(buffer[:read])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != read {
				return total, io.ErrShortWrite
			}
		}
		switch readErr {
		case nil:
			continue
		case io.EOF, io.ErrUnexpectedEOF:
			return total, nil
		default:
			return total, readErr
		}
	}
}
