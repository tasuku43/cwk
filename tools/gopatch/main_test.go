package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLatestStablePatchSelection(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		versions  []string
		candidate string
		updated   bool
	}{
		{name: "equal and older releases are ignored", current: "1.26.9", versions: []string{"1.26.9", "1.26.8"}},
		{name: "numerically newer patch is selected", current: "1.26.9", versions: []string{"1.26.10", "1.26.11", "1.26.8"}, candidate: "1.26.11", updated: true},
		{name: "different minors are ignored", current: "1.26.9", versions: []string{"1.27.3", "1.25.99"}},
		{name: "patch ordering is numeric", current: "1.26.9", versions: []string{"1.26.10", "1.26.9"}, candidate: "1.26.10", updated: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			goModPath := filepath.Join(root, "go.mod")
			original := []byte("module example.com/synthetic\n\ngo 1.26.9\n\nrequire example.com/dependency v1.2.3\n")
			if !bytes.Contains(original, []byte("go "+test.current)) {
				original = bytes.Replace(original, []byte("go 1.26.9"), []byte("go "+test.current), 1)
			}
			if err := os.WriteFile(goModPath, original, 0o644); err != nil {
				t.Fatal(err)
			}
			markerPath := filepath.Join(root, "unrelated.txt")
			marker := []byte("unchanged\n")
			if err := os.WriteFile(markerPath, marker, 0o600); err != nil {
				t.Fatal(err)
			}

			body := releaseIndexJSON(t, test.versions...)
			server := newJSONServer(t, http.StatusOK, "application/json; charset=utf-8", body)
			client := &http.Client{Timeout: time.Second}
			candidate, updated, err := updateFromEndpoint(context.Background(), goModPath, client, server.URL)
			if err != nil {
				t.Fatalf("updateFromEndpoint returned an error: %v", err)
			}
			if candidate != test.candidate || updated != test.updated {
				t.Fatalf("update result = (%q, %v), want (%q, %v)", candidate, updated, test.candidate, test.updated)
			}

			wantGoMod := original
			if test.updated {
				wantGoMod = bytes.Replace(original, []byte("go "+test.current), []byte("go "+test.candidate), 1)
			}
			gotGoMod, err := os.ReadFile(goModPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotGoMod, wantGoMod) {
				t.Fatalf("go.mod changed unexpectedly\n got: %q\nwant: %q", gotGoMod, wantGoMod)
			}
			gotMarker, err := os.ReadFile(markerPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotMarker, marker) {
				t.Fatalf("unrelated file changed: %q", gotMarker)
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 2 {
				t.Fatalf("persistent files = %d, want only go.mod and unrelated.txt", len(entries))
			}
		})
	}
}

func TestUpdateRejectsInvalidSourceWithoutChangingFiles(t *testing.T) {
	valid := releaseIndexJSON(t, "1.26.10")
	invalidCases := []struct {
		name        string
		status      int
		contentType string
		body        []byte
	}{
		{name: "malformed json", status: http.StatusOK, contentType: "application/json", body: []byte("[broken")},
		{name: "duplicate keys", status: http.StatusOK, contentType: "application/json", body: bytes.Replace(valid, []byte(`"stable":true`), []byte(`"stable":true,"stable":true`), 1)},
		{name: "unknown field", status: http.StatusOK, contentType: "application/json", body: bytes.Replace(valid, []byte(`"stable":true`), []byte(`"stable":true,"unexpected":true`), 1)},
		{name: "trailing value", status: http.StatusOK, contentType: "application/json", body: append(append([]byte(nil), valid...), []byte(` {}`)...)},
		{name: "not stable", status: http.StatusOK, contentType: "application/json", body: []byte(strings.Replace(string(valid), `"stable":true`, `"stable":false`, 1))},
		{name: "wrong content type", status: http.StatusOK, contentType: "text/plain", body: valid},
		{name: "server failure", status: http.StatusServiceUnavailable, contentType: "application/json", body: valid},
	}

	for _, test := range invalidCases {
		t.Run(test.name, func(t *testing.T) {
			root, goModPath, markerPath, original, marker := syntheticRepo(t)
			server := newJSONServer(t, test.status, test.contentType, test.body)
			_, updated, err := updateFromEndpoint(context.Background(), goModPath, &http.Client{Timeout: time.Second}, server.URL)
			if err == nil || updated {
				t.Fatalf("updateFromEndpoint = (updated %v, error %v), want error and no update", updated, err)
			}
			assertSyntheticRepoUnchanged(t, root, goModPath, markerPath, original, marker)
		})
	}
}

