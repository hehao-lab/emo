package data

import (
	"context"
	"encoding/json"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

type adminRepoImpl struct {
	db *gorm.DB
}

func NewAdminRepo(d *Data) biz.AdminRepo {
	return &adminRepoImpl{db: d.db}
}

func (r *adminRepoImpl) GetDashboardOverview(ctx context.Context, today time.Time) (*biz.AdminDashboardOverview, error) {
	out := &biz.AdminDashboardOverview{}
	counts := []struct {
		model any
		where string
		args  []any
		dest  *int64
	}{
		{&UserModel{}, "", nil, &out.UserCount},
		{&UserModel{}, "created_at >= ?", []any{today}, &out.TodayNewUsers},
		{&MoodDiaryModel{}, "", nil, &out.DiaryCount},
		{&MoodDiaryModel{}, "created_at >= ?", []any{today}, &out.TodayDiaries},
		{&ChatSessionModel{}, "", nil, &out.ChatSessionCount},
		{&ChatMessageModel{}, "created_at >= ?", []any{today}, &out.TodayChatMessages},
		{&EmotionAnalysisModel{}, "", nil, &out.EmotionAnalysisCount},
		{&EmotionAnalysisModel{}, "risk_level IN ?", []any{[]string{"high", "crisis"}}, &out.HighRiskAnalysisCount},
	}
	for _, item := range counts {
		q := r.db.WithContext(ctx).Model(item.model)
		if item.where != "" {
			q = q.Where(item.where, item.args...)
		}
		if err := q.Count(item.dest).Error; err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *adminRepoImpl) GetDashboardTrends(ctx context.Context, start, end time.Time) ([]*biz.AdminDashboardTrendPoint, error) {
	type countRow struct {
		Date  string
		Count int64
	}
	endExclusive := end.AddDate(0, 0, 1)
	readCounts := func(model any) (map[string]int64, error) {
		var rows []countRow
		err := r.db.WithContext(ctx).Model(model).
			Select("DATE(created_at) AS date, COUNT(*) AS count").
			Where("created_at >= ? AND created_at < ?", start, endExclusive).
			Group("DATE(created_at)").Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		out := make(map[string]int64, len(rows))
		for _, row := range rows {
			out[row.Date] = row.Count
		}
		return out, nil
	}
	users, err := readCounts(&UserModel{})
	if err != nil {
		return nil, err
	}
	diaries, err := readCounts(&MoodDiaryModel{})
	if err != nil {
		return nil, err
	}
	messages, err := readCounts(&ChatMessageModel{})
	if err != nil {
		return nil, err
	}
	analyses, err := readCounts(&EmotionAnalysisModel{})
	if err != nil {
		return nil, err
	}
	points := make([]*biz.AdminDashboardTrendPoint, 0, int(end.Sub(start).Hours()/24)+1)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		points = append(points, &biz.AdminDashboardTrendPoint{
			Date: date, NewUsers: users[date], Diaries: diaries[date],
			ChatMessages: messages[date], EmotionAnalyses: analyses[date],
		})
	}
	return points, nil
}

func (r *adminRepoImpl) ListUsers(ctx context.Context, opt biz.AdminUserListOption) ([]*biz.AdminUser, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&UserModel{})
	if opt.Keyword != "" {
		like := "%" + opt.Keyword + "%"
		q = q.Where("username LIKE ? OR phone LIKE ? OR email LIKE ?", like, like, like)
	}
	if opt.Status > 0 {
		q = q.Where("status = ?", opt.Status)
	}
	if opt.Role != "" {
		q = q.Where("roles LIKE ?", "%\""+opt.Role+"\"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []UserModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	profiles, err := r.loadProfiles(ctx, userIDs(models))
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminUser, 0, len(models))
	for i := range models {
		out = append(out, toAdminUser(&models[i], profiles[models[i].ID]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) GetUser(ctx context.Context, userID int64) (*biz.AdminUser, error) {
	var model UserModel
	err := r.db.WithContext(ctx).First(&model, userID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, biz.AdminNotFound("user")
	}
	if err != nil {
		return nil, err
	}
	profiles, err := r.loadProfiles(ctx, []int64{userID})
	if err != nil {
		return nil, err
	}
	return toAdminUser(&model, profiles[userID]), nil
}

func (r *adminRepoImpl) UpdateUserStatus(ctx context.Context, userID int64, status int32, reason string) (*biz.AdminUser, error) {
	if err := r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", userID).Update("status", status).Error; err != nil {
		return nil, err
	}
	if reason != "" {
		metadata, _ := json.Marshal(map[string]any{"status": status, "reason": reason})
		_ = r.db.WithContext(ctx).Create(&SecurityEventModel{UserID: userID, EventType: "account_status_changed", RiskLevel: "medium", MetadataJSON: string(metadata)}).Error
	}
	return r.GetUser(ctx, userID)
}

func (r *adminRepoImpl) UpdateUserRoles(ctx context.Context, userID int64, roles []string) (*biz.AdminUser, error) {
	raw, err := json.Marshal(roles)
	if err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", userID).Update("roles", string(raw)).Error; err != nil {
		return nil, err
	}
	return r.GetUser(ctx, userID)
}

func (r *adminRepoImpl) UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error {
	result := r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", userID).Update("password_hash", passwordHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return biz.AdminNotFound("user")
		}
	}
	return nil
}

