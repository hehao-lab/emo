package biz

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

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

// AIChatUsecase validates BFF-level input, proxies model work to FastAPI,
// and stores the frontend-facing chat history in the local chat tables.
type AIChatUsecase struct {
	repo     AIChatRepo
	chatRepo ChatRepo
}

// NewAIChatUsecase 构建 AI 聊天应用场景。
func NewAIChatUsecase(repo AIChatRepo, chatRepo ChatRepo) *AIChatUsecase {
	return &AIChatUsecase{repo: repo, chatRepo: chatRepo}
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
	if uc.chatRepo == nil {
		return uc.repo.Chat(ctx, req)
	}

	session, upstreamConversationID, err := uc.ensureLocalChatSession(ctx, req)
	if err != nil {
		return nil, err
	}
	userMessage, err := uc.chatRepo.CreateMessage(ctx, &ChatMessage{
		SessionID:   session.ID,
		UserID:      req.UserID,
		Role:        "user",
		Content:     strings.TrimSpace(req.Message),
		ContentType: "text",
		Status:      "success",
	})
	if err != nil {
		return nil, err
	}

	upstreamReq := *req
	upstreamReq.ConversationID = optionalString(upstreamConversationID)
	reply, err := uc.repo.Chat(ctx, &upstreamReq)
	if err != nil {
		return nil, err
	}

	assistantContent := ""
	modelName := ""
	if reply != nil && reply.AssistantMessage != nil {
		assistantContent = reply.AssistantMessage.Content
		modelName = stringValue(reply.AssistantMessage.ModelName)
	}
	assistantMessage, err := uc.chatRepo.CreateMessage(ctx, &ChatMessage{
		SessionID:           session.ID,
		UserID:              req.UserID,
		Role:                "assistant",
		Content:             assistantContent,
		ContentType:         "text",
		Model:               modelName,
		EmotionSnapshotJSON: "{}",
		SafetyResultJSON:    "{}",
		Status:              "success",
	})
	if err != nil {
		return nil, err
	}

	if reply != nil && reply.Conversation != nil {
		if err := uc.chatRepo.BindSessionUpstream(ctx, session.ID, reply.Conversation.ID); err != nil {
			return nil, err
		}
	}
	if err := uc.chatRepo.TouchSession(ctx, session.ID, time.Now(), 2); err != nil {
		return nil, err
	}

	return &AIChatReply{
		Conversation:     localAIConversation(session),
		UserMessage:      localAIMessage(userMessage),
		AssistantMessage: localAIMessage(assistantMessage),
	}, nil
}

// StreamChat 打开上游 SSE 响应体，保存本地聊天记录，并把前端可见 ID 改成本地数据库 ID。
func (uc *AIChatUsecase) StreamChat(ctx context.Context, req *AIChatRequest) (*AIChatStream, error) {
	if err := validateAIChatRequest(req); err != nil {
		return nil, err
	}
	if uc.chatRepo == nil {
		return uc.repo.StreamChat(ctx, req)
	}

	session, upstreamConversationID, err := uc.ensureLocalChatSession(ctx, req)
	if err != nil {
		return nil, err
	}
	if _, err := uc.chatRepo.CreateMessage(ctx, &ChatMessage{
		SessionID:   session.ID,
		UserID:      req.UserID,
		Role:        "user",
		Content:     strings.TrimSpace(req.Message),
		ContentType: "text",
		Status:      "success",
	}); err != nil {
		return nil, err
	}

	upstreamReq := *req
	upstreamReq.ConversationID = optionalString(upstreamConversationID)
	stream, err := uc.repo.StreamChat(ctx, &upstreamReq)
	if err != nil {
		return nil, err
	}

	reader, writer := io.Pipe()
	go uc.copyAndPersistSSE(ctx, stream.Body, writer, session)

	return &AIChatStream{
		StatusCode: stream.StatusCode,
		Body:       reader,
	}, nil
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

func (uc *AIChatUsecase) ensureLocalChatSession(ctx context.Context, req *AIChatRequest) (*ChatSession, string, error) {
	rawConversationID := strings.TrimSpace(stringValue(req.ConversationID))
	if rawConversationID == "" {
		session, err := uc.chatRepo.CreateSession(ctx, &ChatSession{
			UserID:   req.UserID,
			Title:    chatTitleFromMessage(req.Message),
			Scenario: "emotional_support",
			Status:   "active",
		})
		return session, "", err
	}

	localSessionID, err := strconv.ParseInt(rawConversationID, 10, 64)
	if err != nil {
		session, createErr := uc.chatRepo.CreateSession(ctx, &ChatSession{
			UserID:                 req.UserID,
			Title:                  chatTitleFromMessage(req.Message),
			Scenario:               "emotional_support",
			Status:                 "active",
			UpstreamConversationID: rawConversationID,
		})
		return session, rawConversationID, createErr
	}

	session, err := uc.chatRepo.GetSession(ctx, req.UserID, localSessionID)
	if err != nil {
		return nil, "", err
	}
	if session == nil {
		return nil, "", kerrors.NotFound("CHAT_SESSION_NOT_FOUND", "chat session not found")
	}
	return session, session.UpstreamConversationID, nil
}

func (uc *AIChatUsecase) copyAndPersistSSE(ctx context.Context, source io.ReadCloser, writer *io.PipeWriter, session *ChatSession) {
	defer source.Close()

	state := &ssePersistState{
		ctx:     ctx,
		uc:      uc,
		session: session,
	}
	reader := bufio.NewReader(source)
	var eventLines []string

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if isSSEBlankLine(line) {
				if err := state.writeEvent(writer, eventLines); err != nil {
					_ = writer.CloseWithError(err)
					return
				}
				eventLines = nil
			} else {
				eventLines = append(eventLines, line)
			}
		}
		if err == io.EOF {
			if len(eventLines) > 0 {
				if err := state.writeEvent(writer, eventLines); err != nil {
					_ = writer.CloseWithError(err)
					return
				}
			}
			_ = writer.Close()
			return
		}
		if err != nil {
			_ = writer.CloseWithError(err)
			return
		}
	}
}

