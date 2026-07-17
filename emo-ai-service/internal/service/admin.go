package service

import (
	"context"
	"time"

	v1 "emo-ai-service/api/admin/v1"
	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AdminService struct {
	uc *biz.AdminUsecase
}

func NewAdminService(uc *biz.AdminUsecase) *AdminService {
	return &AdminService{uc: uc}
}

var _ v1.AdminServiceHTTPServer = (*AdminService)(nil)

func requireAdmin(ctx context.Context) (int64, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return 0, err
	}
	if !auth.HasRole(ctx, "admin") {
		return 0, kerrors.New(403, "PERMISSION_DENIED", "admin permission required")
	}
	return userID, nil
}

func (s *AdminService) GetDashboardOverview(ctx context.Context, req *v1.GetDashboardOverviewRequest) (*v1.DashboardOverview, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	out, err := s.uc.DashboardOverview(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.DashboardOverview{
		UserCount:             out.UserCount,
		TodayNewUsers:         out.TodayNewUsers,
		DiaryCount:            out.DiaryCount,
		TodayDiaries:          out.TodayDiaries,
		ChatSessionCount:      out.ChatSessionCount,
		TodayChatMessages:     out.TodayChatMessages,
		EmotionAnalysisCount:  out.EmotionAnalysisCount,
		HighRiskAnalysisCount: out.HighRiskAnalysisCount,
	}, nil
}

func (s *AdminService) GetDashboardTrends(ctx context.Context, req *v1.GetDashboardTrendsRequest) (*v1.DashboardTrends, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, err := s.uc.DashboardTrends(ctx, req.GetStartDate(), req.GetEndDate())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.DashboardTrendPoint, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.DashboardTrendPoint{
			Date:            item.Date,
			NewUsers:        item.NewUsers,
			Diaries:         item.Diaries,
			ChatMessages:    item.ChatMessages,
			EmotionAnalyses: item.EmotionAnalyses,
		})
	}
	return &v1.DashboardTrends{Points: out}, nil
}

func (s *AdminService) ListUsers(ctx context.Context, req *v1.ListUsersRequest) (*v1.ListUsersResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListUsers(ctx, biz.AdminUserListOption{AdminPageOption: pageOption(req.GetPage(), req.GetPageSize()), Keyword: req.GetKeyword(), Status: req.GetStatus(), Role: req.GetRole()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminUser, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminUserDTO(item))
	}
	return &v1.ListUsersResponse{Users: out, Total: total}, nil
}

func (s *AdminService) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.AdminUserDetail, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.GetUser(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, kerrors.NotFound("USER_NOT_FOUND", "user not found")
	}
	return &v1.AdminUserDetail{User: toAdminUserDTO(item.User), Profile: toAdminUserProfileDTO(item.Profile)}, nil
}

func (s *AdminService) UpdateUserStatus(ctx context.Context, req *v1.UpdateUserStatusRequest) (*v1.AdminUser, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateUserStatus(ctx, req.GetUserId(), req.GetStatus())
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, kerrors.NotFound("USER_NOT_FOUND", "user not found")
	}
	return toAdminUserDTO(item), nil
}

func (s *AdminService) UpdateUserRoles(ctx context.Context, req *v1.UpdateUserRolesRequest) (*v1.AdminUser, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateUserRoles(ctx, req.GetUserId(), req.GetRoles())
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, kerrors.NotFound("USER_NOT_FOUND", "user not found")
	}
	return toAdminUserDTO(item), nil
}

func (s *AdminService) ListSystemConfigs(ctx context.Context, req *v1.ListSystemConfigsRequest) (*v1.ListSystemConfigsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	var isPublic *bool
	if req.IsPublic != nil {
		v := req.GetIsPublic()
		isPublic = &v
	}
	items, total, err := s.uc.ListSystemConfigs(ctx, biz.AdminSystemConfigListOption{AdminPageOption: pageOption(req.GetPage(), req.GetPageSize()), Keyword: req.GetKeyword(), IsPublic: isPublic})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.SystemConfig, 0, len(items))
	for _, item := range items {
		out = append(out, toSystemConfigDTO(item))
	}
	return &v1.ListSystemConfigsResponse{Configs: out, Total: total}, nil
}