func (r *adminRepoImpl) ListDiaries(ctx context.Context, opt biz.AdminDiaryListOption) ([]*biz.AdminDiary, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&MoodDiaryModel{})
	if opt.Keyword != "" {
		like := "%" + opt.Keyword + "%"
		q = q.Where("title LIKE ? OR content LIKE ?", like, like)
	}
	if opt.UserID > 0 {
		q = q.Where("user_id = ?", opt.UserID)
	}
	if opt.Mood != "" {
		q = q.Where("mood = ?", opt.Mood)
	}
	if opt.StartDate != "" {
		q = q.Where("occurred_on >= ?", opt.StartDate)
	}
	if opt.EndDate != "" {
		q = q.Where("occurred_on <= ?", opt.EndDate)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []MoodDiaryModel
	if err := q.Order("occurred_on desc, created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	names, err := r.loadUsernames(ctx, diaryUserIDs(models))
	if err != nil {
		return nil, 0, err
	}
	filler := diaryRepoImpl{db: r.db}
	out := make([]*biz.AdminDiary, 0, len(models))
	for i := range models {
		diary, err := filler.fillDiary(ctx, &models[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, &biz.AdminDiary{MoodDiary: diary, Username: names[models[i].UserID]})
	}
	return out, total, nil
}

func (r *adminRepoImpl) GetDiary(ctx context.Context, id int64) (*biz.AdminDiary, error) {
	var model MoodDiaryModel
	err := r.db.WithContext(ctx).First(&model, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, biz.AdminNotFound("diary")
	}
	if err != nil {
		return nil, err
	}
	diary, err := (&diaryRepoImpl{db: r.db}).fillDiary(ctx, &model)
	if err != nil {
		return nil, err
	}
	names, err := r.loadUsernames(ctx, []int64{model.UserID})
	if err != nil {
		return nil, err
	}
	return &biz.AdminDiary{MoodDiary: diary, Username: names[model.UserID]}, nil
}

func (r *adminRepoImpl) DeleteDiary(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&MoodDiaryModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return biz.AdminNotFound("diary")
	}
	return nil
}

func (r *adminRepoImpl) ListEmotionAnalyses(ctx context.Context, opt biz.AdminEmotionListOption) ([]*biz.AdminEmotionAnalysis, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&EmotionAnalysisModel{})
	if opt.UserID > 0 {
		q = q.Where("user_id = ?", opt.UserID)
	}
	if opt.RiskLevel != "" {
		q = q.Where("risk_level = ?", opt.RiskLevel)
	}
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
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	names, err := r.loadUsernames(ctx, analysisUserIDs(models))
	if err != nil {
		return nil, 0, err
	}
	filler := emotionRepoImpl{db: r.db}
	out := make([]*biz.AdminEmotionAnalysis, 0, len(models))
	for i := range models {
		analysis, err := filler.fillAnalysis(ctx, &models[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, &biz.AdminEmotionAnalysis{EmotionAnalysis: analysis, Username: names[models[i].UserID]})
	}
	return out, total, nil
}

func (r *adminRepoImpl) GetEmotionAnalysis(ctx context.Context, id int64) (*biz.AdminEmotionAnalysis, error) {
	var model EmotionAnalysisModel
	err := r.db.WithContext(ctx).First(&model, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, biz.AdminNotFound("emotion analysis")
	}
	if err != nil {
		return nil, err
	}
	analysis, err := (&emotionRepoImpl{db: r.db}).fillAnalysis(ctx, &model)
	if err != nil {
		return nil, err
	}
	names, err := r.loadUsernames(ctx, []int64{model.UserID})
	if err != nil {
		return nil, err
	}
	return &biz.AdminEmotionAnalysis{EmotionAnalysis: analysis, Username: names[model.UserID]}, nil
}

func (r *adminRepoImpl) ListChatSessions(ctx context.Context, opt biz.AdminChatListOption) ([]*biz.AdminChatSession, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&ChatSessionModel{})
	if opt.UserID > 0 {
		q = q.Where("user_id = ?", opt.UserID)
	}
	if opt.Status != "" {
		q = q.Where("status = ?", opt.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []ChatSessionModel
	if err := q.Order("COALESCE(last_message_at, created_at) desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	names, err := r.loadUsernames(ctx, sessionUserIDs(models))
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminChatSession, 0, len(models))
	for i := range models {
		out = append(out, &biz.AdminChatSession{ChatSession: toBizChatSession(&models[i]), Username: names[models[i].UserID]})
	}
	return out, total, nil
}

func (r *adminRepoImpl) ListChatMessages(ctx context.Context, sessionID int64, page, pageSize int32) ([]*biz.ChatMessage, int64, error) {
	var sessionCount int64
	if err := r.db.WithContext(ctx).Model(&ChatSessionModel{}).Where("id = ?", sessionID).Count(&sessionCount).Error; err != nil {
		return nil, 0, err
	}
	if sessionCount == 0 {
		return nil, 0, biz.AdminNotFound("chat session")
	}
	p, size := normalizePage(page, pageSize)
	q := r.db.WithContext(ctx).Model(&ChatMessageModel{}).Where("session_id = ?", sessionID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []ChatMessageModel
	if err := q.Order("created_at asc, id asc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.ChatMessage, 0, len(models))
	for i := range models {
		out = append(out, toBizChatMessage(&models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) ListLoginLogs(ctx context.Context, opt biz.AdminLoginLogListOption) ([]*biz.AdminLoginLog, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&LoginLogModel{})
	if opt.UserID > 0 {
		q = q.Where("user_id = ?", opt.UserID)
	}
	if opt.SuccessOnly {
		q = q.Where("success = ?", true)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []LoginLogModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.UserID)
	}
	names, err := r.loadUsernames(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminLoginLog, 0, len(models))
	for i := range models {
		username := models[i].Username
		if username == "" {
			username = names[models[i].UserID]
		}
		out = append(out, &biz.AdminLoginLog{LoginLog: toBizLoginLog(&models[i]), Username: username})
	}
	return out, total, nil
}

func (r *adminRepoImpl) ListSecurityEvents(ctx context.Context, opt biz.AdminSecurityEventListOption) ([]*biz.AdminSecurityEvent, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&SecurityEventModel{})
	if opt.UserID > 0 {
		q = q.Where("user_id = ?", opt.UserID)
	}
	if opt.RiskLevel != "" {
		q = q.Where("risk_level = ?", opt.RiskLevel)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []SecurityEventModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.UserID)
	}
	names, err := r.loadUsernames(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminSecurityEvent, 0, len(models))
	for i := range models {
		event := &biz.SecurityEvent{
			ID: models[i].ID, UserID: models[i].UserID, EventType: models[i].EventType,
			RiskLevel: models[i].RiskLevel, IP: models[i].IP, UserAgent: models[i].UserAgent,
			MetadataJSON: models[i].MetadataJSON, CreatedAt: models[i].CreatedAt,
		}
		out = append(out, &biz.AdminSecurityEvent{SecurityEvent: event, Username: names[models[i].UserID]})
	}
	return out, total, nil
}

func (r *adminRepoImpl) ListConfigs(ctx context.Context, opt biz.AdminConfigListOption) ([]*biz.AdminSystemConfig, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&SystemConfigModel{})
	if opt.Keyword != "" {
		like := "%" + opt.Keyword + "%"
		q = q.Where("config_key LIKE ? OR description LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []SystemConfigModel
	if err := q.Order("config_key asc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminSystemConfig, 0, len(models))
	for i := range models {
		out = append(out, toAdminConfig(&models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) CreateConfig(ctx context.Context, item *biz.AdminSystemConfig) (*biz.AdminSystemConfig, error) {
	model := &SystemConfigModel{ConfigKey: item.Key, ConfigValue: item.ValueJSON, Description: item.Description, IsPublic: item.IsPublic}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toAdminConfig(model), nil
}

func (r *adminRepoImpl) UpdateConfig(ctx context.Context, item *biz.AdminSystemConfig) (*biz.AdminSystemConfig, error) {
	result := r.db.WithContext(ctx).Model(&SystemConfigModel{}).Where("id = ?", item.ID).Updates(map[string]any{
		"config_value": item.ValueJSON, "description": item.Description, "is_public": item.IsPublic,
	})
	if result.Error != nil {
		return nil, result.Error
	}
	var model SystemConfigModel
	if err := r.db.WithContext(ctx).First(&model, item.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, biz.AdminNotFound("config")
		}
		return nil, err
	}
	return toAdminConfig(&model), nil
}

func (r *adminRepoImpl) DeleteConfig(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&SystemConfigModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return biz.AdminNotFound("config")
	}
	return nil
}

func (r *adminRepoImpl) ListAnnouncements(ctx context.Context, opt biz.AdminAnnouncementListOption) ([]*biz.AdminAnnouncement, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&SystemAnnouncementModel{})
	if opt.Platform != "" {
		q = q.Where("target_platform = ?", opt.Platform)
	}
	if opt.Status != nil {
		q = q.Where("status = ?", *opt.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []SystemAnnouncementModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminAnnouncement, 0, len(models))
	for i := range models {
		out = append(out, toAdminAnnouncement(&models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) CreateAnnouncement(ctx context.Context, item *biz.AdminAnnouncement) (*biz.AdminAnnouncement, error) {
	model := &SystemAnnouncementModel{
		Title: item.Title, Content: item.Content, TargetPlatform: item.TargetPlatform,
		StartAt: optionalTime(item.StartAt), EndAt: optionalTime(item.EndAt), Status: item.Status,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toAdminAnnouncement(model), nil
}

func (r *adminRepoImpl) UpdateAnnouncement(ctx context.Context, item *biz.AdminAnnouncement) (*biz.AdminAnnouncement, error) {
	updates := map[string]any{
		"title": item.Title, "content": item.Content, "target_platform": item.TargetPlatform,
		"start_at": optionalTime(item.StartAt), "end_at": optionalTime(item.EndAt), "status": item.Status,
	}
	if err := r.db.WithContext(ctx).Model(&SystemAnnouncementModel{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	var model SystemAnnouncementModel
	if err := r.db.WithContext(ctx).First(&model, item.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, biz.AdminNotFound("announcement")
		}
		return nil, err
	}
	return toAdminAnnouncement(&model), nil
}

func (r *adminRepoImpl) DeleteAnnouncement(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&SystemAnnouncementModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return biz.AdminNotFound("announcement")
	}
	return nil
}

func (r *adminRepoImpl) ListVersions(ctx context.Context, opt biz.AdminVersionListOption) ([]*biz.AdminAppVersion, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&AppVersionModel{})
	if opt.Platform != "" {
		q = q.Where("platform = ?", opt.Platform)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []AppVersionModel
	if err := q.Order("published_at desc, build_no desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminAppVersion, 0, len(models))
	for i := range models {
		out = append(out, toAdminVersion(&models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) CreateVersion(ctx context.Context, item *biz.AdminAppVersion) (*biz.AdminAppVersion, error) {
	model := &AppVersionModel{
		Platform: item.Platform, Version: item.Version, BuildNo: item.BuildNo, ForceUpdate: item.ForceUpdate,
		DownloadURL: item.DownloadURL, Changelog: item.Changelog, MinSupportedVersion: item.MinSupportedVersion,
		PublishedAt: optionalTime(item.PublishedAt),
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toAdminVersion(model), nil
}

func (r *adminRepoImpl) UpdateVersion(ctx context.Context, item *biz.AdminAppVersion) (*biz.AdminAppVersion, error) {
	updates := map[string]any{
		"platform": item.Platform, "version": item.Version, "build_no": item.BuildNo,
		"force_update": item.ForceUpdate, "download_url": item.DownloadURL, "changelog": item.Changelog,
		"min_supported_version": item.MinSupportedVersion, "published_at": optionalTime(item.PublishedAt),
	}
	if err := r.db.WithContext(ctx).Model(&AppVersionModel{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	var model AppVersionModel
	if err := r.db.WithContext(ctx).First(&model, item.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, biz.AdminNotFound("version")
		}
		return nil, err
	}
	return toAdminVersion(&model), nil
}

func (r *adminRepoImpl) DeleteVersion(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&AppVersionModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return biz.AdminNotFound("version")
	}
	return nil
}

func (r *adminRepoImpl) ListMoodTags(ctx context.Context, opt biz.AdminMoodTagListOption) ([]*biz.MoodTag, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&MoodTagModel{}).Where("user_id = 0")
	if opt.Keyword != "" {
		q = q.Where("name LIKE ?", "%"+opt.Keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []MoodTagModel
	if err := q.Order("sort asc, id asc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.MoodTag, 0, len(models))
	for i := range models {
		out = append(out, toBizTag(&models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) CreateMoodTag(ctx context.Context, tag *biz.MoodTag) (*biz.MoodTag, error) {
	model := &MoodTagModel{UserID: 0, Name: tag.Name, Color: tag.Color, Icon: tag.Icon, Sort: tag.Sort}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toBizTag(model), nil
}

func (r *adminRepoImpl) UpdateMoodTag(ctx context.Context, tag *biz.MoodTag) (*biz.MoodTag, error) {
	updates := map[string]any{"name": tag.Name, "color": tag.Color, "icon": tag.Icon, "sort": tag.Sort}
	if err := r.db.WithContext(ctx).Model(&MoodTagModel{}).Where("id = ? AND user_id = 0", tag.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	var model MoodTagModel
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = 0", tag.ID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, biz.AdminNotFound("system mood tag")
		}
		return nil, err
	}
	return toBizTag(&model), nil
}

func (r *adminRepoImpl) DeleteMoodTag(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = 0", id).Delete(&MoodTagModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return biz.AdminNotFound("system mood tag")
	}
	return nil
}

func (r *adminRepoImpl) ListFiles(ctx context.Context, opt biz.AdminFileListOption) ([]*biz.AdminFile, int64, error) {
	page, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&FileAssetModel{})
	if opt.BizType != "" {
		q = q.Where("biz_type = ?", opt.BizType)
	}
	if opt.OwnerUserID > 0 {
		q = q.Where("owner_user_id = ?", opt.OwnerUserID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []FileAssetModel
	if err := q.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.OwnerUserID)
	}
	names, err := r.loadUsernames(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminFile, 0, len(models))
	for i := range models {
		out = append(out, &biz.AdminFile{FileAsset: toBizFile(&models[i]), Username: names[models[i].OwnerUserID]})
	}
	return out, total, nil
}

func (r *adminRepoImpl) GetFile(ctx context.Context, id int64) (*biz.AdminFile, error) {
	var model FileAssetModel
	err := r.db.WithContext(ctx).First(&model, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, biz.AdminNotFound("file")
	}
	if err != nil {
		return nil, err
	}
	names, err := r.loadUsernames(ctx, []int64{model.OwnerUserID})
	if err != nil {
		return nil, err
	}
	return &biz.AdminFile{FileAsset: toBizFile(&model), Username: names[model.OwnerUserID]}, nil
}

func (r *adminRepoImpl) DeleteFile(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&FileAssetModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return biz.AdminNotFound("file")
	}
	return nil
}

func (r *adminRepoImpl) loadProfiles(ctx context.Context, ids []int64) (map[int64]*UserProfileModel, error) {
	out := make(map[int64]*UserProfileModel)
	if len(ids) == 0 {
		return out, nil
	}
	var models []UserProfileModel
	if err := r.db.WithContext(ctx).Where("user_id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}
	for i := range models {
		out[models[i].UserID] = &models[i]
	}
	return out, nil
}

func (r *adminRepoImpl) loadUsernames(ctx context.Context, ids []int64) (map[int64]string, error) {
	out := make(map[int64]string)
	if len(ids) == 0 {
		return out, nil
	}
	var users []UserModel
	if err := r.db.WithContext(ctx).Select("id", "username").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		out[user.ID] = user.Username
	}
	return out, nil
}

func toAdminUser(model *UserModel, profile *UserProfileModel) *biz.AdminUser {
	var lastLogin time.Time
	if model.LastLoginAt != nil {
		lastLogin = *model.LastLoginAt
	}
	out := &biz.AdminUser{
		ID: model.ID, Username: model.Username, Phone: model.Phone, Email: model.Email,
		Avatar: model.Avatar, Roles: rolesFromJSON(model.Roles), Status: model.Status,
		LastLoginAt: lastLogin, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}
	if profile != nil {
		out.Profile = &biz.AdminUserProfile{
			Nickname: profile.Nickname, Gender: profile.Gender, Birthday: profile.Birthday,
			Bio: profile.Bio, Location: profile.Location, Occupation: profile.Occupation,
			Industry: profile.Industry, Language: profile.Language, Timezone: profile.Timezone,
		}
	}
	return out
}

func toAdminConfig(model *SystemConfigModel) *biz.AdminSystemConfig {
	return &biz.AdminSystemConfig{
		ID: model.ID, Key: model.ConfigKey, ValueJSON: model.ConfigValue, Description: model.Description,
		IsPublic: model.IsPublic, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}
}

func toAdminAnnouncement(model *SystemAnnouncementModel) *biz.AdminAnnouncement {
	return &biz.AdminAnnouncement{
		ID: model.ID, Title: model.Title, Content: model.Content, TargetPlatform: model.TargetPlatform,
		StartAt: valueTime(model.StartAt), EndAt: valueTime(model.EndAt), Status: model.Status,
		CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}
}

func toAdminVersion(model *AppVersionModel) *biz.AdminAppVersion {
	return &biz.AdminAppVersion{
		ID: model.ID, Platform: model.Platform, Version: model.Version, BuildNo: model.BuildNo,
		ForceUpdate: model.ForceUpdate, DownloadURL: model.DownloadURL, Changelog: model.Changelog,
		MinSupportedVersion: model.MinSupportedVersion, PublishedAt: valueTime(model.PublishedAt),
		CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func valueTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func userIDs(models []UserModel) []int64 {
	out := make([]int64, 0, len(models))
	for _, model := range models {
		out = append(out, model.ID)
	}
	return out
}

func diaryUserIDs(models []MoodDiaryModel) []int64 {
	out := make([]int64, 0, len(models))
	for _, model := range models {
		out = append(out, model.UserID)
	}
	return out
}

func analysisUserIDs(models []EmotionAnalysisModel) []int64 {
	out := make([]int64, 0, len(models))
	for _, model := range models {
		out = append(out, model.UserID)
	}
	return out
}

func sessionUserIDs(models []ChatSessionModel) []int64 {
	out := make([]int64, 0, len(models))
	for _, model := range models {
		out = append(out, model.UserID)
	}
	return out
}
