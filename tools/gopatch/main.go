// Command gopatch proposes a newer stable Go patch from the official Go
// download index. It intentionally edits only go.mod's go directive.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	officialReleaseIndexURL = "https://go.dev/dl/?mode=json"
	maxReleaseIndexBytes    = 8 << 20
	maxGoModBytes           = 1 << 20
	maxReleaseRecords       = 256
	maxFilesPerRelease      = 64
	requestTimeout          = 15 * time.Second
	responseHeaderTimeout   = 5 * time.Second
	dialTimeout             = 5 * time.Second
)

var (
	goVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	goDirectiveLine  = regexp.MustCompile(`^go([ \t]+)([0-9]+\.[0-9]+\.[0-9]+)([ \t]*)(\r?)$`)
	filenamePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,159}$`)
	platformPattern  = regexp.MustCompile(`^[a-z0-9]{0,16}$`)
	sha256Pattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type goVersion struct {
	major int64
	minor int64
	patch int64
}

type releaseRecord struct {
	Version string        `json:"version"`
	Stable  *bool         `json:"stable"`
	Files   []releaseFile `json:"files"`
}

type releaseFile struct {
	Filename *string `json:"filename"`
	OS       *string `json:"os"`
	Arch     *string `json:"arch"`
	Version  *string `json:"version"`
	SHA256   *string `json:"sha256"`
	Size     *int64  `json:"size"`
	Kind     *string `json:"kind"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	switch {
	case len(args) == 0:
		candidate, updated, err := updateFromOfficialSource(context.Background(), "go.mod")
		if err != nil {
			return err
		}
		if updated {
			_, err = fmt.Fprintln(stdout, candidate)
		}
		return err
	case len(args) == 1 && args[0] == "version":
		current, _, err := readGoMod("go.mod")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, current)
		return err
	case len(args) == 1 && args[0] == "minor":
		current, _, err := readGoMod("go.mod")
		if err != nil {
			return err
		}
		version, err := parseGoVersion(current)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "%d.%d\n", version.major, version.minor)
		return err
	case len(args) == 2 && args[0] == "apply":
		return applyCandidate("go.mod", args[1])
	default:
		return fmt.Errorf("使い方: go run ./tools/gopatch [version|minor|apply <version>]")
	}
}

func updateFromOfficialSource(ctx context.Context, goModPath string) (string, bool, error) {
	return updateFromEndpoint(ctx, goModPath, newOfficialHTTPClient(), officialReleaseIndexURL)
}

func updateFromEndpoint(ctx context.Context, goModPath string, client *http.Client, endpoint string) (string, bool, error) {
	current, original, info, err := readGoModFile(goModPath)
	if err != nil {
		return "", false, err
	}
	currentVersion, err := parseGoVersion(current)
	if err != nil {
		return "", false, err
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	candidate, found, err := latestStablePatch(ctx, client, endpoint, currentVersion)
	if err != nil {
		return "", false, err
	}
	if !found {
		return "", false, nil
	}

	updated, err := rewriteGoDirective(original, candidate.String())
	if err != nil {
		return "", false, err
	}
	if err := writeUpdatedFile(goModPath, original, updated, info); err != nil {
		return "", false, err
	}
	return candidate.String(), true, nil
}

func applyCandidate(goModPath, candidate string) error {
	current, original, info, err := readGoModFile(goModPath)
	if err != nil {
		return err
	}
	currentVersion, err := parseGoVersion(current)
	if err != nil {
		return err
	}
	candidateVersion, err := parseGoVersion(candidate)
	if err != nil {
		return fmt.Errorf("候補 Go バージョンが不正です: %w", err)
	}
	if candidateVersion.major != currentVersion.major || candidateVersion.minor != currentVersion.minor {
		return fmt.Errorf("Go の major/minor は変更できません: 現在 %s、候補 %s", current, candidate)
	}
	if compareVersions(candidateVersion, currentVersion) <= 0 {
		return fmt.Errorf("候補 Go バージョンは現在より新しい patch である必要があります: 現在 %s、候補 %s", current, candidate)
	}
	updated, err := rewriteGoDirective(original, candidate)
	if err != nil {
		return err
	}
	return writeUpdatedFile(goModPath, original, updated, info)
}

func latestStablePatch(ctx context.Context, client *http.Client, endpoint string, current goVersion) (goVersion, bool, error) {
	records, err := fetchReleaseRecords(ctx, client, endpoint)
	if err != nil {
		return goVersion{}, false, err
	}

	var candidate goVersion
	found := false
	for _, record := range records {
		version, err := parseReleaseVersion(record.Version)
		if err != nil {
			return goVersion{}, false, err
		}
		if version.major != current.major || version.minor != current.minor || compareVersions(version, current) <= 0 {
			continue
		}
		if !found || compareVersions(version, candidate) > 0 {
			candidate = version
			found = true
		}
	}
	return candidate, found, nil
}

func newOfficialHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: dialTimeout, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   dialTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		IdleConnTimeout:       30 * time.Second,
		DisableCompression:    true,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 2 || req.URL.Scheme != "https" || req.URL.Host != "go.dev" || req.URL.Path != "/dl/" || req.URL.RawQuery != "mode=json" || req.URL.User != nil {
				return errors.New("Go release index redirected outside its fixed HTTPS endpoint")
			}
			return nil
		},
	}
}

