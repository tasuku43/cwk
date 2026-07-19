package chatworkapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/tasuku43/cwk/internal/domain/apicall"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

// CallPolicy exposes the finite transport declaration used for a task. All
// mutations are conservatively unsafe and every operation has one attempt.
func CallPolicy(task chatwork.Task) (apicall.Policy, error) {
	if !task.Valid() {
		return apicall.Policy{}, invalidRequest("Chatwork タスクが呼び出しポリシーにマッピングされていません")
	}
	timeout := RequestTimeout
	idempotency := apicall.IdempotencySafe
	if mutating(task) {
		idempotency = apicall.IdempotencyUnsafe
	}
	if task == chatwork.TaskFilesUpload {
		timeout = UploadTimeout
	}
	return apicall.SingleAttempt(timeout, idempotency), nil
}

const (
	ProductionBaseURL       = "https://api.chatwork.com/v2"
	RequestTimeout          = 20 * time.Second
	UploadTimeout           = 60 * time.Second
	MaxAttempts             = 1
	MaxSuccessResponseBytes = 8 * 1024 * 1024
	MaxErrorResponseBytes   = 64 * 1024
	MaxUploadBytes          = 5 * 1024 * 1024
	generalRateLimitWindow  = 5 * time.Minute
	roomPostRateLimitWait   = 10 * time.Second
)

const documentedRoomPostRateLimitMessage = "Rate limit for message posting per room exceeded."

const (
	messageLimitationHeader        = "chatwork-message-limitation"
	messageLimitationSummaryHeader = "chatwork-message-limitation-summary"
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func productionHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: RequestTimeout,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{
		// Per-operation contexts enforce the shorter metadata/read budget.
		// Client.Timeout is the outer ceiling needed by file upload.
		Timeout:   UploadTimeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// Execute maps one provider-independent Chatwork task to exactly one bounded
// official API operation. It never retries or follows a redirect.
func (c *Client) Execute(ctx context.Context, binding authn.BindingID, input chatwork.Request) (chatwork.Result, error) {
	if ctx == nil {
		return chatwork.Result{}, fault.New(fault.KindContract, "missing_context", "Chatwork タスクコンテキストが設定されていません", false)
	}
	if err := ctx.Err(); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindCanceled, "operation_canceled", "実行前に Chatwork タスクがキャンセルされました", true, err)
	}
	if err := input.Validate(); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindInvalidInput, "invalid_chatwork_request", "Chatwork タスク入力は無効です", false, err)
	}
	record, err := c.resolve(binding)
	if err != nil {
		return chatwork.Result{}, err
	}
	if input.Task == chatwork.TaskRoomsCreate &&
		(record.session.AccountID == "" || record.session.AccountID != input.Account.Value) {
		return chatwork.Result{}, fault.New(fault.KindAuthentication, "authentication_context_mismatch", "認証が要求された Chatwork アカウントと一致しません", false)
	}
	spec, err := c.buildRequest(input)
	if err != nil {
		return chatwork.Result{}, err
	}
	policy, err := CallPolicy(input.Task)
	if err != nil {
		return chatwork.Result{}, err
	}
	callCtx, cancel := context.WithTimeout(ctx, policy.Timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(callCtx, spec.method, c.baseURL+spec.path, spec.body)
	if err != nil {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_request_contract_invalid", "Chatwork リクエストを構築できませんでした", false)
	}
	request.Header.Set("Accept", "application/json")
	if err := record.credential.authorize(request); err != nil {
		return chatwork.Result{}, err
	}
	if spec.contentType != "" {
		request.Header.Set("Content-Type", spec.contentType)
	}
	if c.http == nil {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_transport_missing", "Chatwork トランスポートが設定されていません", false)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return chatwork.Result{}, transportFault(input.Task, callCtx, err)
	}
	if response == nil || response.Body == nil {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_response_invalid", "Chatwork が無効なレスポンスを返しました", false)
	}
	defer response.Body.Close()
	messageAccess, err := messageAccessFromResponse(input.Task, response.StatusCode, response.Header)
	if err != nil {
		return chatwork.Result{}, err
	}
	if input.Task == chatwork.TaskMessagesShow && response.StatusCode == http.StatusNotFound && messageAccess != chatwork.MessageAccessNone {
		return chatwork.Result{}, fault.New(fault.KindPermission, "chatwork_message_restricted", "指定した Chatwork メッセージは閲覧制限により取得できません", false)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return chatwork.Result{}, providerFault(input.Task, response, c.currentTime())
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_unexpected_response", "Chatwork が文書化されていない成功ステータスを返しました", false)
	}
	if response.StatusCode == http.StatusNoContent {
		if !allowsNoContent(input.Task) {
			return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_unexpected_response", "Chatwork が文書化されていない空レスポンスを返しました", false)
		}
		result := emptyResult(input)
		result.MessageAccess = messageAccess
		return result, nil
	}
	body, err := readBounded(response.Body, MaxSuccessResponseBytes)
	if err != nil {
		return chatwork.Result{}, err
	}
	result, err := mapResponse(input, body)
	if err != nil {
		return chatwork.Result{}, err
	}
	if input.Task == chatwork.TaskMessagesList {
		result.MessageAccess = messageAccess
		if len(result.Messages) == 0 {
			return chatwork.Result{}, malformedResponse()
		}
	}
	return result, nil
}

