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
	"sync"
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

type preparedMessageBody struct {
	mu      sync.Mutex
	owner   *Client
	binding authn.BindingID
	source  chatwork.Request
	bytes   []byte
	used    bool
}

func (*preparedMessageBody) IsPreparedMessageBody() {}

func (*preparedMessageBody) String() string { return "[prepared Chatwork message body]" }

func (*preparedMessageBody) GoString() string { return "[prepared Chatwork message body]" }

func (p *preparedMessageBody) consume(owner *Client, binding authn.BindingID, request chatwork.Request) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.used || p.owner != owner || p.binding != binding || !samePreparedMessageBodySource(p.source, request) {
		return "", false
	}
	p.used = true
	body := string(p.bytes)
	clear(p.bytes)
	p.bytes = nil
	p.source = chatwork.Request{}
	p.binding = authn.BindingID{}
	p.owner = nil
	return body, true
}

func (p *preparedMessageBody) discard(owner *Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.owner != owner {
		return
	}
	clear(p.bytes)
	p.bytes = nil
	p.source = chatwork.Request{}
	p.binding = authn.BindingID{}
	p.owner = nil
	p.used = true
}

func samePreparedMessageBodySource(source, request chatwork.Request) bool {
	if request.Task != chatwork.TaskMessagesSend || request.Body != "" || request.BodyFile != source.BodyFile ||
		request.Room != source.Room || request.SelfUnread != source.SelfUnread ||
		len(request.MessageRecipients) != len(source.MessageRecipients) {
		return false
	}
	for index := range source.MessageRecipients {
		if request.MessageRecipients[index] != source.MessageRecipients[index] {
			return false
		}
	}
	if source.MessageReplyTarget == nil || request.MessageReplyTarget == nil ||
		source.MessageReplyTarget.Message != request.MessageReplyTarget.Message {
		return false
	}
	return true
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
		return chatwork.Result{}, cancellationBeforeDispatch(input.Task, err)
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
	spec, err := c.buildRequestContext(ctx, input)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return chatwork.Result{}, cancellationBeforeDispatch(input.Task, ctxErr)
		}
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
	request.Header.Set("User-Agent", c.userAgent)
	if err := record.credential.authorize(request); err != nil {
		return chatwork.Result{}, err
	}
	if spec.contentType != "" {
		request.Header.Set("Content-Type", spec.contentType)
	}
	if c.http == nil {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_transport_missing", "Chatwork トランスポートが設定されていません", false)
	}
	// This check is the last provable no-dispatch boundary. Cancellation after
	// it may race with Do and must be classified as an unknown mutation outcome.
	if err := callCtx.Err(); err != nil {
		return chatwork.Result{}, cancellationBeforeDispatch(input.Task, err)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return chatwork.Result{}, transportFault(input.Task, callCtx, err)
	}
	if response == nil || response.Body == nil {
		return chatwork.Result{}, mutationResponseFailure(input.Task,
			fault.New(fault.KindContract, "chatwork_response_invalid", "Chatwork が無効なレスポンスを返しました", false),
		)
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
		return chatwork.Result{}, mutationResponseFailure(input.Task,
			fault.New(fault.KindContract, "chatwork_unexpected_response", "Chatwork が文書化されていない成功ステータスを返しました", false),
		)
	}
	if response.StatusCode == http.StatusNoContent {
		if !allowsNoContent(input.Task) {
			return chatwork.Result{}, mutationResponseFailure(input.Task,
				fault.New(fault.KindContract, "chatwork_unexpected_response", "Chatwork が文書化されていない空レスポンスを返しました", false),
			)
		}
		result := emptyResult(input)
		result.MessageAccess = messageAccess
		return result, nil
	}
	body, err := readBounded(response.Body, MaxSuccessResponseBytes)
	if err != nil {
		return chatwork.Result{}, mutationResponseFailure(input.Task, err)
	}
	result, err := mapResponse(input, body)
	if err != nil {
		return chatwork.Result{}, mutationResponseFailure(input.Task, err)
	}
	if mutating(input.Task) {
		if err := result.ValidateFor(input); err != nil {
			return chatwork.Result{}, mutationOutcomeUnknown()
		}
	}
	if input.Task == chatwork.TaskMessagesList {
		result.MessageAccess = messageAccess
		if len(result.Messages) == 0 {
			return chatwork.Result{}, malformedResponse()
		}
	}
	return result, nil
}

