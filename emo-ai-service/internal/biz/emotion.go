package biz

import (
	"context"
	"time"
)

type EmotionAnalysis struct {
	ID                  int64
	UserID              int64
	SourceType          string
	SourceID            int64
	PrimaryEmotion      string
	Sentiment           string
	SentimentScore      float64
	StressScore         int32
	AnxietyScore        int32
	DepressionRiskScore int32
	EnergyScore         int32
	Confidence          float64
	Summary             string
	Advice              string
	RiskLevel           string
	Model               string
	Dimensions          []*EmotionDimensionScore
	RawResultJSON       string
	CreatedAt           time.Time
}

type EmotionDimensionScore struct {
	Dimension string
	Score     float64
}

type EmotionDailyStat struct {
	Date              string
	DiaryCount        int32
	ChatCount         int32
	AvgMoodScore      float64
	AvgSentimentScore float64
	DominantEmotion   string
	HighRiskCount     int64
	Dimensions        []*EmotionDimensionScore
}

type EmotionListOption struct {
	Page       int32
	PageSize   int32
	SourceType string
	StartDate  string
	EndDate    string
}

type EmotionOverview struct {
	TotalAnalyses     int64
	AvgSentimentScore float64
	AvgMoodScore      float64
	DominantEmotion   string
	HighRiskCount     int64
	Dimensions        []*EmotionDimensionScore
}

type EmotionRepo interface {
	CreateAnalysis(ctx context.Context, analysis *EmotionAnalysis) (*EmotionAnalysis, error)
	ListAnalyses(ctx context.Context, userID int64, opt EmotionListOption) ([]*EmotionAnalysis, int64, error)
	GetAnalysis(ctx context.Context, userID, id int64) (*EmotionAnalysis, error)
	Overview(ctx context.Context, userID int64, startDate string) (*EmotionOverview, error)
	Trends(ctx context.Context, userID int64, startDate, endDate string) ([]*EmotionDailyStat, error)
	Calendar(ctx context.Context, userID int64, month string) ([]*EmotionDailyStat, error)
}

type EmotionAnalyzer interface {
	Analyze(ctx context.Context, userID int64, sourceType string, sourceID int64, text string) (*EmotionAnalysis, error)
}

type EmotionUsecase struct {
	repo     EmotionRepo
	analyzer EmotionAnalyzer
}

func NewEmotionUsecase(repo EmotionRepo, analyzer EmotionAnalyzer) *EmotionUsecase {
	return &EmotionUsecase{repo: repo, analyzer: analyzer}
}

func (uc *EmotionUsecase) CreateAnalysis(ctx context.Context, userID int64, sourceType string, sourceID int64, text string) (*EmotionAnalysis, error) {
	analysis, err := uc.analyzer.Analyze(ctx, userID, sourceType, sourceID, text)
	if err != nil {
		return nil, err
	}
	return uc.repo.CreateAnalysis(ctx, analysis)
}

func (uc *EmotionUsecase) ListAnalyses(ctx context.Context, userID int64, opt EmotionListOption) ([]*EmotionAnalysis, int64, error) {
	return uc.repo.ListAnalyses(ctx, userID, opt)
}

func (uc *EmotionUsecase) GetAnalysis(ctx context.Context, userID, id int64) (*EmotionAnalysis, error) {
	return uc.repo.GetAnalysis(ctx, userID, id)
}

func (uc *EmotionUsecase) Overview(ctx context.Context, userID int64, period string) (*EmotionOverview, error) {
	startDate := dateByRange(period)
	return uc.repo.Overview(ctx, userID, startDate)
}

func (uc *EmotionUsecase) Trends(ctx context.Context, userID int64, startDate, endDate string) ([]*EmotionDailyStat, error) {
	return uc.repo.Trends(ctx, userID, startDate, endDate)
}

func (uc *EmotionUsecase) Calendar(ctx context.Context, userID int64, month string) ([]*EmotionDailyStat, error) {
	return uc.repo.Calendar(ctx, userID, month)
}

func dateByRange(period string) string {
	now := time.Now()
	switch period {
	case "90d":
		return now.AddDate(0, 0, -90).Format("2006-01-02")
	case "30d":
		return now.AddDate(0, 0, -30).Format("2006-01-02")
	default:
		return now.AddDate(0, 0, -7).Format("2006-01-02")
	}
}