func TestUpdateRejectsOversizedResponse(t *testing.T) {
	root, goModPath, markerPath, original, marker := syntheticRepo(t)
	body := append([]byte("["), bytes.Repeat([]byte(" "), maxReleaseIndexBytes)...)
	server := newJSONServer(t, http.StatusOK, "application/json", body)
	_, updated, err := updateFromEndpoint(context.Background(), goModPath, &http.Client{Timeout: 3 * time.Second}, server.URL)
	if err == nil || updated {
		t.Fatalf("updateFromEndpoint = (updated %v, error %v), want oversized-response error", updated, err)
	}
	assertSyntheticRepoUnchanged(t, root, goModPath, markerPath, original, marker)
}

func TestUpdateHonorsContextDeadline(t *testing.T) {
	root, goModPath, markerPath, original, marker := syntheticRepo(t)
	body := releaseIndexJSON(t, "1.26.10")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, updated, err := updateFromEndpoint(ctx, goModPath, &http.Client{Timeout: time.Second}, server.URL)
	if err == nil || updated {
		t.Fatalf("updateFromEndpoint = (updated %v, error %v), want timeout error", updated, err)
	}
	assertSyntheticRepoUnchanged(t, root, goModPath, markerPath, original, marker)
}

func TestUpdateNetworkFailureLeavesRepositoryUnchanged(t *testing.T) {
	root, goModPath, markerPath, original, marker := syntheticRepo(t)
	server := httptest.NewServer(http.NotFoundHandler())
	serverURL := server.URL
	server.Close()
	_, updated, err := updateFromEndpoint(context.Background(), goModPath, &http.Client{Timeout: time.Second}, serverURL)
	if err == nil || updated {
		t.Fatalf("updateFromEndpoint = (updated %v, error %v), want network error", updated, err)
	}
	assertSyntheticRepoUnchanged(t, root, goModPath, markerPath, original, marker)
}

func TestUpdateRejectsCrossOriginRedirect(t *testing.T) {
	root, goModPath, markerPath, original, marker := syntheticRepo(t)
	body := releaseIndexJSON(t, "1.26.10")
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(target.Close)
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(redirect.Close)
	_, updated, err := updateFromEndpoint(context.Background(), goModPath, newOfficialHTTPClient(), redirect.URL)
	if err == nil || updated {
		t.Fatalf("updateFromEndpoint = (updated %v, error %v), want redirect error", updated, err)
	}
	assertSyntheticRepoUnchanged(t, root, goModPath, markerPath, original, marker)
}

func TestApplyCandidateOnlyChangesGoDirective(t *testing.T) {
	root, goModPath, markerPath, original, marker := syntheticRepo(t)
	if err := applyCandidate(goModPath, "1.26.10"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Replace(original, []byte("go 1.26.9"), []byte("go 1.26.10"), 1)
	if !bytes.Equal(got, want) {
		t.Fatalf("go.mod update = %q, want %q", got, want)
	}
	gotMarker, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotMarker, marker) {
		t.Fatalf("unrelated file changed: %q", gotMarker)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("persistent files = %d, want 2", len(entries))
	}
}

func TestApplyCandidateRejectsMinorChangesAndNonNewerVersions(t *testing.T) {
	tests := []string{"1.27.1", "1.25.99", "1.26.9", "1.26.8", "1.026.10", "1.26.10rc1", "1.26.10.1"}
	for _, candidate := range tests {
		t.Run(candidate, func(t *testing.T) {
			root, goModPath, markerPath, original, marker := syntheticRepo(t)
			if err := applyCandidate(goModPath, candidate); err == nil {
				t.Fatalf("applyCandidate(%q) unexpectedly succeeded", candidate)
			}
			assertSyntheticRepoUnchanged(t, root, goModPath, markerPath, original, marker)
		})
	}
}