func (s *AdminService) CreateSystemConfig(ctx context.Context, req *v1.CreateSystemConfigRequest) (*v1.SystemConfig, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.CreateSystemConfig(ctx, &biz.AdminSystemConfig{Key: req.GetKey(), ValueJSON: req.GetValueJson(), Description: req.GetDescription(), IsPublic: req.GetIsPublic()})
	if err != nil {
		return nil, err
	}
	return toSystemConfigDTO(item), nil
}

func (s *AdminService) UpdateSystemConfig(ctx context.Context, req *v1.UpdateSystemConfigRequest) (*v1.SystemConfig, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateSystemConfig(ctx, &biz.AdminSystemConfig{ID: req.GetId(), Key: req.GetKey(), ValueJSON: req.GetValueJson(), Description: req.GetDescription(), IsPublic: req.GetIsPublic()}, req.IsPublic != nil)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, kerrors.NotFound("CONFIG_NOT_FOUND", "system config not found")
	}
	return toSystemConfigDTO(item), nil
}

func (s *AdminService) DeleteSystemConfig(ctx context.Context, req *v1.DeleteSystemConfigRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.DeleteSystemConfig(ctx, req.GetId())
}

func (s *AdminService) ListAnnouncements(ctx context.Context, req *v1.ListAnnouncementsRequest) (*v1.ListAnnouncementsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	var status *int32
	if req.Status != nil {
		v := req.GetStatus()
		status = &v
	}
	items, total, err := s.uc.ListAnnouncements(ctx, biz.AdminAnnouncementListOption{AdminPageOption: pageOption(req.GetPage(), req.GetPageSize()), Platform: req.GetPlatform(), Status: status, Keyword: req.GetKeyword()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.SystemAnnouncement, 0, len(items))
	for _, item := range items {
		out = append(out, toAnnouncementDTO(item))
	}
	return &v1.ListAnnouncementsResponse{Announcements: out, Total: total}, nil
}

func (s *AdminService) CreateAnnouncement(ctx context.Context, req *v1.CreateAnnouncementRequest) (*v1.SystemAnnouncement, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.CreateAnnouncement(ctx, &biz.AdminAnnouncement{Title: req.GetTitle(), Content: req.GetContent(), TargetPlatform: req.GetTargetPlatform(), StartAt: timeFromUnix(req.GetStartAt()), EndAt: timeFromUnix(req.GetEndAt()), Status: req.GetStatus()})
	if err != nil {
		return nil, err
	}
	return toAnnouncementDTO(item), nil
}

func (s *AdminService) UpdateAnnouncement(ctx context.Context, req *v1.UpdateAnnouncementRequest) (*v1.SystemAnnouncement, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateAnnouncement(ctx, &biz.AdminAnnouncement{ID: req.GetId(), Title: req.GetTitle(), Content: req.GetContent(), TargetPlatform: req.GetTargetPlatform(), StartAt: timeFromUnix(req.GetStartAt()), EndAt: timeFromUnix(req.GetEndAt()), Status: req.GetStatus()}, req.Status != nil)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, kerrors.NotFound("ANNOUNCEMENT_NOT_FOUND", "announcement not found")
	}
	return toAnnouncementDTO(item), nil
}

func (s *AdminService) DeleteAnnouncement(ctx context.Context, req *v1.DeleteAnnouncementRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.DeleteAnnouncement(ctx, req.GetId())
}

func (s *AdminService) ListAppVersions(ctx context.Context, req *v1.ListAppVersionsRequest) (*v1.ListAppVersionsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	var status *int32
	if req.Status != nil {
		v := req.GetStatus()
		status = &v
	}
	items, total, err := s.uc.ListAppVersions(ctx, biz.AdminAppVersionListOption{AdminPageOption: pageOption(req.GetPage(), req.GetPageSize()), Platform: req.GetPlatform(), Status: status})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AppVersion, 0, len(items))
	for _, item := range items {
		out = append(out, toAppVersionDTO(item))
	}
	return &v1.ListAppVersionsResponse{Versions: out, Total: total}, nil
}

func (s *AdminService) CreateAppVersion(ctx context.Context, req *v1.CreateAppVersionRequest) (*v1.AppVersion, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.CreateAppVersion(ctx, &biz.AdminAppVersion{Platform: req.GetPlatform(), Version: req.GetVersion(), BuildNo: req.GetBuildNo(), ForceUpdate: req.GetForceUpdate(), DownloadURL: req.GetDownloadUrl(), Changelog: req.GetChangelog(), MinSupportedVersion: req.GetMinSupportedVersion(), PublishedAt: timeFromUnix(req.GetPublishedAt()), Status: req.GetStatus()})
	if err != nil {
		return nil, err
	}
	return toAppVersionDTO(item), nil
}

