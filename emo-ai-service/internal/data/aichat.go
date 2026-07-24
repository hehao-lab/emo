package data

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"emo-ai-service/internal/biz"
	"emo-ai-service/internal/conf"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/transport"
)

// defaultAIServiceBaseURL keeps local development working when config is omitted.
const defaultAIServiceBaseURL = "http://127.0.0.1:8000"

// aiChatRepo implements biz.AIChatRepo by calling the downstream FastAPI service.
type aiChatRepo struct {
	baseURL string
	client  *http.Client
	// streamClient has no whole-request timeout because SSE duration is controlled by context cancellation.
	streamClient   *http.Client
	identitySecret []byte
}

// NewAIChatRepo creates the FastAPI-backed AI chat repository.
func NewAIChatRepo(c *conf.AIService) biz.AIChatRepo {
	if c == nil {
		c = &conf.AIService{}
	}
	baseURL := strings.TrimRight(c.GetBaseUrl(), "/")
	if baseURL == "" {
		baseURL = defaultAIServiceBaseURL
	}
	identitySecret := os.Getenv("EMO_AI_SERVICE_SHARED_SECRET")
	if os.Getenv("EMO_ENV") == "production" && identitySecret == "" {
		panic("EMO_AI_SERVICE_SHARED_SECRET must be configured in production")
	}
	return &aiChatRepo{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: protoDurationOrDefault(c.GetTimeout(), 120*time.Second),
		},
		streamClient:   &http.Client{},
		identitySecret: []byte(identitySecret),
	}
}

// Health calls the FastAPI health endpoint.
func (r *aiChatRepo) Health(ctx context.Context) error {
	var out struct {
		Status string `json:"status"`
	}
	return r.doJSON(ctx, http.MethodGet, "/health/ready", "", nil, &out)
}

// CreateConversation forwards conversation creation with a signed BFF identity.
func (r *aiChatRepo) CreateConversation(ctx context.Context, userID string, title string) (*biz.AIConversation, error) {
	var out conversationPO
	err := r.doJSON(ctx, http.MethodPost, "/api/v1/conversations", userID, map[string]string{"title": title}, &out)
	if err != nil {
		return nil, err
	}
	return out.toBiz(), nil
}

// ListConversations forwards the current user's conversation list request.
func (r *aiChatRepo) ListConversations(ctx context.Context, userID string) ([]*biz.AIConversation, error) {
	var out conversationSetPO
	if err := r.doJSON(ctx, http.MethodGet, "/api/v1/conversations", userID, nil, &out); err != nil {
		return nil, err
	}
	return out.toBiz(), nil
}

// ListMessages forwards the message-list request for one conversation.
func (r *aiChatRepo) ListMessages(ctx context.Context, userID string, conversationID string) ([]*biz.AIMessage, error) {
	var out messageSetPO
	path := fmt.Sprintf("/api/v1/conversations/%s/messages", conversationID)
	if err := r.doJSON(ctx, http.MethodGet, path, userID, nil, &out); err != nil {
		return nil, err
	}
	return out.toBiz(), nil
}

// Chat forwards a non-streaming chat request and decodes the complete response.
func (r *aiChatRepo) Chat(ctx context.Context, req *biz.AIChatRequest) (*biz.AIChatReply, error) {
	var out chatReplyPO
	if err := r.doJSON(ctx, http.MethodPost, "/api/v1/chat", req.UpstreamUserID, chatRequestPO{
		ConversationID:  req.ConversationID,
		Message:         req.Message,
		SystemPrompt:    req.SystemPrompt,
		ClientRequestID: optionalStringPO(req.ClientRequestID),
	}, &out); err != nil {
		return nil, err
	}
	return out.toBiz(), nil
}

