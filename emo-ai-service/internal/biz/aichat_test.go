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

func (m *mockAIChatRepo) CreateKnowledgeDocument(ctx context.Context, req *AICreateKnowledgeDocument) (*AIKnowledgeDocument, error) {
	return nil, nil
}

func (m *mockAIChatRepo) ListKnowledgeDocuments(ctx context.Context, userID string) ([]*AIKnowledgeDocument, error) {
	return nil, nil
}

type mockChatRepo struct {
	nextSessionID int64
	nextMessageID int64
	sessions      map[int64]*ChatSession
	messages      []*ChatMessage
	touchedID     int64
	touchedDelta  int
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
		out := *existing
		return &out, nil
	}
	return nil, nil
}

func (m *mockChatRepo) FindMessagesByRequestID(ctx context.Context, userID int64, clientRequestID string) ([]*ChatMessage, error) {
	var out []*ChatMessage
	for _, message := range m.messages {
		if message.UserID == userID && message.ClientRequestID != nil && *message.ClientRequestID == clientRequestID {
			copyMessage := *message
			out = append(out, &copyMessage)
		}
	}
	return out, nil
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
		`data: {"conversation_id":"upstream-session-1","assistant_message_id":"upstream-message-2","content":"Hello world"}`,
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
		UserID:         7,
		UpstreamUserID: "7",
		Message:        "你好",
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
