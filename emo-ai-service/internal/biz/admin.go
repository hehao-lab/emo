package biz

import (
	"context"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type AdminPageOption struct {
	Page     int32
	PageSize int32
}

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

type AdminUserListOption struct {
	AdminPageOption
	Keyword string
	Status  int32
	Role    string
}

type AdminUser struct {
	UserID      int64
	Username    string
	Phone       string
	Email       string
	Avatar      string
	Roles       []string
	Status      int32
	LastLoginAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AdminUserProfile struct {
	Nickname   string
	AvatarURL  string
	Gender     string
	Birthday   string
	Bio        string
	Location   string
	Occupation string
	Industry   string
	Language   string
	Timezone   string
}

type AdminUserDetail struct {
	User    *AdminUser
	Profile *AdminUserProfile
}

type AdminSystemConfigListOption struct {
	AdminPageOption
	Keyword  string
	IsPublic *bool
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

type AdminAnnouncementListOption struct {
	AdminPageOption
	Platform string
	Status   *int32
	Keyword  string
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

type AdminAppVersionListOption struct {
	AdminPageOption
	Platform string
	Status   *int32
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
	Status              int32
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type AdminMoodTagListOption struct {
	AdminPageOption
	Keyword string
}

type AdminMoodTag struct {
	ID        int64
	Name      string
	Color     string
	Icon      string
	Sort      int32
	System    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AdminDiaryListOption struct {
	AdminPageOption
	Keyword   string
	UserID    int64
	Mood      string
	StartDate string
	EndDate   string
}

type AdminDiary struct {
	ID             int64
	UserID         int64
	Username       string
	Title          string
	Content        string
	Mood           string
	MoodScore      int32
	Weather        string
	Location       string
	OccurredOn     string
	Visibility     string
	Tags           []*AdminMoodTag
	AttachmentURLs []string
	AnalysisID     int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AdminEmotionAnalysisListOption struct {
	AdminPageOption
	UserID     int64
	RiskLevel  string
	SourceType string
	StartDate  string
	EndDate    string
}

type AdminEmotionDimensionScore struct {
	Dimension string
	Score     float64
}

type AdminEmotionAnalysis struct {
	ID                  int64
	UserID              int64
	Username            string
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
	Dimensions          []*AdminEmotionDimensionScore
	RawResultJSON       string
	CreatedAt           time.Time
}

type AdminChatSessionListOption struct {
	AdminPageOption
	UserID int64
	Status string
}

type AdminChatSession struct {
	ID            int64
	UserID        int64
	Username      string
	Title         string
	Scenario      string
	Status        string
	Summary       string
	MessageCount  int32
	LastMessageAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AdminChatMessageListOption struct {
	AdminPageOption
	SessionID int64
}

type AdminChatMessage struct {
	ID                  int64
	SessionID           int64
	UserID              int64
	Username            string
	Role                string
	Content             string
	ContentType         string
	Model               string
	PromptTokens        int32
	CompletionTokens    int32
	TotalTokens         int32
	LatencyMS           int32
	EmotionSnapshotJSON string
	SafetyResultJSON    string
	Status              string
	ErrorMessage        string
	CreatedAt           time.Time
}

type AdminFileListOption struct {
	AdminPageOption
	BizType     string
	OwnerUserID int64
}

type AdminFileAsset struct {
	ID              int64
	OwnerUserID     int64
	Username        string
	BizType         string
	StorageProvider string
	Bucket          string
	ObjectKey       string
	URL             string
	MimeType        string
	SizeBytes       int64
	Checksum        string
	Status          int32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type AdminRepo interface {
	DashboardOverview(ctx context.Context) (*AdminDashboardOverview, error)
	DashboardTrends(ctx context.Context, startDate, endDate string) ([]*AdminDashboardTrendPoint, error)

	ListUsers(ctx context.Context, opt AdminUserListOption) ([]*AdminUser, int64, error)
	GetUser(ctx context.Context, userID int64) (*AdminUserDetail, error)
	UpdateUserStatus(ctx context.Context, userID int64, status int32) (*AdminUser, error)
	UpdateUserRoles(ctx context.Context, userID int64, roles []string) (*AdminUser, error)

	ListSystemConfigs(ctx context.Context, opt AdminSystemConfigListOption) ([]*AdminSystemConfig, int64, error)
	CreateSystemConfig(ctx context.Context, item *AdminSystemConfig) (*AdminSystemConfig, error)
	UpdateSystemConfig(ctx context.Context, item *AdminSystemConfig, updateIsPublic bool) (*AdminSystemConfig, error)
	DeleteSystemConfig(ctx context.Context, id int64) error

	ListAnnouncements(ctx context.Context, opt AdminAnnouncementListOption) ([]*AdminAnnouncement, int64, error)
	CreateAnnouncement(ctx context.Context, item *AdminAnnouncement) (*AdminAnnouncement, error)
	UpdateAnnouncement(ctx context.Context, item *AdminAnnouncement, updateStatus bool) (*AdminAnnouncement, error)
	DeleteAnnouncement(ctx context.Context, id int64) error

	ListAppVersions(ctx context.Context, opt AdminAppVersionListOption) ([]*AdminAppVersion, int64, error)
	CreateAppVersion(ctx context.Context, item *AdminAppVersion) (*AdminAppVersion, error)
	UpdateAppVersion(ctx context.Context, item *AdminAppVersion, updateForceUpdate bool, updateStatus bool) (*AdminAppVersion, error)
	DeleteAppVersion(ctx context.Context, id int64) error

	ListMoodTags(ctx context.Context, opt AdminMoodTagListOption) ([]*AdminMoodTag, int64, error)
	CreateMoodTag(ctx context.Context, item *AdminMoodTag) (*AdminMoodTag, error)
	UpdateMoodTag(ctx context.Context, item *AdminMoodTag) (*AdminMoodTag, error)
	DeleteMoodTag(ctx context.Context, id int64) error

	ListDiaries(ctx context.Context, opt AdminDiaryListOption) ([]*AdminDiary, int64, error)
	GetDiary(ctx context.Context, id int64) (*AdminDiary, error)
	DeleteDiary(ctx context.Context, id int64) error

	ListEmotionAnalyses(ctx context.Context, opt AdminEmotionAnalysisListOption) ([]*AdminEmotionAnalysis, int64, error)
	GetEmotionAnalysis(ctx context.Context, id int64) (*AdminEmotionAnalysis, error)

	ListChatSessions(ctx context.Context, opt AdminChatSessionListOption) ([]*AdminChatSession, int64, error)
	ListChatMessages(ctx context.Context, opt AdminChatMessageListOption) ([]*AdminChatMessage, int64, error)

	ListFiles(ctx context.Context, opt AdminFileListOption) ([]*AdminFileAsset, int64, error)
	GetFile(ctx context.Context, id int64) (*AdminFileAsset, error)
	DeleteFile(ctx context.Context, id int64) error
}

type AdminUsecase struct {
	repo AdminRepo
}

func NewAdminUsecase(repo AdminRepo) *AdminUsecase {
	return &AdminUsecase{repo: repo}
}

func (uc *AdminUsecase) DashboardOverview(ctx context.Context) (*AdminDashboardOverview, error) {
	return uc.repo.DashboardOverview(ctx)
}

func (uc *AdminUsecase) DashboardTrends(ctx context.Context, startDate, endDate string) ([]*AdminDashboardTrendPoint, error) {
	return uc.repo.DashboardTrends(ctx, startDate, endDate)
}

func (uc *AdminUsecase) ListUsers(ctx context.Context, opt AdminUserListOption) ([]*AdminUser, int64, error) {
	return uc.repo.ListUsers(ctx, opt)
}

func (uc *AdminUsecase) GetUser(ctx context.Context, userID int64) (*AdminUserDetail, error) {
	if userID <= 0 {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "user_id is required")
	}
	return uc.repo.GetUser(ctx, userID)
}

func (uc *AdminUsecase) UpdateUserStatus(ctx context.Context, userID int64, status int32) (*AdminUser, error) {
	if userID <= 0 || status <= 0 {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "user_id and status are required")
	}
	return uc.repo.UpdateUserStatus(ctx, userID, status)
}

func (uc *AdminUsecase) UpdateUserRoles(ctx context.Context, userID int64, roles []string) (*AdminUser, error) {
	if userID <= 0 || len(roles) == 0 {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "user_id and roles are required")
	}
	return uc.repo.UpdateUserRoles(ctx, userID, normalizeRoles(roles))
}

func (uc *AdminUsecase) ListSystemConfigs(ctx context.Context, opt AdminSystemConfigListOption) ([]*AdminSystemConfig, int64, error) {
	return uc.repo.ListSystemConfigs(ctx, opt)
}

func (uc *AdminUsecase) CreateSystemConfig(ctx context.Context, item *AdminSystemConfig) (*AdminSystemConfig, error) {
	if item == nil || strings.TrimSpace(item.Key) == "" || strings.TrimSpace(item.ValueJSON) == "" {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "key and value_json are required")
	}
	return uc.repo.CreateSystemConfig(ctx, item)
}

func (uc *AdminUsecase) UpdateSystemConfig(ctx context.Context, item *AdminSystemConfig, updateIsPublic bool) (*AdminSystemConfig, error) {
	if item == nil || item.ID <= 0 {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "id is required")
	}
	return uc.repo.UpdateSystemConfig(ctx, item, updateIsPublic)
}

func (uc *AdminUsecase) DeleteSystemConfig(ctx context.Context, id int64) error {
	return uc.repo.DeleteSystemConfig(ctx, id)
}

func (uc *AdminUsecase) ListAnnouncements(ctx context.Context, opt AdminAnnouncementListOption) ([]*AdminAnnouncement, int64, error) {
	return uc.repo.ListAnnouncements(ctx, opt)
}

func (uc *AdminUsecase) CreateAnnouncement(ctx context.Context, item *AdminAnnouncement) (*AdminAnnouncement, error) {
	if item == nil || strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Content) == "" {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "title and content are required")
	}
	if item.TargetPlatform == "" {
		item.TargetPlatform = "all"
	}
	if item.Status == 0 {
		item.Status = 1
	}
	return uc.repo.CreateAnnouncement(ctx, item)
}

func (uc *AdminUsecase) UpdateAnnouncement(ctx context.Context, item *AdminAnnouncement, updateStatus bool) (*AdminAnnouncement, error) {
	if item == nil || item.ID <= 0 {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "id is required")
	}
	return uc.repo.UpdateAnnouncement(ctx, item, updateStatus)
}

func (uc *AdminUsecase) DeleteAnnouncement(ctx context.Context, id int64) error {
	return uc.repo.DeleteAnnouncement(ctx, id)
}

func (uc *AdminUsecase) ListAppVersions(ctx context.Context, opt AdminAppVersionListOption) ([]*AdminAppVersion, int64, error) {
	return uc.repo.ListAppVersions(ctx, opt)
}

func (uc *AdminUsecase) CreateAppVersion(ctx context.Context, item *AdminAppVersion) (*AdminAppVersion, error) {
	if item == nil || strings.TrimSpace(item.Platform) == "" || strings.TrimSpace(item.Version) == "" || item.BuildNo <= 0 {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "platform, version and build_no are required")
	}
	if item.Status == 0 {
		item.Status = 1
	}
	return uc.repo.CreateAppVersion(ctx, item)
}

