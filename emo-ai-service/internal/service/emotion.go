package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	v1 "emo-ai-service/api/emotion/v1"
	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type EmotionService struct {
	uc           *biz.EmotionUsecase
	tokenManager *auth.TokenManager
}

func NewEmotionService(uc *biz.EmotionUsecase, tokenManager *auth.TokenManager) *EmotionService {
	return &EmotionService{uc: uc, tokenManager: tokenManager}
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

type relationshipHealthReportDTO struct {
	PersonalPortrait *personalPortraitReportDTO     `json:"personal_portrait"`
	TargetReports    []*targetRelationshipReportDTO `json:"target_reports"`
}

type personalPortraitReportDTO struct {
	Title               string   `json:"title"`
	Summary             string   `json:"summary"`
	Traits              []string `json:"traits"`
	RelationshipPattern string   `json:"relationship_pattern"`
	RiskNotes           []string `json:"risk_notes"`
	Suggestions         []string `json:"suggestions"`
}

type targetRelationshipReportDTO struct {
	TargetID          int64    `json:"target_id"`
	TargetName        string   `json:"target_name"`
	RelationshipLabel string   `json:"relationship_label"`
	HealthScore       int32    `json:"health_score"`
	HealthLevel       string   `json:"health_level"`
	Summary           string   `json:"summary"`
	Evidence          []string `json:"evidence"`
	RiskNotes         []string `json:"risk_notes"`
	Suggestions       []string `json:"suggestions"`
	GeneratedAt       string   `json:"generated_at"`
}

func (s *EmotionService) RelationshipHealthReportHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID, err := s.userIDFromHTTPRequest(r)
	if err != nil {
		writeJSONError(w, err)
		return
	}

	report, err := s.uc.RelationshipHealthReport(r.Context(), userID)
	if err != nil {
		writeJSONError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(toRelationshipHealthReportDTO(report))
}

func (s *EmotionService) userIDFromHTTPRequest(r *http.Request) (int64, error) {
	if s.tokenManager == nil {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "login required")
	}

	token := emotionBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "missing access token")
	}

	claims, err := s.tokenManager.Parse(token)
	if err != nil || claims.UserID <= 0 {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "invalid access token")
	}

	return claims.UserID, nil
}

func toRelationshipHealthReportDTO(report *biz.RelationshipHealthReport) *relationshipHealthReportDTO {
	if report == nil {
		return &relationshipHealthReportDTO{
			PersonalPortrait: &personalPortraitReportDTO{},
			TargetReports:    []*targetRelationshipReportDTO{},
		}
	}

	targets := make([]*targetRelationshipReportDTO, 0, len(report.TargetReports))
	for _, target := range report.TargetReports {
		targets = append(targets, toTargetRelationshipReportDTO(target))
	}

	return &relationshipHealthReportDTO{
		PersonalPortrait: toPersonalPortraitReportDTO(report.PersonalPortrait),
		TargetReports:    targets,
	}
}

func toPersonalPortraitReportDTO(report *biz.PersonalPortraitReport) *personalPortraitReportDTO {
	if report == nil {
		return &personalPortraitReportDTO{}
	}

	return &personalPortraitReportDTO{
		Title:               report.Title,
		Summary:             report.Summary,
		Traits:              report.Traits,
		RelationshipPattern: report.RelationshipPattern,
		RiskNotes:           report.RiskNotes,
		Suggestions:         report.Suggestions,
	}
}

func toTargetRelationshipReportDTO(report *biz.TargetRelationshipHealthReport) *targetRelationshipReportDTO {
	if report == nil {
		return &targetRelationshipReportDTO{}
	}

	return &targetRelationshipReportDTO{
		TargetID:          report.TargetID,
		TargetName:        report.TargetName,
		RelationshipLabel: report.RelationshipLabel,
		HealthScore:       report.HealthScore,
		HealthLevel:       report.HealthLevel,
		Summary:           report.Summary,
		Evidence:          report.Evidence,
		RiskNotes:         report.RiskNotes,
		Suggestions:       report.Suggestions,
		GeneratedAt:       report.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func emotionBearerToken(value string) string {
	parts := strings.Fields(value)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