// StreamChat opens the upstream POST SSE response and returns its body for pass-through.
func (r *aiChatRepo) StreamChat(ctx context.Context, req *biz.AIChatRequest) (*biz.AIChatStream, error) {
	body, err := json.Marshal(chatRequestPO{
		ConversationID:  req.ConversationID,
		Message:         req.Message,
		SystemPrompt:    req.SystemPrompt,
		ClientRequestID: optionalStringPO(req.ClientRequestID),
	})
	if err != nil {
		return nil, kerrors.BadRequest("INVALID_REQUEST", err.Error())
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/api/v1/chat/stream", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	requestTraceparent := traceparentForRequest(req.Traceparent)
	httpReq.Header.Set("traceparent", requestTraceparent)
	r.setInternalIdentity(httpReq, req.UpstreamUserID)

	resp, err := r.streamClient.Do(httpReq)
	if err != nil {
		return nil, kerrors.New(http.StatusBadGateway, "AI_SERVICE_UNAVAILABLE", "AI service unavailable")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		return nil, r.decodeUpstreamError(resp)
	}
	responseTraceparent := traceparentForRequest(resp.Header.Get("traceparent"))
	if resp.Header.Get("traceparent") == "" {
		responseTraceparent = requestTraceparent
	}
	return &biz.AIChatStream{
		StatusCode:          resp.StatusCode,
		Body:                resp.Body,
		Traceparent:         responseTraceparent,
		IdempotencyReplayed: strings.EqualFold(resp.Header.Get("Idempotency-Replayed"), "true"),
	}, nil
}

// CreateKnowledgeDocument forwards document indexing to FastAPI.
func (r *aiChatRepo) CreateKnowledgeDocument(ctx context.Context, req *biz.AICreateKnowledgeDocument) (*biz.AICreateKnowledgeDocumentReply, error) {
	var out createKnowledgeDocumentReplyPO
	err := r.doJSON(ctx, http.MethodPost, "/api/v1/knowledge/documents", req.UpstreamUserID, createKnowledgeDocumentPO{
		Title:           req.Title,
		Content:         req.Content,
		Source:          req.Source,
		ObjectReference: req.ObjectReference,
		Metadata:        jsonObjectPO(req.MetadataJSON),
	}, &out)
	if err != nil {
		return nil, err
	}
	return &biz.AICreateKnowledgeDocumentReply{ID: out.ID, Status: out.Status, JobID: out.JobID}, nil
}

// ListKnowledgeDocuments forwards the knowledge document list request.
func (r *aiChatRepo) ListKnowledgeDocuments(ctx context.Context, userID string, options biz.AIKnowledgeListOptions) (*biz.AIKnowledgeDocumentSet, error) {
	var out knowledgeDocumentSetPO
	query := url.Values{}
	if options.Page > 0 {
		query.Set("page", strconv.Itoa(int(options.Page)))
	}
	if options.PageSize > 0 {
		query.Set("page_size", strconv.Itoa(int(options.PageSize)))
	}
	setOptionalQuery(query, "status", options.Status)
	setOptionalQuery(query, "query", options.Query)
	setOptionalQuery(query, "cursor", options.Cursor)
	path := "/api/v1/knowledge/documents"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	if err := r.doJSON(ctx, http.MethodGet, path, userID, nil, &out); err != nil {
		return nil, err
	}
	return out.toBiz(), nil
}

func (r *aiChatRepo) GetKnowledgeDocument(ctx context.Context, userID, documentID string) (*biz.AIKnowledgeDocument, error) {
	var out knowledgeDocumentPO
	path := "/api/v1/knowledge/documents/" + url.PathEscape(documentID)
	if err := r.doJSON(ctx, http.MethodGet, path, userID, nil, &out); err != nil {
		return nil, err
	}
	return out.toBiz(), nil
}

func (r *aiChatRepo) UpdateKnowledgeDocument(ctx context.Context, req *biz.AIUpdateKnowledgeDocument) (*biz.AIKnowledgeDocument, error) {
	var out knowledgeDocumentPO
	path := "/api/v1/knowledge/documents/" + url.PathEscape(req.DocumentID)
	payload := updateKnowledgeDocumentPO{Title: req.Title, Source: req.Source}
	if req.MetadataJSON != nil {
		metadata := jsonObjectPO(*req.MetadataJSON)
		payload.Metadata = &metadata
	}
	if err := r.doJSON(ctx, http.MethodPatch, path, req.UpstreamUserID, payload, &out); err != nil {
		return nil, err
	}
	return out.toBiz(), nil
}

func (r *aiChatRepo) DeleteKnowledgeDocument(ctx context.Context, userID, documentID string) error {
	path := "/api/v1/knowledge/documents/" + url.PathEscape(documentID)
	return r.doJSON(ctx, http.MethodDelete, path, userID, nil, nil)
}

func (r *aiChatRepo) ReindexKnowledgeDocument(ctx context.Context, userID, documentID string) (*biz.AIReindexKnowledgeDocumentReply, error) {
	var out reindexKnowledgeDocumentReplyPO
	path := "/api/v1/knowledge/documents/" + url.PathEscape(documentID) + ":reindex"
	if err := r.doJSON(ctx, http.MethodPost, path, userID, map[string]any{}, &out); err != nil {
		return nil, err
	}
	return &biz.AIReindexKnowledgeDocumentReply{JobID: out.JobID, Status: out.Status}, nil
}

func (r *aiChatRepo) GetKnowledgeJob(ctx context.Context, userID, jobID string) (*biz.AIKnowledgeJob, error) {
	var out knowledgeJobPO
	path := "/api/v1/knowledge/jobs/" + url.PathEscape(jobID)
	if err := r.doJSON(ctx, http.MethodGet, path, userID, nil, &out); err != nil {
		return nil, err
	}
	return out.toBiz(), nil
}

// doJSON centralizes normal JSON proxy calls and error mapping.
func (r *aiChatRepo) doJSON(ctx context.Context, method, path, userID string, in any, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return kerrors.BadRequest("INVALID_REQUEST", err.Error())
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("traceparent", traceparentFromContext(ctx))
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	r.setInternalIdentity(req, userID)

	resp, err := r.client.Do(req)
	if err != nil {
		return kerrors.New(http.StatusBadGateway, "AI_SERVICE_UNAVAILABLE", "AI service unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return r.decodeUpstreamError(resp)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return kerrors.New(http.StatusBadGateway, "AI_SERVICE_BAD_RESPONSE", "AI service returned invalid JSON")
	}
	return nil
}

// setInternalIdentity prevents a browser supplied user ID from becoming a
// trusted upstream identity. The model service verifies this short-lived HMAC
// assertion and rejects direct traffic lacking it.
func (r *aiChatRepo) setInternalIdentity(req *http.Request, userID string) {
	if userID == "" {
		return
	}
	if len(r.identitySecret) == 0 {
		req.Header.Set("X-User-Id", userID)
		return
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	payload := userID + "." + timestamp
	mac := hmac.New(sha256.New, r.identitySecret)
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	req.Header.Set("X-Internal-User-Assertion", payload+"."+signature)
}

// decodeUpstreamError maps FastAPI error responses into Kratos typed errors.
func (r *aiChatRepo) decodeUpstreamError(resp *http.Response) error {
	var payload struct {
		Detail any `json:"detail"`
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	detail := strings.TrimSpace(string(data))
	if len(data) > 0 && json.Unmarshal(data, &payload) == nil && payload.Detail != nil {
		switch v := payload.Detail.(type) {
		case string:
			detail = v
		default:
			if b, err := json.Marshal(v); err == nil {
				detail = string(b)
			}
		}
	}
	if detail == "" {
		detail = http.StatusText(resp.StatusCode)
	}
	switch resp.StatusCode {
	case http.StatusBadRequest:
		return kerrors.BadRequest("AI_BAD_REQUEST", detail)
	case http.StatusUnauthorized:
		return kerrors.Unauthorized("AI_UNAUTHORIZED", detail)
	case http.StatusForbidden:
		return kerrors.Forbidden("AI_FORBIDDEN", detail)
	case http.StatusNotFound:
		return kerrors.NotFound("AI_NOT_FOUND", detail)
	case http.StatusConflict:
		return kerrors.Conflict("AI_IDEMPOTENCY_CONFLICT", detail)
	case http.StatusTooManyRequests:
		return kerrors.New(http.StatusTooManyRequests, "AI_RATE_LIMITED", detail)
	case http.StatusUnprocessableEntity:
		return kerrors.BadRequest("AI_VALIDATION_FAILED", detail)
	default:
		return kerrors.New(http.StatusBadGateway, "AI_SERVICE_ERROR", detail)
	}
}

// conversationPO is the downstream persistent/transport shape owned by data.
type conversationPO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// toBiz converts a downstream conversation into the biz domain shape.
func (p conversationPO) toBiz() *biz.AIConversation {
	return &biz.AIConversation{
		ID:        p.ID,
		Title:     p.Title,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// conversationSetPO matches FastAPI list responses.
type conversationSetPO struct {
	Items []conversationPO `json:"items"`
}

// toBiz converts a downstream conversation list into domain objects.
func (p conversationSetPO) toBiz() []*biz.AIConversation {
	out := make([]*biz.AIConversation, 0, len(p.Items))
	for _, item := range p.Items {
		out = append(out, item.toBiz())
	}
	return out
}

// messagePO is the downstream message shape owned by data.
type messagePO struct {
	ID                string  `json:"id"`
	ConversationID    string  `json:"conversation_id"`
	Role              string  `json:"role"`
	Content           string  `json:"content"`
	Sequence          int32   `json:"sequence"`
	ModelName         *string `json:"model_name"`
	ProviderRequestID *string `json:"provider_request_id"`
	RequestID         *string `json:"request_id"`
	ClientRequestID   *string `json:"client_request_id"`
	TurnStatus        string  `json:"turn_status"`
	ReferencesJSON    *string `json:"references_json"`
	UsageJSON         *string `json:"usage_json"`
	CreatedAt         string  `json:"created_at"`
}

// toBiz converts a downstream message into the biz domain shape.
func (p messagePO) toBiz() *biz.AIMessage {
	return &biz.AIMessage{
		ID:                p.ID,
		ConversationID:    p.ConversationID,
		Role:              p.Role,
		Content:           p.Content,
		Sequence:          p.Sequence,
		ModelName:         p.ModelName,
		ProviderRequestID: p.ProviderRequestID,
		RequestID:         p.RequestID,
		ClientRequestID:   p.ClientRequestID,
		TurnStatus:        p.TurnStatus,
		ReferencesJSON:    stringValuePO(p.ReferencesJSON, "[]"),
		UsageJSON:         stringValuePO(p.UsageJSON, "{}"),
		CreatedAt:         p.CreatedAt,
	}
}

// messageSetPO matches FastAPI message list responses.
type messageSetPO struct {
	Items []messagePO `json:"items"`
}

// toBiz converts a downstream message list into domain objects.
func (p messageSetPO) toBiz() []*biz.AIMessage {
	out := make([]*biz.AIMessage, 0, len(p.Items))
	for _, item := range p.Items {
		out = append(out, item.toBiz())
	}
	return out
}

// chatRequestPO is the JSON payload expected by FastAPI chat endpoints.
type chatRequestPO struct {
	ConversationID  *string `json:"conversation_id,omitempty"`
	Message         string  `json:"message"`
	SystemPrompt    *string `json:"system_prompt,omitempty"`
	ClientRequestID *string `json:"client_request_id,omitempty"`
}

// chatReplyPO is the full non-streaming chat response from FastAPI.
type chatReplyPO struct {
	Conversation     conversationPO `json:"conversation"`
	UserMessage      messagePO      `json:"user_message"`
	AssistantMessage messagePO      `json:"assistant_message"`
}

// toBiz converts a full non-streaming chat response into domain objects.
func (p chatReplyPO) toBiz() *biz.AIChatReply {
	return &biz.AIChatReply{
		Conversation:     p.Conversation.toBiz(),
		UserMessage:      p.UserMessage.toBiz(),
		AssistantMessage: p.AssistantMessage.toBiz(),
	}
}

// knowledgeDocumentPO is the downstream knowledge document summary shape.
type knowledgeDocumentPO struct {
	ID                 string             `json:"id"`
	Title              string             `json:"title"`
	Source             *string            `json:"source"`
	ChunkCount         int32              `json:"chunk_count"`
	CreatedAt          string             `json:"created_at"`
	Status             string             `json:"status"`
	Progress           int32              `json:"progress"`
	ErrorCode          *string            `json:"error_code"`
	ErrorDetail        *string            `json:"error_detail"`
	IndexVersion       int32              `json:"index_version"`
	EmbeddingModel     *string            `json:"embedding_model"`
	EmbeddingDimension int32              `json:"embedding_dimension"`
	UpdatedAt          string             `json:"updated_at"`
	Metadata           map[string]any     `json:"metadata"`
	Preview            *string            `json:"preview"`
	Chunks             []knowledgeChunkPO `json:"chunks"`
}

// toBiz converts a downstream knowledge document into the biz domain shape.
func (p knowledgeDocumentPO) toBiz() *biz.AIKnowledgeDocument {
	chunks := make([]*biz.AIKnowledgeChunk, 0, len(p.Chunks))
	for _, chunk := range p.Chunks {
		chunks = append(chunks, &biz.AIKnowledgeChunk{ID: chunk.ID, Content: chunk.Content, Sequence: chunk.Sequence})
	}
	return &biz.AIKnowledgeDocument{
		ID:                 p.ID,
		Title:              p.Title,
		Source:             p.Source,
		ChunkCount:         p.ChunkCount,
		CreatedAt:          p.CreatedAt,
		Status:             p.Status,
		Progress:           p.Progress,
		ErrorCode:          p.ErrorCode,
		ErrorDetail:        p.ErrorDetail,
		IndexVersion:       p.IndexVersion,
		EmbeddingModel:     p.EmbeddingModel,
		EmbeddingDimension: p.EmbeddingDimension,
		UpdatedAt:          p.UpdatedAt,
		MetadataJSON:       marshalJSONPO(p.Metadata, "{}"),
		Preview:            p.Preview,
		Chunks:             chunks,
	}
}

type knowledgeChunkPO struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Sequence int32  `json:"sequence"`
}

// knowledgeDocumentSetPO matches FastAPI knowledge document list responses.
type knowledgeDocumentSetPO struct {
	Items      []knowledgeDocumentPO `json:"items"`
	Total      int64                 `json:"total"`
	NextCursor *string               `json:"next_cursor"`
	Page       int32                 `json:"page"`
	PageSize   int32                 `json:"page_size"`
}

// toBiz converts a downstream knowledge document list into domain objects.
func (p knowledgeDocumentSetPO) toBiz() *biz.AIKnowledgeDocumentSet {
	out := make([]*biz.AIKnowledgeDocument, 0, len(p.Items))
	for _, item := range p.Items {
		out = append(out, item.toBiz())
	}
	return &biz.AIKnowledgeDocumentSet{
		Items: out, Total: p.Total, NextCursor: p.NextCursor, Page: p.Page, PageSize: p.PageSize,
	}
}

// createKnowledgeDocumentPO is the JSON payload expected by FastAPI indexing.
type createKnowledgeDocumentPO struct {
	Title           string         `json:"title"`
	Content         *string        `json:"content,omitempty"`
	Source          *string        `json:"source,omitempty"`
	ObjectReference *string        `json:"object_reference,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type createKnowledgeDocumentReplyPO struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	JobID  string `json:"job_id"`
}

type updateKnowledgeDocumentPO struct {
	Title    *string         `json:"title,omitempty"`
	Source   *string         `json:"source,omitempty"`
	Metadata *map[string]any `json:"metadata,omitempty"`
}

type reindexKnowledgeDocumentReplyPO struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type knowledgeJobPO struct {
	ID                 string  `json:"id"`
	DocumentID         string  `json:"document_id"`
	Kind               string  `json:"kind"`
	Status             string  `json:"status"`
	Progress           int32   `json:"progress"`
	TargetIndexVersion int32   `json:"target_index_version"`
	ErrorCode          *string `json:"error_code"`
	ErrorDetail        *string `json:"error_detail"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

func (p knowledgeJobPO) toBiz() *biz.AIKnowledgeJob {
	return &biz.AIKnowledgeJob{
		ID: p.ID, DocumentID: p.DocumentID, Kind: p.Kind, Status: p.Status,
		Progress: p.Progress, TargetIndexVersion: p.TargetIndexVersion,
		ErrorCode: p.ErrorCode, ErrorDetail: p.ErrorDetail,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func optionalStringPO(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func stringValuePO(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return *value
}

func jsonObjectPO(value string) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal([]byte(value), &out)
	return out
}

func marshalJSONPO(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fallback
	}
	return string(encoded)
}

func setOptionalQuery(values url.Values, key string, value *string) {
	if value != nil && strings.TrimSpace(*value) != "" {
		values.Set(key, strings.TrimSpace(*value))
	}
}

func traceparentFromContext(ctx context.Context) string {
	if tr, ok := transport.FromServerContext(ctx); ok {
		return traceparentForRequest(tr.RequestHeader().Get("traceparent"))
	}
	return traceparentForRequest("")
}

func traceparentForRequest(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	parts := strings.Split(value, "-")
	if len(parts) == 4 && len(parts[0]) == 2 && parts[0] != "ff" &&
		validTraceHex(parts[0], 2, false) && validTraceHex(parts[1], 32, true) &&
		validTraceHex(parts[2], 16, true) && validTraceHex(parts[3], 2, false) {
		return value
	}
	traceID := make([]byte, 16)
	parentID := make([]byte, 8)
	if _, err := rand.Read(traceID); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	if _, err := rand.Read(parentID); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return "00-" + hex.EncodeToString(traceID) + "-" + hex.EncodeToString(parentID) + "-01"
}

func validTraceHex(value string, length int, nonZero bool) bool {
	if len(value) != length {
		return false
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return false
	}
	if !nonZero {
		return true
	}
	for _, item := range decoded {
		if item != 0 {
			return true
		}
	}
	return false
}
