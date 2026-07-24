package biz

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
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
	ProviderRequestID *string
	RequestID      *string
	ClientRequestID *string
	TurnStatus     string
	ReferencesJSON string
	UsageJSON      string
	CreatedAt      string
}

// AIKnowledgeDocument is the domain shape for a knowledge-base document summary.
type AIKnowledgeDocument struct {
	ID                 string
	Title              string
	Source             *string
	ChunkCount         int32
	CreatedAt          string
	Status             string
	Progress           int32
	ErrorCode          *string
	ErrorDetail        *string
	IndexVersion       int32
	EmbeddingModel     *string
	EmbeddingDimension int32
	UpdatedAt          string
	MetadataJSON       string
	Preview            *string
	Chunks             []*AIKnowledgeChunk
}

type AIKnowledgeChunk struct {
	ID       string
	Content  string
	Sequence int32
}

type AIKnowledgeJob struct {
	ID                 string
	DocumentID         string
	Kind               string
	Status             string
	Progress           int32
	TargetIndexVersion int32
	ErrorCode          *string
	ErrorDetail        *string
	CreatedAt          string
	UpdatedAt          string
}

type AIKnowledgeDocumentSet struct {
	Items      []*AIKnowledgeDocument
	Total      int64
	NextCursor *string
	Page       int32
	PageSize   int32
}

type AIKnowledgeListOptions struct {
	Page     int32
	PageSize int32
	Status   *string
	Query    *string
	Cursor   *string
}

// AIChatRequest carries the authenticated user identity and chat input.
//
// UserID is the Kratos user identity. UpstreamUserID is used only to construct
// the signed internal assertion accepted by the model service.
type AIChatRequest struct {
	UserID         int64
	UpstreamUserID string
	ConversationID *string
	Message        string
	SystemPrompt   *string
	ClientRequestID string
	IdempotencyKey string
	Traceparent    string
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
	Content        *string
	Source         *string
	ObjectReference *string
	MetadataJSON   string
}

type AICreateKnowledgeDocumentReply struct {
	ID     string
	Status string
	JobID  string
}

type AIUpdateKnowledgeDocument struct {
	UserID         int64
	UpstreamUserID string
	DocumentID     string
	Title          *string
	Source         *string
	MetadataJSON   *string
}

type AIReindexKnowledgeDocumentReply struct {
	JobID  string
	Status string
}

// AIChatStream owns the upstream SSE body returned by the data layer.
//
// Callers must close Body after they finish forwarding the stream.
type AIChatStream struct {
	StatusCode          int
	Body                io.ReadCloser
	Traceparent         string
	IdempotencyReplayed bool
}

// AIChatRepo is implemented by data and hides the downstream FastAPI protocol.
type AIChatRepo interface {
	Health(ctx context.Context) error
	CreateConversation(ctx context.Context, userID string, title string) (*AIConversation, error)
	ListConversations(ctx context.Context, userID string) ([]*AIConversation, error)
	ListMessages(ctx context.Context, userID string, conversationID string) ([]*AIMessage, error)
	Chat(ctx context.Context, req *AIChatRequest) (*AIChatReply, error)
	StreamChat(ctx context.Context, req *AIChatRequest) (*AIChatStream, error)
	CreateKnowledgeDocument(ctx context.Context, req *AICreateKnowledgeDocument) (*AICreateKnowledgeDocumentReply, error)
	ListKnowledgeDocuments(ctx context.Context, userID string, options AIKnowledgeListOptions) (*AIKnowledgeDocumentSet, error)
	GetKnowledgeDocument(ctx context.Context, userID, documentID string) (*AIKnowledgeDocument, error)
	UpdateKnowledgeDocument(ctx context.Context, req *AIUpdateKnowledgeDocument) (*AIKnowledgeDocument, error)
	DeleteKnowledgeDocument(ctx context.Context, userID, documentID string) error
	ReindexKnowledgeDocument(ctx context.Context, userID, documentID string) (*AIReindexKnowledgeDocumentReply, error)
	GetKnowledgeJob(ctx context.Context, userID, jobID string) (*AIKnowledgeJob, error)
}

