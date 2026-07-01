package service

import (
	"context"
	"strconv"

	v1 "emo-ai-service/api/aichat/v1"
	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/biz"
)

// AIChatService adapts frontend DTOs to biz objects for the AI chat BFF.
type AIChatService struct {
	v1.UnimplementedAIChatServiceServer

	uc           *biz.AIChatUsecase
	tokenManager *auth.TokenManager
}

// NewAIChatService builds the AI chat HTTP service.
//
// tokenManager is also used by the raw SSE handler, which does not pass through
// Kratos unary service middleware.
func NewAIChatService(uc *biz.AIChatUsecase, tokenManager *auth.TokenManager) *AIChatService {
	return &AIChatService{uc: uc, tokenManager: tokenManager}
}

var _ v1.AIChatServiceHTTPServer = (*AIChatService)(nil)

// Health proxies downstream health without requiring an authenticated user.
func (s *AIChatService) Health(ctx context.Context, _ *v1.HealthRequest) (*v1.HealthReply, error) {
	if err := s.uc.Health(ctx); err != nil {
		return nil, err
	}
	return &v1.HealthReply{Status: "ok"}, nil
}

// CreateConversation creates a conversation for the current authenticated user.
func (s *AIChatService) CreateConversation(ctx context.Context, req *v1.CreateConversationRequest) (*v1.Conversation, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	conversation, err := s.uc.CreateConversation(ctx, userID, upstreamUserID(userID), req.GetTitle())
	if err != nil {
		return nil, err
	}
	return toAIConversationDTO(conversation), nil
}

// ListConversations returns the current user's conversation list.
func (s *AIChatService) ListConversations(ctx context.Context, _ *v1.ListConversationsRequest) (*v1.ConversationSet, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.uc.ListConversations(ctx, userID, upstreamUserID(userID))
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Conversation, 0, len(items))
	for _, item := range items {
		out = append(out, toAIConversationDTO(item))
	}
	return &v1.ConversationSet{Items: out}, nil
}

// ListMessages returns messages for one conversation owned by the current user.
func (s *AIChatService) ListMessages(ctx context.Context, req *v1.ListMessagesRequest) (*v1.MessageSet, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.uc.ListMessages(ctx, userID, upstreamUserID(userID), req.GetConversationId())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Message, 0, len(items))
	for _, item := range items {
		out = append(out, toAIMessageDTO(item))
	}
	return &v1.MessageSet{Items: out}, nil
}

// Chat sends a non-streaming chat request through the BFF.
func (s *AIChatService) Chat(ctx context.Context, req *v1.ChatRequest) (*v1.ChatReply, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	reply, err := s.uc.Chat(ctx, &biz.AIChatRequest{
		UserID:         userID,
		UpstreamUserID: upstreamUserID(userID),
		ConversationID: req.ConversationId,
		Message:        req.GetMessage(),
		SystemPrompt:   req.SystemPrompt,
	})
	if err != nil {
		return nil, err
	}
	return &v1.ChatReply{
		Conversation:     toAIConversationDTO(reply.Conversation),
		UserMessage:      toAIMessageDTO(reply.UserMessage),
		AssistantMessage: toAIMessageDTO(reply.AssistantMessage),
	}, nil
}

// CreateKnowledgeDocument indexes a knowledge document for the current user.
func (s *AIChatService) CreateKnowledgeDocument(ctx context.Context, req *v1.CreateKnowledgeDocumentRequest) (*v1.KnowledgeDocument, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := s.uc.CreateKnowledgeDocument(ctx, &biz.AICreateKnowledgeDocument{
		UserID:         userID,
		UpstreamUserID: upstreamUserID(userID),
		Title:          req.GetTitle(),
		Content:        req.GetContent(),
		Source:         req.Source,
	})
	if err != nil {
		return nil, err
	}
	return toAIKnowledgeDocumentDTO(doc), nil
}

// ListKnowledgeDocuments returns the current user's knowledge documents.
func (s *AIChatService) ListKnowledgeDocuments(ctx context.Context, _ *v1.ListKnowledgeDocumentsRequest) (*v1.KnowledgeDocumentSet, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.uc.ListKnowledgeDocuments(ctx, userID, upstreamUserID(userID))
	if err != nil {
		return nil, err
	}
	out := make([]*v1.KnowledgeDocument, 0, len(items))
	for _, item := range items {
		out = append(out, toAIKnowledgeDocumentDTO(item))
	}
	return &v1.KnowledgeDocumentSet{Items: out}, nil
}

// upstreamUserID is the current mapping from local users to FastAPI X-User-Id.
func upstreamUserID(userID int64) string {
	return strconv.FormatInt(userID, 10)
}

// toAIConversationDTO converts a biz conversation into the proto DTO.
func toAIConversationDTO(in *biz.AIConversation) *v1.Conversation {
	if in == nil {
		return nil
	}
	return &v1.Conversation{
		Id:        in.ID,
		Title:     in.Title,
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
	}
}

// toAIMessageDTO converts a biz message into the proto DTO.
func toAIMessageDTO(in *biz.AIMessage) *v1.Message {
	if in == nil {
		return nil
	}
	return &v1.Message{
		Id:             in.ID,
		ConversationId: in.ConversationID,
		Role:           in.Role,
		Content:        in.Content,
		Sequence:       in.Sequence,
		ModelName:      in.ModelName,
		CreatedAt:      in.CreatedAt,
	}
}

// toAIKnowledgeDocumentDTO converts a biz knowledge document into the proto DTO.
func toAIKnowledgeDocumentDTO(in *biz.AIKnowledgeDocument) *v1.KnowledgeDocument {
	if in == nil {
		return nil
	}
	return &v1.KnowledgeDocument{
		Id:         in.ID,
		Title:      in.Title,
		Source:     in.Source,
		ChunkCount: in.ChunkCount,
		CreatedAt:  in.CreatedAt,
	}
}