func (s *AdminService) UpdateAppVersion(ctx context.Context, req *v1.UpdateAppVersionRequest) (*v1.AppVersion, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateAppVersion(ctx, &biz.AdminAppVersion{ID: req.GetId(), Platform: req.GetPlatform(), Version: req.GetVersion(), BuildNo: req.GetBuildNo(), ForceUpdate: req.GetForceUpdate(), DownloadURL: req.GetDownloadUrl(), Changelog: req.GetChangelog(), MinSupportedVersion: req.GetMinSupportedVersion(), PublishedAt: timeFromUnix(req.GetPublishedAt()), Status: req.GetStatus()}, req.ForceUpdate != nil, req.Status != nil)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, kerrors.NotFound("VERSION_NOT_FOUND", "app version not found")
	}
	return toAppVersionDTO(item), nil
}

func (s *AdminService) DeleteAppVersion(ctx context.Context, req *v1.DeleteAppVersionRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.DeleteAppVersion(ctx, req.GetId())
}

func (s *AdminService) ListMoodTags(ctx context.Context, req *v1.ListMoodTagsRequest) (*v1.ListMoodTagsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListMoodTags(ctx, biz.AdminMoodTagListOption{AdminPageOption: pageOption(req.GetPage(), req.GetPageSize()), Keyword: req.GetKeyword()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.MoodTag, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminMoodTagDTO(item))
	}
	return &v1.ListMoodTagsResponse{Tags: out, Total: total}, nil
}

func (s *AdminService) CreateMoodTag(ctx context.Context, req *v1.CreateMoodTagRequest) (*v1.MoodTag, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.CreateMoodTag(ctx, &biz.AdminMoodTag{Name: req.GetName(), Color: req.GetColor(), Icon: req.GetIcon(), Sort: req.GetSort()})
	if err != nil {
		return nil, err
	}
	return toAdminMoodTagDTO(item), nil
}

func (s *AdminService) UpdateMoodTag(ctx context.Context, req *v1.UpdateMoodTagRequest) (*v1.MoodTag, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateMoodTag(ctx, &biz.AdminMoodTag{ID: req.GetId(), Name: req.GetName(), Color: req.GetColor(), Icon: req.GetIcon(), Sort: req.GetSort()})
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, kerrors.NotFound("TAG_NOT_FOUND", "mood tag not found")
	}
	return toAdminMoodTagDTO(item), nil
}

func (s *AdminService) DeleteMoodTag(ctx context.Context, req *v1.DeleteMoodTagRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.DeleteMoodTag(ctx, req.GetId())
}

func (s *AdminService) ListDiaries(ctx context.Context, req *v1.ListAdminDiariesRequest) (*v1.ListAdminDiariesResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListDiaries(ctx, biz.AdminDiaryListOption{AdminPageOption: pageOption(req.GetPage(), req.GetPageSize()), Keyword: req.GetKeyword(), UserID: req.GetUserId(), Mood: req.GetMood(), StartDate: req.GetStartDate(), EndDate: req.GetEndDate()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminDiary, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminDiaryDTO(item))
	}
	return &v1.ListAdminDiariesResponse{Diaries: out, Total: total}, nil
}

func (s *AdminService) GetDiary(ctx context.Context, req *v1.GetAdminDiaryRequest) (*v1.AdminDiary, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.GetDiary(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, kerrors.NotFound("DIARY_NOT_FOUND", "diary not found")
	}
	return toAdminDiaryDTO(item), nil
}

func (s *AdminService) DeleteDiary(ctx context.Context, req *v1.DeleteAdminDiaryRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.DeleteDiary(ctx, req.GetId())
}

func (s *AdminService) ListEmotionAnalyses(ctx context.Context, req *v1.ListAdminEmotionAnalysesRequest) (*v1.ListAdminEmotionAnalysesResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListEmotionAnalyses(ctx, biz.AdminEmotionAnalysisListOption{AdminPageOption: pageOption(req.GetPage(), req.GetPageSize()), UserID: req.GetUserId(), RiskLevel: req.GetRiskLevel(), SourceType: req.GetSourceType(), StartDate: req.GetStartDate(), EndDate: req.GetEndDate()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminEmotionAnalysis, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminEmotionAnalysisDTO(item))
	}
	return &v1.ListAdminEmotionAnalysesResponse{Analyses: out, Total: total}, nil
}

