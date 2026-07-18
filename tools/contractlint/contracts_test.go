package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/cli"
)

func TestRepositoryContractsMatchDefaultCatalog(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	catalog := cli.DefaultCatalog()
	if err := catalog.Validate(); err != nil {
		t.Fatalf("DefaultCatalog.Validate() error = %v", err)
	}
	issues, err := inspectContracts(root, catalogCapabilityIDs(catalog))
	if err != nil {
		t.Fatalf("inspectContracts() error = %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("contract issues = %+v", issues)
	}
}

func TestInspectContractsAcceptsPublicCapabilitiesAndSchemaFixture(t *testing.T) {
	root := t.TempDir()
	writeJSON(t, root, capabilitiesPath, []capability{
		{ID: "items.read", Status: "public"},
		{ID: "items.write", Status: "deferred", Reason: "The mutation policy is not selected."},
		{ID: "chatwork.items.read", Status: "deferred", Reason: "Synthetic upstream mapping for this contract fixture."},
	})
	writeJSON(t, root, chatworkAPIPath, validChatworkSnapshot("chatwork.items.read"))
	fixturePath := "internal/infra/example/testdata/schema.json"
	fixture := []byte("{\"version\":1}\n")
	writeFixture(t, root, fixturePath, fixture)
	writeJSON(t, root, schemasPath, []schemaFixture{{
		ID:         "example.v1",
		Path:       fixturePath,
		SHA256:     digest(fixture),
		Provenance: "A publishable synthetic provider fixture.",
		License:    "CC0-1.0",
	}})

	issues, err := inspectContracts(root, set("items.read"))
	if err != nil {
		t.Fatalf("inspectContracts() error = %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("contract issues = %+v", issues)
	}
}

func TestChatworkSnapshotRejectsCoverageAndShapeErrors(t *testing.T) {
	snapshot := validChatworkSnapshot("items.read")
	snapshot.SnapshotDate = "latest"
	snapshot.Source = "https://example.com"
	snapshot.BaseURL = "http://api.chatwork.com/v2"
	snapshot.Operations = snapshot.Operations[:2]
	snapshot.Operations[0].ID = "Bad ID"
	snapshot.Operations[0].Method = "PATCH"
	snapshot.Operations[0].Path = "https://api.chatwork.com/v2/me"
	snapshot.Operations[0].CapabilityIDs = []string{"items.read", "items.read", "chatwork.unknown"}
	snapshot.Operations[1] = snapshot.Operations[0]
	issues := validateChatworkSnapshot(snapshot, []capability{
		{ID: "items.read", Status: "public"},
		{ID: "chatwork.unmapped", Status: "deferred", Reason: "Later."},
	})
	assertIssuesContain(t, issues,
		"snapshot_date must remain",
		"source must be the official",
		"base_url must be the fixed",
		"exactly the reviewed 32-operation",
		"lowercase upstream operation syntax",
		"method must be",
		"relative resource-template syntax",
		"duplicate operation id",
		"duplicate operation route",
		"is not in the reviewed official",
		"reviewed operation \"get-me\" is missing",
		"duplicate capability id",
		"absent from .harness/capabilities.json",
		"must use the chatwork namespace",
		"has no upstream operation mapping",
	)
}

func TestChatworkSnapshotPinsEveryOfficialOperationIdentity(t *testing.T) {
	snapshot := validChatworkSnapshot("chatwork.items.read")
	snapshot.Operations[0].Method = "POST"
	snapshot.Operations[1].Path = "/my/changed"
	snapshot.Operations[2].ID = "get-invented-operation"
	issues := validateChatworkSnapshot(snapshot, []capability{{
		ID: "chatwork.items.read", Status: "deferred", Reason: "Synthetic contract fixture.",
	}})
	assertIssuesContain(t, issues,
		"operation \"get-me\" method must remain GET",
		"operation \"get-my-status\" path must remain /my/status",
		"operation id \"get-invented-operation\" is not in the reviewed official",
		"reviewed operation \"get-my-tasks\" is missing",
	)
}

func TestChatworkSnapshotCompleteCoverageRequiresPublicOwnerForEveryOperation(t *testing.T) {
	snapshot := validChatworkSnapshot("chatwork.items.read")
	snapshot.CoverageStatus = "complete"
	issues := validateChatworkSnapshot(snapshot, []capability{{
		ID: "chatwork.items.read", Status: "deferred", Reason: "Synthetic contract fixture.",
	}})
	assertIssuesContain(t, issues, "complete coverage requires operation \"get-me\" to map to at least one public capability")

	issues = validateChatworkSnapshot(snapshot, []capability{{ID: "chatwork.items.read", Status: "public"}})
	if len(issues) != 0 {
		t.Fatalf("public complete coverage issues = %+v", issues)
	}
}

func TestChatworkSnapshotPinsLimitsAndMutationPolicy(t *testing.T) {
	snapshot := validChatworkSnapshot("chatwork.items.read")
	snapshot.Limits.MetadataReadTimeoutSeconds = 21
	snapshot.Limits.UploadBytes = 1
	snapshot.CoverageStatus = "latest"
	snapshot.Documented100ItemOperationIDs = append(snapshot.Documented100ItemOperationIDs, "get-rooms")
	snapshot.MutationPolicy.DefaultConfirmation = "always-confirm"
	snapshot.MutationPolicy.AccessChangeOperationIDs = snapshot.MutationPolicy.AccessChangeOperationIDs[1:]
	snapshot.MutationPolicy.DestructiveConfirmation = "none"
	snapshot.MutationPolicy.UncertainOutcome = "retry"
	issues := validateChatworkSnapshot(snapshot, []capability{{
		ID: "chatwork.items.read", Status: "deferred", Reason: "Synthetic contract fixture.",
	}})
	assertIssuesContain(t, issues,
		"coverage_status must be planned during implementation or complete at goal closure",
		"metadata_read_timeout_seconds must remain 20",
		"upload_bytes must remain 5242880",
		"operation id \"get-rooms\" is not in the reviewed documented_100_item_operation_ids set",
		"default_confirmation must remain exact-invocation",
		"reviewed operation id \"post-rooms\" is missing",
		"destructive_confirmation must remain --confirm=destructive",
		"uncertain_outcome must remain read-only-reconciliation",
	)
}

func TestStrictChatworkSnapshotRejectsUnknownAndDuplicateFields(t *testing.T) {
	for name, data := range map[string]string{
		"unknown":   `{"snapshot_date":"2026-07-18","source":"x","base_url":"x","operations":[],"latest":true}`,
		"duplicate": `{"snapshot_date":"2026-07-18","snapshot_date":"2026-07-19","source":"x","base_url":"x","operations":[]}`,
		"trailing":  `{"snapshot_date":"2026-07-18","source":"x","base_url":"x","operations":[]} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, chatworkAPIPath, []byte(data))
			if _, err := loadStrictObject[chatworkAPISnapshot](root, chatworkAPIPath); err == nil {
				t.Fatal("strict snapshot parsing succeeded")
			}
		})
	}
}

func TestCapabilitiesRejectCoverageAndLedgerErrors(t *testing.T) {
	entries := []capability{
		{ID: "items.read", Status: "public"},
		{ID: "Items.Write", Status: "public"},
		{ID: "items.read", Status: "public"},
		{ID: "items.internal", Status: "internal"},
		{ID: "items.deferred", Status: "deferred", Reason: "Later milestone."},
		{ID: "items.excluded", Status: "excluded", Reason: "Outside the product thesis."},
		{ID: "items.unknown", Status: "planned", Reason: "Not a supported status."},
	}
	issues := validateCapabilities(entries, set("items.read", "items.internal", "items.missing"))
	assertIssuesContain(t, issues,
		"lowercase dot syntax",
		"duplicate capability id",
		"requires a non-empty reason",
		"non-public capability \"items.internal\" is exposed",
		"status must be public, internal, deferred, or excluded",
		"catalog capability \"items.missing\" is not declared public",
	)
}

func TestCapabilitiesRejectPublicEntryWithoutCatalogExposure(t *testing.T) {
	issues := validateCapabilities([]capability{{ID: "items.read", Status: "public"}}, map[string]struct{}{})
	assertIssuesContain(t, issues, "has no command catalog entry")
}

func TestStrictManifestParsingRejectsUnknownDuplicateAndTrailingJSON(t *testing.T) {
	tests := []struct {
		name string
		path string
		data string
	}{
		{name: "capability unknown field", path: capabilitiesPath, data: `[{"id":"items.read","status":"public","reason":"","command":"items read"}]`},
		{name: "capability duplicate key", path: capabilitiesPath, data: `[{"id":"items.read","id":"items.write","status":"public","reason":""}]`},
		{name: "schema unknown field", path: schemasPath, data: `[{"id":"provider.v1","path":"testdata/schema.json","sha256":"","provenance":"fixture","license":"CC0-1.0","url":"https://example.com"}]`},
		{name: "schema duplicate key", path: schemasPath, data: `[{"id":"provider.v1","path":"testdata/a.json","path":"testdata/b.json","sha256":"","provenance":"fixture","license":"CC0-1.0"}]`},
		{name: "null is not explicit empty", path: schemasPath, data: `null`},
		{name: "object is not array", path: schemasPath, data: `{}`},
		{name: "multiple values", path: schemasPath, data: `[] []`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, test.path, []byte(test.data))
			var err error
			if test.path == capabilitiesPath {
				_, err = loadStrictArray[capability](root, test.path)
			} else {
				_, err = loadStrictArray[schemaFixture](root, test.path)
			}
			if err == nil {
				t.Fatal("strict manifest parsing succeeded")
			}
		})
	}
}

func TestManifestsMustBeRegularFilesWithoutSymbolicLinks(t *testing.T) {
	t.Run("symbolic link", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, ".harness/target.json", []byte("[]\n"))
		link := filepath.Join(root, filepath.FromSlash(capabilitiesPath))
		if err := os.Symlink("target.json", link); err != nil {
			t.Skipf("symbolic links are unavailable: %v", err)
		}
		if _, err := loadStrictArray[capability](root, capabilitiesPath); err == nil || !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("error = %v, want symbolic-link rejection", err)
		}
	})

	t.Run("symbolic link parent", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "real-harness/capabilities.json", []byte("[]\n"))
		if err := os.Symlink("real-harness", filepath.Join(root, ".harness")); err != nil {
			t.Skipf("symbolic links are unavailable: %v", err)
		}
		if _, err := loadStrictArray[capability](root, capabilitiesPath); err == nil || !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("error = %v, want symbolic-link rejection", err)
		}
	})

	t.Run("directory", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, filepath.FromSlash(schemasPath))
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := loadStrictArray[schemaFixture](root, schemasPath); err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("error = %v, want regular-file rejection", err)
		}
	})
}

func TestSchemasAllowExplicitEmptyManifest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, schemasPath, []byte("[]\n"))
	entries, err := loadStrictArray[schemaFixture](root, schemasPath)
	if err != nil {
		t.Fatalf("loadStrictArray() error = %v", err)
	}
	if entries == nil || len(entries) != 0 {
		t.Fatalf("entries = %#v, want explicit empty slice", entries)
	}
	if issues := validateSchemas(root, entries); len(issues) != 0 {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestSchemasRejectInvalidMetadataAndDigest(t *testing.T) {
	root := t.TempDir()
	fixturePath := "testdata/provider/schema.json"
	fixture := []byte("{\"version\":1}\n")
	writeFixture(t, root, fixturePath, fixture)
	entries := []schemaFixture{
		{ID: "Provider.V1", Path: fixturePath, SHA256: digest(fixture), Provenance: "fixture", License: "CC0-1.0"},
		{ID: "provider.v1", Path: fixturePath, SHA256: strings.ToUpper(digest(fixture)), Provenance: "", License: ""},
		{ID: "provider.v1", Path: fixturePath, SHA256: strings.Repeat("0", 64), Provenance: "fixture", License: "CC0-1.0"},
	}
	issues := validateSchemas(root, entries)
	assertIssuesContain(t, issues,
		"lowercase dot syntax",
		"duplicate schema id",
		"duplicate schema path",
		"provenance is required",
		"license is required",
		"sha256 must be exactly 64 lowercase hexadecimal characters",
		"sha256 mismatch",
	)
}

func TestSchemaFixturePathsRejectTraversalHiddenNonFixtureAndNonRegularTargets(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "testdata/valid.json", []byte("{}\n"))
	if err := os.MkdirAll(filepath.Join(root, "testdata", "directory"), 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "traversal", path: "testdata/../outside.json", want: "canonical repository-relative"},
		{name: "absolute", path: filepath.Join(root, "testdata", "valid.json"), want: "canonical repository-relative"},
		{name: "hidden", path: "testdata/.schema.json", want: "hidden components"},
		{name: "not fixture", path: "schemas/provider.json", want: "below a testdata directory"},
		{name: "missing", path: "testdata/missing.json", want: "cannot be inspected"},
		{name: "directory", path: "testdata/directory", want: "not a regular file"},
		{name: "credential-like", path: "testdata/credentials.json", want: "credential-bearing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, issues := readSchemaFixture(root, test.path)
			if !messagesContain(issues, test.want) {
				t.Fatalf("issues = %q, want substring %q", issues, test.want)
			}
		})
	}
}

func TestSchemaFixtureRejectsSymbolicLinks(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "testdata/target.json", []byte("{}\n"))
	link := filepath.Join(root, "testdata", "schema.json")
	if err := os.Symlink("target.json", link); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	_, issues := readSchemaFixture(root, "testdata/schema.json")
	if !messagesContain(issues, "symbolic link") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestSchemaFixtureDigestIsByteExact(t *testing.T) {
	root := t.TempDir()
	one := []byte("{\"version\":1}\n")
	two := []byte("{\"version\":1}\r\n")
	writeFixture(t, root, "testdata/schema.json", two)
	issues := validateSchemas(root, []schemaFixture{{
		ID:         "provider.v1",
		Path:       "testdata/schema.json",
		SHA256:     digest(one),
		Provenance: "Synthetic fixture.",
		License:    "CC0-1.0",
	}})
	assertIssuesContain(t, issues, "sha256 mismatch")
}

func writeJSON(t *testing.T, root, relative string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, relative, append(data, '\n'))
}

func validChatworkSnapshot(capabilityID string) chatworkAPISnapshot {
	operations := make([]chatworkOperation, len(reviewedChatworkOperations))
	for index, reviewed := range reviewedChatworkOperations {
		operations[index] = chatworkOperation{
			ID:            reviewed.ID,
			Method:        reviewed.Method,
			Path:          reviewed.Path,
			CapabilityIDs: []string{capabilityID},
		}
	}
	return chatworkAPISnapshot{
		SnapshotDate:   "2026-07-18",
		Source:         "https://developer.chatwork.com/llms.txt",
		BaseURL:        "https://api.chatwork.com/v2",
		CoverageStatus: "planned",
		Limits: chatworkLimits{
			MetadataReadTimeoutSeconds:  20,
			UploadTimeoutSeconds:        60,
			MaxAttempts:                 1,
			SuccessResponseBytes:        8 * 1024 * 1024,
			ProviderErrorResponseBytes:  64 * 1024,
			OutputBytes:                 16 * 1024 * 1024,
			ListItems:                   10_000,
			DocumentedEndpointListItems: 100,
			UploadBytes:                 5 * 1024 * 1024,
		},
		Documented100ItemOperationIDs: append([]string(nil), documented100ItemOperationIDs...),
		MutationPolicy: chatworkMutationPolicy{
			DefaultConfirmation:      "exact-invocation",
			AccessChangeConfirmation: "--confirm=access-change",
			AccessChangeOperationIDs: append([]string(nil), accessChangeOperationIDs...),
			DestructiveConfirmation:  "--confirm=destructive",
			DestructiveOperationIDs:  append([]string(nil), destructiveOperationIDs...),
			UncertainOutcome:         "read-only-reconciliation",
		},
		Operations: operations,
	}
}

func writeFixture(t *testing.T, root, relative string, data []byte) {
	t.Helper()
	writeFile(t, root, relative, data)
}

func writeFile(t *testing.T, root, relative string, data []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func set(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func assertIssuesContain(t *testing.T, issues []issue, wanted ...string) {
	t.Helper()
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		messages = append(messages, issue.Message)
	}
	for _, substring := range wanted {
		if !messagesContain(messages, substring) {
			t.Errorf("issues = %q, want substring %q", messages, substring)
		}
	}
}

func messagesContain(messages []string, wanted string) bool {
	for _, message := range messages {
		if strings.Contains(message, wanted) {
			return true
		}
	}
	return false
}
