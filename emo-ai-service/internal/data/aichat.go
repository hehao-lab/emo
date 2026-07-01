package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"emo-ai-service/internal/biz"
	"emo-ai-service/internal/conf"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

// defaultAIServiceBaseURL keeps local development working when config is omitted.
const defaultAIServiceBaseURL = "http://127.0.0.1:8000"

// aiChatRepo implements biz.AIChatRepo by calling the downstream FastAPI service.
type aiChatRepo struct {
	baseURL string
	client  *http.Client
	// streamClient has no whole-request timeout because SSE duration is controlled by context cancellation.
	streamClient *http.Client
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
	return &aiChatRepo{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: protoDurationOrDefault(c.GetTimeout(), 120*time.Second),
		},
		streamClient: &http.Client{},
	}
}

// Health calls the FastAPI health endpoint.
func (r *aiChatRepo) Health(ctx context.Context) error {
	var out struct {
		Status string `json:"status"`
	}
	return r.doJSON(ctx, http.MethodGet, "/health", "", nil, &out)
}

// CreateConversation forwards conversation creation to FastAPI with X-User-Id.
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
		ConversationID: req.ConversationID,
		Message:        req.Message,
		SystemPrompt:   req.SystemPrompt,
	}, &out); err != nil {
		return nil, err
	}
	return out.toBiz(), nil
}

// StreamChat opens the upstream POST SSE response and returns its body for pass-through.
func (r *aiChatRepo) StreamChat(ctx context.Context, req *biz.AIChatRequest) (*biz.AIChatStream, error) {
	body, err := json.Marshal(chatRequestPO{
		ConversationID: req.ConversationID,
		Message:        req.Message,
		SystemPrompt:   req.SystemPrompt,
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
	httpReq.Header.Set("X-User-Id", req.UpstreamUserID)

	resp, err := r.streamClient.Do(httpReq)
	if err != nil {
		return nil, kerrors.New(http.StatusBadGateway, "AI_SERVICE_UNAVAILABLE", "AI service unavailable")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		return nil, r.decodeUpstreamError(resp)
	}
	return &biz.AIChatStream{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
	}, nil
}

// CreateKnowledgeDocument forwards document indexing to FastAPI.
func (r *aiChatRepo) CreateKnowledgeDocument(ctx context.Context, req *biz.AICreateKnowledgeDocument) (*biz.AIKnowledgeDocument, error) {
	var out knowledgeDocumentPO
	err := r.doJSON(ctx, http.MethodPost, "/api/v1/knowledge/documents", req.UpstreamUserID, createKnowledgeDocumentPO{
		Title:   req.Title,
		Content: req.Content,
		Source:  req.Source,
	}, &out)
	if err != nil {
		return nil, err
	}
	return out.toBiz(), nil
}

// ListKnowledgeDocuments forwards the knowledge document list request.
func (r *aiChatRepo) ListKnowledgeDocuments(ctx context.Context, userID string) ([]*biz.AIKnowledgeDocument, error) {
	var out knowledgeDocumentSetPO
	if err := r.doJSON(ctx, http.MethodGet, "/api/v1/knowledge/documents", userID, nil, &out); err != nil {
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
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if userID != "" {
		req.Header.Set("X-User-Id", userID)
	}

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
	case http.StatusNotFound:
		return kerrors.NotFound("AI_NOT_FOUND", detail)
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
	ID             string  `json:"id"`
	ConversationID string  `json:"conversation_id"`
	Role           string  `json:"role"`
	Content        string  `json:"content"`
	Sequence       int32   `json:"sequence"`
	ModelName      *string `json:"model_name"`
	CreatedAt      string  `json:"created_at"`
}

// toBiz converts a downstream message into the biz domain shape.
func (p messagePO) toBiz() *biz.AIMessage {
	return &biz.AIMessage{
		ID:             p.ID,
		ConversationID: p.ConversationID,
		Role:           p.Role,
		Content:        p.Content,
		Sequence:       p.Sequence,
		ModelName:      p.ModelName,
		CreatedAt:      p.CreatedAt,
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
	ConversationID *string `json:"conversation_id,omitempty"`
	Message        string  `json:"message"`
	SystemPrompt   *string `json:"system_prompt,omitempty"`
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
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Source     *string `json:"source"`
	ChunkCount int32   `json:"chunk_count"`
	CreatedAt  string  `json:"created_at"`
}

// toBiz converts a downstream knowledge document into the biz domain shape.
func (p knowledgeDocumentPO) toBiz() *biz.AIKnowledgeDocument {
	return &biz.AIKnowledgeDocument{
		ID:         p.ID,
		Title:      p.Title,
		Source:     p.Source,
		ChunkCount: p.ChunkCount,
		CreatedAt:  p.CreatedAt,
	}
}

// knowledgeDocumentSetPO matches FastAPI knowledge document list responses.
type knowledgeDocumentSetPO struct {
	Items []knowledgeDocumentPO `json:"items"`
}

// toBiz converts a downstream knowledge document list into domain objects.
func (p knowledgeDocumentSetPO) toBiz() []*biz.AIKnowledgeDocument {
	out := make([]*biz.AIKnowledgeDocument, 0, len(p.Items))
	for _, item := range p.Items {
		out = append(out, item.toBiz())
	}
	return out
}

// createKnowledgeDocumentPO is the JSON payload expected by FastAPI indexing.
type createKnowledgeDocumentPO struct {
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Source  *string `json:"source,omitempty"`
}