// AIChatUsecase validates BFF-level input, proxies model work to FastAPI,
// and stores the frontend-facing chat history in the local chat tables.
type AIChatUsecase struct {
	repo     AIChatRepo
	chatRepo ChatRepo
	turnLocks sync.Map
	dailyTokenLimit int64
	dailyCostMicrosLimit int64
}

// NewAIChatUsecase 构建 AI 聊天应用场景。
func NewAIChatUsecase(repo AIChatRepo, chatRepo ChatRepo) *AIChatUsecase {
	return &AIChatUsecase{
		repo: repo,
		chatRepo: chatRepo,
		dailyTokenLimit: envInt64("EMO_AI_DAILY_TOKEN_LIMIT"),
		dailyCostMicrosLimit: envInt64("EMO_AI_DAILY_COST_MICROS_LIMIT"),
	}
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
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, kerrors.BadRequest("MISSING_IDEMPOTENCY_KEY", "Idempotency-Key is required")
	}
	if len(req.IdempotencyKey) > 128 {
		return nil, kerrors.BadRequest("INVALID_IDEMPOTENCY_KEY", "idempotency key is too long")
	}
	if uc.chatRepo == nil {
		return uc.repo.StreamChat(ctx, req)
	}
	if err := uc.enforceDailyQuota(ctx, req.UserID); err != nil {
		return nil, err
	}
	requestHash := hashAIChatRequest(req)
	requestLockKey := "request:" + strconv.FormatInt(req.UserID, 10) + ":" + req.IdempotencyKey
	if !uc.tryLock(requestLockKey) {
		return nil, kerrors.Conflict("CHAT_TURN_IN_PROGRESS", "this request is already being processed")
	}
	requestLocked := true
	releaseRequest := func() {
		if requestLocked {
			uc.unlock(requestLockKey)
			requestLocked = false
		}
	}

	existing, err := uc.chatRepo.FindMessagesByIdempotencyKey(ctx, req.UserID, req.IdempotencyKey)
	if err != nil {
		releaseRequest()
		return nil, err
	}
	if len(existing) > 0 {
		replay, replayErr := replayChatTurn(existing, requestHash)
		releaseRequest()
		return replay, replayErr
	}

	session, upstreamConversationID, err := uc.ensureLocalChatSession(ctx, req)
	if err != nil {
		releaseRequest()
		return nil, err
	}
	sessionLockKey := "session:" + strconv.FormatInt(session.ID, 10)
	if !uc.tryLock(sessionLockKey) {
		releaseRequest()
		return nil, kerrors.Conflict("CHAT_TURN_IN_PROGRESS", "wait for the active reply to finish")
	}
	releaseLocks := func() {
		uc.unlock(sessionLockKey)
		releaseRequest()
	}
	userMessage, err := uc.chatRepo.CreateMessage(ctx, &ChatMessage{
		SessionID:   session.ID,
		UserID:      req.UserID,
		Role:        "user",
		Content:     strings.TrimSpace(req.Message),
		ContentType: "text",
		Status:      "completed",
		ClientRequestID: optionalString(req.ClientRequestID),
		IdempotencyKey: optionalString(req.IdempotencyKey),
		RequestPayloadHash: requestHash,
	})
	if err != nil {
		releaseLocks()
		return nil, err
	}
	assistantMessage, err := uc.chatRepo.CreateMessage(ctx, &ChatMessage{
		SessionID:       session.ID,
		UserID:          req.UserID,
		Role:            "assistant",
		Content:         "",
		ContentType:     "text",
		Status:          "pending",
		ClientRequestID: optionalString(req.ClientRequestID),
		IdempotencyKey:  optionalString(req.IdempotencyKey),
		RequestPayloadHash: requestHash,
		ReferencesJSON:  "[]",
		UsageJSON:       "{}",
	})
	if err != nil {
		releaseLocks()
		return nil, err
	}

	upstreamReq := *req
	upstreamReq.ConversationID = optionalString(upstreamConversationID)
	stream, err := uc.repo.StreamChat(ctx, &upstreamReq)
	if err != nil {
		_, _ = uc.chatRepo.UpdateMessage(context.WithoutCancel(ctx), &ChatMessage{
			ID: assistantMessage.ID, UserID: req.UserID, Status: "failed", ErrorMessage: err.Error(), ReferencesJSON: "[]", UsageJSON: "{}",
		})
		_ = uc.chatRepo.TouchSession(context.WithoutCancel(ctx), session.ID, time.Now(), 2)
		releaseLocks()
		return nil, err
	}

	reader, writer := io.Pipe()
	go uc.copyAndPersistSSE(ctx, stream.Body, writer, session, assistantMessage, releaseLocks)
	_ = userMessage

	return &AIChatStream{
		StatusCode:          stream.StatusCode,
		Body:                reader,
		Traceparent:         stream.Traceparent,
		IdempotencyReplayed: stream.IdempotencyReplayed,
	}, nil
}

