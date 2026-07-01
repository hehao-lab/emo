package data

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

type EmotionAnalysisModel struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement;comment:情绪分析ID"`
	UserID              int64     `gorm:"index:idx_user_created;not null;comment:用户ID"`
	SourceType          string    `gorm:"type:varchar(32);index;not null;comment:分析来源类型 diary chat_message manual"`
	SourceID            int64     `gorm:"index;comment:来源数据ID"`
	PrimaryEmotion      string    `gorm:"type:varchar(32);default:'';comment:主导情绪"`
	Sentiment           string    `gorm:"type:varchar(16);default:'neutral';comment:情感倾向 positive neutral negative"`
	SentimentScore      float64   `gorm:"type:decimal(5,4);default:0;comment:情感分数 -1到1"`
	StressScore         int32     `gorm:"default:0;comment:压力分数 0到100"`
	AnxietyScore        int32     `gorm:"default:0;comment:焦虑分数 0到100"`
	DepressionRiskScore int32     `gorm:"default:0;comment:抑郁风险分数 0到100"`
	EnergyScore         int32     `gorm:"default:0;comment:能量分数 0到100"`
	Confidence          float64   `gorm:"type:decimal(5,4);default:0;comment:分析置信度"`
	Summary             string    `gorm:"type:text;comment:分析摘要"`
	Advice              string    `gorm:"type:text;comment:建议内容"`
	RiskLevel           string    `gorm:"type:varchar(16);default:'low';comment:风险等级 low medium high crisis"`
	Model               string    `gorm:"type:varchar(64);default:'';comment:分析模型名称"`
	RawResultJSON       string    `gorm:"type:json;comment:原始分析结果JSON"`
	CreatedAt           time.Time `gorm:"autoCreateTime;index:idx_user_created;comment:创建时间"`
}

func (EmotionAnalysisModel) TableName() string { return "emotion_analyses" }

type EmotionDimensionScoreModel struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;comment:情绪维度分数ID"`
	AnalysisID int64     `gorm:"index;not null;comment:情绪分析ID"`
	Dimension  string    `gorm:"type:varchar(64);not null;comment:情绪维度名称"`
	Score      float64   `gorm:"type:decimal(5,4);not null;comment:维度分数 0到1"`
	CreatedAt  time.Time `gorm:"autoCreateTime;comment:创建时间"`
}

func (EmotionDimensionScoreModel) TableName() string { return "emotion_dimension_scores" }

type EmotionDailyStatModel struct {
	ID                int64     `gorm:"primaryKey;autoIncrement;comment:每日情绪统计ID"`
	UserID            int64     `gorm:"uniqueIndex:uk_user_date;not null;comment:用户ID"`
	StatDate          string    `gorm:"type:date;uniqueIndex:uk_user_date;not null;comment:统计日期"`
	DiaryCount        int32     `gorm:"default:0;comment:当日心情日记数量"`
	ChatCount         int32     `gorm:"default:0;comment:当日聊天消息数量"`
	AvgMoodScore      float64   `gorm:"type:decimal(5,2);default:0;comment:平均心情分数"`
	AvgSentimentScore float64   `gorm:"type:decimal(5,4);default:0;comment:平均情感分数"`
	DominantEmotion   string    `gorm:"type:varchar(32);default:'';comment:当日主导情绪"`
	HighRiskCount     int64     `gorm:"default:0;comment:高风险分析次数"`
	DimensionSummary  string    `gorm:"type:json;comment:情绪维度汇总JSON"`
	CreatedAt         time.Time `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime;comment:更新时间"`
}

func (EmotionDailyStatModel) TableName() string { return "emotion_daily_stats" }

type emotionRepoImpl struct {
	db *gorm.DB
}

func NewEmotionRepo(d *Data) biz.EmotionRepo {
	return &emotionRepoImpl{db: d.db}
}

