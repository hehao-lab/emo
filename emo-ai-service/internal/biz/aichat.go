package biz

import (
	"context"
	"io"
	"strings"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

// AIConversation is the domain shape for a conversation returned by FastAPI.
type AIConversation struct {
	ID        string
	Title     string
	CreatedAt string
	UpdatedAt string
}

// AIMessage is the domain shape for a persisted chat message.
type AIMessage struct {
	ID             string
	ConversationID string
	Role           string
	Content        string
	Sequence       int32
	ModelName      *string
	CreatedAt      string
}

// AIKnowledgeDocument is the domain shape for a knowledge-base document summary.
type AIKnowledgeDocument struct {
	ID         string
	Title      string
	Source     *string
	ChunkCount int32
	CreatedAt  string
}

// AIChatRequest carries the authenticated user identity and chat input.
//
// UserID is the Kratos user identity. UpstreamUserID is the value sent to the
// FastAPI service as X-User-Id, which keeps the translation explicit.
type AIChatRequest struct {
	UserID         int64
	UpstreamUserID string
	ConversationID *string
	Message        string
	SystemPrompt   *string
}

// AIChatReply is the full response from the non-streaming chat endpoint.
type AIChatReply struct {
	Conversation     *AIConversation
	UserMessage      *AIMessage
	AssistantMessage *AIMessage
}

// AICreateKnowledgeDocument carries the text that FastAPI should index.
type AICreateKnowledgeDocument struct {
	UserID         int64
	UpstreamUserID string
	Title          string
	Content        string
	Source         *string
}

// AIChatStream owns the upstream SSE body returned by the data layer.
//
// Callers must close Body after they finish forwarding the stream.
type AIChatStream struct {
	StatusCode int
	Body       io.ReadCloser
}

// AIChatRepo is implemented by data and hides the downstream FastAPI protocol.
type AIChatRepo interface {
	Health(ctx context.Context) error
	CreateConversation(ctx context.Context, userID string, title string) (*AIConversation, error)
	ListConversations(ctx context.Context, userID string) ([]*AIConversation, error)
	ListMessages(ctx context.Context, userID string, conversationID string) ([]*AIMessage, error)
	Chat(ctx context.Context, req *AIChatRequest) (*AIChatReply, error)
	StreamChat(ctx context.Context, req *AIChatRequest) (*AIChatStream, error)
	CreateKnowledgeDocument(ctx context.Context, req *AICreateKnowledgeDocument) (*AIKnowledgeDocument, error)
	ListKnowledgeDocuments(ctx context.Context, userID string) ([]*AIKnowledgeDocument, error)
}

// AIChatUsecase validates BFF-level input and delegates persistence/model work to FastAPI.
type AIChatUsecase struct {
	repo AIChatRepo
}

// NewAIChatUsecase 构建 AI 聊天应用场景。
func NewAIChatUsecase(repo AIChatRepo) *AIChatUsecase {
	return &AIChatUsecase{repo: repo}
}

// Health 检查下游 FastAPI 服务是否可达。
func (uc *AIChatUsecase) Health(ctx context.Context) error {
	return uc.repo.Health(ctx)
}

// CreateConversation 在转发给 FastAPI 之前验证标题。
func (uc *AIChatUsecase) CreateConversation(ctx context.Context, userID int64, upstreamUserID, title string) (*AIConversation, error) {
	if strings.TrimSpace(title) == "" {
		return nil, kerrors.BadRequest("INVALID_TITLE", "title is required")
	}
	if len([]rune(title)) > 200 {
		return nil, kerrors.BadRequest("INVALID_TITLE", "title is too long")
	}
	return uc.repo.CreateConversation(ctx, upstreamUserID, title)
}

// ListConversations 返回已认证用户的对话。
func (uc *AIChatUsecase) ListConversations(ctx context.Context, userID int64, upstreamUserID string) ([]*AIConversation, error) {
	return uc.repo.ListConversations(ctx, upstreamUserID)
}

// ListMessages 在转发给 FastAPI 之前验证会话 ID。
func (uc *AIChatUsecase) ListMessages(ctx context.Context, userID int64, upstreamUserID, conversationID string) ([]*AIMessage, error) {
	if strings.TrimSpace(conversationID) == "" {
		return nil, kerrors.BadRequest("INVALID_CONVERSATION_ID", "conversation_id is required")
	}
	return uc.repo.ListMessages(ctx, upstreamUserID, conversationID)
}

// Chat 转发非流式聊天请求。
func (uc *AIChatUsecase) Chat(ctx context.Context, req *AIChatRequest) (*AIChatReply, error) {
	if err := validateAIChatRequest(req); err != nil {
		return nil, err
	}
	return uc.repo.Chat(ctx, req)
}

// StreamChat 打开上游 SSE 响应体，供服务层转发。
func (uc *AIChatUsecase) StreamChat(ctx context.Context, req *AIChatRequest) (*AIChatStream, error) {
	if err := validateAIChatRequest(req); err != nil {
		return nil, err
	}
	return uc.repo.StreamChat(ctx, req)
}

// CreateKnowledgeDocument 在建立索引之前，验证上传文件的大小和元数据。
func (uc *AIChatUsecase) CreateKnowledgeDocument(ctx context.Context, req *AICreateKnowledgeDocument) (*AIKnowledgeDocument, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, kerrors.BadRequest("INVALID_TITLE", "title is required")
	}
	if strings.TrimSpace(req.Content) == "" {
		return nil, kerrors.BadRequest("INVALID_CONTENT", "content is required")
	}
	if len([]rune(req.Title)) > 200 {
		return nil, kerrors.BadRequest("INVALID_TITLE", "title is too long")
	}
	if len([]rune(req.Content)) > 200000 {
		return nil, kerrors.BadRequest("INVALID_CONTENT", "content is too long")
	}
	if req.Source != nil && len([]rune(*req.Source)) > 200 {
		return nil, kerrors.BadRequest("INVALID_SOURCE", "source is too long")
	}
	return uc.repo.CreateKnowledgeDocument(ctx, req)
}

// ListKnowledgeDocuments 返回用户已索引的知识文档。
func (uc *AIChatUsecase) ListKnowledgeDocuments(ctx context.Context, userID int64, upstreamUserID string) ([]*AIKnowledgeDocument, error) {
	return uc.repo.ListKnowledgeDocuments(ctx, upstreamUserID)
}

// validateAIChatRequest` 在进行代理之前，会强制执行面向前端的聊天限制。
func validateAIChatRequest(req *AIChatRequest) error {
	if req == nil {
		return kerrors.BadRequest("INVALID_REQUEST", "request is required")
	}
	if strings.TrimSpace(req.Message) == "" {
		return kerrors.BadRequest("INVALID_MESSAGE", "message is required")
	}
	if len([]rune(req.Message)) > 8000 {
		return kerrors.BadRequest("INVALID_MESSAGE", "message is too long")
	}
	if req.SystemPrompt != nil && len([]rune(*req.SystemPrompt)) > 4000 {
		return kerrors.BadRequest("INVALID_SYSTEM_PROMPT", "system_prompt is too long")
	}
	return nil
}
