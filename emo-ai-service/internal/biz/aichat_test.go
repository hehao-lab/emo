package biz

import (
	"context"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"
)

type mockAIChatRepo struct {
	stream *AIChatStream
}

func (m *mockAIChatRepo) Health(ctx context.Context) error { return nil }

func (m *mockAIChatRepo) CreateConversation(ctx context.Context, userID string, title string) (*AIConversation, error) {
	return nil, nil
}

func (m *mockAIChatRepo) ListConversations(ctx context.Context, userID string) ([]*AIConversation, error) {
	return nil, nil
}

func (m *mockAIChatRepo) ListMessages(ctx context.Context, userID string, conversationID string) ([]*AIMessage, error) {
	return nil, nil
}

func (m *mockAIChatRepo) Chat(ctx context.Context, req *AIChatRequest) (*AIChatReply, error) {
	return nil, nil
}

func (m *mockAIChatRepo) StreamChat(ctx context.Context, req *AIChatRequest) (*AIChatStream, error) {
	return m.stream, nil
}

func (m *mockAIChatRepo) CreateKnowledgeDocument(ctx context.Context, req *AICreateKnowledgeDocument) (*AICreateKnowledgeDocumentReply, error) {
	return nil, nil
}

func (m *mockAIChatRepo) ListKnowledgeDocuments(ctx context.Context, userID string, options AIKnowledgeListOptions) (*AIKnowledgeDocumentSet, error) {
	return nil, nil
}

func (m *mockAIChatRepo) GetKnowledgeDocument(context.Context, string, string) (*AIKnowledgeDocument, error) {
	return nil, nil
}

func (m *mockAIChatRepo) UpdateKnowledgeDocument(context.Context, *AIUpdateKnowledgeDocument) (*AIKnowledgeDocument, error) {
	return nil, nil
}

func (m *mockAIChatRepo) DeleteKnowledgeDocument(context.Context, string, string) error { return nil }

func (m *mockAIChatRepo) ReindexKnowledgeDocument(context.Context, string, string) (*AIReindexKnowledgeDocumentReply, error) {
	return nil, nil
}

func (m *mockAIChatRepo) GetKnowledgeJob(context.Context, string, string) (*AIKnowledgeJob, error) {
	return nil, nil
}

type mockChatRepo struct {
	nextSessionID int64
	nextMessageID int64
	sessions      map[int64]*ChatSession
	messages      []*ChatMessage
	touchedID     int64
	touchedDelta  int
	dailyUsage    *ChatDailyUsage
}

func newMockChatRepo() *mockChatRepo {
	return &mockChatRepo{
		nextSessionID: 1,
		nextMessageID: 1,
		sessions:      map[int64]*ChatSession{},
	}
}

func (m *mockChatRepo) CreateSession(ctx context.Context, session *ChatSession) (*ChatSession, error) {
	now := time.Now()
	out := *session
	out.ID = m.nextSessionID
	out.CreatedAt = now
	out.UpdatedAt = now
	m.nextSessionID++
	m.sessions[out.ID] = &out
	return &out, nil
}

func (m *mockChatRepo) ListSessions(ctx context.Context, userID int64, opt ChatListOption) ([]*ChatSession, int64, error) {
	return nil, 0, nil
}

func (m *mockChatRepo) GetSession(ctx context.Context, userID, id int64) (*ChatSession, error) {
	session := m.sessions[id]
	if session == nil || session.UserID != userID {
		return nil, nil
	}
	out := *session
	return &out, nil
}

func (m *mockChatRepo) UpdateSession(ctx context.Context, session *ChatSession) (*ChatSession, error) {
	return session, nil
}

func (m *mockChatRepo) DeleteSession(ctx context.Context, userID, id int64) error { return nil }

func (m *mockChatRepo) CreateMessage(ctx context.Context, message *ChatMessage) (*ChatMessage, error) {
	out := *message
	out.ID = m.nextMessageID
	out.CreatedAt = time.Now()
	m.nextMessageID++
	m.messages = append(m.messages, &out)
	return &out, nil
}