func (r *emotionRepoImpl) CreateAnalysis(ctx context.Context, analysis *biz.EmotionAnalysis) (*biz.EmotionAnalysis, error) {
	model := &EmotionAnalysisModel{
		UserID:              analysis.UserID,
		SourceType:          analysis.SourceType,
		SourceID:            analysis.SourceID,
		PrimaryEmotion:      analysis.PrimaryEmotion,
		Sentiment:           analysis.Sentiment,
		SentimentScore:      analysis.SentimentScore,
		StressScore:         analysis.StressScore,
		AnxietyScore:        analysis.AnxietyScore,
		DepressionRiskScore: analysis.DepressionRiskScore,
		EnergyScore:         analysis.EnergyScore,
		Confidence:          analysis.Confidence,
		Summary:             analysis.Summary,
		Advice:              analysis.Advice,
		RiskLevel:           analysis.RiskLevel,
		Model:               analysis.Model,
		RawResultJSON:       jsonObject(analysis.RawResultJSON),
	}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(model).Error; err != nil {
			return err
		}
		for _, item := range analysis.Dimensions {
			if err := tx.Create(&EmotionDimensionScoreModel{AnalysisID: model.ID, Dimension: item.Dimension, Score: item.Score}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetAnalysis(ctx, analysis.UserID, model.ID)
}

func (r *emotionRepoImpl) ListAnalyses(ctx context.Context, userID int64, opt biz.EmotionListOption) ([]*biz.EmotionAnalysis, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&EmotionAnalysisModel{}).Where("user_id = ?", userID)
	if opt.SourceType != "" {
		q = q.Where("source_type = ?", opt.SourceType)
	}
	if opt.StartDate != "" {
		q = q.Where("DATE(created_at) >= ?", opt.StartDate)
	}
	if opt.EndDate != "" {
		q = q.Where("DATE(created_at) <= ?", opt.EndDate)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []EmotionAnalysisModel
	if err := q.Order("created_at desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.EmotionAnalysis, 0, len(models))
	for i := range models {
		item, err := r.fillAnalysis(ctx, &models[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (r *emotionRepoImpl) GetAnalysis(ctx context.Context, userID, id int64) (*biz.EmotionAnalysis, error) {
	var model EmotionAnalysisModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, id).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.fillAnalysis(ctx, &model)
}

func (r *emotionRepoImpl) Overview(ctx context.Context, userID int64, startDate string) (*biz.EmotionOverview, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&EmotionAnalysisModel{}).Where("user_id = ?", userID)
	if startDate != "" {
		q = q.Where("DATE(created_at) >= ?", startDate)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var avg sql.NullFloat64
	if err := q.Session(&gorm.Session{}).Select("AVG(sentiment_score)").Scan(&avg).Error; err != nil {
		return nil, err
	}
	var highRisk int64
	if err := q.Session(&gorm.Session{}).Where("risk_level IN ?", []string{"high", "crisis"}).Count(&highRisk).Error; err != nil {
		return nil, err
	}
	dominant := dominantEmotion(ctx, q.Session(&gorm.Session{}))
	dimensions, err := r.averageDimensions(ctx, userID, startDate)
	if err != nil {
		return nil, err
	}
	return &biz.EmotionOverview{
		TotalAnalyses:     total,
		AvgSentimentScore: nullFloat(avg),
		DominantEmotion:   dominant,
		HighRiskCount:     highRisk,
		Dimensions:        dimensions,
	}, nil
}

func (r *emotionRepoImpl) Trends(ctx context.Context, userID int64, startDate, endDate string) ([]*biz.EmotionDailyStat, error) {
	q := r.db.WithContext(ctx).Model(&EmotionAnalysisModel{}).
		Select("DATE(created_at) AS date, AVG(sentiment_score) AS avg_sentiment_score, SUM(CASE WHEN risk_level IN ('high','crisis') THEN 1 ELSE 0 END) AS high_risk_count").
		Where("user_id = ?", userID).
		Group("DATE(created_at)").
		Order("date asc")
	if startDate != "" {
		q = q.Where("DATE(created_at) >= ?", startDate)
	}
	if endDate != "" {
		q = q.Where("DATE(created_at) <= ?", endDate)
	}
	type row struct {
		Date              string
		AvgSentimentScore float64
		HighRiskCount     int64
	}
	var rows []row
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.EmotionDailyStat, 0, len(rows))
	for _, item := range rows {
		out = append(out, &biz.EmotionDailyStat{Date: item.Date, AvgSentimentScore: item.AvgSentimentScore, HighRiskCount: item.HighRiskCount})
	}
	return out, nil
}

func (r *emotionRepoImpl) Calendar(ctx context.Context, userID int64, month string) ([]*biz.EmotionDailyStat, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	return r.Trends(ctx, userID, month+"-01", month+"-31")
}

func (r *emotionRepoImpl) fillAnalysis(ctx context.Context, model *EmotionAnalysisModel) (*biz.EmotionAnalysis, error) {
	var dimensions []EmotionDimensionScoreModel
	if err := r.db.WithContext(ctx).Where("analysis_id = ?", model.ID).Find(&dimensions).Error; err != nil {
		return nil, err
	}
	outDims := make([]*biz.EmotionDimensionScore, 0, len(dimensions))
	for i := range dimensions {
		outDims = append(outDims, &biz.EmotionDimensionScore{Dimension: dimensions[i].Dimension, Score: dimensions[i].Score})
	}
	return &biz.EmotionAnalysis{
		ID:                  model.ID,
		UserID:              model.UserID,
		SourceType:          model.SourceType,
		SourceID:            model.SourceID,
		PrimaryEmotion:      model.PrimaryEmotion,
		Sentiment:           model.Sentiment,
		SentimentScore:      model.SentimentScore,
		StressScore:         model.StressScore,
		AnxietyScore:        model.AnxietyScore,
		DepressionRiskScore: model.DepressionRiskScore,
		EnergyScore:         model.EnergyScore,
		Confidence:          model.Confidence,
		Summary:             model.Summary,
		Advice:              model.Advice,
		RiskLevel:           model.RiskLevel,
		Model:               model.Model,
		Dimensions:          outDims,
		RawResultJSON:       model.RawResultJSON,
		CreatedAt:           model.CreatedAt,
	}, nil
}

func (r *emotionRepoImpl) averageDimensions(ctx context.Context, userID int64, startDate string) ([]*biz.EmotionDimensionScore, error) {
	q := r.db.WithContext(ctx).Table("emotion_dimension_scores").
		Select("emotion_dimension_scores.dimension, AVG(emotion_dimension_scores.score) AS score").
		Joins("JOIN emotion_analyses ON emotion_analyses.id = emotion_dimension_scores.analysis_id").
		Where("emotion_analyses.user_id = ?", userID).
		Group("emotion_dimension_scores.dimension")
	if startDate != "" {
		q = q.Where("DATE(emotion_analyses.created_at) >= ?", startDate)
	}
	var rows []struct {
		Dimension string
		Score     float64
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.EmotionDimensionScore, 0, len(rows))
	for _, item := range rows {
		out = append(out, &biz.EmotionDimensionScore{Dimension: item.Dimension, Score: item.Score})
	}
	return out, nil
}

func dominantEmotion(ctx context.Context, q *gorm.DB) string {
	var row struct {
		PrimaryEmotion string
		Count          int64
	}
	if err := q.Select("primary_emotion, COUNT(*) AS count").Where("primary_emotion <> ''").Group("primary_emotion").Order("count desc").Limit(1).Scan(&row).Error; err != nil {
		return ""
	}
	return row.PrimaryEmotion
}

func nullFloat(v sql.NullFloat64) float64 {
	if v.Valid {
		return v.Float64
	}
	return 0
}

type localEmotionAnalyzer struct{}

func NewEmotionAnalyzer(*Data) biz.EmotionAnalyzer {
	return &localEmotionAnalyzer{}
}

func (a *localEmotionAnalyzer) Analyze(ctx context.Context, userID int64, sourceType string, sourceID int64, text string) (*biz.EmotionAnalysis, error) {
	lower := strings.ToLower(text)
	primary := "calm"
	sentiment := "neutral"
	sentimentScore := 0.0
	stress := int32(20)
	anxiety := int32(15)
	risk := "low"
	if strings.Contains(lower, "焦虑") || strings.Contains(lower, "anxious") || strings.Contains(lower, "担心") {
		primary = "anxious"
		sentiment = "negative"
		sentimentScore = -0.55
		stress = 70
		anxiety = 80
		risk = "medium"
	}
	if strings.Contains(lower, "难过") || strings.Contains(lower, "sad") || strings.Contains(lower, "崩溃") {
		primary = "sadness"
		sentiment = "negative"
		sentimentScore = -0.65
		stress = 75
		anxiety = 60
		risk = "medium"
	}
	if strings.Contains(lower, "开心") || strings.Contains(lower, "happy") || strings.Contains(lower, "高兴") {
		primary = "joy"
		sentiment = "positive"
		sentimentScore = 0.72
		stress = 10
		anxiety = 8
	}
	if strings.Contains(lower, "自杀") || strings.Contains(lower, "轻生") || strings.Contains(lower, "suicide") {
		risk = "crisis"
		stress = 95
		anxiety = 90
	}
	return &biz.EmotionAnalysis{
		UserID:              userID,
		SourceType:          sourceType,
		SourceID:            sourceID,
		PrimaryEmotion:      primary,
		Sentiment:           sentiment,
		SentimentScore:      sentimentScore,
		StressScore:         stress,
		AnxietyScore:        anxiety,
		DepressionRiskScore: scoreByRisk(risk),
		EnergyScore:         60 - stress/2,
		Confidence:          0.72,
		Summary:             "系统已根据文本内容生成情绪画像。",
		Advice:              "建议持续记录触发情绪的场景，并在压力升高时优先保证休息和支持性沟通。",
		RiskLevel:           risk,
		Model:               "local-emotion-v1",
		RawResultJSON:       `{"provider":"local","version":"v1"}`,
		Dimensions: []*biz.EmotionDimensionScore{
			{Dimension: "joy", Score: dimensionScore(primary, "joy", sentimentScore)},
			{Dimension: "sadness", Score: dimensionScore(primary, "sadness", sentimentScore)},
			{Dimension: "anxiety", Score: float64(anxiety) / 100},
			{Dimension: "stress", Score: float64(stress) / 100},
			{Dimension: "calm", Score: dimensionScore(primary, "calm", sentimentScore)},
		},
	}, nil
}

func scoreByRisk(risk string) int32 {
	switch risk {
	case "crisis":
		return 95
	case "high":
		return 80
	case "medium":
		return 55
	default:
		return 10
	}
}

func dimensionScore(primary, target string, sentimentScore float64) float64 {
	if primary == target {
		return 0.85
	}
	if target == "calm" && sentimentScore >= 0 {
		return 0.55
	}
	if target == "joy" && sentimentScore > 0 {
		return 0.7
	}
	if target == "sadness" && sentimentScore < 0 {
		return 0.45
	}
	return 0.1
}