// CreateKnowledgeDocument 在建立索引之前，验证上传文件的大小和元数据。

func (uc *AIChatUsecase) CreateKnowledgeDocument(ctx context.Context, req *AICreateKnowledgeDocument) (*AICreateKnowledgeDocumentReply, error) {
	if req == nil {
		return nil, kerrors.BadRequest("INVALID_REQUEST", "request is required")
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, kerrors.BadRequest("INVALID_TITLE", "title is required")
	}
	content := strings.TrimSpace(stringValue(req.Content))
	objectReference := strings.TrimSpace(stringValue(req.ObjectReference))
	if content == "" && objectReference == "" {
		return nil, kerrors.BadRequest("INVALID_CONTENT", "content or object_reference is required")
	}
	if len([]rune(req.Title)) > 200 {
		return nil, kerrors.BadRequest("INVALID_TITLE", "title is too long")
	}
	if len([]rune(content)) > 200000 {
		return nil, kerrors.BadRequest("INVALID_CONTENT", "content is too long")
	}
	if req.Source != nil && len([]rune(*req.Source)) > 200 {
		return nil, kerrors.BadRequest("INVALID_SOURCE", "source is too long")
	}
	if objectReference != "" && len(objectReference) > 2048 {
		return nil, kerrors.BadRequest("INVALID_OBJECT_REFERENCE", "object_reference is too long")
	}
	if err := validateJSONObject(req.MetadataJSON); err != nil {
		return nil, err
	}
	return uc.repo.CreateKnowledgeDocument(ctx, req)
}

// ListKnowledgeDocuments 返回用户已索引的知识文档。
func (uc *AIChatUsecase) ListKnowledgeDocuments(ctx context.Context, userID int64, upstreamUserID string, options AIKnowledgeListOptions) (*AIKnowledgeDocumentSet, error) {
	if options.Page < 0 || options.PageSize < 0 || options.PageSize > 100 {
		return nil, kerrors.BadRequest("INVALID_PAGINATION", "invalid knowledge pagination")
	}
	return uc.repo.ListKnowledgeDocuments(ctx, upstreamUserID, options)
}

func (uc *AIChatUsecase) GetKnowledgeDocument(ctx context.Context, userID int64, upstreamUserID, documentID string) (*AIKnowledgeDocument, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, kerrors.BadRequest("INVALID_DOCUMENT_ID", "document_id is required")
	}
	return uc.repo.GetKnowledgeDocument(ctx, upstreamUserID, documentID)
}

func (uc *AIChatUsecase) UpdateKnowledgeDocument(ctx context.Context, req *AIUpdateKnowledgeDocument) (*AIKnowledgeDocument, error) {
	if req == nil || strings.TrimSpace(req.DocumentID) == "" {
		return nil, kerrors.BadRequest("INVALID_DOCUMENT_ID", "document_id is required")
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, kerrors.BadRequest("INVALID_TITLE", "title cannot be empty")
	}
	if req.Title != nil && len([]rune(*req.Title)) > 200 {
		return nil, kerrors.BadRequest("INVALID_TITLE", "title is too long")
	}
	if req.MetadataJSON != nil {
		if err := validateJSONObject(*req.MetadataJSON); err != nil {
			return nil, err
		}
	}
	return uc.repo.UpdateKnowledgeDocument(ctx, req)
}

