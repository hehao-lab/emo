package biz

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"golang.org/x/crypto/bcrypt"
)

type AdminDashboardOverview struct {
	UserCount             int64
	TodayNewUsers         int64
	DiaryCount            int64
	TodayDiaries          int64
	ChatSessionCount      int64
	TodayChatMessages     int64
	EmotionAnalysisCount  int64
	HighRiskAnalysisCount int64
}

type AdminDashboardTrendPoint struct {
	Date            string
	NewUsers        int64
	Diaries         int64
	ChatMessages    int64
	EmotionAnalyses int64
}

type AdminUserProfile struct {
	Nickname   string
	Gender     string
	Birthday   string
	Bio        string
	Location   string
	Occupation string
	Industry   string
	Language   string
	Timezone   string
}

type AdminUser struct {
	ID          int64
	Username    string
	Phone       string
	Email       string
	Avatar      string
	Roles       []string
	Status      int32
	LastLoginAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Profile     *AdminUserProfile
}

type AdminDiary struct {
	*MoodDiary
	Username string
}

type AdminEmotionAnalysis struct {
	*EmotionAnalysis
	Username string
}

type AdminChatSession struct {
	*ChatSession
	Username string
}

type AdminLoginLog struct {
	*LoginLog
	Username string
}

type AdminSecurityEvent struct {
	*SecurityEvent
	Username string
}