func (uc *AdminUsecase) UpdateAppVersion(ctx context.Context, item *AdminAppVersion, updateForceUpdate bool, updateStatus bool) (*AdminAppVersion, error) {
	if item == nil || item.ID <= 0 {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "id is required")
	}
	return uc.repo.UpdateAppVersion(ctx, item, updateForceUpdate, updateStatus)
}

func (uc *AdminUsecase) DeleteAppVersion(ctx context.Context, id int64) error {
	return uc.repo.DeleteAppVersion(ctx, id)
}

func (uc *AdminUsecase) ListMoodTags(ctx context.Context, opt AdminMoodTagListOption) ([]*AdminMoodTag, int64, error) {
	return uc.repo.ListMoodTags(ctx, opt)
}

func (uc *AdminUsecase) CreateMoodTag(ctx context.Context, item *AdminMoodTag) (*AdminMoodTag, error) {
	if item == nil || strings.TrimSpace(item.Name) == "" {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "name is required")
	}
	return uc.repo.CreateMoodTag(ctx, item)
}

func (uc *AdminUsecase) UpdateMoodTag(ctx context.Context, item *AdminMoodTag) (*AdminMoodTag, error) {
	if item == nil || item.ID <= 0 {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "id is required")
	}
	return uc.repo.UpdateMoodTag(ctx, item)
}

