package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	capabilitiesPath = ".harness/capabilities.json"
	chatworkAPIPath  = ".harness/chatwork_api_v2.json"
	schemasPath      = ".harness/schemas.json"
)

var contractID = regexp.MustCompile(`^[a-z][a-z0-9-]*\.[a-z][a-z0-9-]*(?:\.[a-z][a-z0-9-]*)*$`)
var operationID = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
var operationPath = regexp.MustCompile(`^/[a-z_]+(?:/(?:[a-z_]+|\{[a-z_]+\}))*$`)

type issue struct {
	Path    string
	Message string
}

type capability struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type schemaFixture struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	Provenance string `json:"provenance"`
	License    string `json:"license"`
}

type chatworkAPISnapshot struct {
	SnapshotDate                  string                 `json:"snapshot_date"`
	Source                        string                 `json:"source"`
	BaseURL                       string                 `json:"base_url"`
	CoverageStatus                string                 `json:"coverage_status"`
	Limits                        chatworkLimits         `json:"limits"`
	Documented100ItemOperationIDs []string               `json:"documented_100_item_operation_ids"`
	MutationPolicy                chatworkMutationPolicy `json:"mutation_policy"`
	Operations                    []chatworkOperation    `json:"operations"`
}

type chatworkLimits struct {
	MetadataReadTimeoutSeconds  int `json:"metadata_read_timeout_seconds"`
	UploadTimeoutSeconds        int `json:"upload_timeout_seconds"`
	MaxAttempts                 int `json:"max_attempts"`
	SuccessResponseBytes        int `json:"success_response_bytes"`
	ProviderErrorResponseBytes  int `json:"provider_error_response_bytes"`
	OutputBytes                 int `json:"output_bytes"`
	ListItems                   int `json:"list_items"`
	DocumentedEndpointListItems int `json:"documented_endpoint_list_items"`
	UploadBytes                 int `json:"upload_bytes"`
}

type chatworkMutationPolicy struct {
	DefaultConfirmation      string   `json:"default_confirmation"`
	AccessChangeConfirmation string   `json:"access_change_confirmation"`
	AccessChangeOperationIDs []string `json:"access_change_operation_ids"`
	DestructiveConfirmation  string   `json:"destructive_confirmation"`
	DestructiveOperationIDs  []string `json:"destructive_operation_ids"`
	UncertainOutcome         string   `json:"uncertain_outcome"`
}

type chatworkOperation struct {
	ID            string   `json:"id"`
	Method        string   `json:"method"`
	Path          string   `json:"path"`
	CapabilityIDs []string `json:"capability_ids"`
}

type chatworkOperationIdentity struct {
	ID     string
	Method string
	Path   string
}