func (m *mockChatRepo) UpdateMessage(ctx context.Context, message *ChatMessage) (*ChatMessage, error) {
	for _, existing := range m.messages {
		if existing.ID != message.ID || existing.UserID != message.UserID {
			continue
		}
		existing.Content = message.Content
		existing.Model = message.Model
		existing.Status = message.Status
		existing.ErrorMessage = message.ErrorMessage
		existing.ReferencesJSON = message.ReferencesJSON
		existing.UsageJSON = message.UsageJSON
		existing.PromptTokens = message.PromptTokens
		existing.CompletionTokens = message.CompletionTokens
		existing.TotalTokens = message.TotalTokens
		existing.CachedTokens = message.CachedTokens
		existing.CostMicros = message.CostMicros
		existing.RequestID = message.RequestID
		existing.Provider = message.Provider
		existing.ProviderRequestID = message.ProviderRequestID
		out := *existing
		return &out, nil
	}
	return nil, nil
}

func (m *mockChatRepo) FindMessagesByIdempotencyKey(ctx context.Context, userID int64, idempotencyKey string) ([]*ChatMessage, error) {
	var out []*ChatMessage
	for _, message := range m.messages {
		if message.UserID == userID && message.IdempotencyKey != nil && *message.IdempotencyKey == idempotencyKey {
			copyMessage := *message
			out = append(out, &copyMessage)
		}
	}
	return out, nil
}

func (m *mockChatRepo) DailyUsage(context.Context, int64, time.Time) (*ChatDailyUsage, error) {
	if m.dailyUsage != nil {
		return m.dailyUsage, nil
	}
	return &ChatDailyUsage{}, nil
}

func (m *mockChatRepo) ListMessages(ctx context.Context, userID, sessionID int64, page, pageSize int32) ([]*ChatMessage, int64, error) {
	return nil, 0, nil
}

func (m *mockChatRepo) RecentMessages(ctx context.Context, sessionID int64, limit int) ([]*ChatMessage, error) {
	return nil, nil
}

func (m *mockChatRepo) TouchSession(ctx context.Context, sessionID int64, lastMessageAt time.Time, deltaCount int) error {
	m.touchedID = sessionID
	m.touchedDelta = deltaCount
	return nil
}

func (m *mockChatRepo) BindSessionUpstream(ctx context.Context, sessionID int64, upstreamConversationID string) error {
	m.sessions[sessionID].UpstreamConversationID = upstreamConversationID
	return nil
}

func (m *mockChatRepo) CreateFeedback(ctx context.Context, feedback *ChatFeedback) (*ChatFeedback, error) {
	return nil, nil
}

func (m *mockChatRepo) CreateSummary(ctx context.Context, summary *ChatContextSummary) (*ChatContextSummary, error) {
	return nil, nil
}

func TestAIChatUsecase_StreamChatPersistsMessages(t *testing.T) {
	upstreamBody := strings.Join([]string{
		`event: delta`,
		`data: {"content":"Hello "}`,
		``,
		`event: delta`,
		`data: {"content":"world"}`,
		``,
		`event: done`,
		`data: {"conversation_id":"upstream-session-1","assistant_message_id":"upstream-message-2","content":"Hello world","model_name":"gpt-test","provider":"openai","provider_request_id":"resp-1","request_id":"req-1","usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12,"cached_tokens":3,"cost_micros":55},"references":[{"key":"K1","document_id":"doc-1","title":"Guide","chunk_id":"chunk-1","snippet":"text","score":0.9}]}`,
		``,
	}, "\n")
	chatRepo := newMockChatRepo()
	uc := NewAIChatUsecase(&mockAIChatRepo{
		stream: &AIChatStream{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
	}, chatRepo)

	stream, err := uc.StreamChat(context.Background(), &AIChatRequest{
		UserID:          7,
		UpstreamUserID:  "7",
		Message:         "你好",
		ClientRequestID: "client-1",
		IdempotencyKey:  "turn-1",
	})
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}
	data, err := io.ReadAll(stream.Body)
	if err != nil {
		t.Fatalf("ReadAll(stream.Body) error = %v", err)
	}

	if len(chatRepo.messages) != 2 {
		t.Fatalf("stored messages = %d, want 2", len(chatRepo.messages))
	}
	if chatRepo.messages[0].Role != "user" || chatRepo.messages[0].Content != "你好" {
		t.Fatalf("user message = %#v", chatRepo.messages[0])
	}
	if chatRepo.messages[1].Role != "assistant" || chatRepo.messages[1].Content != "Hello world" {
		t.Fatalf("assistant message = %#v", chatRepo.messages[1])
	}
	if chatRepo.messages[1].TotalTokens != 12 || chatRepo.messages[1].CostMicros != 55 || chatRepo.messages[1].CachedTokens != 3 {
		t.Fatalf("assistant usage = %#v", chatRepo.messages[1])
	}
	if chatRepo.messages[1].Provider != "openai" || chatRepo.messages[1].ProviderRequestID != "resp-1" || chatRepo.messages[1].RequestID != "req-1" {
		t.Fatalf("assistant provider metadata = %#v", chatRepo.messages[1])
	}
	if got := chatRepo.sessions[1].UpstreamConversationID; got != "upstream-session-1" {
		t.Fatalf("upstream conversation id = %q, want upstream-session-1", got)
	}
	if chatRepo.touchedID != 1 || chatRepo.touchedDelta != 2 {
		t.Fatalf("TouchSession() = (%d, %d), want (1, 2)", chatRepo.touchedID, chatRepo.touchedDelta)
	}

	output := string(data)
	localSessionID := strconv.FormatInt(chatRepo.sessions[1].ID, 10)
	localAssistantID := strconv.FormatInt(chatRepo.messages[1].ID, 10)
	if !strings.Contains(output, `"conversation_id":"`+localSessionID+`"`) {
		t.Fatalf("stream output does not contain local conversation id %q: %s", localSessionID, output)
	}
	if !strings.Contains(output, `"assistant_message_id":"`+localAssistantID+`"`) {
		t.Fatalf("stream output does not contain local assistant id %q: %s", localAssistantID, output)
	}
	if strings.Contains(output, "upstream-session-1") || strings.Contains(output, "upstream-message-2") {
		t.Fatalf("stream output leaked upstream ids: %s", output)
	}
}

