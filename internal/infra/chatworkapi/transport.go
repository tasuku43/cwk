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
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return chatwork.Result{}, providerFault(input.Task, response)
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_unexpected_response", "Chatwork が文書化されていない成功ステータスを返しました", false)
	}
	if response.StatusCode == http.StatusNoContent {
		if !allowsNoContent(input.Task) {
			return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_unexpected_response", "Chatwork が文書化されていない空レスポンスを返しました", false)
		}
		return emptyResult(input), nil
	}
	body, err := readBounded(response.Body, MaxSuccessResponseBytes)
	if err != nil {
		return chatwork.Result{}, err
	}
	result, err := mapResponse(input, body)
	if err != nil {
		return chatwork.Result{}, err
	}
	return result, nil
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

func providerFault(task chatwork.Task, response *http.Response) error {
	// Drain a bounded amount so a connection can be reused, but never expose or
	// parse provider prose into the public fault.
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, MaxErrorResponseBytes+1))
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
		err := fault.New(fault.KindRateLimited, "chatwork_rate_limited", "Chatwork のレート上限に達しました", true)
		err.RetryAfter = retryAfter(response.Header.Get("Retry-After"))
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

func retryAfter(value string) time.Duration {
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds <= 0 || seconds > 86400 {
		return 0
	}
	return time.Duration(seconds) * time.Second
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