type ssePersistState struct {
	ctx      context.Context
	uc       *AIChatUsecase
	session  *ChatSession
	content  strings.Builder
}

func (s *ssePersistState) writeEvent(writer io.Writer, lines []string) error {
	if len(lines) == 0 {
		_, err := io.WriteString(writer, "\n")
		return err
	}

	eventName, dataText := parseSSEEvent(lines)
	if eventName == "delta" {
		s.content.WriteString(readSSEContent(dataText))
		return writeRawSSEEvent(writer, lines)
	}
	if eventName != "done" {
		return writeRawSSEEvent(writer, lines)
	}

	payload := map[string]any{}
	if err := json.Unmarshal([]byte(dataText), &payload); err != nil {
		return writeRawSSEEvent(writer, lines)
	}

	assistantContent := stringFromPayload(payload, "content")
	if assistantContent == "" {
		assistantContent = s.content.String()
	}
	assistantMessage, err := s.uc.chatRepo.CreateMessage(s.ctx, &ChatMessage{
		SessionID:           s.session.ID,
		UserID:              s.session.UserID,
		Role:                "assistant",
		Content:             assistantContent,
		ContentType:         "text",
		Model:               stringFromPayload(payload, "model_name"),
		EmotionSnapshotJSON: "{}",
		SafetyResultJSON:    "{}",
		Status:              "success",
	})
	if err != nil {
		return err
	}

	upstreamConversationID := stringFromPayload(payload, "conversation_id")
	if upstreamConversationID != "" {
		if err := s.uc.chatRepo.BindSessionUpstream(s.ctx, s.session.ID, upstreamConversationID); err != nil {
			return err
		}
	}
	if err := s.uc.chatRepo.TouchSession(s.ctx, s.session.ID, time.Now(), 2); err != nil {
		return err
	}

	payload["conversation_id"] = strconv.FormatInt(s.session.ID, 10)
	payload["assistant_message_id"] = strconv.FormatInt(assistantMessage.ID, 10)
	payload["content"] = assistantContent

	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte("event: done\ndata: " + string(encoded) + "\n\n"))
	return err
}

func parseSSEEvent(lines []string) (string, string) {
	var eventName string
	var dataLines []string

	for _, line := range lines {
		normalized := strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(normalized, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(normalized, "event:"))
			continue
		}
		if strings.HasPrefix(normalized, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(normalized, "data:")))
		}
	}

	return eventName, strings.Join(dataLines, "\n")
}

func writeRawSSEEvent(writer io.Writer, lines []string) error {
	for _, line := range lines {
		if _, err := io.WriteString(writer, line); err != nil {
			return err
		}
	}
	_, err := io.WriteString(writer, "\n")
	return err
}

func readSSEContent(dataText string) string {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(dataText), &payload); err != nil {
		return ""
	}
	return stringFromPayload(payload, "content")
}

func stringFromPayload(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typedValue := value.(type) {
	case string:
		return typedValue
	default:
		return strings.TrimSpace(fmt.Sprint(typedValue))
	}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func isSSEBlankLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

func chatTitleFromMessage(message string) string {
	title := strings.TrimSpace(message)
	if title == "" {
		return "新的情感咨询"
	}
	runes := []rune(title)
	if len(runes) > 24 {
		title = string(runes[:24])
	}
	return title
}

func localAIConversation(session *ChatSession) *AIConversation {
	if session == nil {
		return nil
	}
	return &AIConversation{
		ID:        strconv.FormatInt(session.ID, 10),
		Title:     session.Title,
		CreatedAt: formatUnixTime(session.CreatedAt),
		UpdatedAt: formatUnixTime(session.UpdatedAt),
	}
}

func localAIMessage(message *ChatMessage) *AIMessage {
	if message == nil {
		return nil
	}
	return &AIMessage{
		ID:             strconv.FormatInt(message.ID, 10),
		ConversationID: strconv.FormatInt(message.SessionID, 10),
		Role:           message.Role,
		Content:        message.Content,
		ModelName:      optionalString(message.Model),
		CreatedAt:      formatUnixTime(message.CreatedAt),
	}
}

func formatUnixTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