func TestReadGoModRejectsInvalidOrIndirectFiles(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "missing directive", content: "module example.com/synthetic\n"},
		{name: "duplicate directive", content: "module example.com/synthetic\ngo 1.26.9\ngo 1.26.10\n"},
		{name: "directive comment", content: "module example.com/synthetic\ngo 1.26.9 // comment\n"},
		{name: "two-component version", content: "module example.com/synthetic\ngo 1.26\n"},
		{name: "leading zero", content: "module example.com/synthetic\ngo 1.026.9\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "go.mod")
			if err := os.WriteFile(path, []byte(test.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, _, err := readGoMod(path); err == nil {
				t.Fatal("readGoMod accepted invalid go.mod")
			}
		})
	}

	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "target")
		link := filepath.Join(root, "go.mod")
		if err := os.WriteFile(target, []byte("module example.com/synthetic\ngo 1.26.9\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, _, err := readGoMod(link); err == nil {
			t.Fatal("readGoMod accepted a symlink")
		}
	})
}

func TestRunHasNoSourceOverride(t *testing.T) {
	if err := run([]string{"--url", "https://example.com"}, &bytes.Buffer{}); err == nil {
		t.Fatal("run accepted a source override")
	}
}

func TestOfficialReleaseRecordSchema(t *testing.T) {
	body := releaseIndexJSON(t, "1.26.10")
	records, err := decodeReleaseRecords(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Version != "go1.26.10" {
		t.Fatalf("decoded records = %#v", records)
	}
}

func syntheticRepo(t *testing.T) (root, goModPath, markerPath string, goMod, marker []byte) {
	t.Helper()
	root = t.TempDir()
	goModPath = filepath.Join(root, "go.mod")
	goMod = []byte("module example.com/synthetic\n\ngo 1.26.9\n\nrequire example.com/dependency v1.2.3\n")
	if err := os.WriteFile(goModPath, goMod, 0o644); err != nil {
		t.Fatal(err)
	}
	markerPath = filepath.Join(root, "unrelated.txt")
	marker = []byte("unchanged\n")
	if err := os.WriteFile(markerPath, marker, 0o600); err != nil {
		t.Fatal(err)
	}
	return root, goModPath, markerPath, goMod, marker
}

func assertSyntheticRepoUnchanged(t *testing.T, root, goModPath, markerPath string, goMod, marker []byte) {
	t.Helper()
	gotGoMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotGoMod, goMod) {
		t.Fatalf("go.mod changed after rejected update: %q", gotGoMod)
	}
	gotMarker, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotMarker, marker) {
		t.Fatalf("unrelated file changed: %q", gotMarker)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("persistent files after rejected update = %d, want 2", len(entries))
	}
}

func newJSONServer(t *testing.T, status int, contentType string, body []byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server
}

func releaseIndexJSON(t *testing.T, versions ...string) []byte {
	t.Helper()
	records := make([]releaseRecord, 0, len(versions))
	for _, version := range versions {
		fullVersion := "go" + version
		stable := true
		filename := fullVersion + ".src.tar.gz"
		osName := ""
		arch := ""
		fileVersion := fullVersion
		checksum := strings.Repeat("a", 64)
		size := int64(10)
		kind := "source"
		records = append(records, releaseRecord{
			Version: fullVersion,
			Stable:  &stable,
			Files: []releaseFile{{
				Filename: &filename,
				OS:       &osName,
				Arch:     &arch,
				Version:  &fileVersion,
				SHA256:   &checksum,
				Size:     &size,
				Kind:     &kind,
			}},
		})
	}
	body, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal synthetic Go release index: %v", err)
	}
	return body
}

func TestFetchReleaseIndexRejectsTopLevelArrayViolation(t *testing.T) {
	server := newJSONServer(t, http.StatusOK, "application/json", []byte(`{"version":"go1.26.10"}`))
	if _, err := fetchReleaseRecords(context.Background(), &http.Client{Timeout: time.Second}, server.URL); err == nil {
		t.Fatal("fetchReleaseRecords accepted a non-array response")
	}
}

func ExamplecompareVersions() {
	older, _ := parseGoVersion("1.26.9")
	newer, _ := parseGoVersion("1.26.10")
	fmt.Println(compareVersions(newer, older) > 0)
	// Output: true
}