var reviewedChatworkOperations = []chatworkOperationIdentity{
	{ID: "get-me", Method: "GET", Path: "/me"},
	{ID: "get-my-status", Method: "GET", Path: "/my/status"},
	{ID: "get-my-tasks", Method: "GET", Path: "/my/tasks"},
	{ID: "get-contacts", Method: "GET", Path: "/contacts"},
	{ID: "get-rooms", Method: "GET", Path: "/rooms"},
	{ID: "post-rooms", Method: "POST", Path: "/rooms"},
	{ID: "get-rooms-room_id", Method: "GET", Path: "/rooms/{room_id}"},
	{ID: "put-rooms-room_id", Method: "PUT", Path: "/rooms/{room_id}"},
	{ID: "delete-rooms-room_id", Method: "DELETE", Path: "/rooms/{room_id}"},
	{ID: "get-rooms-room_id-members", Method: "GET", Path: "/rooms/{room_id}/members"},
	{ID: "put-rooms-room_id-members", Method: "PUT", Path: "/rooms/{room_id}/members"},
	{ID: "get-rooms-room_id-messages", Method: "GET", Path: "/rooms/{room_id}/messages"},
	{ID: "post-rooms-room_id-messages", Method: "POST", Path: "/rooms/{room_id}/messages"},
	{ID: "put-rooms-room_id-messages-read", Method: "PUT", Path: "/rooms/{room_id}/messages/read"},
	{ID: "put-rooms-room_id-messages-unread", Method: "PUT", Path: "/rooms/{room_id}/messages/unread"},
	{ID: "get-rooms-room_id-messages-message_id", Method: "GET", Path: "/rooms/{room_id}/messages/{message_id}"},
	{ID: "put-rooms-room_id-messages-message_id", Method: "PUT", Path: "/rooms/{room_id}/messages/{message_id}"},
	{ID: "delete-rooms-room_id-messages-message_id", Method: "DELETE", Path: "/rooms/{room_id}/messages/{message_id}"},
	{ID: "get-rooms-room_id-tasks", Method: "GET", Path: "/rooms/{room_id}/tasks"},
	{ID: "post-rooms-room_id-tasks", Method: "POST", Path: "/rooms/{room_id}/tasks"},
	{ID: "get-rooms-room_id-tasks-task_id", Method: "GET", Path: "/rooms/{room_id}/tasks/{task_id}"},
	{ID: "put-rooms-room_id-tasks-task_id-status", Method: "PUT", Path: "/rooms/{room_id}/tasks/{task_id}/status"},
	{ID: "get-rooms-room_id-files", Method: "GET", Path: "/rooms/{room_id}/files"},
	{ID: "post-rooms-room_id-files", Method: "POST", Path: "/rooms/{room_id}/files"},
	{ID: "get-rooms-room_id-files-file_id", Method: "GET", Path: "/rooms/{room_id}/files/{file_id}"},
	{ID: "get-rooms-room_id-link", Method: "GET", Path: "/rooms/{room_id}/link"},
	{ID: "post-rooms-room_id-link", Method: "POST", Path: "/rooms/{room_id}/link"},
	{ID: "put-rooms-room_id-link", Method: "PUT", Path: "/rooms/{room_id}/link"},
	{ID: "delete-rooms-room_id-link", Method: "DELETE", Path: "/rooms/{room_id}/link"},
	{ID: "get-incoming_requests", Method: "GET", Path: "/incoming_requests"},
	{ID: "put-incoming_requests-request_id", Method: "PUT", Path: "/incoming_requests/{request_id}"},
	{ID: "delete-incoming_requests-request_id", Method: "DELETE", Path: "/incoming_requests/{request_id}"},
}

var documented100ItemOperationIDs = []string{
	"get-my-tasks",
	"get-rooms-room_id-messages",
	"get-rooms-room_id-tasks",
	"get-rooms-room_id-files",
	"get-incoming_requests",
}

var accessChangeOperationIDs = []string{
	"post-rooms",
	"put-rooms-room_id-members",
	"post-rooms-room_id-link",
	"put-rooms-room_id-link",
	"put-incoming_requests-request_id",
}

var destructiveOperationIDs = []string{
	"delete-rooms-room_id",
	"delete-rooms-room_id-messages-message_id",
	"delete-rooms-room_id-link",
	"delete-incoming_requests-request_id",
}

func inspectContracts(root string, catalogIDs map[string]struct{}) ([]issue, error) {
	capabilities, err := loadStrictArray[capability](root, capabilitiesPath)
	if err != nil {
		return nil, err
	}
	schemas, err := loadStrictArray[schemaFixture](root, schemasPath)
	if err != nil {
		return nil, err
	}
	snapshot, err := loadStrictObject[chatworkAPISnapshot](root, chatworkAPIPath)
	if err != nil {
		return nil, err
	}
	issues := validateCapabilities(capabilities, catalogIDs)
	issues = append(issues, validateChatworkSnapshot(snapshot, capabilities)...)
	issues = append(issues, validateSchemas(root, schemas)...)
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		return issues[i].Message < issues[j].Message
	})
	return issues, nil
}