func fetchReleaseRecords(ctx context.Context, client *http.Client, endpoint string) ([]releaseRecord, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return nil, errors.New("Go release index URL is invalid")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("Go release index request could not be constructed: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Encoding", "identity")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("official Go release index request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("official Go release index returned HTTP %d", response.StatusCode)
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return nil, errors.New("official Go release index did not return application/json")
	}
	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return nil, errors.New("official Go release index used an unsupported content encoding")
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxReleaseIndexBytes+1))
	if err != nil {
		return nil, fmt.Errorf("official Go release index body could not be read: %w", err)
	}
	if len(body) > maxReleaseIndexBytes {
		return nil, fmt.Errorf("official Go release index exceeds %d bytes", maxReleaseIndexBytes)
	}
	return decodeReleaseRecords(body)
}

func decodeReleaseRecords(body []byte) ([]releaseRecord, error) {
	if len(body) == 0 || len(body) > maxReleaseIndexBytes || !utf8.Valid(body) {
		return nil, errors.New("official Go release index is empty, oversized, or invalid UTF-8")
	}
	if err := validateJSONWithoutDuplicateKeys(body); err != nil {
		return nil, fmt.Errorf("official Go release index is invalid JSON: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var records []releaseRecord
	if err := decoder.Decode(&records); err != nil {
		return nil, fmt.Errorf("official Go release index does not match the reviewed schema: %w", err)
	}
	if records == nil || len(records) == 0 || len(records) > maxReleaseRecords {
		return nil, errors.New("official Go release index has an invalid release count")
	}
	if err := validateReleaseRecords(records); err != nil {
		return nil, err
	}
	return records, nil
}

func validateJSONWithoutDuplicateKeys(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate object key %q", key)
			}
			seen[key] = struct{}{}
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("unterminated JSON object")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("unterminated JSON array")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func validateReleaseRecords(records []releaseRecord) error {
	seenVersions := make(map[string]struct{}, len(records))
	for _, record := range records {
		if record.Stable == nil || !*record.Stable {
			return fmt.Errorf("Go release %q is not explicitly stable", record.Version)
		}
		if _, err := parseReleaseVersion(record.Version); err != nil {
			return err
		}
		if _, exists := seenVersions[record.Version]; exists {
			return fmt.Errorf("official Go release index repeats version %q", record.Version)
		}
		seenVersions[record.Version] = struct{}{}
		if record.Files == nil || len(record.Files) == 0 || len(record.Files) > maxFilesPerRelease {
			return fmt.Errorf("Go release %q has an invalid file list", record.Version)
		}
		seenFiles := make(map[string]struct{}, len(record.Files))
		for _, file := range record.Files {
			if file.Filename == nil || file.OS == nil || file.Arch == nil || file.Version == nil || file.SHA256 == nil || file.Size == nil || file.Kind == nil {
				return fmt.Errorf("Go release %q has a file with missing metadata", record.Version)
			}
			if !filenamePattern.MatchString(*file.Filename) || strings.ContainsAny(*file.Filename, `/\\`) {
				return fmt.Errorf("Go release %q has an invalid filename", record.Version)
			}
			if !platformPattern.MatchString(*file.OS) || !platformPattern.MatchString(*file.Arch) {
				return fmt.Errorf("Go release %q has invalid platform metadata", record.Version)
			}
			if *file.Version != record.Version || !sha256Pattern.MatchString(*file.SHA256) || *file.Size <= 0 {
				return fmt.Errorf("Go release %q has inconsistent file metadata", record.Version)
			}
			switch *file.Kind {
			case "source", "archive", "installer":
			default:
				return fmt.Errorf("Go release %q has an unknown file kind", record.Version)
			}
			if _, exists := seenFiles[*file.Filename]; exists {
				return fmt.Errorf("Go release %q repeats file %q", record.Version, *file.Filename)
			}
			seenFiles[*file.Filename] = struct{}{}
		}
	}
	return nil
}

func parseReleaseVersion(raw string) (goVersion, error) {
	version, err := parseGoVersion(strings.TrimPrefix(raw, "go"))
	if !strings.HasPrefix(raw, "go") || err != nil {
		return goVersion{}, fmt.Errorf("official Go release index contains invalid stable version %q", raw)
	}
	return version, nil
}

func parseGoVersion(raw string) (goVersion, error) {
	parts := goVersionPattern.FindStringSubmatch(raw)
	if parts == nil {
		return goVersion{}, fmt.Errorf("Go version %q must be numeric major.minor.patch", raw)
	}
	major, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return goVersion{}, fmt.Errorf("Go version %q has an invalid major value", raw)
	}
	minor, err := strconv.ParseInt(parts[2], 10, 32)
	if err != nil {
		return goVersion{}, fmt.Errorf("Go version %q has an invalid minor value", raw)
	}
	patch, err := strconv.ParseInt(parts[3], 10, 32)
	if err != nil {
		return goVersion{}, fmt.Errorf("Go version %q has an invalid patch value", raw)
	}
	return goVersion{major: major, minor: minor, patch: patch}, nil
}

func (version goVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", version.major, version.minor, version.patch)
}

func compareVersions(left, right goVersion) int {
	if left.major != right.major {
		if left.major < right.major {
			return -1
		}
		return 1
	}
	if left.minor != right.minor {
		if left.minor < right.minor {
			return -1
		}
		return 1
	}
	if left.patch < right.patch {
		return -1
	}
	if left.patch > right.patch {
		return 1
	}
	return 0
}

func readGoMod(path string) (string, []byte, error) {
	version, content, _, err := readGoModFile(path)
	return version, content, err
}

func readGoModFile(path string) (string, []byte, os.FileInfo, error) {
	content, info, err := readRegularFile(path, maxGoModBytes)
	if err != nil {
		return "", nil, nil, fmt.Errorf("go.mod cannot be read safely: %w", err)
	}
	if !utf8.Valid(content) {
		return "", nil, nil, errors.New("go.mod is oversized or invalid UTF-8")
	}
	version, err := findGoDirective(content)
	if err != nil {
		return "", nil, nil, err
	}
	return version, content, info, nil
}

func readRegularFile(path string, maxBytes int64) ([]byte, os.FileInfo, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()

	name := filepath.Base(path)
	linkInfo, err := root.Lstat(name)
	if err != nil {
		return nil, nil, err
	}
	if !linkInfo.Mode().IsRegular() {
		return nil, nil, errors.New("file is not regular")
	}
	if linkInfo.Size() < 0 || linkInfo.Size() > maxBytes {
		return nil, nil, fmt.Errorf("file exceeds %d bytes", maxBytes)
	}

	file, err := root.Open(name)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(linkInfo, info) {
		return nil, nil, errors.New("file changed during open")
	}
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if int64(len(content)) > maxBytes {
		return nil, nil, fmt.Errorf("file exceeds %d bytes", maxBytes)
	}
	return content, info, nil
}

func findGoDirective(content []byte) (string, error) {
	var version string
	directiveCount := 0
	for _, line := range bytes.Split(content, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		fields := strings.Fields(string(trimmed))
		if len(fields) == 0 || fields[0] != "go" {
			continue
		}
		directiveCount++
		match := goDirectiveLine.FindSubmatch(bytes.TrimSuffix(line, []byte("\n")))
		if match == nil {
			return "", errors.New("go.mod must contain one plain go major.minor.patch directive")
		}
		version = string(match[2])
		if _, err := parseGoVersion(version); err != nil {
			return "", err
		}
	}
	if directiveCount != 1 {
		return "", errors.New("go.mod must contain exactly one go directive")
	}
	return version, nil
}

func rewriteGoDirective(content []byte, version string) ([]byte, error) {
	if _, err := parseGoVersion(version); err != nil {
		return nil, err
	}
	lines := bytes.SplitAfter(content, []byte("\n"))
	var output bytes.Buffer
	directiveCount := 0
	for _, completeLine := range lines {
		line := bytes.TrimSuffix(completeLine, []byte("\n"))
		fields := strings.Fields(string(bytes.TrimSpace(line)))
		if len(fields) == 0 || fields[0] != "go" {
			output.Write(completeLine)
			continue
		}
		directiveCount++
		match := goDirectiveLine.FindSubmatch(line)
		if match == nil {
			return nil, errors.New("go.mod must contain one plain go major.minor.patch directive")
		}
		output.WriteString("go")
		output.Write(match[1])
		output.WriteString(version)
		output.Write(match[3])
		output.Write(match[4])
		if bytes.HasSuffix(completeLine, []byte("\n")) {
			output.WriteByte('\n')
		}
	}
	if directiveCount != 1 {
		return nil, errors.New("go.mod must contain exactly one go directive")
	}
	return output.Bytes(), nil
}

func writeUpdatedFile(path string, original, updated []byte, info os.FileInfo) error {
	if bytes.Equal(original, updated) {
		return errors.New("go.mod update did not change the go directive")
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".gopatch-*.tmp")
	if err != nil {
		return fmt.Errorf("go.mod update file could not be created: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("go.mod update permissions could not be set: %w", err)
	}
	if _, err := temporary.Write(updated); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("go.mod update could not be written: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("go.mod update could not be synced: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("go.mod update could not be closed: %w", err)
	}

	currentBytes, _, err := readRegularFile(path, maxGoModBytes)
	if err != nil || !bytes.Equal(currentBytes, original) {
		return errors.New("go.mod changed during update; refusing to overwrite it")
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("go.mod update could not be installed: %w", err)
	}
	return nil
}