type AdminSystemConfig struct {
	ID          int64
	Key         string
	ValueJSON   string
	Description string
	IsPublic    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AdminAnnouncement struct {
	ID             int64
	Title          string
	Content        string
	TargetPlatform string
	StartAt        time.Time
	EndAt          time.Time
	Status         int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AdminAppVersion struct {
	ID                  int64
	Platform            string
	Version             string
	BuildNo             int32
	ForceUpdate         bool
	DownloadURL         string
	Changelog           string
	MinSupportedVersion string
	PublishedAt         time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type AdminFile struct {
	*FileAsset
	Username string
}

type AdminUserListOption struct {
	Page, PageSize int32
	Keyword        string
	Status         int32
	Role           string
}

type AdminDiaryListOption struct {
	Page, PageSize int32
	Keyword        string
	UserID         int64
	Mood           string
	StartDate      string
	EndDate        string
}

type AdminEmotionListOption struct {
	Page, PageSize int32
	UserID         int64
	RiskLevel      string
	SourceType     string
	StartDate      string
	EndDate        string
}

type AdminChatListOption struct {
	Page, PageSize int32
	UserID         int64
	Status         string
}

type AdminLoginLogListOption struct {
	Page, PageSize int32
	UserID         int64
	SuccessOnly    bool
}

type AdminSecurityEventListOption struct {
	Page, PageSize int32
	UserID         int64
	RiskLevel      string
}

type AdminConfigListOption struct {
	Page, PageSize int32
	Keyword        string
}

type AdminAnnouncementListOption struct {
	Page, PageSize int32
	Platform       string
	Status         *int32
}

type AdminVersionListOption struct {
	Page, PageSize int32
	Platform       string
}

type AdminMoodTagListOption struct {
	Page, PageSize int32
	Keyword        string
}

type AdminFileListOption struct {
	Page, PageSize int32
	BizType        string
	OwnerUserID    int64
}

type AdminRepo interface {
	GetDashboardOverview(context.Context, time.Time) (*AdminDashboardOverview, error)
	GetDashboardTrends(context.Context, time.Time, time.Time) ([]*AdminDashboardTrendPoint, error)
	ListUsers(context.Context, AdminUserListOption) ([]*AdminUser, int64, error)
	GetUser(context.Context, int64) (*AdminUser, error)
	UpdateUserStatus(context.Context, int64, int32, string) (*AdminUser, error)
	UpdateUserRoles(context.Context, int64, []string) (*AdminUser, error)
	UpdateUserPassword(context.Context, int64, string) error
	ListDiaries(context.Context, AdminDiaryListOption) ([]*AdminDiary, int64, error)
	GetDiary(context.Context, int64) (*AdminDiary, error)
	DeleteDiary(context.Context, int64) error
	ListEmotionAnalyses(context.Context, AdminEmotionListOption) ([]*AdminEmotionAnalysis, int64, error)
	GetEmotionAnalysis(context.Context, int64) (*AdminEmotionAnalysis, error)
	ListChatSessions(context.Context, AdminChatListOption) ([]*AdminChatSession, int64, error)
	ListChatMessages(context.Context, int64, int32, int32) ([]*ChatMessage, int64, error)
	ListLoginLogs(context.Context, AdminLoginLogListOption) ([]*AdminLoginLog, int64, error)
	ListSecurityEvents(context.Context, AdminSecurityEventListOption) ([]*AdminSecurityEvent, int64, error)
	ListConfigs(context.Context, AdminConfigListOption) ([]*AdminSystemConfig, int64, error)
	CreateConfig(context.Context, *AdminSystemConfig) (*AdminSystemConfig, error)
	UpdateConfig(context.Context, *AdminSystemConfig) (*AdminSystemConfig, error)
	DeleteConfig(context.Context, int64) error
	ListAnnouncements(context.Context, AdminAnnouncementListOption) ([]*AdminAnnouncement, int64, error)
	CreateAnnouncement(context.Context, *AdminAnnouncement) (*AdminAnnouncement, error)
	UpdateAnnouncement(context.Context, *AdminAnnouncement) (*AdminAnnouncement, error)
	DeleteAnnouncement(context.Context, int64) error
	ListVersions(context.Context, AdminVersionListOption) ([]*AdminAppVersion, int64, error)
	CreateVersion(context.Context, *AdminAppVersion) (*AdminAppVersion, error)
	UpdateVersion(context.Context, *AdminAppVersion) (*AdminAppVersion, error)
	DeleteVersion(context.Context, int64) error
	ListMoodTags(context.Context, AdminMoodTagListOption) ([]*MoodTag, int64, error)
	CreateMoodTag(context.Context, *MoodTag) (*MoodTag, error)
	UpdateMoodTag(context.Context, *MoodTag) (*MoodTag, error)
	DeleteMoodTag(context.Context, int64) error
	ListFiles(context.Context, AdminFileListOption) ([]*AdminFile, int64, error)
	GetFile(context.Context, int64) (*AdminFile, error)
	DeleteFile(context.Context, int64) error
}

type AdminUsecase struct {
	repo AdminRepo
}

func NewAdminUsecase(repo AdminRepo) *AdminUsecase { return &AdminUsecase{repo: repo} }

func (uc *AdminUsecase) GetDashboardOverview(ctx context.Context) (*AdminDashboardOverview, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return uc.repo.GetDashboardOverview(ctx, today)
}

func (uc *AdminUsecase) GetDashboardTrends(ctx context.Context, startDate, endDate string) ([]*AdminDashboardTrendPoint, error) {
	end := time.Now()
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	start := end.AddDate(0, 0, -6)
	var err error
	if strings.TrimSpace(startDate) != "" {
		start, err = time.ParseInLocation("2006-01-02", startDate, time.Local)
		if err != nil {
			return nil, badAdminRequest("start_date must use YYYY-MM-DD")
		}
	}
	if strings.TrimSpace(endDate) != "" {
		end, err = time.ParseInLocation("2006-01-02", endDate, time.Local)
		if err != nil {
			return nil, badAdminRequest("end_date must use YYYY-MM-DD")
		}
	}
	if end.Before(start) || end.Sub(start) > 366*24*time.Hour {
		return nil, badAdminRequest("date range must be between 1 and 366 days")
	}
	return uc.repo.GetDashboardTrends(ctx, start, end)
}

func (uc *AdminUsecase) ListUsers(ctx context.Context, opt AdminUserListOption) ([]*AdminUser, int64, error) {
	return uc.repo.ListUsers(ctx, opt)
}

func (uc *AdminUsecase) GetUser(ctx context.Context, userID int64) (*AdminUser, error) {
	return uc.repo.GetUser(ctx, userID)
}

func (uc *AdminUsecase) UpdateUserStatus(ctx context.Context, actorID, userID int64, status int32, reason string) (*AdminUser, error) {
	if status < 1 || status > 3 {
		return nil, badAdminRequest("status must be 1, 2, or 3")
	}
	if actorID == userID && status != 1 {
		return nil, kerrors.Forbidden("ADMIN_SELF_LOCK", "an administrator cannot freeze or close their own account")
	}
	return uc.repo.UpdateUserStatus(ctx, userID, status, strings.TrimSpace(reason))
}

func (uc *AdminUsecase) UpdateUserRoles(ctx context.Context, actorID, userID int64, roles []string) (*AdminUser, error) {
	roles = normalizeAdminRoles(roles)
	if len(roles) == 0 {
		return nil, badAdminRequest("at least one role is required")
	}
	if actorID == userID && !containsFold(roles, "admin") {
		return nil, kerrors.Forbidden("ADMIN_SELF_DEMOTE", "an administrator cannot remove their own admin role")
	}
	return uc.repo.UpdateUserRoles(ctx, userID, roles)
}

func (uc *AdminUsecase) UpdateUserPassword(ctx context.Context, userID int64, password string) error {
	if len(password) < 8 {
		return badAdminRequest("password must contain at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return uc.repo.UpdateUserPassword(ctx, userID, string(hash))
}

func (uc *AdminUsecase) ListDiaries(ctx context.Context, opt AdminDiaryListOption) ([]*AdminDiary, int64, error) {
	return uc.repo.ListDiaries(ctx, opt)
}
func (uc *AdminUsecase) GetDiary(ctx context.Context, id int64) (*AdminDiary, error) {
	return uc.repo.GetDiary(ctx, id)
}
func (uc *AdminUsecase) DeleteDiary(ctx context.Context, id int64) error {
	return uc.repo.DeleteDiary(ctx, id)
}
func (uc *AdminUsecase) ListEmotionAnalyses(ctx context.Context, opt AdminEmotionListOption) ([]*AdminEmotionAnalysis, int64, error) {
	return uc.repo.ListEmotionAnalyses(ctx, opt)
}
func (uc *AdminUsecase) GetEmotionAnalysis(ctx context.Context, id int64) (*AdminEmotionAnalysis, error) {
	return uc.repo.GetEmotionAnalysis(ctx, id)
}
func (uc *AdminUsecase) ListChatSessions(ctx context.Context, opt AdminChatListOption) ([]*AdminChatSession, int64, error) {
	return uc.repo.ListChatSessions(ctx, opt)
}
func (uc *AdminUsecase) ListChatMessages(ctx context.Context, sessionID int64, page, pageSize int32) ([]*ChatMessage, int64, error) {
	return uc.repo.ListChatMessages(ctx, sessionID, page, pageSize)
}
func (uc *AdminUsecase) ListLoginLogs(ctx context.Context, opt AdminLoginLogListOption) ([]*AdminLoginLog, int64, error) {
	return uc.repo.ListLoginLogs(ctx, opt)
}
func (uc *AdminUsecase) ListSecurityEvents(ctx context.Context, opt AdminSecurityEventListOption) ([]*AdminSecurityEvent, int64, error) {
	return uc.repo.ListSecurityEvents(ctx, opt)
}
func (uc *AdminUsecase) ListConfigs(ctx context.Context, opt AdminConfigListOption) ([]*AdminSystemConfig, int64, error) {
	return uc.repo.ListConfigs(ctx, opt)
}

func (uc *AdminUsecase) CreateConfig(ctx context.Context, config *AdminSystemConfig) (*AdminSystemConfig, error) {
	if err := validateAdminConfig(config, true); err != nil {
		return nil, err
	}
	return uc.repo.CreateConfig(ctx, config)
}
func (uc *AdminUsecase) UpdateConfig(ctx context.Context, config *AdminSystemConfig) (*AdminSystemConfig, error) {
	if err := validateAdminConfig(config, false); err != nil {
		return nil, err
	}
	return uc.repo.UpdateConfig(ctx, config)
}
func (uc *AdminUsecase) DeleteConfig(ctx context.Context, id int64) error {
	return uc.repo.DeleteConfig(ctx, id)
}
func (uc *AdminUsecase) ListAnnouncements(ctx context.Context, opt AdminAnnouncementListOption) ([]*AdminAnnouncement, int64, error) {
	return uc.repo.ListAnnouncements(ctx, opt)
}
func (uc *AdminUsecase) CreateAnnouncement(ctx context.Context, item *AdminAnnouncement) (*AdminAnnouncement, error) {
	if err := validateAdminAnnouncement(item); err != nil {
		return nil, err
	}
	return uc.repo.CreateAnnouncement(ctx, item)
}
func (uc *AdminUsecase) UpdateAnnouncement(ctx context.Context, item *AdminAnnouncement) (*AdminAnnouncement, error) {
	if err := validateAdminAnnouncement(item); err != nil {
		return nil, err
	}
	return uc.repo.UpdateAnnouncement(ctx, item)
}
func (uc *AdminUsecase) DeleteAnnouncement(ctx context.Context, id int64) error {
	return uc.repo.DeleteAnnouncement(ctx, id)
}
func (uc *AdminUsecase) ListVersions(ctx context.Context, opt AdminVersionListOption) ([]*AdminAppVersion, int64, error) {
	return uc.repo.ListVersions(ctx, opt)
}
func (uc *AdminUsecase) CreateVersion(ctx context.Context, item *AdminAppVersion) (*AdminAppVersion, error) {
	if err := validateAdminVersion(item); err != nil {
		return nil, err
	}
	return uc.repo.CreateVersion(ctx, item)
}
func (uc *AdminUsecase) UpdateVersion(ctx context.Context, item *AdminAppVersion) (*AdminAppVersion, error) {
	if err := validateAdminVersion(item); err != nil {
		return nil, err
	}
	return uc.repo.UpdateVersion(ctx, item)
}
func (uc *AdminUsecase) DeleteVersion(ctx context.Context, id int64) error {
	return uc.repo.DeleteVersion(ctx, id)
}
func (uc *AdminUsecase) ListMoodTags(ctx context.Context, opt AdminMoodTagListOption) ([]*MoodTag, int64, error) {
	return uc.repo.ListMoodTags(ctx, opt)
}
func (uc *AdminUsecase) CreateMoodTag(ctx context.Context, tag *MoodTag) (*MoodTag, error) {
	if strings.TrimSpace(tag.Name) == "" {
		return nil, badAdminRequest("tag name is required")
	}
	tag.UserID, tag.System, tag.Name = 0, true, strings.TrimSpace(tag.Name)
	return uc.repo.CreateMoodTag(ctx, tag)
}
func (uc *AdminUsecase) UpdateMoodTag(ctx context.Context, tag *MoodTag) (*MoodTag, error) {
	if strings.TrimSpace(tag.Name) == "" {
		return nil, badAdminRequest("tag name is required")
	}
	tag.UserID, tag.System, tag.Name = 0, true, strings.TrimSpace(tag.Name)
	return uc.repo.UpdateMoodTag(ctx, tag)
}
func (uc *AdminUsecase) DeleteMoodTag(ctx context.Context, id int64) error {
	return uc.repo.DeleteMoodTag(ctx, id)
}
func (uc *AdminUsecase) ListFiles(ctx context.Context, opt AdminFileListOption) ([]*AdminFile, int64, error) {
	return uc.repo.ListFiles(ctx, opt)
}
func (uc *AdminUsecase) GetFile(ctx context.Context, id int64) (*AdminFile, error) {
	return uc.repo.GetFile(ctx, id)
}
func (uc *AdminUsecase) DeleteFile(ctx context.Context, id int64) error {
	return uc.repo.DeleteFile(ctx, id)
}

func validateAdminConfig(config *AdminSystemConfig, requireKey bool) error {
	if config == nil || (requireKey && strings.TrimSpace(config.Key) == "") {
		return badAdminRequest("config key is required")
	}
	if !json.Valid([]byte(config.ValueJSON)) {
		return badAdminRequest("value_json must be valid JSON")
	}
	config.Key = strings.TrimSpace(config.Key)
	return nil
}

func validateAdminAnnouncement(item *AdminAnnouncement) error {
	if item == nil || strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Content) == "" {
		return badAdminRequest("announcement title and content are required")
	}
	item.Title, item.Content = strings.TrimSpace(item.Title), strings.TrimSpace(item.Content)
	item.TargetPlatform = strings.ToLower(strings.TrimSpace(item.TargetPlatform))
	if item.TargetPlatform == "" {
		item.TargetPlatform = "all"
	}
	if !containsFold([]string{"all", "ios", "android", "web"}, item.TargetPlatform) {
		return badAdminRequest("target_platform must be all, ios, android, or web")
	}
	if item.Status != 0 && item.Status != 1 {
		return badAdminRequest("status must be 0 or 1")
	}
	if !item.StartAt.IsZero() && !item.EndAt.IsZero() && item.EndAt.Before(item.StartAt) {
		return badAdminRequest("end_at must not be before start_at")
	}
	return nil
}

func validateAdminVersion(item *AdminAppVersion) error {
	if item == nil {
		return badAdminRequest("version is required")
	}
	item.Platform = strings.ToLower(strings.TrimSpace(item.Platform))
	item.Version = strings.TrimSpace(item.Version)
	if !containsFold([]string{"ios", "android", "web"}, item.Platform) {
		return badAdminRequest("platform must be ios, android, or web")
	}
	if item.Version == "" || item.BuildNo <= 0 {
		return badAdminRequest("version and a positive build_no are required")
	}
	return nil
}

func normalizeAdminRoles(roles []string) []string {
	seen := make(map[string]struct{}, len(roles))
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.ToLower(strings.TrimSpace(role))
		if role == "" || !validRoleName(role) {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	return out
}

func validRoleName(role string) bool {
	for _, r := range role {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func badAdminRequest(message string) error {
	return kerrors.BadRequest("ADMIN_INVALID_ARGUMENT", message)
}

func AdminNotFound(resource string) error {
	return kerrors.NotFound("ADMIN_NOT_FOUND", resource+" not found")
}
