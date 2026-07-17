package data

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

type AdminOperationLogModel struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;comment:管理操作日志ID"`
	AdminUserID  int64     `gorm:"index;not null;comment:管理员用户ID"`
	Action       string    `gorm:"type:varchar(64);not null;comment:操作类型"`
	ResourceType string    `gorm:"type:varchar(64);index:idx_admin_logs_resource;not null;comment:资源类型"`
	ResourceID   int64     `gorm:"index:idx_admin_logs_resource;comment:资源ID"`
	DetailJSON   string    `gorm:"type:json;comment:操作详情JSON"`
	IP           string    `gorm:"type:varchar(64);default:'';comment:操作IP"`
	UserAgent    string    `gorm:"type:varchar(512);default:'';comment:User-Agent"`
	CreatedAt    time.Time `gorm:"autoCreateTime;index;comment:创建时间"`
}

func (AdminOperationLogModel) TableName() string { return "admin_operation_logs" }

type adminRepoImpl struct {
	db *gorm.DB
}

func NewAdminRepo(d *Data) biz.AdminRepo {
	return &adminRepoImpl{db: d.db}
}

func (r *adminRepoImpl) DashboardOverview(ctx context.Context) (*biz.AdminDashboardOverview, error) {
	today := time.Now().Format("2006-01-02")
	out := &biz.AdminDashboardOverview{}
	if err := r.db.WithContext(ctx).Model(&UserModel{}).Count(&out.UserCount).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&UserModel{}).Where("DATE(created_at) = ?", today).Count(&out.TodayNewUsers).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&MoodDiaryModel{}).Count(&out.DiaryCount).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&MoodDiaryModel{}).Where("DATE(created_at) = ?", today).Count(&out.TodayDiaries).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&ChatSessionModel{}).Count(&out.ChatSessionCount).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&ChatMessageModel{}).Where("DATE(created_at) = ?", today).Count(&out.TodayChatMessages).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&EmotionAnalysisModel{}).Count(&out.EmotionAnalysisCount).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&EmotionAnalysisModel{}).Where("risk_level IN ?", []string{"high", "crisis"}).Count(&out.HighRiskAnalysisCount).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *adminRepoImpl) DashboardTrends(ctx context.Context, startDate, endDate string) ([]*biz.AdminDashboardTrendPoint, error) {
	start, end := normalizeAdminDateRange(startDate, endDate)
	points := map[string]*biz.AdminDashboardTrendPoint{}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		points[key] = &biz.AdminDashboardTrendPoint{Date: key}
	}
	fill := func(model any, apply func(*biz.AdminDashboardTrendPoint, int64)) error {
		rows, err := r.dailyCounts(ctx, model, start, end)
		if err != nil {
			return err
		}
		for date, count := range rows {
			if point, ok := points[date]; ok {
				apply(point, count)
			}
		}
		return nil
	}
	if err := fill(&UserModel{}, func(p *biz.AdminDashboardTrendPoint, count int64) { p.NewUsers = count }); err != nil {
		return nil, err
	}
	if err := fill(&MoodDiaryModel{}, func(p *biz.AdminDashboardTrendPoint, count int64) { p.Diaries = count }); err != nil {
		return nil, err
	}
	if err := fill(&ChatMessageModel{}, func(p *biz.AdminDashboardTrendPoint, count int64) { p.ChatMessages = count }); err != nil {
		return nil, err
	}
	if err := fill(&EmotionAnalysisModel{}, func(p *biz.AdminDashboardTrendPoint, count int64) { p.EmotionAnalyses = count }); err != nil {
		return nil, err
	}
	out := make([]*biz.AdminDashboardTrendPoint, 0, len(points))
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		out = append(out, points[d.Format("2006-01-02")])
	}
	return out, nil
}

func (r *adminRepoImpl) dailyCounts(ctx context.Context, model any, start, end time.Time) (map[string]int64, error) {
	var rows []struct {
		Date  string
		Count int64
	}
	err := r.db.WithContext(ctx).Model(model).
		Select("DATE(created_at) AS date, COUNT(*) AS count").
		Where("DATE(created_at) BETWEEN ? AND ?", start.Format("2006-01-02"), end.Format("2006-01-02")).
		Group("DATE(created_at)").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, row := range rows {
		out[row.Date] = row.Count
	}
	return out, nil
}

