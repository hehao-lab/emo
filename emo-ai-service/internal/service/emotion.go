package service

import (
	"context"

	v1 "emo-ai-service/api/emotion/v1"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type EmotionService struct {
	uc *biz.EmotionUsecase
}

func NewEmotionService(uc *biz.EmotionUsecase) *EmotionService {
	return &EmotionService{uc: uc}
}

var _ v1.EmotionServiceHTTPServer = (*EmotionService)(nil)

// CreateAnalysis 实现情绪分析创建接口：对文本进行情绪识别并保存分析结果和维度分数。
func (s *EmotionService) CreateAnalysis(ctx context.Context, req *v1.CreateAnalysisRequest) (*v1.EmotionAnalysis, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetText() == "" {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "text is required")
	}
	analysis, err := s.uc.CreateAnalysis(ctx, userID, req.GetSourceType(), req.GetSourceId(), req.GetText())
	if err != nil {
		return nil, err
	}
	return toEmotionDTO(analysis), nil
}

// ListAnalyses 实现情绪分析历史接口：按来源类型和日期范围分页查询分析记录。
func (s *EmotionService) ListAnalyses(ctx context.Context, req *v1.ListAnalysesRequest) (*v1.ListAnalysesResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListAnalyses(ctx, userID, biz.EmotionListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), SourceType: req.GetSourceType(), StartDate: req.GetStartDate(), EndDate: req.GetEndDate()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.EmotionAnalysis, 0, len(items))
	for _, item := range items {
		out = append(out, toEmotionDTO(item))
	}
	return &v1.ListAnalysesResponse{Analyses: out, Total: total}, nil
}

// GetAnalysis 实现情绪分析详情接口：读取单次分析的情绪、风险、建议和维度分数。
func (s *EmotionService) GetAnalysis(ctx context.Context, req *v1.GetAnalysisRequest) (*v1.EmotionAnalysis, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	analysis, err := s.uc.GetAnalysis(ctx, userID, req.GetId())
	if err != nil {
		return nil, err
	}
	if analysis == nil {
		return nil, kerrors.NotFound("ANALYSIS_NOT_FOUND", "emotion analysis not found")
	}
	return toEmotionDTO(analysis), nil
}

// GetOverview 实现情感分析报告总览接口：聚合情绪数量、平均分、主导情绪和风险次数。
func (s *EmotionService) GetOverview(ctx context.Context, req *v1.GetOverviewRequest) (*v1.EmotionOverview, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	overview, err := s.uc.Overview(ctx, userID, req.GetRange())
	if err != nil {
		return nil, err
	}
	dims := make([]*v1.EmotionDimensionScore, 0, len(overview.Dimensions))
	for _, item := range overview.Dimensions {
		dims = append(dims, &v1.EmotionDimensionScore{Dimension: item.Dimension, Score: item.Score})
	}
	return &v1.EmotionOverview{TotalAnalyses: overview.TotalAnalyses, AvgSentimentScore: overview.AvgSentimentScore, AvgMoodScore: overview.AvgMoodScore, DominantEmotion: overview.DominantEmotion, HighRiskCount: overview.HighRiskCount, Dimensions: dims}, nil
}

// GetTrends 实现情绪趋势接口：按日期返回情绪分数、心情分数和高风险次数变化。
func (s *EmotionService) GetTrends(ctx context.Context, req *v1.GetTrendsRequest) (*v1.EmotionTrends, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	points, err := s.uc.Trends(ctx, userID, req.GetStartDate(), req.GetEndDate())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.EmotionTrendPoint, 0, len(points))
	for _, item := range points {
		out = append(out, &v1.EmotionTrendPoint{Date: item.Date, AvgSentimentScore: item.AvgSentimentScore, AvgMoodScore: item.AvgMoodScore, DominantEmotion: item.DominantEmotion, HighRiskCount: item.HighRiskCount})
	}
	return &v1.EmotionTrends{Points: out}, nil
}

// GetCalendar 实现情绪日历接口：按月份返回每日情绪概览，供报告页日历视图使用。
func (s *EmotionService) GetCalendar(ctx context.Context, req *v1.GetCalendarRequest) (*v1.EmotionCalendar, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	days, err := s.uc.Calendar(ctx, userID, req.GetMonth())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.EmotionCalendarDay, 0, len(days))
	for _, item := range days {
		out = append(out, &v1.EmotionCalendarDay{Date: item.Date, DominantEmotion: item.DominantEmotion, AvgMoodScore: item.AvgMoodScore, AvgSentimentScore: item.AvgSentimentScore, DiaryCount: item.DiaryCount, ChatCount: item.ChatCount})
	}
	return &v1.EmotionCalendar{Days: out}, nil
}

func toEmotionDTO(analysis *biz.EmotionAnalysis) *v1.EmotionAnalysis {
	if analysis == nil {
		return &v1.EmotionAnalysis{}
	}
	dims := make([]*v1.EmotionDimensionScore, 0, len(analysis.Dimensions))
	for _, item := range analysis.Dimensions {
		dims = append(dims, &v1.EmotionDimensionScore{Dimension: item.Dimension, Score: item.Score})
	}
	return &v1.EmotionAnalysis{Id: analysis.ID, UserId: analysis.UserID, SourceType: analysis.SourceType, SourceId: analysis.SourceID, PrimaryEmotion: analysis.PrimaryEmotion, Sentiment: analysis.Sentiment, SentimentScore: analysis.SentimentScore, StressScore: analysis.StressScore, AnxietyScore: analysis.AnxietyScore, DepressionRiskScore: analysis.DepressionRiskScore, EnergyScore: analysis.EnergyScore, Confidence: analysis.Confidence, Summary: analysis.Summary, Advice: analysis.Advice, RiskLevel: analysis.RiskLevel, Model: analysis.Model, Dimensions: dims, RawResultJson: analysis.RawResultJSON, CreatedAt: analysis.CreatedAt.Unix()}
}