func messageAccessFromResponse(task chatwork.Task, status int, header http.Header) (chatwork.MessageAccessLimitation, error) {
	if task != chatwork.TaskMessagesList && task != chatwork.TaskMessagesShow {
		return chatwork.MessageAccessNone, nil
	}
	limitationValues := header.Values(messageLimitationHeader)
	summaryPresent := len(header.Values(messageLimitationSummaryHeader)) > 0
	if len(limitationValues) == 0 {
		if summaryPresent {
			return chatwork.MessageAccessNone, invalidMessageLimitation()
		}
		return chatwork.MessageAccessNone, nil
	}
	if len(limitationValues) != 1 || limitationValues[0] != "true" {
		return chatwork.MessageAccessNone, invalidMessageLimitation()
	}

	switch {
	case task == chatwork.TaskMessagesList && status == http.StatusOK:
		return chatwork.MessageAccessPartial, nil
	case task == chatwork.TaskMessagesList && status == http.StatusNoContent:
		return chatwork.MessageAccessAll, nil
	case task == chatwork.TaskMessagesShow && status == http.StatusNotFound:
		return chatwork.MessageAccessAll, nil
	default:
		return chatwork.MessageAccessNone, invalidMessageLimitation()
	}
}

func invalidMessageLimitation() error {
	return fault.New(fault.KindContract, "chatwork_message_limitation_invalid", "Chatwork のメッセージ閲覧制限ヘッダーが公式契約と一致しません", false)
}

func allowsNoContent(task chatwork.Task) bool {
	switch task {
	case chatwork.TaskPersonalTasksList, chatwork.TaskContactsList,
		chatwork.TaskMessagesList, chatwork.TaskRoomTasksList,
		chatwork.TaskFilesList, chatwork.TaskContactRequestsList,
		chatwork.TaskRoomsLeave, chatwork.TaskRoomsDelete,
		chatwork.TaskContactRequestsReject:
		return true
	default:
		return false
	}
}

type requestSpec struct {
	method      string
	path        string
	body        io.Reader
	contentType string
}

func formRequest(method, path string, values url.Values) requestSpec {
	return requestSpec{method: method, path: path, body: strings.NewReader(values.Encode()), contentType: "application/x-www-form-urlencoded"}
}

func noBodyRequest(method, path string, query url.Values) requestSpec {
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return requestSpec{method: method, path: path}
}