func validateChatworkSnapshot(snapshot chatworkAPISnapshot, capabilities []capability) []issue {
	var issues []issue
	if snapshot.SnapshotDate != "2026-07-18" {
		issues = append(issues, issue{Path: chatworkAPIPath, Message: "snapshot_date must remain 2026-07-18 unless the governing coverage decision is reviewed"})
	}
	if snapshot.Source != "https://developer.chatwork.com/llms.txt" {
		issues = append(issues, issue{Path: chatworkAPIPath, Message: "source must be the official Chatwork documentation index"})
	}
	if snapshot.BaseURL != "https://api.chatwork.com/v2" {
		issues = append(issues, issue{Path: chatworkAPIPath, Message: "base_url must be the fixed Chatwork API v2 production destination"})
	}
	switch snapshot.CoverageStatus {
	case "planned", "complete":
	default:
		issues = append(issues, issue{Path: chatworkAPIPath, Message: "coverage_status must be planned during implementation or complete at goal closure"})
	}
	issues = append(issues, validateChatworkLimits(snapshot.Limits)...)
	issues = append(issues, validateExactOperationIDSet("documented_100_item_operation_ids", snapshot.Documented100ItemOperationIDs, documented100ItemOperationIDs)...)
	issues = append(issues, validateChatworkMutationPolicy(snapshot.MutationPolicy)...)
	if len(snapshot.Operations) != 32 {
		issues = append(issues, issue{Path: chatworkAPIPath, Message: fmt.Sprintf("operations must contain exactly the reviewed 32-operation snapshot, got %d", len(snapshot.Operations))})
	}
	knownCapabilities := make(map[string]struct{}, len(capabilities))
	publicCapabilities := make(map[string]struct{}, len(capabilities))
	chatworkCapabilities := make(map[string]struct{})
	for _, entry := range capabilities {
		knownCapabilities[entry.ID] = struct{}{}
		if entry.Status == "public" {
			publicCapabilities[entry.ID] = struct{}{}
		}
		if strings.HasPrefix(entry.ID, "chatwork.") {
			chatworkCapabilities[entry.ID] = struct{}{}
		}
	}
	reviewedByID := make(map[string]chatworkOperationIdentity, len(reviewedChatworkOperations))
	for _, operation := range reviewedChatworkOperations {
		reviewedByID[operation.ID] = operation
	}
	seenIDs := make(map[string]int, len(snapshot.Operations))
	seenRoutes := make(map[string]int, len(snapshot.Operations))
	mappedCapabilities := make(map[string]struct{})
	for index, operation := range snapshot.Operations {
		location := fmt.Sprintf("%s.operations[%d]", chatworkAPIPath, index)
		if len(operation.ID) > 160 || !operationID.MatchString(operation.ID) {
			issues = append(issues, issue{Path: location, Message: "id must use the bounded lowercase upstream operation syntax"})
		}
		if first, exists := seenIDs[operation.ID]; exists {
			issues = append(issues, issue{Path: location, Message: fmt.Sprintf("duplicate operation id %q; first declared at index %d", operation.ID, first)})
		} else {
			seenIDs[operation.ID] = index
		}
		reviewed, isReviewed := reviewedByID[operation.ID]
		if !isReviewed {
			issues = append(issues, issue{Path: location, Message: fmt.Sprintf("operation id %q is not in the reviewed official 2026-07-18 snapshot", operation.ID)})
		} else {
			if operation.Method != reviewed.Method {
				issues = append(issues, issue{Path: location, Message: fmt.Sprintf("operation %q method must remain %s", operation.ID, reviewed.Method)})
			}
			if operation.Path != reviewed.Path {
				issues = append(issues, issue{Path: location, Message: fmt.Sprintf("operation %q path must remain %s", operation.ID, reviewed.Path)})
			}
		}
		switch operation.Method {
		case "GET", "POST", "PUT", "DELETE":
		default:
			issues = append(issues, issue{Path: location, Message: "method must be GET, POST, PUT, or DELETE"})
		}
		if len(operation.Path) > 256 || !operationPath.MatchString(operation.Path) {
			issues = append(issues, issue{Path: location, Message: "path must use the reviewed relative resource-template syntax"})
		}
		route := operation.Method + " " + operation.Path
		if first, exists := seenRoutes[route]; exists {
			issues = append(issues, issue{Path: location, Message: fmt.Sprintf("duplicate operation route %q; first declared at index %d", route, first)})
		} else {
			seenRoutes[route] = index
		}
		if len(operation.CapabilityIDs) == 0 {
			issues = append(issues, issue{Path: location, Message: "capability_ids must contain at least one task capability"})
		}
		seenCapability := make(map[string]struct{}, len(operation.CapabilityIDs))
		hasPublicCapability := false
		for _, id := range operation.CapabilityIDs {
			if _, duplicate := seenCapability[id]; duplicate {
				issues = append(issues, issue{Path: location, Message: fmt.Sprintf("duplicate capability id %q", id)})
				continue
			}
			seenCapability[id] = struct{}{}
			if _, exists := knownCapabilities[id]; !exists {
				issues = append(issues, issue{Path: location, Message: fmt.Sprintf("capability %q is absent from %s", id, capabilitiesPath)})
			}
			if !strings.HasPrefix(id, "chatwork.") {
				issues = append(issues, issue{Path: location, Message: fmt.Sprintf("capability %q must use the chatwork namespace", id)})
			}
			if _, isPublic := publicCapabilities[id]; isPublic {
				hasPublicCapability = true
			}
			mappedCapabilities[id] = struct{}{}
		}
		if snapshot.CoverageStatus == "complete" && !hasPublicCapability {
			issues = append(issues, issue{Path: location, Message: fmt.Sprintf("complete coverage requires operation %q to map to at least one public capability", operation.ID)})
		}
	}
	for _, reviewed := range reviewedChatworkOperations {
		if _, exists := seenIDs[reviewed.ID]; !exists {
			issues = append(issues, issue{Path: chatworkAPIPath, Message: fmt.Sprintf("reviewed operation %q is missing from the official snapshot", reviewed.ID)})
		}
	}
	for id := range chatworkCapabilities {
		if _, exists := mappedCapabilities[id]; !exists {
			issues = append(issues, issue{Path: chatworkAPIPath, Message: fmt.Sprintf("Chatwork capability %q has no upstream operation mapping", id)})
		}
	}
	return issues
}

