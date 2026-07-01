package service

import (
	"context"

	v1 "emo-ai-service/api/chat/v1"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ChatService struct {
	uc *biz.ChatUsecase
}

func NewChatService(uc *biz.ChatUsecase) *ChatService {
	return &ChatService{uc: uc}
}

var _ v1.ChatServiceHTTPServer = (*ChatService)(nil)

// CreateSession 实现历史咨询会话创建接口：为当前用户创建一条 AI 咨询会话。
func (s *ChatService) CreateSession(ctx context.Context, req *v1.CreateSessionRequest) (*v1.ChatSession, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.uc.CreateSession(ctx, &biz.ChatSession{UserID: userID, Title: req.GetTitle(), Scenario: req.GetScenario()})
	if err != nil {
		return nil, err
	}
	return toChatSessionDTO(session), nil
}

// ListSessions 实现历史咨询列表接口：分页返回当前用户的咨询会话记录。
func (s *ChatService) ListSessions(ctx context.Context, req *v1.ListSessionsRequest) (*v1.ListSessionsResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListSessions(ctx, userID, biz.ChatListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), Status: req.GetStatus()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.ChatSession, 0, len(items))
	for _, item := range items {
		out = append(out, toChatSessionDTO(item))
	}
	return &v1.ListSessionsResponse{Sessions: out, Total: total}, nil
}

// GetSession 实现咨询会话详情接口：读取当前用户指定会话的基础信息和摘要。
func (s *ChatService) GetSession(ctx context.Context, req *v1.GetSessionRequest) (*v1.ChatSession, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.uc.GetSession(ctx, userID, req.GetId())
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, kerrors.NotFound("CHAT_SESSION_NOT_FOUND", "chat session not found")
	}
	return toChatSessionDTO(session), nil
}

// UpdateSession 实现咨询会话编辑接口：更新会话标题或归档状态。
func (s *ChatService) UpdateSession(ctx context.Context, req *v1.UpdateSessionRequest) (*v1.ChatSession, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.uc.UpdateSession(ctx, &biz.ChatSession{ID: req.GetId(), UserID: userID, Title: req.GetTitle(), Status: req.GetStatus()})
	if err != nil {
		return nil, err
	}
	return toChatSessionDTO(session), nil
}

// DeleteSession 实现咨询会话删除接口：软删除当前用户的指定会话。
func (s *ChatService) DeleteSession(ctx context.Context, req *v1.DeleteSessionRequest) (*emptypb.Empty, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.DeleteSession(ctx, userID, req.GetId())
}

// ListMessages 实现聊天记录接口：分页查询某个咨询会话下的用户消息和 AI 回复。
func (s *ChatService) ListMessages(ctx context.Context, req *v1.ListMessagesRequest) (*v1.ListMessagesResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListMessages(ctx, userID, req.GetSessionId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.ChatMessage, 0, len(items))
	for _, item := range items {
		out = append(out, toChatMessageDTO(item))
	}
	return &v1.ListMessagesResponse{Messages: out, Total: total}, nil
}

// SendMessage 实现 AI 聊天发送接口：保存用户消息，调用 AI 回复，并保存助手消息。
func (s *ChatService) SendMessage(ctx context.Context, req *v1.SendMessageRequest) (*v1.SendMessageResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetContent() == "" {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "content is required")
	}
	userMsg, assistantMsg, err := s.uc.SendMessage(ctx, userID, req.GetSessionId(), req.GetContent(), req.GetContentType())
	if err != nil {
		return nil, err
	}
	return &v1.SendMessageResponse{UserMessage: toChatMessageDTO(userMsg), AssistantMessage: toChatMessageDTO(assistantMsg)}, nil
}

// CreateFeedback 实现 AI 回复反馈接口：记录用户对某条 AI 消息的评分和反馈原因。
func (s *ChatService) CreateFeedback(ctx context.Context, req *v1.CreateFeedbackRequest) (*v1.ChatFeedback, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	feedback, err := s.uc.CreateFeedback(ctx, &biz.ChatFeedback{UserID: userID, MessageID: req.GetMessageId(), Rating: req.GetRating(), FeedbackType: req.GetFeedbackType(), Content: req.GetContent()})
	if err != nil {
		return nil, err
	}
	return &v1.ChatFeedback{Id: feedback.ID, UserId: feedback.UserID, SessionId: feedback.SessionID, MessageId: feedback.MessageID, Rating: feedback.Rating, FeedbackType: feedback.FeedbackType, Content: feedback.Content, CreatedAt: feedback.CreatedAt.Unix()}, nil
}

// SummarizeSession 实现会话摘要接口：根据历史消息生成并保存当前咨询会话摘要。
func (s *ChatService) SummarizeSession(ctx context.Context, req *v1.SummarizeSessionRequest) (*v1.ChatContextSummary, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	summary, err := s.uc.SummarizeSession(ctx, userID, req.GetSessionId())
	if err != nil {
		return nil, err
	}
	return &v1.ChatContextSummary{Id: summary.ID, SessionId: summary.SessionID, Summary: summary.Summary, MessageStartId: summary.MessageStartID, MessageEndId: summary.MessageEndID, Model: summary.Model, CreatedAt: summary.CreatedAt.Unix()}, nil
}

func toChatSessionDTO(session *biz.ChatSession) *v1.ChatSession {
	if session == nil {
		return &v1.ChatSession{}
	}
	return &v1.ChatSession{Id: session.ID, UserId: session.UserID, Title: session.Title, Scenario: session.Scenario, Status: session.Status, Summary: session.Summary, MessageCount: session.MessageCount, LastMessageAt: session.LastMessageAt.Unix(), CreatedAt: session.CreatedAt.Unix(), UpdatedAt: session.UpdatedAt.Unix()}
}

func toChatMessageDTO(message *biz.ChatMessage) *v1.ChatMessage {
	if message == nil {
		return &v1.ChatMessage{}
	}
	return &v1.ChatMessage{Id: message.ID, SessionId: message.SessionID, UserId: message.UserID, Role: message.Role, Content: message.Content, ContentType: message.ContentType, Model: message.Model, PromptTokens: message.PromptTokens, CompletionTokens: message.CompletionTokens, TotalTokens: message.TotalTokens, LatencyMs: message.LatencyMS, EmotionSnapshotJson: message.EmotionSnapshotJSON, SafetyResultJson: message.SafetyResultJSON, Status: message.Status, ErrorMessage: message.ErrorMessage, CreatedAt: message.CreatedAt.Unix()}
}