// PrepareMessageBody reads and validates a reply-send body file through the
// existing infrastructure-owned reader before the application performs its
// reply GET. The returned one-use handle keeps the exact bytes in this adapter
// so the later POST does not reread a mutable path.
func (c *Client) PrepareMessageBody(ctx context.Context, binding authn.BindingID, input chatwork.Request) (chatwork.PreparedMessageBody, error) {
	if ctx == nil {
		return nil, fault.New(fault.KindContract, "missing_context", "Chatwork タスクコンテキストが設定されていません", false)
	}
	if err := ctx.Err(); err != nil {
		return nil, cancellationBeforeDispatch(input.Task, err)
	}
	if input.Task != chatwork.TaskMessagesSend || input.BodyFile == "" || input.Body != "" || input.MessageReplyTarget == nil {
		return nil, invalidRequest("reply body-file preparation requires one messages.send body file and exact reply target")
	}
	if err := input.Validate(); err != nil {
		return nil, fault.Wrap(fault.KindInvalidInput, "invalid_chatwork_request", "Chatwork タスク入力は無効です", false, err)
	}
	if _, err := c.resolve(binding); err != nil {
		return nil, err
	}
	body, err := c.messageBody(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := chatwork.ValidateMessageBody(body); err != nil {
		return nil, fault.New(fault.KindInvalidInput, "chatwork_message_body_invalid", "メッセージ本文ファイルには空でない65535バイト以下の有効なUTF-8が必要です", false)
	}
	if err := ctx.Err(); err != nil {
		return nil, cancellationBeforeDispatch(input.Task, err)
	}
	return &preparedMessageBody{
		owner: c, binding: binding, source: clonePreparedMessageBodySource(input), bytes: []byte(body),
	}, nil
}

// ExecutePreparedMessageSend consumes a body-file handle created by this same
// client and binding, then executes the ordinary single-attempt send path with
// the prepared bytes as its literal body source.
func (c *Client) ExecutePreparedMessageSend(ctx context.Context, binding authn.BindingID, input chatwork.Request, prepared chatwork.PreparedMessageBody) (chatwork.Result, error) {
	body, ok := prepared.(*preparedMessageBody)
	if !ok || body == nil {
		return chatwork.Result{}, invalidRequest("prepared message body handle is invalid")
	}
	value, ok := body.consume(c, binding, input)
	if !ok {
		return chatwork.Result{}, invalidRequest("prepared message body handle does not match this send")
	}
	input.Body = value
	input.BodyFile = ""
	return c.Execute(ctx, binding, input)
}

// DiscardPreparedMessageBody releases an unused body-file snapshot after a
// reply preflight failure. Foreign and already-consumed handles are ignored.
func (c *Client) DiscardPreparedMessageBody(prepared chatwork.PreparedMessageBody) {
	if body, ok := prepared.(*preparedMessageBody); ok && body != nil {
		body.discard(c)
	}
}

func clonePreparedMessageBodySource(input chatwork.Request) chatwork.Request {
	copy := input
	copy.MessageRecipients = append([]chatwork.Reference(nil), input.MessageRecipients...)
	if input.MessageReplyTarget != nil {
		target := *input.MessageReplyTarget
		copy.MessageReplyTarget = &target
	}
	return copy
}

func cancellationBeforeDispatch(task chatwork.Task, cause error) error {
	if mutating(task) {
		return fault.New(
			fault.KindCanceled,
			"chatwork_mutation_not_dispatched",
			"キャンセルされたため、変更要求は Chatwork への送信前に停止しました",
			true,
		)
	}
	if errors.Is(cause, context.DeadlineExceeded) {
		return fault.New(fault.KindCanceled, "operation_canceled", "Chatwork の読み取りが時間内に完了しませんでした", true)
	}
	return fault.Wrap(fault.KindCanceled, "operation_canceled", "実行前に Chatwork タスクがキャンセルされました", true, cause)
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

func (c *Client) messageBody(ctx context.Context, input chatwork.Request) (string, error) {
	data, err := c.messageBodyBytes(ctx, input)
	if err != nil {
		return "", err
	}
	defer clear(data)
	return string(data), nil
}

func expandMessageBody(input chatwork.Request, body string) (string, error) {
	bodyBytes := []byte(body)
	defer clear(bodyBytes)
	expanded, err := expandMessageBodyBytes(input, bodyBytes)
	if err != nil {
		return "", err
	}
	defer clear(expanded)
	return string(expanded), nil
}

func expandMessageBodyBytes(input chatwork.Request, body []byte) ([]byte, error) {
	tags := make([]string, 0, len(input.MessageRecipients)+1)
	for _, recipient := range input.MessageRecipients {
		tags = append(tags, "[To:"+recipient.Value+"]")
	}
	if input.MessageReplyTarget != nil {
		target := input.MessageReplyTarget
		if target.Sender.Kind != chatwork.ReferenceAccount || chatwork.ValidateReference(chatwork.ReferenceAccount, target.Sender.Value) != nil {
			return nil, fault.New(fault.KindContract, "chatwork_request_contract_invalid", "返信記法を作成する投稿者参照がありません", false)
		}
		tags = append(tags, fmt.Sprintf("[rp aid=%s to=%s-%s]", target.Sender.Value, input.Room.Value, target.Message.Value))
	}
	var expanded []byte
	if len(tags) > 0 {
		prefix := strings.Join(tags, " ") + " "
		expanded = make([]byte, 0, len(prefix)+len(body))
		expanded = append(expanded, prefix...)
		expanded = append(expanded, body...)
	} else {
		expanded = append([]byte(nil), body...)
	}
	if err := chatwork.ValidateMessageBodyBytes(expanded); err != nil {
		code := "chatwork_message_body_invalid"
		message := "展開後のメッセージ本文が空か、有効なUTF-8ではありません"
		if len(expanded) > chatwork.MaxMessageBodyBytes {
			code = "chatwork_message_body_too_large"
			message = "展開後のメッセージ本文が 65535 バイトの上限を超えています"
		}
		clear(expanded)
		return nil, fault.New(fault.KindInvalidInput, code, message, false)
	}
	return expanded, nil
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
	return readBoundedRegularFile(path, MaxUploadBytes+1)
}

func boundedReadMessageBodyFile(ctx context.Context, path string) ([]byte, error) {
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	value, err := readBoundedRegularFile(path, int64(chatwork.MaxMessageBodyBytes)+1)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return value, nil
}

func readBoundedRegularFile(path string, limit int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("selected path is not a readable regular file")
	}
	file, err := os.Open(path) // #nosec G304 -- upload and message-body tasks intentionally read the exact user-selected path and apply fixed byte ceilings before HTTP.
	if err != nil {
		return nil, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("selected path changed before its bounded read")
	}
	return io.ReadAll(io.LimitReader(file, limit))
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
	case http.StatusBadRequest:
		return fault.New(fault.KindInvalidInput, "chatwork_invalid_request", "Chatwork がタスク入力を拒否しました", false)
	case http.StatusUnprocessableEntity:
		if mutating(task) {
			// Chatwork's reviewed endpoint contracts document 400, not 422. A
			// status outside that contract cannot prove a mutation was rejected.
			return mutationOutcomeUnknown()
		}
		return fault.New(fault.KindInvalidInput, "chatwork_invalid_request", "Chatwork がタスク入力を拒否しました", false)
	case http.StatusUnauthorized:
		return fault.New(fault.KindAuthentication, "chatwork_authentication_failed", "Chatwork が設定済みの API トークンを拒否しました", false)
	case http.StatusForbidden:
		message := "Chatwork がこのタスクの権限を拒否しました"
		if hasRoomTarget(task) {
			message = "Chatwork が対象ルームの権限を拒否しました。ルームに参加していない可能性があります"
		}
		return fault.New(fault.KindPermission, "chatwork_permission_denied", message, false)
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
		if mutating(task) {
			// Only documented provider rejection statuses above prove that the
			// mutation was not applied. An unfamiliar status leaves its outcome
			// unknown regardless of whether it is an error or redirect response.
			return mutationOutcomeUnknown()
		}
		if response.StatusCode >= 500 {
			return fault.New(fault.KindUnavailable, "chatwork_unavailable", "Chatwork は一時的に利用できません", true)
		}
		return fault.New(fault.KindContract, "chatwork_unexpected_response", "Chatwork が予期しないレスポンスステータスを返しました", false)
	}
}