func validateChatworkLimits(limits chatworkLimits) []issue {
	expected := map[string][2]int{
		"metadata_read_timeout_seconds":  {limits.MetadataReadTimeoutSeconds, 20},
		"upload_timeout_seconds":         {limits.UploadTimeoutSeconds, 60},
		"max_attempts":                   {limits.MaxAttempts, 1},
		"success_response_bytes":         {limits.SuccessResponseBytes, 8 * 1024 * 1024},
		"provider_error_response_bytes":  {limits.ProviderErrorResponseBytes, 64 * 1024},
		"output_bytes":                   {limits.OutputBytes, 16 * 1024 * 1024},
		"list_items":                     {limits.ListItems, 10_000},
		"documented_endpoint_list_items": {limits.DocumentedEndpointListItems, 100},
		"upload_bytes":                   {limits.UploadBytes, 5 * 1024 * 1024},
	}
	var issues []issue
	for name, values := range expected {
		if values[0] != values[1] {
			issues = append(issues, issue{Path: chatworkAPIPath + ".limits", Message: fmt.Sprintf("%s must remain %d for the first complete implementation", name, values[1])})
		}
	}
	return issues
}

func validateChatworkMutationPolicy(policy chatworkMutationPolicy) []issue {
	var issues []issue
	if policy.DefaultConfirmation != "exact-invocation" {
		issues = append(issues, issue{Path: chatworkAPIPath + ".mutation_policy", Message: "default_confirmation must remain exact-invocation"})
	}
	if policy.AccessChangeConfirmation != "--confirm access-change" {
		issues = append(issues, issue{Path: chatworkAPIPath + ".mutation_policy", Message: "access_change_confirmation must remain --confirm access-change"})
	}
	issues = append(issues, validateExactOperationIDSet("mutation_policy.access_change_operation_ids", policy.AccessChangeOperationIDs, accessChangeOperationIDs)...)
	if policy.DestructiveConfirmation != "--confirm destructive" {
		issues = append(issues, issue{Path: chatworkAPIPath + ".mutation_policy", Message: "destructive_confirmation must remain --confirm destructive"})
	}
	issues = append(issues, validateExactOperationIDSet("mutation_policy.destructive_operation_ids", policy.DestructiveOperationIDs, destructiveOperationIDs)...)
	if policy.UncertainOutcome != "read-only-reconciliation" {
		issues = append(issues, issue{Path: chatworkAPIPath + ".mutation_policy", Message: "uncertain_outcome must remain read-only-reconciliation"})
	}
	return issues
}