func (s *AdminService) GetEmotionAnalysis(ctx context.Context, req *v1.GetAdminEmotionAnalysisRequest) (*v1.AdminEmotionAnalysis, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.GetEmotionAnalysis(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, kerrors.NotFound("ANALYSIS_NOT_FOUND", "emotion analysis not found")
	}
	return toAdminEmotionAnalysisDTO(item), nil
}

func (s *AdminService) ListChatSessions(ctx context.Context, req *v1.ListAdminChatSessionsRequest) (*v1.ListAdminChatSessionsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListChatSessions(ctx, biz.AdminChatSessionListOption{AdminPageOption: pageOption(req.GetPage(), req.GetPageSize()), UserID: req.GetUserId(), Status: req.GetStatus()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminChatSession, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminChatSessionDTO(item))
	}
	return &v1.ListAdminChatSessionsResponse{Sessions: out, Total: total}, nil
}

func (s *AdminService) ListChatMessages(ctx context.Context, req *v1.ListAdminChatMessagesRequest) (*v1.ListAdminChatMessagesResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListChatMessages(ctx, biz.AdminChatMessageListOption{AdminPageOption: pageOption(req.GetPage(), req.GetPageSize()), SessionID: req.GetSessionId()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminChatMessage, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminChatMessageDTO(item))
	}
	return &v1.ListAdminChatMessagesResponse{Messages: out, Total: total}, nil
}

func (s *AdminService) ListFiles(ctx context.Context, req *v1.ListAdminFilesRequest) (*v1.ListAdminFilesResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListFiles(ctx, biz.AdminFileListOption{AdminPageOption: pageOption(req.GetPage(), req.GetPageSize()), BizType: req.GetBizType(), OwnerUserID: req.GetOwnerUserId()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminFileAsset, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminFileDTO(item))
	}
	return &v1.ListAdminFilesResponse{Files: out, Total: total}, nil
}

func (s *AdminService) GetFile(ctx context.Context, req *v1.GetAdminFileRequest) (*v1.AdminFileAsset, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.GetFile(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, kerrors.NotFound("FILE_NOT_FOUND", "file not found")
	}
	return toAdminFileDTO(item), nil
}

func (s *AdminService) DeleteFile(ctx context.Context, req *v1.DeleteAdminFileRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.DeleteFile(ctx, req.GetId())
}

func pageOption(page, pageSize int32) biz.AdminPageOption {
	return biz.AdminPageOption{Page: page, PageSize: pageSize}
}

func timeFromUnix(v int64) time.Time {
	if v <= 0 {
		return time.Time{}
	}
	return time.Unix(v, 0)
}

func adminUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func toAdminUserDTO(item *biz.AdminUser) *v1.AdminUser {
	if item == nil {
		return nil
	}
	return &v1.AdminUser{UserId: item.UserID, Username: item.Username, Phone: item.Phone, Email: item.Email, Avatar: item.Avatar, Roles: item.Roles, Status: item.Status, LastLoginAt: adminUnix(item.LastLoginAt), CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminUserProfileDTO(item *biz.AdminUserProfile) *v1.AdminUserProfile {
	if item == nil {
		return nil
	}
	return &v1.AdminUserProfile{Nickname: item.Nickname, AvatarUrl: item.AvatarURL, Gender: item.Gender, Birthday: item.Birthday, Bio: item.Bio, Location: item.Location, Occupation: item.Occupation, Industry: item.Industry, Language: item.Language, Timezone: item.Timezone}
}

func toSystemConfigDTO(item *biz.AdminSystemConfig) *v1.SystemConfig {
	if item == nil {
		return nil
	}
	return &v1.SystemConfig{Id: item.ID, Key: item.Key, ValueJson: item.ValueJSON, Description: item.Description, IsPublic: item.IsPublic, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAnnouncementDTO(item *biz.AdminAnnouncement) *v1.SystemAnnouncement {
	if item == nil {
		return nil
	}
	return &v1.SystemAnnouncement{Id: item.ID, Title: item.Title, Content: item.Content, TargetPlatform: item.TargetPlatform, StartAt: adminUnix(item.StartAt), EndAt: adminUnix(item.EndAt), Status: item.Status, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAppVersionDTO(item *biz.AdminAppVersion) *v1.AppVersion {
	if item == nil {
		return nil
	}
	return &v1.AppVersion{Id: item.ID, Platform: item.Platform, Version: item.Version, BuildNo: item.BuildNo, ForceUpdate: item.ForceUpdate, DownloadUrl: item.DownloadURL, Changelog: item.Changelog, MinSupportedVersion: item.MinSupportedVersion, PublishedAt: adminUnix(item.PublishedAt), Status: item.Status, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminMoodTagDTO(item *biz.AdminMoodTag) *v1.MoodTag {
	if item == nil {
		return nil
	}
	return &v1.MoodTag{Id: item.ID, Name: item.Name, Color: item.Color, Icon: item.Icon, Sort: item.Sort, System: item.System, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminDiaryDTO(item *biz.AdminDiary) *v1.AdminDiary {
	if item == nil {
		return nil
	}
	tags := make([]*v1.MoodTag, 0, len(item.Tags))
	for _, tag := range item.Tags {
		tags = append(tags, toAdminMoodTagDTO(tag))
	}
	return &v1.AdminDiary{Id: item.ID, UserId: item.UserID, Username: item.Username, Title: item.Title, Content: item.Content, Mood: item.Mood, MoodScore: item.MoodScore, Weather: item.Weather, Location: item.Location, OccurredOn: item.OccurredOn, Visibility: item.Visibility, Tags: tags, AttachmentUrls: item.AttachmentURLs, AnalysisId: item.AnalysisID, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminEmotionAnalysisDTO(item *biz.AdminEmotionAnalysis) *v1.AdminEmotionAnalysis {
	if item == nil {
		return nil
	}
	dims := make([]*v1.AdminEmotionDimensionScore, 0, len(item.Dimensions))
	for _, dim := range item.Dimensions {
		dims = append(dims, &v1.AdminEmotionDimensionScore{Dimension: dim.Dimension, Score: dim.Score})
	}
	return &v1.AdminEmotionAnalysis{Id: item.ID, UserId: item.UserID, Username: item.Username, SourceType: item.SourceType, SourceId: item.SourceID, PrimaryEmotion: item.PrimaryEmotion, Sentiment: item.Sentiment, SentimentScore: item.SentimentScore, StressScore: item.StressScore, AnxietyScore: item.AnxietyScore, DepressionRiskScore: item.DepressionRiskScore, EnergyScore: item.EnergyScore, Confidence: item.Confidence, Summary: item.Summary, Advice: item.Advice, RiskLevel: item.RiskLevel, Model: item.Model, Dimensions: dims, RawResultJson: item.RawResultJSON, CreatedAt: adminUnix(item.CreatedAt)}
}

func toAdminChatSessionDTO(item *biz.AdminChatSession) *v1.AdminChatSession {
	if item == nil {
		return nil
	}
	return &v1.AdminChatSession{Id: item.ID, UserId: item.UserID, Username: item.Username, Title: item.Title, Scenario: item.Scenario, Status: item.Status, Summary: item.Summary, MessageCount: item.MessageCount, LastMessageAt: adminUnix(item.LastMessageAt), CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminChatMessageDTO(item *biz.AdminChatMessage) *v1.AdminChatMessage {
	if item == nil {
		return nil
	}
	return &v1.AdminChatMessage{Id: item.ID, SessionId: item.SessionID, UserId: item.UserID, Username: item.Username, Role: item.Role, Content: item.Content, ContentType: item.ContentType, Model: item.Model, PromptTokens: item.PromptTokens, CompletionTokens: item.CompletionTokens, TotalTokens: item.TotalTokens, LatencyMs: item.LatencyMS, EmotionSnapshotJson: item.EmotionSnapshotJSON, SafetyResultJson: item.SafetyResultJSON, Status: item.Status, ErrorMessage: item.ErrorMessage, CreatedAt: adminUnix(item.CreatedAt)}
}

func toAdminFileDTO(item *biz.AdminFileAsset) *v1.AdminFileAsset {
	if item == nil {
		return nil
	}
	return &v1.AdminFileAsset{Id: item.ID, OwnerUserId: item.OwnerUserID, Username: item.Username, BizType: item.BizType, StorageProvider: item.StorageProvider, Bucket: item.Bucket, ObjectKey: item.ObjectKey, Url: item.URL, MimeType: item.MimeType, SizeBytes: item.SizeBytes, Checksum: item.Checksum, Status: item.Status, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}
