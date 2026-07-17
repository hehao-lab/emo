package biz

import (
	"context"
	"fmt"
	"strings"
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

type PersonalPortraitReport struct {
	Title               string
	Summary             string
	Traits              []string
	RelationshipPattern string
	RiskNotes           []string
	Suggestions         []string
}

type TargetRelationshipHealthReport struct {
	TargetID          int64
	TargetName        string
	RelationshipLabel string
	HealthScore       int32
	HealthLevel       string
	Summary           string
	Evidence          []string
	RiskNotes         []string
	Suggestions       []string
	GeneratedAt       time.Time
}

type RelationshipHealthReport struct {
	PersonalPortrait *PersonalPortraitReport
	TargetReports    []*TargetRelationshipHealthReport
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
	profile  ProfileRepo
	analyzer EmotionAnalyzer
}

func NewEmotionUsecase(repo EmotionRepo, profile ProfileRepo, analyzer EmotionAnalyzer) *EmotionUsecase {
	return &EmotionUsecase{repo: repo, profile: profile, analyzer: analyzer}
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

func (uc *EmotionUsecase) RelationshipHealthReport(ctx context.Context, userID int64) (*RelationshipHealthReport, error) {
	personal, err := uc.profile.FindPersonalProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if personal == nil {
		personal = &PersonalProfile{UserID: userID}
	}

	targets, err := uc.profile.ListTargetProfiles(ctx, userID)
	if err != nil {
		return nil, err
	}

	report := &RelationshipHealthReport{
		PersonalPortrait: buildPersonalPortraitReport(personal),
		TargetReports:    make([]*TargetRelationshipHealthReport, 0, len(targets)),
	}

	for _, target := range targets {
		records, err := uc.profile.ListImportantRecords(ctx, userID, target.ID)
		if err != nil {
			return nil, err
		}
		report.TargetReports = append(report.TargetReports, buildTargetRelationshipReport(target, records))
	}

	return report, nil
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

func buildPersonalPortraitReport(profile *PersonalProfile) *PersonalPortraitReport {
	traits := compactStrings([]string{
		profile.MBTI,
		profile.RelationshipStatus,
	})

	summary := profile.PersonalitySummary
	if summary == "" {
		summary = "个人画像资料仍在积累中，建议先补充 MBTI、关系状态和性格描述。"
	}

	return &PersonalPortraitReport{
		Title:               "个人画像",
		Summary:             summary,
		Traits:              traits,
		RelationshipPattern: personalRelationshipPattern(profile),
		RiskNotes:           personalRiskNotes(profile),
		Suggestions:         personalSuggestions(profile),
	}
}

func buildTargetRelationshipReport(target *TargetProfile, records []*ImportantRecord) *TargetRelationshipHealthReport {
	score := relationshipHealthScore(target, records)
	level := relationshipHealthLevel(score)
	name := strings.TrimSpace(target.Name)
	if name == "" {
		name = "未命名目标"
	}

	return &TargetRelationshipHealthReport{
		TargetID:          target.ID,
		TargetName:        name,
		RelationshipLabel: fallbackText(target.CurrentRelationship, "关系对象"),
		HealthScore:       score,
		HealthLevel:       level,
		Summary:           targetRelationshipSummary(target, records, score),
		Evidence:          targetEvidence(target, records),
		RiskNotes:         targetRiskNotes(target, records),
		Suggestions:       targetSuggestions(target, records),
		GeneratedAt:       time.Now(),
	}
}

func relationshipHealthScore(target *TargetProfile, records []*ImportantRecord) int32 {
	score := int32(62)
	if strings.TrimSpace(target.InteractionFrequency) != "" {
		score += 8
	}
	if strings.TrimSpace(target.RelationshipGoal) != "" {
		score += 7
	}
	if strings.TrimSpace(target.PersonalityTraits) != "" {
		score += 6
	}
	if strings.TrimSpace(target.RecentInteraction) != "" {
		score += 5
	}
	if len(records) > 0 {
		score += 5
	}
	for _, record := range records {
		switch strings.TrimSpace(record.Satisfaction) {
		case "不满意", "低", "bad", "low":
			score -= 8
		case "满意", "高", "good", "high":
			score += 4
		}
		if strings.TrimSpace(record.ConcernPoint) != "" {
			score -= 2
		}
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func relationshipHealthLevel(score int32) string {
	switch {
	case score >= 85:
		return "excellent"
	case score >= 70:
		return "stable"
	case score >= 50:
		return "watch"
	default:
		return "risk"
	}
}

func personalRelationshipPattern(profile *PersonalProfile) string {
	if profile.RelationshipStatus == "" && profile.MBTI == "" {
		return "资料不足时，系统会先以持续记录和关键互动为主，逐步识别稳定模式。"
	}
	return fmt.Sprintf("当前关系状态为%s，MBTI 为%s；报告会优先观察你在亲密关系中的安全感、确认需求和沟通节奏。", fallbackText(profile.RelationshipStatus, "未填写"), fallbackText(profile.MBTI, "未填写"))
}

func personalRiskNotes(profile *PersonalProfile) []string {
	notes := []string{}
	if profile.PersonalitySummary == "" {
		notes = append(notes, "个人性格描述不足，报告可能偏保守。")
	}
	if profile.RelationshipStatus == "" {
		notes = append(notes, "关系状态未填写，难以判断当前亲密关系阶段。")
	}
	if len(notes) == 0 {
		notes = append(notes, "当回应节奏变化时，留意是否出现过度解读或情绪消耗。")
	}
	return notes
}

func personalSuggestions(profile *PersonalProfile) []string {
	suggestions := []string{"持续补充心情日记和关键互动记录，让报告拥有更稳定的判断依据。"}
	if profile.MBTI == "" {
		suggestions = append(suggestions, "补充 MBTI 或相处偏好，可以提升画像可读性。")
	}
	return suggestions
}

func targetRelationshipSummary(target *TargetProfile, records []*ImportantRecord, score int32) string {
	return fmt.Sprintf("%s 当前健康度为 %d 分。系统结合关系状态、互动频率、关系目标和 %d 条关键记录生成该判断。", fallbackText(target.Name, "该目标"), score, len(records))
}

func targetEvidence(target *TargetProfile, records []*ImportantRecord) []string {
	evidence := compactStrings([]string{
		"当前关系：" + fallbackText(target.CurrentRelationship, "未填写"),
		"互动频率：" + fallbackText(target.InteractionFrequency, "未填写"),
		"关系目标：" + fallbackText(target.RelationshipGoal, "未填写"),
	})
	for _, record := range records {
		if strings.TrimSpace(record.Title) != "" {
			evidence = append(evidence, "关键记录："+record.Title)
		}
	}
	return evidence
}

func targetRiskNotes(target *TargetProfile, records []*ImportantRecord) []string {
	notes := []string{}
	if target.RecentInteraction == "" {
		notes = append(notes, "最近互动未填写，短期关系温度判断有限。")
	}
	for _, record := range records {
		if strings.TrimSpace(record.ConcernPoint) != "" {
			notes = append(notes, record.ConcernPoint)
		}
	}
	if len(notes) == 0 {
		notes = append(notes, "暂无明显高风险信号，继续观察回应稳定性。")
	}
	return notes
}

func targetSuggestions(target *TargetProfile, records []*ImportantRecord) []string {
	suggestions := []string{}
	if target.RelationshipGoal != "" {
		suggestions = append(suggestions, "围绕关系目标推进沟通："+target.RelationshipGoal)
	}
	if len(records) == 0 {
		suggestions = append(suggestions, "记录一次关键互动后，报告会给出更具体建议。")
	} else {
		suggestions = append(suggestions, "复盘近期关键记录，优先处理重复出现的担忧点。")
	}
	return suggestions
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func splitNotes(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '；' || r == ';'
	})
	return compactStrings(parts)
}

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