func (uc *AIChatUsecase) DeleteKnowledgeDocument(ctx context.Context, userID int64, upstreamUserID, documentID string) error {
	if strings.TrimSpace(documentID) == "" {
		return kerrors.BadRequest("INVALID_DOCUMENT_ID", "document_id is required")
	}
	return uc.repo.DeleteKnowledgeDocument(ctx, upstreamUserID, documentID)
}

func (uc *AIChatUsecase) ReindexKnowledgeDocument(ctx context.Context, userID int64, upstreamUserID, documentID string) (*AIReindexKnowledgeDocumentReply, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, kerrors.BadRequest("INVALID_DOCUMENT_ID", "document_id is required")
	}
	return uc.repo.ReindexKnowledgeDocument(ctx, upstreamUserID, documentID)
}

func (uc *AIChatUsecase) GetKnowledgeJob(ctx context.Context, userID int64, upstreamUserID, jobID string) (*AIKnowledgeJob, error) {
	if strings.TrimSpace(jobID) == "" {
		return nil, kerrors.BadRequest("INVALID_JOB_ID", "job_id is required")
	}
	return uc.repo.GetKnowledgeJob(ctx, upstreamUserID, jobID)
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
	if len(req.ClientRequestID) > 128 {
		return kerrors.BadRequest("INVALID_CLIENT_REQUEST_ID", "client_request_id is too long")
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

func (uc *AIChatUsecase) copyAndPersistSSE(ctx context.Context, source io.ReadCloser, writer *io.PipeWriter, session *ChatSession, assistant *ChatMessage, release func()) {
	defer source.Close()
	defer release()

	state := &ssePersistState{
		ctx:       context.WithoutCancel(ctx),
		requestCtx: ctx,
		uc:        uc,
		session:   session,
		assistant: assistant,
	}
	defer state.finishIfNeeded()
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
	ctx        context.Context
	requestCtx context.Context
	uc         *AIChatUsecase
	session    *ChatSession
	assistant  *ChatMessage
	content    strings.Builder
	finished   bool
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
	if eventName == "error" {
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(dataText), &payload)
		status := "failed"
		if stringFromPayload(payload, "code") == "TURN_CANCELLED" {
			status = "cancelled"
		}
		detail := stringFromPayload(payload, "detail")
		if detail == "" {
			detail = "AI service stream failed"
		}
		_, err := s.uc.chatRepo.UpdateMessage(s.ctx, &ChatMessage{
			ID: s.assistant.ID, UserID: s.assistant.UserID, Status: status,
			ErrorMessage: detail, ReferencesJSON: "[]", UsageJSON: "{}",
		})
		if err != nil {
			return err
		}
		if err := s.uc.chatRepo.TouchSession(s.ctx, s.session.ID, time.Now(), 2); err != nil {
			return err
		}
		s.finished = true
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
	referencesJSON := referencesFromPayload(payload)
	usageJSON, usage := usageFromPayload(payload)
	turnStatus := stringFromPayload(payload, "turn_status")
	if turnStatus == "" {
		turnStatus = "completed"
	}
	assistantMessage, err := s.uc.chatRepo.UpdateMessage(s.ctx, &ChatMessage{
		ID:              s.assistant.ID,
		UserID:          s.assistant.UserID,
		Content:         assistantContent,
		Model:           stringFromPayload(payload, "model_name"),
		Status:          turnStatus,
		RequestID:       stringFromPayload(payload, "request_id"),
		Provider:        stringFromPayload(payload, "provider"),
		ProviderRequestID: stringFromPayload(payload, "provider_request_id"),
		ReferencesJSON:  referencesJSON,
		UsageJSON:       usageJSON,
		PromptTokens:    usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:     usage.TotalTokens,
		CachedTokens:    usage.CachedTokens,
		CostMicros:      usage.CostMicros,
		LatencyMS:       int32FromPayload(payload, "latency_ms"),
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
	if _, ok := payload["references"]; !ok {
		payload["references"] = json.RawMessage(referencesJSON)
	}
	s.finished = true

	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte("event: done\ndata: " + string(encoded) + "\n\n"))
	return err
}

func (s *ssePersistState) finishIfNeeded() {
	if s.finished || s.assistant == nil {
		return
	}
	status, detail := "failed", "AI service stream interrupted"
	if s.requestCtx.Err() != nil {
		status, detail = "cancelled", "generation cancelled"
	}
	_, _ = s.uc.chatRepo.UpdateMessage(s.ctx, &ChatMessage{
		ID: s.assistant.ID, UserID: s.assistant.UserID, Status: status,
		ErrorMessage: detail, ReferencesJSON: "[]", UsageJSON: "{}",
	})
	_ = s.uc.chatRepo.TouchSession(s.ctx, s.session.ID, time.Now(), 2)
}

func referencesFromPayload(payload map[string]any) string {
	references := payload["references"]
	if references == nil {
		references = payload["citations"]
	}
	if references == nil {
		return "[]"
	}
	encoded, err := json.Marshal(references)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func usageFromPayload(payload map[string]any) (string, ChatUsage) {
	usageMap, _ := payload["usage"].(map[string]any)
	if usageMap == nil {
		return "{}", ChatUsage{}
	}
	encoded, err := json.Marshal(usageMap)
	if err != nil {
		return "{}", ChatUsage{}
	}
	return string(encoded), ChatUsage{
		PromptTokens:     int32FromPayload(usageMap, "prompt_tokens"),
		CompletionTokens: int32FromPayload(usageMap, "completion_tokens"),
		TotalTokens:      int32FromPayload(usageMap, "total_tokens"),
		CachedTokens:     int32FromPayload(usageMap, "cached_tokens"),
		CostMicros:       int64FromPayload(usageMap, "cost_micros"),
	}
}

type ChatUsage struct {
	PromptTokens     int32
	CompletionTokens int32
	TotalTokens      int32
	CachedTokens     int32
	CostMicros       int64
}

func (uc *AIChatUsecase) tryLock(key string) bool {
	_, loaded := uc.turnLocks.LoadOrStore(key, struct{}{})
	return !loaded
}

func (uc *AIChatUsecase) unlock(key string) {
	uc.turnLocks.Delete(key)
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

func int32FromPayload(payload map[string]any, key string) int32 {
	return int32(int64FromPayload(payload, key))
}

func int64FromPayload(payload map[string]any, key string) int64 {
	value := payload[key]
	switch typedValue := value.(type) {
	case float64:
		return int64(typedValue)
	case float32:
		return int64(typedValue)
	case int:
		return int64(typedValue)
	case int32:
		return int64(typedValue)
	case int64:
		return typedValue
	case json.Number:
		parsed, _ := typedValue.Int64()
		return parsed
	case string:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(typedValue), 10, 64)
		return parsed
	default:
		return 0
	}
}

func (uc *AIChatUsecase) enforceDailyQuota(ctx context.Context, userID int64) error {
	if uc.chatRepo == nil || (uc.dailyTokenLimit <= 0 && uc.dailyCostMicrosLimit <= 0) {
		return nil
	}
	now := time.Now().UTC()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	usage, err := uc.chatRepo.DailyUsage(ctx, userID, since)
	if err != nil {
		return err
	}
	if uc.dailyTokenLimit > 0 && usage.TotalTokens >= uc.dailyTokenLimit {
		return kerrors.New(429, "DAILY_TOKEN_LIMIT_EXCEEDED", "daily token limit reached")
	}
	if uc.dailyCostMicrosLimit > 0 && usage.CostMicros >= uc.dailyCostMicrosLimit {
		return kerrors.New(429, "DAILY_COST_LIMIT_EXCEEDED", "daily cost limit reached")
	}
	return nil
}

func hashAIChatRequest(req *AIChatRequest) string {
	payload, _ := json.Marshal(struct {
		ConversationID  string `json:"conversation_id"`
		Message         string `json:"message"`
		SystemPrompt    string `json:"system_prompt"`
		ClientRequestID string `json:"client_request_id"`
	}{
		ConversationID:  strings.TrimSpace(stringValue(req.ConversationID)),
		Message:         strings.TrimSpace(req.Message),
		SystemPrompt:    stringValue(req.SystemPrompt),
		ClientRequestID: req.ClientRequestID,
	})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func replayChatTurn(messages []*ChatMessage, requestHash string) (*AIChatStream, error) {
	var assistant *ChatMessage
	for _, message := range messages {
		if message == nil {
			continue
		}
		if message.RequestPayloadHash != "" && message.RequestPayloadHash != requestHash {
			return nil, kerrors.Conflict("IDEMPOTENCY_PAYLOAD_CONFLICT", "Idempotency-Key was already used with a different payload")
		}
		if message.Role == "assistant" {
			assistant = message
		}
	}
	if assistant == nil || assistant.Status == "pending" || assistant.Status == "streaming" {
		return nil, kerrors.Conflict("CHAT_TURN_IN_PROGRESS", "this request is still being processed")
	}

	if assistant.Status == "completed" || assistant.Status == "success" || assistant.Status == "done" {
		payload := map[string]any{
			"conversation_id":      strconv.FormatInt(assistant.SessionID, 10),
			"assistant_message_id": strconv.FormatInt(assistant.ID, 10),
			"content":              assistant.Content,
			"model_name":           assistant.Model,
			"provider":             assistant.Provider,
			"provider_request_id":  assistant.ProviderRequestID,
			"request_id":           assistant.RequestID,
			"turn_status":          assistant.Status,
			"references":           rawJSONOrDefault(assistant.ReferencesJSON, "[]"),
			"usage":                rawJSONOrDefault(assistant.UsageJSON, "{}"),
		}
		encoded, _ := json.Marshal(payload)
		return &AIChatStream{
			StatusCode:          201,
			Body:                io.NopCloser(strings.NewReader("event: done\ndata: " + string(encoded) + "\n\n")),
			IdempotencyReplayed: true,
		}, nil
	}

	code := "TURN_FAILED"
	if assistant.Status == "cancelled" {
		code = "TURN_CANCELLED"
	}
	payload, _ := json.Marshal(map[string]any{
		"code": code, "detail": assistant.ErrorMessage, "retryable": assistant.Status == "failed",
	})
	return &AIChatStream{
		StatusCode:          201,
		Body:                io.NopCloser(strings.NewReader("event: error\ndata: " + string(payload) + "\n\n")),
		IdempotencyReplayed: true,
	}, nil
}

func rawJSONOrDefault(value, fallback string) json.RawMessage {
	value = strings.TrimSpace(value)
	if !json.Valid([]byte(value)) {
		value = fallback
	}
	return json.RawMessage(value)
}

func validateJSONObject(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var object map[string]any
	if err := json.Unmarshal([]byte(value), &object); err != nil || object == nil {
		return kerrors.BadRequest("INVALID_METADATA", "metadata_json must be a JSON object")
	}
	return nil
}

func envInt64(key string) int64 {
	value, _ := strconv.ParseInt(strings.TrimSpace(os.Getenv(key)), 10, 64)
	if value < 0 {
		return 0
	}
	return value
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
		ProviderRequestID: optionalString(message.ProviderRequestID),
		RequestID:      optionalString(message.RequestID),
		ClientRequestID: message.ClientRequestID,
		TurnStatus:     message.Status,
		ReferencesJSON: jsonArrayString(message.ReferencesJSON),
		UsageJSON:      jsonObjectString(message.UsageJSON),
		CreatedAt:      formatUnixTime(message.CreatedAt),
	}
}

func jsonArrayString(value string) string {
	if !json.Valid([]byte(value)) {
		return "[]"
	}
	return value
}

func jsonObjectString(value string) string {
	if !json.Valid([]byte(value)) {
		return "{}"
	}
	return value
}

func formatUnixTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