func (uc *AdminUsecase) DeleteMoodTag(ctx context.Context, id int64) error {
	return uc.repo.DeleteMoodTag(ctx, id)
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

func (uc *AdminUsecase) ListEmotionAnalyses(ctx context.Context, opt AdminEmotionAnalysisListOption) ([]*AdminEmotionAnalysis, int64, error) {
	return uc.repo.ListEmotionAnalyses(ctx, opt)
}

func (uc *AdminUsecase) GetEmotionAnalysis(ctx context.Context, id int64) (*AdminEmotionAnalysis, error) {
	return uc.repo.GetEmotionAnalysis(ctx, id)
}

func (uc *AdminUsecase) ListChatSessions(ctx context.Context, opt AdminChatSessionListOption) ([]*AdminChatSession, int64, error) {
	return uc.repo.ListChatSessions(ctx, opt)
}

func (uc *AdminUsecase) ListChatMessages(ctx context.Context, opt AdminChatMessageListOption) ([]*AdminChatMessage, int64, error) {
	if opt.SessionID <= 0 {
		return nil, 0, kerrors.BadRequest("INVALID_ARGUMENT", "session_id is required")
	}
	return uc.repo.ListChatMessages(ctx, opt)
}

func (uc *AdminUsecase) ListFiles(ctx context.Context, opt AdminFileListOption) ([]*AdminFileAsset, int64, error) {
	return uc.repo.ListFiles(ctx, opt)
}

func (uc *AdminUsecase) GetFile(ctx context.Context, id int64) (*AdminFileAsset, error) {
	return uc.repo.GetFile(ctx, id)
}

func (uc *AdminUsecase) DeleteFile(ctx context.Context, id int64) error {
	return uc.repo.DeleteFile(ctx, id)
}

func normalizeRoles(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		role := strings.TrimSpace(item)
		if role == "" || seen[role] {
			continue
		}
		seen[role] = true
		out = append(out, role)
	}
	return out
}
