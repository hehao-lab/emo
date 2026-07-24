package service

import (
	"context"
	"strconv"

	v1 "emo-ai-service/api/aichat/v1"
	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"google.golang.org/protobuf/types/known/emptypb"
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
	if req.SystemPrompt != nil && !auth.HasRole(ctx, "admin") {
		return nil, kerrors.Forbidden("SYSTEM_PROMPT_FORBIDDEN", "system prompt is restricted to administrators")
	}
	reply, err := s.uc.Chat(ctx, &biz.AIChatRequest{
		UserID:         userID,
		UpstreamUserID: upstreamUserID(userID),
		ConversationID: req.ConversationId,
		Message:        req.GetMessage(),
		SystemPrompt:   req.SystemPrompt,
		ClientRequestID: req.GetClientRequestId(),
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
func (s *AIChatService) CreateKnowledgeDocument(ctx context.Context, req *v1.CreateKnowledgeDocumentRequest) (*v1.CreateKnowledgeDocumentReply, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	reply, err := s.uc.CreateKnowledgeDocument(ctx, &biz.AICreateKnowledgeDocument{
		UserID:         userID,
		UpstreamUserID: upstreamUserID(userID),
		Title:          req.GetTitle(),
		Content:        req.Content,
		Source:         req.Source,
		ObjectReference: req.ObjectReference,
		MetadataJSON:   req.GetMetadataJson(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateKnowledgeDocumentReply{Id: reply.ID, Status: reply.Status, JobId: reply.JobID}, nil
}

// ListKnowledgeDocuments returns the current user's knowledge documents.
func (s *AIChatService) ListKnowledgeDocuments(ctx context.Context, req *v1.ListKnowledgeDocumentsRequest) (*v1.KnowledgeDocumentSet, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	set, err := s.uc.ListKnowledgeDocuments(ctx, userID, upstreamUserID(userID), biz.AIKnowledgeListOptions{
		Page: req.GetPage(), PageSize: req.GetPageSize(), Status: req.Status, Query: req.Query, Cursor: req.Cursor,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.KnowledgeDocument, 0, len(set.Items))
	for _, item := range set.Items {
		out = append(out, toAIKnowledgeDocumentDTO(item))
	}
	return &v1.KnowledgeDocumentSet{
		Items: out, Total: set.Total, NextCursor: set.NextCursor, Page: set.Page, PageSize: set.PageSize,
	}, nil
}

func (s *AIChatService) GetKnowledgeDocument(ctx context.Context, req *v1.GetKnowledgeDocumentRequest) (*v1.KnowledgeDocument, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := s.uc.GetKnowledgeDocument(ctx, userID, upstreamUserID(userID), req.GetDocumentId())
	if err != nil {
		return nil, err
	}
	return toAIKnowledgeDocumentDTO(doc), nil
}

func (s *AIChatService) UpdateKnowledgeDocument(ctx context.Context, req *v1.UpdateKnowledgeDocumentRequest) (*v1.KnowledgeDocument, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := s.uc.UpdateKnowledgeDocument(ctx, &biz.AIUpdateKnowledgeDocument{
		UserID: userID, UpstreamUserID: upstreamUserID(userID), DocumentID: req.GetDocumentId(),
		Title: req.Title, Source: req.Source, MetadataJSON: req.MetadataJson,
	})
	if err != nil {
		return nil, err
	}
	return toAIKnowledgeDocumentDTO(doc), nil
}

func (s *AIChatService) DeleteKnowledgeDocument(ctx context.Context, req *v1.DeleteKnowledgeDocumentRequest) (*emptypb.Empty, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.DeleteKnowledgeDocument(ctx, userID, upstreamUserID(userID), req.GetDocumentId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AIChatService) ReindexKnowledgeDocument(ctx context.Context, req *v1.ReindexKnowledgeDocumentRequest) (*v1.ReindexKnowledgeDocumentReply, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	reply, err := s.uc.ReindexKnowledgeDocument(ctx, userID, upstreamUserID(userID), req.GetDocumentId())
	if err != nil {
		return nil, err
	}
	return &v1.ReindexKnowledgeDocumentReply{JobId: reply.JobID, Status: reply.Status}, nil
}

func (s *AIChatService) GetKnowledgeJob(ctx context.Context, req *v1.GetKnowledgeJobRequest) (*v1.KnowledgeJob, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	job, err := s.uc.GetKnowledgeJob(ctx, userID, upstreamUserID(userID), req.GetJobId())
	if err != nil {
		return nil, err
	}
	return toAIKnowledgeJobDTO(job), nil
}

// upstreamUserID is an internal representation used to build the signed model-service assertion.
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
		ProviderRequestId: in.ProviderRequestID,
		RequestId:      in.RequestID,
		ClientRequestId: in.ClientRequestID,
		TurnStatus:     in.TurnStatus,
		ReferencesJson: in.ReferencesJSON,
		UsageJson:      in.UsageJSON,
		CreatedAt:      in.CreatedAt,
	}
}

// toAIKnowledgeDocumentDTO converts a biz knowledge document into the proto DTO.
func toAIKnowledgeDocumentDTO(in *biz.AIKnowledgeDocument) *v1.KnowledgeDocument {
	if in == nil {
		return nil
	}
	chunks := make([]*v1.KnowledgeChunk, 0, len(in.Chunks))
	for _, chunk := range in.Chunks {
		chunks = append(chunks, &v1.KnowledgeChunk{Id: chunk.ID, Content: chunk.Content, Sequence: chunk.Sequence})
	}
	return &v1.KnowledgeDocument{
		Id:         in.ID,
		Title:      in.Title,
		Source:     in.Source,
		ChunkCount: in.ChunkCount,
		CreatedAt:  in.CreatedAt,
		Status: in.Status,
		Progress: in.Progress,
		ErrorCode: in.ErrorCode,
		ErrorDetail: in.ErrorDetail,
		IndexVersion: in.IndexVersion,
		EmbeddingModel: in.EmbeddingModel,
		EmbeddingDimension: in.EmbeddingDimension,
		UpdatedAt: in.UpdatedAt,
		MetadataJson: in.MetadataJSON,
		Preview: in.Preview,
		Chunks: chunks,
	}
}

func toAIKnowledgeJobDTO(in *biz.AIKnowledgeJob) *v1.KnowledgeJob {
	if in == nil {
		return nil
	}
	return &v1.KnowledgeJob{
		Id: in.ID, DocumentId: in.DocumentID, Kind: in.Kind, Status: in.Status,
		Progress: in.Progress, TargetIndexVersion: in.TargetIndexVersion,
		ErrorCode: in.ErrorCode, ErrorDetail: in.ErrorDetail,
		CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt,
	}
}