func validateExactOperationIDSet(name string, actual, expected []string) []issue {
	location := chatworkAPIPath + "." + name
	wanted := make(map[string]struct{}, len(expected))
	for _, id := range expected {
		wanted[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(actual))
	var issues []issue
	for _, id := range actual {
		if _, duplicate := seen[id]; duplicate {
			issues = append(issues, issue{Path: location, Message: fmt.Sprintf("duplicate operation id %q", id)})
			continue
		}
		seen[id] = struct{}{}
		if _, exists := wanted[id]; !exists {
			issues = append(issues, issue{Path: location, Message: fmt.Sprintf("operation id %q is not in the reviewed %s set", id, name)})
		}
	}
	for _, id := range expected {
		if _, exists := seen[id]; !exists {
			issues = append(issues, issue{Path: location, Message: fmt.Sprintf("reviewed operation id %q is missing", id)})
		}
	}
	return issues
}

func validateCapabilities(entries []capability, catalogIDs map[string]struct{}) []issue {
	seen := make(map[string]int, len(entries))
	public := make(map[string]struct{})
	var issues []issue
	for index, entry := range entries {
		location := fmt.Sprintf("%s[%d]", capabilitiesPath, index)
		if !validContractID(entry.ID) {
			issues = append(issues, issue{Path: location, Message: "id must use lowercase dot syntax within 128 bytes, for example items.read"})
		}
		if first, exists := seen[entry.ID]; exists {
			issues = append(issues, issue{Path: location, Message: fmt.Sprintf("duplicate capability id %q; first declared at index %d", entry.ID, first)})
		} else {
			seen[entry.ID] = index
		}
		switch entry.Status {
		case "public":
			public[entry.ID] = struct{}{}
			if _, exists := catalogIDs[entry.ID]; !exists {
				issues = append(issues, issue{Path: location, Message: fmt.Sprintf("public capability %q has no command catalog entry; add its CapabilityID to a command or change its status", entry.ID)})
			}
		case "internal", "deferred", "excluded":
			if strings.TrimSpace(entry.Reason) == "" {
				issues = append(issues, issue{Path: location, Message: fmt.Sprintf("%s capability %q requires a non-empty reason", entry.Status, entry.ID)})
			}
			if _, exists := catalogIDs[entry.ID]; exists {
				issues = append(issues, issue{Path: location, Message: fmt.Sprintf("non-public capability %q is exposed by the command catalog; mark it public or remove the catalog exposure", entry.ID)})
			}
		default:
			issues = append(issues, issue{Path: location, Message: "status must be public, internal, deferred, or excluded"})
		}
		if err := validatePublicText("reason", entry.Reason, false); err != nil {
			issues = append(issues, issue{Path: location, Message: err.Error()})
		}
	}
	for id := range catalogIDs {
		if _, exists := public[id]; !exists {
			issues = append(issues, issue{
				Path:    capabilitiesPath,
				Message: fmt.Sprintf("catalog capability %q is not declared public; add one public ledger entry", id),
			})
		}
	}
	return issues
}

func validateSchemas(root string, entries []schemaFixture) []issue {
	seenIDs := make(map[string]int, len(entries))
	seenPaths := make(map[string]int, len(entries))
	var issues []issue
	for index, entry := range entries {
		location := fmt.Sprintf("%s[%d]", schemasPath, index)
		if !validContractID(entry.ID) {
			issues = append(issues, issue{Path: location, Message: "id must use lowercase dot syntax within 128 bytes, for example provider.v1"})
		}
		if first, exists := seenIDs[entry.ID]; exists {
			issues = append(issues, issue{Path: location, Message: fmt.Sprintf("duplicate schema id %q; first declared at index %d", entry.ID, first)})
		} else {
			seenIDs[entry.ID] = index
		}
		if first, exists := seenPaths[entry.Path]; exists {
			issues = append(issues, issue{Path: location, Message: fmt.Sprintf("duplicate schema path %q; first declared at index %d", entry.Path, first)})
		} else {
			seenPaths[entry.Path] = index
		}
		for name, value := range map[string]string{
			"provenance": entry.Provenance,
			"license":    entry.License,
		} {
			if err := validatePublicText(name, value, true); err != nil {
				issues = append(issues, issue{Path: location, Message: err.Error()})
			}
		}

		fixture, pathIssues := readSchemaFixture(root, entry.Path)
		for _, message := range pathIssues {
			issues = append(issues, issue{Path: location, Message: message})
		}
		if len(pathIssues) != 0 {
			continue
		}
		if !validDigest(entry.SHA256) {
			issues = append(issues, issue{Path: location, Message: "sha256 must be exactly 64 lowercase hexadecimal characters"})
			continue
		}
		actual := sha256.Sum256(fixture)
		actualText := hex.EncodeToString(actual[:])
		if actualText != entry.SHA256 {
			issues = append(issues, issue{
				Path:    location,
				Message: fmt.Sprintf("sha256 mismatch for %q: manifest has %s, computed %s; review the fixture change and update the digest deliberately", entry.Path, entry.SHA256, actualText),
			})
		}
	}
	return issues
}

func readSchemaFixture(root, relative string) ([]byte, []string) {
	if relative == "" {
		return nil, []string{"path is required"}
	}
	if strings.Contains(relative, `\`) || filepath.IsAbs(relative) || !filepath.IsLocal(relative) || filepath.ToSlash(filepath.Clean(relative)) != relative {
		return nil, []string{"path must be a canonical repository-relative path without traversal"}
	}
	parts := strings.Split(relative, "/")
	hasTestdata := false
	for _, part := range parts {
		if part == "testdata" {
			hasTestdata = true
		}
		if part == "" || strings.HasPrefix(part, ".") {
			return nil, []string{"path must not contain empty or hidden components"}
		}
	}
	if !hasTestdata {
		return nil, []string{"path must identify a publishable fixture below a testdata directory"}
	}
	lowerBase := strings.ToLower(filepath.Base(relative))
	if forbiddenFixtureName(lowerBase) {
		return nil, []string{"path looks credential-bearing; schema fixtures must be publishable test data"}
	}

	rootInfo, err := os.Lstat(root)
	if err != nil {
		return nil, []string{fmt.Sprintf("cannot inspect repository root: %v", err)}
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, []string{"repository root must be a regular directory, not a symbolic link"}
	}
	current := root
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, []string{fmt.Sprintf("fixture %q cannot be inspected: %v", relative, err)}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, []string{fmt.Sprintf("fixture path %q contains a symbolic link", relative)}
		}
		if index < len(parts)-1 && !info.IsDir() {
			return nil, []string{fmt.Sprintf("fixture path component in %q is not a directory", relative)}
		}
		if index == len(parts)-1 && !info.Mode().IsRegular() {
			return nil, []string{fmt.Sprintf("fixture %q is not a regular file", relative)}
		}
	}
	data, err := os.ReadFile(current) // #nosec G304 -- every repository-relative component was validated with Lstat above.
	if err != nil {
		return nil, []string{fmt.Sprintf("fixture %q cannot be read: %v", relative, err)}
	}
	return data, nil
}

func forbiddenFixtureName(base string) bool {
	switch base {
	case ".env", "credentials.json", "secrets.json", "id_rsa", "id_ed25519":
		return true
	}
	for _, suffix := range []string{".pem", ".key", ".p12", ".pfx"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	return false
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func validContractID(value string) bool {
	return len(value) > 0 && len(value) <= 128 && contractID.MatchString(value)
}

func validatePublicText(name, value string, required bool) error {
	if value == "" {
		if required {
			return fmt.Errorf("%s is required", name)
		}
		return nil
	}
	if len(value) > 2048 || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return fmt.Errorf("%s must be bounded, valid UTF-8 without surrounding whitespace", name)
	}
	for _, r := range value {
		if unicode.Is(unicode.C, r) {
			return fmt.Errorf("%s must not contain control characters", name)
		}
	}
	return nil
}

func loadStrictArray[T any](root, relative string) ([]T, error) {
	data, err := readRegularManifest(root, relative)
	if err != nil {
		return nil, err
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return nil, fmt.Errorf("%s: invalid strict JSON: %w", relative, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var entries []T
	if err := decoder.Decode(&entries); err != nil {
		return nil, fmt.Errorf("%s: decode strict JSON array: %w", relative, err)
	}
	if entries == nil {
		return nil, fmt.Errorf("%s: top level must be an array; use [] when there are no entries", relative)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, fmt.Errorf("%s: %w", relative, err)
	}
	return entries, nil
}

func loadStrictObject[T any](root, relative string) (T, error) {
	var value T
	data, err := readRegularManifest(root, relative)
	if err != nil {
		return value, err
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return value, fmt.Errorf("%s: invalid strict JSON: %w", relative, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("%s: decode strict JSON object: %w", relative, err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return value, fmt.Errorf("%s: %w", relative, err)
	}
	return value, nil
}

func readRegularManifest(root, relative string) ([]byte, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect repository root for %s: %w", relative, err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s: repository root must be a regular directory, not a symbolic link", relative)
	}
	current := root
	parts := strings.Split(filepath.FromSlash(relative), string(filepath.Separator))
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%s: manifest path contains a symbolic link", relative)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return nil, fmt.Errorf("%s: manifest path component is not a directory", relative)
		}
		if index == len(parts)-1 && !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s: manifest must be a regular file", relative)
		}
	}
	data, err := os.ReadFile(current) // #nosec G304 -- this fixed manifest path and all of its components were validated above.
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", relative, err)
	}
	return data, nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bufio.NewReader(bytes.NewReader(data)))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder, "$", 0); err != nil {
		return err
	}
	return requireJSONEOF(decoder)
}

func consumeJSONValue(decoder *json.Decoder, path string, depth int) error {
	if depth > 128 {
		return fmt.Errorf("JSON nesting exceeds 128 levels at %s", path)
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
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
				return fmt.Errorf("object key at %s is not a string", path)
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate object key %q at %s", key, path)
			}
			seen[key] = struct{}{}
			if err := consumeJSONValue(decoder, path+"."+key, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("object at %s is not closed", path)
		}
	case '[':
		index := 0
		for decoder.More() {
			if err := consumeJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index), depth+1); err != nil {
				return err
			}
			index++
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("array at %s is not closed", path)
		}
	default:
		return fmt.Errorf("unexpected delimiter %q at %s", delimiter, path)
	}
	return nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("multiple top-level JSON values are not allowed")
	}
	return fmt.Errorf("invalid trailing JSON: %w", err)
}