func transportFault(task chatwork.Task, ctx context.Context, err error) error {
	if mutating(task) {
		return mutationOutcomeUnknown()
	}
	var contextErr error
	if ctx != nil {
		contextErr = ctx.Err()
	}
	if errors.Is(contextErr, context.Canceled) {
		return fault.Wrap(fault.KindCanceled, "operation_canceled", "実行中に Chatwork タスクがキャンセルされました", true, contextErr)
	}
	if errors.Is(contextErr, context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return fault.New(fault.KindCanceled, "operation_canceled", "Chatwork の読み取りが時間内に完了しませんでした", true)
	}
	if errors.Is(err, context.Canceled) {
		return fault.Wrap(fault.KindCanceled, "operation_canceled", "実行中に Chatwork タスクがキャンセルされました", true, err)
	}
	return fault.Wrap(fault.KindUnavailable, "chatwork_unavailable", "Chatwork は一時的に利用できません", true, err)
}

func hasRoomTarget(task chatwork.Task) bool {
	switch task {
	case chatwork.TaskRoomsShow, chatwork.TaskRoomsUpdate, chatwork.TaskRoomsLeave, chatwork.TaskRoomsDelete,
		chatwork.TaskMembersList, chatwork.TaskMembersReplace,
		chatwork.TaskMessagesList, chatwork.TaskMessagesSend, chatwork.TaskMessagesReconcileSend, chatwork.TaskMessagesMarkRead, chatwork.TaskMessagesMarkUnread,
		chatwork.TaskMessagesShow, chatwork.TaskMessagesUpdate, chatwork.TaskMessagesDelete,
		chatwork.TaskRoomTasksList, chatwork.TaskRoomTasksCreate, chatwork.TaskRoomTasksShow, chatwork.TaskRoomTasksSetStatus,
		chatwork.TaskFilesList, chatwork.TaskFilesUpload, chatwork.TaskFilesShow,
		chatwork.TaskInviteLinkShow, chatwork.TaskInviteLinkCreate, chatwork.TaskInviteLinkUpdate, chatwork.TaskInviteLinkDelete:
		return true
	default:
		return false
	}
}

func mutationResponseFailure(task chatwork.Task, responseFault error) error {
	if !mutating(task) {
		return responseFault
	}
	return mutationOutcomeUnknown()
}

func mutationOutcomeUnknown() error {
	return fault.New(fault.KindContract, "chatwork_mutation_outcome_unknown", "Chatwork が変更結果を確認できませんでした。再試行前に状態を照合してください", false)
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