func TestAIChatUsecase_StreamChatReplaysCompletedTurn(t *testing.T) {
	body := "event: done\ndata: {\"conversation_id\":\"upstream-1\",\"content\":\"answer\",\"usage\":{\"total_tokens\":4}}\n\n"
	chatRepo := newMockChatRepo()
	uc := NewAIChatUsecase(&mockAIChatRepo{stream: &AIChatStream{
		StatusCode: 201,
		Body:       io.NopCloser(strings.NewReader(body)),
	}}, chatRepo)
	req := &AIChatRequest{
		UserID: 7, UpstreamUserID: "7", Message: "same question",
		ClientRequestID: "client-replay", IdempotencyKey: "turn-replay",
	}

	first, err := uc.StreamChat(context.Background(), req)
	if err != nil {
		t.Fatalf("first StreamChat() error = %v", err)
	}
	if _, err := io.ReadAll(first.Body); err != nil {
		t.Fatalf("read first stream: %v", err)
	}
	second, err := uc.StreamChat(context.Background(), req)
	if err != nil {
		t.Fatalf("replay StreamChat() error = %v", err)
	}
	if !second.IdempotencyReplayed {
		t.Fatal("replay did not set IdempotencyReplayed")
	}
	replayed, err := io.ReadAll(second.Body)
	if err != nil {
		t.Fatalf("read replay stream: %v", err)
	}
	if !strings.Contains(string(replayed), "event: done") || !strings.Contains(string(replayed), "answer") {
		t.Fatalf("replay body = %s", replayed)
	}
	if len(chatRepo.messages) != 2 {
		t.Fatalf("replay created duplicate messages: %d", len(chatRepo.messages))
	}

	conflicting := *req
	conflicting.Message = "different question"
	if _, err := uc.StreamChat(context.Background(), &conflicting); err == nil || !strings.Contains(err.Error(), "different payload") {
		t.Fatalf("payload conflict error = %v", err)
	}
}

func TestAIChatUsecase_StreamChatEnforcesDailyQuota(t *testing.T) {
	chatRepo := newMockChatRepo()
	chatRepo.dailyUsage = &ChatDailyUsage{TotalTokens: 100, CostMicros: 400}
	uc := NewAIChatUsecase(&mockAIChatRepo{}, chatRepo)
	uc.dailyTokenLimit = 100
	uc.dailyCostMicrosLimit = 1000

	_, err := uc.StreamChat(context.Background(), &AIChatRequest{
		UserID: 7, UpstreamUserID: "7", Message: "question",
		ClientRequestID: "quota-client", IdempotencyKey: "quota-turn",
	})
	if err == nil || !strings.Contains(err.Error(), "daily token limit") {
		t.Fatalf("quota error = %v", err)
	}
}