func (c *Client) multipartRequest(path string, input chatwork.Request) (requestSpec, error) {
	if c.readFile == nil || input.FilePath == "" {
		return requestSpec{}, invalidRequest("file upload requires an explicit readable path")
	}
	data, err := c.readFile(input.FilePath)
	if err != nil {
		return requestSpec{}, fault.New(fault.KindInvalidInput, "chatwork_file_unreadable", "Chatwork へアップロードするファイルを読み取れませんでした", false)
	}
	if len(data) > MaxUploadBytes {
		return requestSpec{}, fault.New(fault.KindInvalidInput, "chatwork_file_too_large", "Chatwork へアップロードするファイルが 5 MiB の上限を超えています", false)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	filename := filepath.Base(input.FilePath)
	if !validUploadFilename(filename) {
		return requestSpec{}, fault.New(fault.KindInvalidInput, "chatwork_file_name_invalid", "Chatwork へアップロードするファイル名は無効です", false)
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return requestSpec{}, fault.New(fault.KindContract, "chatwork_upload_contract_invalid", "Chatwork アップロードリクエストを構築できませんでした", false)
	}
	if _, err := part.Write(data); err != nil {
		return requestSpec{}, fault.New(fault.KindContract, "chatwork_upload_contract_invalid", "Chatwork アップロードリクエストを構築できませんでした", false)
	}
	if input.FileMessage != "" {
		if err := writer.WriteField("message", input.FileMessage); err != nil {
			return requestSpec{}, fault.New(fault.KindContract, "chatwork_upload_contract_invalid", "Chatwork アップロードリクエストを構築できませんでした", false)
		}
	}
	if err := writer.Close(); err != nil {
		return requestSpec{}, fault.New(fault.KindContract, "chatwork_upload_contract_invalid", "Chatwork アップロードリクエストを構築できませんでした", false)
	}
	return requestSpec{method: http.MethodPost, path: path, body: bytes.NewReader(body.Bytes()), contentType: writer.FormDataContentType()}, nil
}

func validUploadFilename(value string) bool {
	if value == "" || value == "." || value == string(filepath.Separator) || len(value) > 255 || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.Is(unicode.C, r) || r == '\u2028' || r == '\u2029' {
			return false
		}
	}
	return true
}

func boundedReadFile(path string) ([]byte, error) {
	file, err := os.Open(path) // #nosec G304 -- files upload intentionally reads the exact user-selected --path and applies the fixed byte ceiling below.
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fault.New(fault.KindUnavailable, "chatwork_response_unavailable", "Chatwork レスポンスを読み取れませんでした", true)
	}
	if int64(len(body)) > limit {
		return nil, fault.New(fault.KindContract, "chatwork_response_too_large", "Chatwork レスポンスが設定済みのバイト数上限を超えました", false)
	}
	return body, nil
}

func (c *Client) currentTime() time.Time {
	if c != nil && c.now != nil {
		return c.now().UTC().Truncate(time.Second)
	}
	return time.Now().UTC().Truncate(time.Second)
}

func providerFault(task chatwork.Task, response *http.Response, now time.Time) error {
	// Read only the reviewed error-body ceiling. The bytes remain private and are
	// used solely to recognize one exact documented rate-limit condition.
	body, bodyBounded := readProviderErrorBody(response.Body)
	switch response.StatusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return fault.New(fault.KindInvalidInput, "chatwork_invalid_request", "Chatwork がタスク入力を拒否しました", false)
	case http.StatusUnauthorized:
		return fault.New(fault.KindAuthentication, "chatwork_authentication_failed", "Chatwork が設定済みの API トークンを拒否しました", false)
	case http.StatusForbidden:
		return fault.New(fault.KindPermission, "chatwork_permission_denied", "Chatwork がこのタスクの権限を拒否しました", false)
	case http.StatusNotFound:
		return fault.New(fault.KindNotFound, "chatwork_not_found", "要求した Chatwork リソースが見つかりませんでした", false)
	case http.StatusTooManyRequests:
		code := "chatwork_rate_limited"
		message := "Chatwork のレート上限に達しました"
		retryable := true
		if mutating(task) {
			code = "chatwork_mutation_rate_limited"
			message = "Chatwork の変更はレート上限により拒否されました。自動再試行は行いません"
			retryable = false
		}
		err := fault.New(fault.KindRateLimited, code, message, retryable)
		if roomPostRateLimitedTask(task) && bodyBounded && documentedRoomPostRateLimit(body) {
			err.RetryAfter = roomPostRateLimitWait
		} else {
			err.RetryAfter = rateLimitRetryAfter(response.Header, now)
		}
		return err
	default:
		if mutating(task) && response.StatusCode >= 500 {
			return fault.New(fault.KindContract, "chatwork_mutation_outcome_unknown", "Chatwork が変更結果を確認できませんでした。再試行前に状態を照合してください", false)
		}
		if response.StatusCode >= 500 {
			return fault.New(fault.KindUnavailable, "chatwork_unavailable", "Chatwork は一時的に利用できません", true)
		}
		return fault.New(fault.KindContract, "chatwork_unexpected_response", "Chatwork が予期しないレスポンスステータスを返しました", false)
	}
}

