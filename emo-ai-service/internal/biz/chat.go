package biz

import (
	"context"
	"time"
)

type ChatSession struct {
	ID            int64
	UserID        int64
	Title         string
	Scenario      string
	Status        string
	Summary       string
	MessageCount  int32
	LastMessageAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ChatMessage struct {
	ID                  int64
	SessionID           int64
	UserID              int64
	Role                string
	Content             string
	ContentType         string
	Model               string
	PromptTokens        int32
	CompletionTokens    int32
	TotalTokens         int32
	LatencyMS           int32
	EmotionSnapshotJSON string
	SafetyResultJSON    string
	Status              string
	ErrorMessage        string
	CreatedAt           time.Time
}

type ChatFeedback struct {
	ID           int64
	UserID       int64
	SessionID    int64
	MessageID    int64
	Rating       int32
	FeedbackType string
	Content      string
	CreatedAt    time.Time
}

type ChatContextSummary struct {
	ID             int64
	SessionID      int64
	Summary        string
	MessageStartID int64
	MessageEndID   int64
	Model          string
	CreatedAt      time.Time
}

type ChatListOption struct {
	Page     int32
	PageSize int32
	Status   string
}

type ChatRepo interface {
	CreateSession(ctx context.Context, session *ChatSession) (*ChatSession, error)
	ListSessions(ctx context.Context, userID int64, opt ChatListOption) ([]*ChatSession, int64, error)
	GetSession(ctx context.Context, userID, id int64) (*ChatSession, error)
	UpdateSession(ctx context.Context, session *ChatSession) (*ChatSession, error)
	DeleteSession(ctx context.Context, userID, id int64) error
	CreateMessage(ctx context.Context, message *ChatMessage) (*ChatMessage, error)
	ListMessages(ctx context.Context, userID, sessionID int64, page, pageSize int32) ([]*ChatMessage, int64, error)
	RecentMessages(ctx context.Context, sessionID int64, limit int) ([]*ChatMessage, error)
	TouchSession(ctx context.Context, sessionID int64, lastMessageAt time.Time, deltaCount int) error
	CreateFeedback(ctx context.Context, feedback *ChatFeedback) (*ChatFeedback, error)
	CreateSummary(ctx context.Context, summary *ChatContextSummary) (*ChatContextSummary, error)
}

type AIClient interface {
	Reply(ctx context.Context, req AIReplyRequest) (*AIReply, error)
	Summarize(ctx context.Context, messages []*ChatMessage) (*AIReply, error)
}

type AIReplyRequest struct {
	UserID  int64
	Session *ChatSession
	History []*ChatMessage
	Content string
}

type AIReply struct {
	Content          string
	Model            string
	PromptTokens     int32
	CompletionTokens int32
	LatencyMS        int32
	SafetyResultJSON string
}

type ChatUsecase struct {
	repo ChatRepo
	ai   AIClient
}

func NewChatUsecase(repo ChatRepo, ai AIClient) *ChatUsecase {
	return &ChatUsecase{repo: repo, ai: ai}
}

func (uc *ChatUsecase) CreateSession(ctx context.Context, session *ChatSession) (*ChatSession, error) {
	if session.Status == "" {
		session.Status = "active"
	}
	if session.Scenario == "" {
		session.Scenario = "emotional_support"
	}
	if session.Title == "" {
		session.Title = "新的情感咨询"
	}
	return uc.repo.CreateSession(ctx, session)
}

func (uc *ChatUsecase) ListSessions(ctx context.Context, userID int64, opt ChatListOption) ([]*ChatSession, int64, error) {
	return uc.repo.ListSessions(ctx, userID, opt)
}

func (uc *ChatUsecase) GetSession(ctx context.Context, userID, id int64) (*ChatSession, error) {
	return uc.repo.GetSession(ctx, userID, id)
}

func (uc *ChatUsecase) UpdateSession(ctx context.Context, session *ChatSession) (*ChatSession, error) {
	return uc.repo.UpdateSession(ctx, session)
}

func (uc *ChatUsecase) DeleteSession(ctx context.Context, userID, id int64) error {
	return uc.repo.DeleteSession(ctx, userID, id)
}

func (uc *ChatUsecase) SendMessage(ctx context.Context, userID, sessionID int64, content, contentType string) (*ChatMessage, *ChatMessage, error) {
	session, err := uc.repo.GetSession(ctx, userID, sessionID)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, ErrUserNotFound
	}
	if contentType == "" {
		contentType = "text"
	}
	userMsg, err := uc.repo.CreateMessage(ctx, &ChatMessage{
		SessionID:   sessionID,
		UserID:      userID,
		Role:        "user",
		Content:     content,
		ContentType: contentType,
		Status:      "success",
	})
	if err != nil {
		return nil, nil, err
	}
	history, _ := uc.repo.RecentMessages(ctx, sessionID, 20)
	reply, err := uc.ai.Reply(ctx, AIReplyRequest{UserID: userID, Session: session, History: history, Content: content})
	if err != nil {
		assistantMsg, saveErr := uc.repo.CreateMessage(ctx, &ChatMessage{
			SessionID:    sessionID,
			UserID:       userID,
			Role:         "assistant",
			Content:      "抱歉，我现在暂时无法回复，请稍后再试。",
			ContentType:  "text",
			Status:       "failed",
			ErrorMessage: err.Error(),
		})
		if saveErr != nil {
			return userMsg, nil, saveErr
		}
		return userMsg, assistantMsg, err
	}
	assistantMsg, err := uc.repo.CreateMessage(ctx, &ChatMessage{
		SessionID:           sessionID,
		UserID:              userID,
		Role:                "assistant",
		Content:             reply.Content,
		ContentType:         "text",
		Model:               reply.Model,
		PromptTokens:        reply.PromptTokens,
		CompletionTokens:    reply.CompletionTokens,
		TotalTokens:         reply.PromptTokens + reply.CompletionTokens,
		LatencyMS:           reply.LatencyMS,
		SafetyResultJSON:    reply.SafetyResultJSON,
		EmotionSnapshotJSON: "{}",
		Status:              "success",
	})
	if err != nil {
		return userMsg, nil, err
	}
	_ = uc.repo.TouchSession(ctx, sessionID, time.Now(), 2)
	return userMsg, assistantMsg, nil
}

func (uc *ChatUsecase) ListMessages(ctx context.Context, userID, sessionID int64, page, pageSize int32) ([]*ChatMessage, int64, error) {
	return uc.repo.ListMessages(ctx, userID, sessionID, page, pageSize)
}

func (uc *ChatUsecase) CreateFeedback(ctx context.Context, feedback *ChatFeedback) (*ChatFeedback, error) {
	return uc.repo.CreateFeedback(ctx, feedback)
}

func (uc *ChatUsecase) SummarizeSession(ctx context.Context, userID, sessionID int64) (*ChatContextSummary, error) {
	if _, err := uc.repo.GetSession(ctx, userID, sessionID); err != nil {
		return nil, err
	}
	messages, err := uc.repo.RecentMessages(ctx, sessionID, 100)
	if err != nil {
		return nil, err
	}
	reply, err := uc.ai.Summarize(ctx, messages)
	if err != nil {
		return nil, err
	}
	var startID, endID int64
	if len(messages) > 0 {
		startID = messages[0].ID
		endID = messages[len(messages)-1].ID
	}
	return uc.repo.CreateSummary(ctx, &ChatContextSummary{
		SessionID:      sessionID,
		Summary:        reply.Content,
		MessageStartID: startID,
		MessageEndID:   endID,
		Model:          reply.Model,
	})
}