func (r *adminRepoImpl) ListUsers(ctx context.Context, opt biz.AdminUserListOption) ([]*biz.AdminUser, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&UserModel{})
	if keyword := strings.TrimSpace(opt.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username LIKE ? OR phone LIKE ? OR email LIKE ?", like, like, like)
	}
	if opt.Status > 0 {
		q = q.Where("status = ?", opt.Status)
	}
	if role := strings.TrimSpace(opt.Role); role != "" {
		q = q.Where("JSON_CONTAINS(roles, JSON_QUOTE(?))", role)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []UserModel
	if err := q.Order("created_at desc, id desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminUser, 0, len(models))
	for i := range models {
		out = append(out, toAdminUser(&models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) GetUser(ctx context.Context, userID int64) (*biz.AdminUserDetail, error) {
	var user UserModel
	err := r.db.WithContext(ctx).First(&user, userID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var profile UserProfileModel
	err = r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return &biz.AdminUserDetail{
		User:    toAdminUser(&user),
		Profile: toAdminProfile(&profile),
	}, nil
}

func (r *adminRepoImpl) UpdateUserStatus(ctx context.Context, userID int64, status int32) (*biz.AdminUser, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&UserModel{}).Where("id = ?", userID).Update("status", status).Error; err != nil {
			return err
		}
		if status != 1 {
			now := time.Now()
			return tx.Model(&AuthRefreshTokenModel{}).
				Where("user_id = ? AND revoked_at IS NULL", userID).
				Updates(map[string]any{"revoked_at": &now, "revoke_reason": "admin_status_changed"}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var user UserModel
	err = r.db.WithContext(ctx).First(&user, userID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toAdminUser(&user), nil
}

func (r *adminRepoImpl) UpdateUserRoles(ctx context.Context, userID int64, roles []string) (*biz.AdminUser, error) {
	raw, _ := json.Marshal(roles)
	if err := r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", userID).Update("roles", string(raw)).Error; err != nil {
		return nil, err
	}
	var user UserModel
	err := r.db.WithContext(ctx).First(&user, userID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toAdminUser(&user), nil
}

func (r *adminRepoImpl) ListSystemConfigs(ctx context.Context, opt biz.AdminSystemConfigListOption) ([]*biz.AdminSystemConfig, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&SystemConfigModel{})
	if keyword := strings.TrimSpace(opt.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("config_key LIKE ? OR description LIKE ?", like, like)
	}
	if opt.IsPublic != nil {
		q = q.Where("is_public = ?", *opt.IsPublic)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []SystemConfigModel
	if err := q.Order("config_key asc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminSystemConfig, 0, len(models))
	for i := range models {
		out = append(out, toAdminSystemConfig(&models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) CreateSystemConfig(ctx context.Context, item *biz.AdminSystemConfig) (*biz.AdminSystemConfig, error) {
	model := &SystemConfigModel{ConfigKey: item.Key, ConfigValue: item.ValueJSON, Description: item.Description, IsPublic: item.IsPublic}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toAdminSystemConfig(model), nil
}

func (r *adminRepoImpl) UpdateSystemConfig(ctx context.Context, item *biz.AdminSystemConfig, updateIsPublic bool) (*biz.AdminSystemConfig, error) {
	var model SystemConfigModel
	err := r.db.WithContext(ctx).First(&model, item.ID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if item.Key != "" {
		model.ConfigKey = item.Key
	}
	if item.ValueJSON != "" {
		model.ConfigValue = item.ValueJSON
	}
	if item.Description != "" {
		model.Description = item.Description
	}
	if updateIsPublic {
		model.IsPublic = item.IsPublic
	}
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}
	return toAdminSystemConfig(&model), nil
}

func (r *adminRepoImpl) DeleteSystemConfig(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&SystemConfigModel{}, id).Error
}

func (r *adminRepoImpl) ListAnnouncements(ctx context.Context, opt biz.AdminAnnouncementListOption) ([]*biz.AdminAnnouncement, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&SystemAnnouncementModel{})
	if platform := strings.TrimSpace(opt.Platform); platform != "" {
		q = q.Where("target_platform = ?", platform)
	}
	if opt.Status != nil {
		q = q.Where("status = ?", *opt.Status)
	}
	if keyword := strings.TrimSpace(opt.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("title LIKE ? OR content LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []SystemAnnouncementModel
	if err := q.Order("created_at desc, id desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
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
		Title:          item.Title,
		Content:        item.Content,
		TargetPlatform: item.TargetPlatform,
		StartAt:        timePtr(item.StartAt),
		EndAt:          timePtr(item.EndAt),
		Status:         item.Status,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toAdminAnnouncement(model), nil
}

func (r *adminRepoImpl) UpdateAnnouncement(ctx context.Context, item *biz.AdminAnnouncement, updateStatus bool) (*biz.AdminAnnouncement, error) {
	var model SystemAnnouncementModel
	err := r.db.WithContext(ctx).First(&model, item.ID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if item.Title != "" {
		model.Title = item.Title
	}
	if item.Content != "" {
		model.Content = item.Content
	}
	if item.TargetPlatform != "" {
		model.TargetPlatform = item.TargetPlatform
	}
	if !item.StartAt.IsZero() {
		model.StartAt = &item.StartAt
	}
	if !item.EndAt.IsZero() {
		model.EndAt = &item.EndAt
	}
	if updateStatus {
		model.Status = item.Status
	}
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}
	return toAdminAnnouncement(&model), nil
}

func (r *adminRepoImpl) DeleteAnnouncement(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&SystemAnnouncementModel{}, id).Error
}

func (r *adminRepoImpl) ListAppVersions(ctx context.Context, opt biz.AdminAppVersionListOption) ([]*biz.AdminAppVersion, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&AppVersionModel{})
	if platform := strings.TrimSpace(opt.Platform); platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if opt.Status != nil {
		q = q.Where("status = ?", *opt.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []AppVersionModel
	if err := q.Order("platform asc, build_no desc, id desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminAppVersion, 0, len(models))
	for i := range models {
		out = append(out, toAdminAppVersion(&models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) CreateAppVersion(ctx context.Context, item *biz.AdminAppVersion) (*biz.AdminAppVersion, error) {
	model := &AppVersionModel{
		Platform:            item.Platform,
		Version:             item.Version,
		BuildNo:             item.BuildNo,
		ForceUpdate:         item.ForceUpdate,
		DownloadURL:         item.DownloadURL,
		Changelog:           item.Changelog,
		MinSupportedVersion: item.MinSupportedVersion,
		PublishedAt:         timePtr(item.PublishedAt),
		Status:              item.Status,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toAdminAppVersion(model), nil
}

func (r *adminRepoImpl) UpdateAppVersion(ctx context.Context, item *biz.AdminAppVersion, updateForceUpdate bool, updateStatus bool) (*biz.AdminAppVersion, error) {
	var model AppVersionModel
	err := r.db.WithContext(ctx).First(&model, item.ID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if item.Platform != "" {
		model.Platform = item.Platform
	}
	if item.Version != "" {
		model.Version = item.Version
	}
	if item.BuildNo > 0 {
		model.BuildNo = item.BuildNo
	}
	if updateForceUpdate {
		model.ForceUpdate = item.ForceUpdate
	}
	if item.DownloadURL != "" {
		model.DownloadURL = item.DownloadURL
	}
	if item.Changelog != "" {
		model.Changelog = item.Changelog
	}
	if item.MinSupportedVersion != "" {
		model.MinSupportedVersion = item.MinSupportedVersion
	}
	if !item.PublishedAt.IsZero() {
		model.PublishedAt = &item.PublishedAt
	}
	if updateStatus {
		model.Status = item.Status
	}
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}
	return toAdminAppVersion(&model), nil
}

func (r *adminRepoImpl) DeleteAppVersion(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&AppVersionModel{}, id).Error
}

func (r *adminRepoImpl) ListMoodTags(ctx context.Context, opt biz.AdminMoodTagListOption) ([]*biz.AdminMoodTag, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&MoodTagModel{}).Where("user_id = 0")
	if keyword := strings.TrimSpace(opt.Keyword); keyword != "" {
		q = q.Where("name LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []MoodTagModel
	if err := q.Order("sort asc, id asc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminMoodTag, 0, len(models))
	for i := range models {
		out = append(out, toAdminMoodTag(&models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) CreateMoodTag(ctx context.Context, item *biz.AdminMoodTag) (*biz.AdminMoodTag, error) {
	model := &MoodTagModel{UserID: 0, Name: item.Name, Color: item.Color, Icon: item.Icon, Sort: item.Sort}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toAdminMoodTag(model), nil
}

func (r *adminRepoImpl) UpdateMoodTag(ctx context.Context, item *biz.AdminMoodTag) (*biz.AdminMoodTag, error) {
	var model MoodTagModel
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = 0", item.ID).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if item.Name != "" {
		model.Name = item.Name
	}
	if item.Color != "" {
		model.Color = item.Color
	}
	if item.Icon != "" {
		model.Icon = item.Icon
	}
	if item.Sort != 0 {
		model.Sort = item.Sort
	}
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}
	return toAdminMoodTag(&model), nil
}

func (r *adminRepoImpl) DeleteMoodTag(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("user_id = 0 AND id = ?", id).Delete(&MoodTagModel{}).Error
}

func (r *adminRepoImpl) ListDiaries(ctx context.Context, opt biz.AdminDiaryListOption) ([]*biz.AdminDiary, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&MoodDiaryModel{}).Joins("LEFT JOIN users ON users.id = mood_diaries.user_id")
	if keyword := strings.TrimSpace(opt.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("mood_diaries.title LIKE ? OR mood_diaries.content LIKE ? OR users.username LIKE ?", like, like, like)
	}
	if opt.UserID > 0 {
		q = q.Where("mood_diaries.user_id = ?", opt.UserID)
	}
	if opt.Mood != "" {
		q = q.Where("mood_diaries.mood = ?", opt.Mood)
	}
	if opt.StartDate != "" {
		q = q.Where("mood_diaries.occurred_on >= ?", opt.StartDate)
	}
	if opt.EndDate != "" {
		q = q.Where("mood_diaries.occurred_on <= ?", opt.EndDate)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []MoodDiaryModel
	if err := q.Order("mood_diaries.occurred_on desc, mood_diaries.created_at desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminDiary, 0, len(models))
	for i := range models {
		item, err := r.adminFillDiary(ctx, &models[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (r *adminRepoImpl) GetDiary(ctx context.Context, id int64) (*biz.AdminDiary, error) {
	var model MoodDiaryModel
	err := r.db.WithContext(ctx).First(&model, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.adminFillDiary(ctx, &model)
}

func (r *adminRepoImpl) DeleteDiary(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&MoodDiaryModel{}, id).Error
}

func (r *adminRepoImpl) ListEmotionAnalyses(ctx context.Context, opt biz.AdminEmotionAnalysisListOption) ([]*biz.AdminEmotionAnalysis, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
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
	if err := q.Order("created_at desc, id desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminEmotionAnalysis, 0, len(models))
	for i := range models {
		item, err := r.adminFillEmotionAnalysis(ctx, &models[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (r *adminRepoImpl) GetEmotionAnalysis(ctx context.Context, id int64) (*biz.AdminEmotionAnalysis, error) {
	var model EmotionAnalysisModel
	err := r.db.WithContext(ctx).First(&model, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.adminFillEmotionAnalysis(ctx, &model)
}

func (r *adminRepoImpl) ListChatSessions(ctx context.Context, opt biz.AdminChatSessionListOption) ([]*biz.AdminChatSession, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
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
	if err := q.Order("COALESCE(last_message_at, created_at) desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminChatSession, 0, len(models))
	for i := range models {
		out = append(out, r.toAdminChatSession(ctx, &models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) ListChatMessages(ctx context.Context, opt biz.AdminChatMessageListOption) ([]*biz.AdminChatMessage, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&ChatMessageModel{}).Where("session_id = ?", opt.SessionID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []ChatMessageModel
	if err := q.Order("created_at asc, id asc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminChatMessage, 0, len(models))
	for i := range models {
		out = append(out, r.toAdminChatMessage(ctx, &models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) ListFiles(ctx context.Context, opt biz.AdminFileListOption) ([]*biz.AdminFileAsset, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
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
	if err := q.Order("created_at desc, id desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminFileAsset, 0, len(models))
	for i := range models {
		out = append(out, r.toAdminFile(ctx, &models[i]))
	}
	return out, total, nil
}

func (r *adminRepoImpl) GetFile(ctx context.Context, id int64) (*biz.AdminFileAsset, error) {
	var model FileAssetModel
	err := r.db.WithContext(ctx).First(&model, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toAdminFile(ctx, &model), nil
}

func (r *adminRepoImpl) DeleteFile(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&FileAssetModel{}, id).Error
}

func (r *adminRepoImpl) adminFillDiary(ctx context.Context, model *MoodDiaryModel) (*biz.AdminDiary, error) {
	var links []MoodDiaryTagModel
	if err := r.db.WithContext(ctx).Where("diary_id = ?", model.ID).Find(&links).Error; err != nil {
		return nil, err
	}
	tagIDs := make([]int64, 0, len(links))
	for _, link := range links {
		tagIDs = append(tagIDs, link.TagID)
	}
	tags := make([]*biz.AdminMoodTag, 0, len(tagIDs))
	if len(tagIDs) > 0 {
		var tagModels []MoodTagModel
		if err := r.db.WithContext(ctx).Where("id IN ?", tagIDs).Find(&tagModels).Error; err != nil {
			return nil, err
		}
		for i := range tagModels {
			tags = append(tags, toAdminMoodTag(&tagModels[i]))
		}
	}
	var attachments []MoodDiaryAttachmentModel
	if err := r.db.WithContext(ctx).Where("diary_id = ?", model.ID).Order("sort asc").Find(&attachments).Error; err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		urls = append(urls, attachment.URL)
	}
	return &biz.AdminDiary{
		ID:             model.ID,
		UserID:         model.UserID,
		Username:       r.usernameByID(ctx, model.UserID),
		Title:          model.Title,
		Content:        model.Content,
		Mood:           model.Mood,
		MoodScore:      model.MoodScore,
		Weather:        model.Weather,
		Location:       model.Location,
		OccurredOn:     model.OccurredOn,
		Visibility:     model.Visibility,
		Tags:           tags,
		AttachmentURLs: urls,
		AnalysisID:     model.AnalysisID,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}, nil
}

func (r *adminRepoImpl) adminFillEmotionAnalysis(ctx context.Context, model *EmotionAnalysisModel) (*biz.AdminEmotionAnalysis, error) {
	var dims []EmotionDimensionScoreModel
	if err := r.db.WithContext(ctx).Where("analysis_id = ?", model.ID).Find(&dims).Error; err != nil {
		return nil, err
	}
	outDims := make([]*biz.AdminEmotionDimensionScore, 0, len(dims))
	for i := range dims {
		outDims = append(outDims, &biz.AdminEmotionDimensionScore{Dimension: dims[i].Dimension, Score: dims[i].Score})
	}
	return &biz.AdminEmotionAnalysis{
		ID:                  model.ID,
		UserID:              model.UserID,
		Username:            r.usernameByID(ctx, model.UserID),
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

func (r *adminRepoImpl) toAdminChatSession(ctx context.Context, model *ChatSessionModel) *biz.AdminChatSession {
	var last time.Time
	if model.LastMessageAt != nil {
		last = *model.LastMessageAt
	}
	return &biz.AdminChatSession{
		ID:            model.ID,
		UserID:        model.UserID,
		Username:      r.usernameByID(ctx, model.UserID),
		Title:         model.Title,
		Scenario:      model.Scenario,
		Status:        model.Status,
		Summary:       model.Summary,
		MessageCount:  model.MessageCount,
		LastMessageAt: last,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

func (r *adminRepoImpl) toAdminChatMessage(ctx context.Context, model *ChatMessageModel) *biz.AdminChatMessage {
	return &biz.AdminChatMessage{
		ID:                  model.ID,
		SessionID:           model.SessionID,
		UserID:              model.UserID,
		Username:            r.usernameByID(ctx, model.UserID),
		Role:                model.Role,
		Content:             model.Content,
		ContentType:         model.ContentType,
		Model:               model.Model,
		PromptTokens:        model.PromptTokens,
		CompletionTokens:    model.CompletionTokens,
		TotalTokens:         model.TotalTokens,
		LatencyMS:           model.LatencyMS,
		EmotionSnapshotJSON: model.EmotionSnapshotJSON,
		SafetyResultJSON:    model.SafetyResultJSON,
		Status:              model.Status,
		ErrorMessage:        model.ErrorMessage,
		CreatedAt:           model.CreatedAt,
	}
}

func (r *adminRepoImpl) toAdminFile(ctx context.Context, model *FileAssetModel) *biz.AdminFileAsset {
	return &biz.AdminFileAsset{
		ID:              model.ID,
		OwnerUserID:     model.OwnerUserID,
		Username:        r.usernameByID(ctx, model.OwnerUserID),
		BizType:         model.BizType,
		StorageProvider: model.StorageProvider,
		Bucket:          model.Bucket,
		ObjectKey:       model.ObjectKey,
		URL:             model.URL,
		MimeType:        model.MimeType,
		SizeBytes:       model.SizeBytes,
		Checksum:        model.Checksum,
		Status:          model.Status,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

func (r *adminRepoImpl) usernameByID(ctx context.Context, userID int64) string {
	if userID <= 0 {
		return ""
	}
	var user UserModel
	if err := r.db.WithContext(ctx).Select("username").First(&user, userID).Error; err != nil {
		return ""
	}
	return user.Username
}

func toAdminUser(model *UserModel) *biz.AdminUser {
	var lastLoginAt time.Time
	if model.LastLoginAt != nil {
		lastLoginAt = *model.LastLoginAt
	}
	return &biz.AdminUser{
		UserID:      model.ID,
		Username:    model.Username,
		Phone:       model.Phone,
		Email:       model.Email,
		Avatar:      model.Avatar,
		Roles:       rolesFromJSON(model.Roles),
		Status:      model.Status,
		LastLoginAt: lastLoginAt,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

func toAdminProfile(model *UserProfileModel) *biz.AdminUserProfile {
	if model == nil || model.ID == 0 {
		return &biz.AdminUserProfile{}
	}
	return &biz.AdminUserProfile{
		Nickname:   model.Nickname,
		AvatarURL:  model.AvatarURL,
		Gender:     model.Gender,
		Birthday:   model.Birthday,
		Bio:        model.Bio,
		Location:   model.Location,
		Occupation: model.Occupation,
		Industry:   model.Industry,
		Language:   model.Language,
		Timezone:   model.Timezone,
	}
}

func toAdminSystemConfig(model *SystemConfigModel) *biz.AdminSystemConfig {
	return &biz.AdminSystemConfig{
		ID:          model.ID,
		Key:         model.ConfigKey,
		ValueJSON:   model.ConfigValue,
		Description: model.Description,
		IsPublic:    model.IsPublic,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

func toAdminAnnouncement(model *SystemAnnouncementModel) *biz.AdminAnnouncement {
	var startAt, endAt time.Time
	if model.StartAt != nil {
		startAt = *model.StartAt
	}
	if model.EndAt != nil {
		endAt = *model.EndAt
	}
	return &biz.AdminAnnouncement{
		ID:             model.ID,
		Title:          model.Title,
		Content:        model.Content,
		TargetPlatform: model.TargetPlatform,
		StartAt:        startAt,
		EndAt:          endAt,
		Status:         model.Status,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}
}

func toAdminAppVersion(model *AppVersionModel) *biz.AdminAppVersion {
	var publishedAt time.Time
	if model.PublishedAt != nil {
		publishedAt = *model.PublishedAt
	}
	return &biz.AdminAppVersion{
		ID:                  model.ID,
		Platform:            model.Platform,
		Version:             model.Version,
		BuildNo:             model.BuildNo,
		ForceUpdate:         model.ForceUpdate,
		DownloadURL:         model.DownloadURL,
		Changelog:           model.Changelog,
		MinSupportedVersion: model.MinSupportedVersion,
		PublishedAt:         publishedAt,
		Status:              model.Status,
		CreatedAt:           model.CreatedAt,
		UpdatedAt:           model.UpdatedAt,
	}
}

func toAdminMoodTag(model *MoodTagModel) *biz.AdminMoodTag {
	return &biz.AdminMoodTag{
		ID:        model.ID,
		Name:      model.Name,
		Color:     model.Color,
		Icon:      model.Icon,
		Sort:      model.Sort,
		System:    model.UserID == 0,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func normalizeAdminDateRange(startDate, endDate string) (time.Time, time.Time) {
	now := time.Now()
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		end = now
	}
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		start = end.AddDate(0, 0, -6)
	}
	if start.After(end) {
		start, end = end, start
	}
	return start, end
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