func transportFault(task chatwork.Task, ctx context.Context, err error) error {
	if mutating(task) {
		return fault.Wrap(fault.KindContract, "chatwork_mutation_outcome_unknown", "Chatwork が変更結果を確認できませんでした。再試行前に状態を照合してください", false, err)
	}
	if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fault.Wrap(fault.KindCanceled, "operation_canceled", "実行中に Chatwork タスクがキャンセルされました", true, ctx.Err())
	}
	return fault.Wrap(fault.KindUnavailable, "chatwork_unavailable", "Chatwork は一時的に利用できません", true, err)
}

func readProviderErrorBody(reader io.Reader) ([]byte, bool) {
	if reader == nil {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(reader, MaxErrorResponseBytes+1))
	if err != nil || len(body) > MaxErrorResponseBytes {
		return nil, false
	}
	return body, true
}

func roomPostRateLimitedTask(task chatwork.Task) bool {
	return task == chatwork.TaskMessagesSend || task == chatwork.TaskRoomTasksCreate
}

func documentedRoomPostRateLimit(body []byte) bool {
	if len(body) == 0 || len(body) > MaxErrorResponseBytes {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return false
	}
	seenErrors := false
	matched := false
	for decoder.More() {
		key, err := decoder.Token()
		name, ok := key.(string)
		if err != nil || !ok || name != "errors" || seenErrors {
			return false
		}
		seenErrors = true
		var messages []string
		if err := decoder.Decode(&messages); err != nil || len(messages) != 1 {
			return false
		}
		matched = messages[0] == documentedRoomPostRateLimitMessage
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') || !seenErrors || !matched {
		return false
	}
	return decoder.Decode(&struct{}{}) == io.EOF
}

func rateLimitRetryAfter(header http.Header, now time.Time) time.Duration {
	values := header.Values("x-ratelimit-reset")
	if len(values) != 1 || !strictDecimal(values[0]) {
		return 0
	}
	seconds, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return 0
	}
	baseline := now.UTC().Truncate(time.Second)
	reset := time.Unix(seconds, 0).UTC()
	if !withinRateLimitWindow(reset, baseline) {
		return 0
	}
	if dates := header.Values("Date"); len(dates) == 1 {
		if responseTime, parseErr := http.ParseTime(dates[0]); parseErr == nil {
			baseline = responseTime.UTC().Truncate(time.Second)
			if !withinRateLimitWindow(reset, baseline) {
				return 0
			}
		}
	}
	return reset.Sub(baseline)
}

func strictDecimal(value string) bool {
	if value == "" {
		return false
	}
	for index := range len(value) {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func withinRateLimitWindow(reset, baseline time.Time) bool {
	delay := reset.Sub(baseline)
	return delay > 0 && delay <= generalRateLimitWindow
}

func decodeJSON(body []byte, target any) error {
	if len(body) == 0 {
		return fault.New(fault.KindContract, "chatwork_response_malformed", "Chatwork レスポンス本文がありません", false)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return fault.New(fault.KindContract, "chatwork_response_malformed", "Chatwork レスポンス JSON は不正です", false)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fault.New(fault.KindContract, "chatwork_response_malformed", "Chatwork レスポンスに後続データがあります", false)
	}
	return nil
}

func invalidRequest(message string) error {
	return fault.New(fault.KindInvalidInput, "invalid_chatwork_request", message, false)
}

func decimal(ref chatwork.Reference, kind chatwork.ReferenceKind) (string, error) {
	if ref.Kind != kind || chatwork.ValidateReference(kind, ref.Value) != nil {
		return "", invalidRequest(fmt.Sprintf("タスクには正確な %s 参照が必要です", kind))
	}
	return ref.Value, nil
}
