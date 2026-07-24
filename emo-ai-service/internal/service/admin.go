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

func NewAdminService(uc *biz.AdminUsecase) *AdminService { return &AdminService{uc: uc} }

var _ v1.AdminServiceHTTPServer = (*AdminService)(nil)

func requireAdmin(ctx context.Context) (int64, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return 0, err
	}
	if !auth.IsAdmin(ctx) {
		return 0, kerrors.Forbidden("ADMIN_REQUIRED", "admin role required")
	}
	return userID, nil
}

func (s *AdminService) GetDashboardOverview(ctx context.Context, _ *v1.GetDashboardOverviewRequest) (*v1.DashboardOverview, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.GetDashboardOverview(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.DashboardOverview{
		UserCount: item.UserCount, TodayNewUsers: item.TodayNewUsers, DiaryCount: item.DiaryCount,
		TodayDiaries: item.TodayDiaries, ChatSessionCount: item.ChatSessionCount,
		TodayChatMessages: item.TodayChatMessages, EmotionAnalysisCount: item.EmotionAnalysisCount,
		HighRiskAnalysisCount: item.HighRiskAnalysisCount,
	}, nil
}

func (s *AdminService) GetDashboardTrends(ctx context.Context, req *v1.GetDashboardTrendsRequest) (*v1.DashboardTrends, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, err := s.uc.GetDashboardTrends(ctx, req.GetStartDate(), req.GetEndDate())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.DashboardTrendPoint, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.DashboardTrendPoint{Date: item.Date, NewUsers: item.NewUsers, Diaries: item.Diaries, ChatMessages: item.ChatMessages, EmotionAnalyses: item.EmotionAnalyses})
	}
	return &v1.DashboardTrends{Points: out}, nil
}

func (s *AdminService) ListUsers(ctx context.Context, req *v1.ListUsersRequest) (*v1.ListUsersResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListUsers(ctx, biz.AdminUserListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), Keyword: req.GetKeyword(), Status: req.GetStatus(), Role: req.GetRole()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminUser, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminUserDTO(item))
	}
	return &v1.ListUsersResponse{Users: out, Total: total}, nil
}

func (s *AdminService) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.AdminUser, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.GetUser(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return toAdminUserDTO(item), nil
}

func (s *AdminService) UpdateUserStatus(ctx context.Context, req *v1.UpdateUserStatusRequest) (*v1.AdminUser, error) {
	actorID, err := requireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateUserStatus(ctx, actorID, req.GetUserId(), req.GetStatus(), req.GetReason())
	if err != nil {
		return nil, err
	}
	return toAdminUserDTO(item), nil
}

func (s *AdminService) UpdateUserRoles(ctx context.Context, req *v1.UpdateUserRolesRequest) (*v1.AdminUser, error) {
	actorID, err := requireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateUserRoles(ctx, actorID, req.GetUserId(), req.GetRoles())
	if err != nil {
		return nil, err
	}
	return toAdminUserDTO(item), nil
}

func (s *AdminService) UpdateUserPassword(ctx context.Context, req *v1.UpdateUserPasswordRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.UpdateUserPassword(ctx, req.GetUserId(), req.GetPassword()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AdminService) ListDiaries(ctx context.Context, req *v1.ListDiariesRequest) (*v1.ListDiariesResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListDiaries(ctx, biz.AdminDiaryListOption{
		Page: req.GetPage(), PageSize: req.GetPageSize(), Keyword: req.GetKeyword(), UserID: req.GetUserId(),
		Mood: req.GetMood(), StartDate: req.GetStartDate(), EndDate: req.GetEndDate(),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminDiary, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminDiaryDTO(item))
	}
	return &v1.ListDiariesResponse{Diaries: out, Total: total}, nil
}

func (s *AdminService) GetDiary(ctx context.Context, req *v1.GetDiaryRequest) (*v1.AdminDiary, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.GetDiary(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return toAdminDiaryDTO(item), nil
}

func (s *AdminService) DeleteDiary(ctx context.Context, req *v1.DeleteDiaryRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.DeleteDiary(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AdminService) ListEmotionAnalyses(ctx context.Context, req *v1.ListEmotionAnalysesRequest) (*v1.ListEmotionAnalysesResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListEmotionAnalyses(ctx, biz.AdminEmotionListOption{
		Page: req.GetPage(), PageSize: req.GetPageSize(), UserID: req.GetUserId(), RiskLevel: req.GetRiskLevel(),
		SourceType: req.GetSourceType(), StartDate: req.GetStartDate(), EndDate: req.GetEndDate(),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminEmotionAnalysis, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminEmotionDTO(item))
	}
	return &v1.ListEmotionAnalysesResponse{Analyses: out, Total: total}, nil
}

func (s *AdminService) GetEmotionAnalysis(ctx context.Context, req *v1.GetEmotionAnalysisRequest) (*v1.AdminEmotionAnalysis, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.GetEmotionAnalysis(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return toAdminEmotionDTO(item), nil
}

func (s *AdminService) ListChatSessions(ctx context.Context, req *v1.ListChatSessionsRequest) (*v1.ListChatSessionsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListChatSessions(ctx, biz.AdminChatListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), UserID: req.GetUserId(), Status: req.GetStatus()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminChatSession, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminChatSessionDTO(item))
	}
	return &v1.ListChatSessionsResponse{Sessions: out, Total: total}, nil
}

func (s *AdminService) ListChatMessages(ctx context.Context, req *v1.ListChatMessagesRequest) (*v1.ListChatMessagesResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListChatMessages(ctx, req.GetSessionId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminChatMessage, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminChatMessageDTO(item))
	}
	return &v1.ListChatMessagesResponse{Messages: out, Total: total}, nil
}

func (s *AdminService) ListLoginLogs(ctx context.Context, req *v1.ListLoginLogsRequest) (*v1.ListLoginLogsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListLoginLogs(ctx, biz.AdminLoginLogListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), UserID: req.GetUserId(), SuccessOnly: req.GetSuccessOnly()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminLoginLog, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.AdminLoginLog{Id: item.ID, UserId: item.UserID, Username: item.Username, Success: item.Success, FailReason: item.FailReason, Ip: item.IP, UserAgent: item.UserAgent, DeviceId: item.DeviceID, Location: item.Location, CreatedAt: adminUnix(item.CreatedAt)})
	}
	return &v1.ListLoginLogsResponse{Logs: out, Total: total}, nil
}

func (s *AdminService) ListSecurityEvents(ctx context.Context, req *v1.ListSecurityEventsRequest) (*v1.ListSecurityEventsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListSecurityEvents(ctx, biz.AdminSecurityEventListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), UserID: req.GetUserId(), RiskLevel: req.GetRiskLevel()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminSecurityEvent, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.AdminSecurityEvent{Id: item.ID, UserId: item.UserID, Username: item.Username, EventType: item.EventType, RiskLevel: item.RiskLevel, Ip: item.IP, UserAgent: item.UserAgent, MetadataJson: item.MetadataJSON, CreatedAt: adminUnix(item.CreatedAt)})
	}
	return &v1.ListSecurityEventsResponse{Events: out, Total: total}, nil
}

func (s *AdminService) ListConfigs(ctx context.Context, req *v1.ListConfigsRequest) (*v1.ListConfigsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListConfigs(ctx, biz.AdminConfigListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), Keyword: req.GetKeyword()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminSystemConfig, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminConfigDTO(item))
	}
	return &v1.ListConfigsResponse{Configs: out, Total: total}, nil
}

func (s *AdminService) CreateConfig(ctx context.Context, req *v1.CreateConfigRequest) (*v1.AdminSystemConfig, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.CreateConfig(ctx, &biz.AdminSystemConfig{Key: req.GetKey(), ValueJSON: req.GetValueJson(), Description: req.GetDescription(), IsPublic: req.GetIsPublic()})
	if err != nil {
		return nil, err
	}
	return toAdminConfigDTO(item), nil
}

func (s *AdminService) UpdateConfig(ctx context.Context, req *v1.UpdateConfigRequest) (*v1.AdminSystemConfig, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateConfig(ctx, &biz.AdminSystemConfig{ID: req.GetId(), ValueJSON: req.GetValueJson(), Description: req.GetDescription(), IsPublic: req.GetIsPublic()})
	if err != nil {
		return nil, err
	}
	return toAdminConfigDTO(item), nil
}

func (s *AdminService) DeleteConfig(ctx context.Context, req *v1.DeleteConfigRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.DeleteConfig(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AdminService) ListAnnouncements(ctx context.Context, req *v1.ListAnnouncementsRequest) (*v1.ListAnnouncementsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	var status *int32
	if req.GetFilterStatus() {
		value := req.GetStatus()
		status = &value
	}
	items, total, err := s.uc.ListAnnouncements(ctx, biz.AdminAnnouncementListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), Platform: req.GetPlatform(), Status: status})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminAnnouncement, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminAnnouncementDTO(item))
	}
	return &v1.ListAnnouncementsResponse{Announcements: out, Total: total}, nil
}

func (s *AdminService) CreateAnnouncement(ctx context.Context, req *v1.CreateAnnouncementRequest) (*v1.AdminAnnouncement, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.CreateAnnouncement(ctx, &biz.AdminAnnouncement{Title: req.GetTitle(), Content: req.GetContent(), TargetPlatform: req.GetTargetPlatform(), StartAt: adminTime(req.GetStartAt()), EndAt: adminTime(req.GetEndAt()), Status: req.GetStatus()})
	if err != nil {
		return nil, err
	}
	return toAdminAnnouncementDTO(item), nil
}

func (s *AdminService) UpdateAnnouncement(ctx context.Context, req *v1.UpdateAnnouncementRequest) (*v1.AdminAnnouncement, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateAnnouncement(ctx, &biz.AdminAnnouncement{ID: req.GetId(), Title: req.GetTitle(), Content: req.GetContent(), TargetPlatform: req.GetTargetPlatform(), StartAt: adminTime(req.GetStartAt()), EndAt: adminTime(req.GetEndAt()), Status: req.GetStatus()})
	if err != nil {
		return nil, err
	}
	return toAdminAnnouncementDTO(item), nil
}

func (s *AdminService) DeleteAnnouncement(ctx context.Context, req *v1.DeleteAnnouncementRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.DeleteAnnouncement(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AdminService) ListVersions(ctx context.Context, req *v1.ListVersionsRequest) (*v1.ListVersionsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListVersions(ctx, biz.AdminVersionListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), Platform: req.GetPlatform()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminAppVersion, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminVersionDTO(item))
	}
	return &v1.ListVersionsResponse{Versions: out, Total: total}, nil
}

func (s *AdminService) CreateVersion(ctx context.Context, req *v1.CreateVersionRequest) (*v1.AdminAppVersion, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.CreateVersion(ctx, &biz.AdminAppVersion{Platform: req.GetPlatform(), Version: req.GetVersion(), BuildNo: req.GetBuildNo(), ForceUpdate: req.GetForceUpdate(), DownloadURL: req.GetDownloadUrl(), Changelog: req.GetChangelog(), MinSupportedVersion: req.GetMinSupportedVersion(), PublishedAt: adminTime(req.GetPublishedAt())})
	if err != nil {
		return nil, err
	}
	return toAdminVersionDTO(item), nil
}

func (s *AdminService) UpdateVersion(ctx context.Context, req *v1.UpdateVersionRequest) (*v1.AdminAppVersion, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateVersion(ctx, &biz.AdminAppVersion{ID: req.GetId(), Platform: req.GetPlatform(), Version: req.GetVersion(), BuildNo: req.GetBuildNo(), ForceUpdate: req.GetForceUpdate(), DownloadURL: req.GetDownloadUrl(), Changelog: req.GetChangelog(), MinSupportedVersion: req.GetMinSupportedVersion(), PublishedAt: adminTime(req.GetPublishedAt())})
	if err != nil {
		return nil, err
	}
	return toAdminVersionDTO(item), nil
}

func (s *AdminService) DeleteVersion(ctx context.Context, req *v1.DeleteVersionRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.DeleteVersion(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AdminService) ListMoodTags(ctx context.Context, req *v1.ListMoodTagsRequest) (*v1.ListMoodTagsResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListMoodTags(ctx, biz.AdminMoodTagListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), Keyword: req.GetKeyword()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminMoodTag, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminMoodTagDTO(item))
	}
	return &v1.ListMoodTagsResponse{Tags: out, Total: total}, nil
}

func (s *AdminService) CreateMoodTag(ctx context.Context, req *v1.CreateMoodTagRequest) (*v1.AdminMoodTag, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.CreateMoodTag(ctx, &biz.MoodTag{Name: req.GetName(), Color: req.GetColor(), Icon: req.GetIcon(), Sort: req.GetSort()})
	if err != nil {
		return nil, err
	}
	return toAdminMoodTagDTO(item), nil
}

func (s *AdminService) UpdateMoodTag(ctx context.Context, req *v1.UpdateMoodTagRequest) (*v1.AdminMoodTag, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.UpdateMoodTag(ctx, &biz.MoodTag{ID: req.GetId(), Name: req.GetName(), Color: req.GetColor(), Icon: req.GetIcon(), Sort: req.GetSort()})
	if err != nil {
		return nil, err
	}
	return toAdminMoodTagDTO(item), nil
}

func (s *AdminService) DeleteMoodTag(ctx context.Context, req *v1.DeleteMoodTagRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.DeleteMoodTag(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AdminService) ListFiles(ctx context.Context, req *v1.ListFilesRequest) (*v1.ListFilesResponse, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListFiles(ctx, biz.AdminFileListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), BizType: req.GetBizType(), OwnerUserID: req.GetOwnerUserId()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AdminFile, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminFileDTO(item))
	}
	return &v1.ListFilesResponse{Files: out, Total: total}, nil
}

func (s *AdminService) GetFile(ctx context.Context, req *v1.GetFileRequest) (*v1.AdminFile, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	item, err := s.uc.GetFile(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return toAdminFileDTO(item), nil
}

func (s *AdminService) DeleteFile(ctx context.Context, req *v1.DeleteFileRequest) (*emptypb.Empty, error) {
	if _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.uc.DeleteFile(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func toAdminUserDTO(item *biz.AdminUser) *v1.AdminUser {
	out := &v1.AdminUser{UserId: item.ID, Username: item.Username, Phone: item.Phone, Email: item.Email, Avatar: item.Avatar, Roles: item.Roles, Status: item.Status, LastLoginAt: adminUnix(item.LastLoginAt), CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
	if item.Profile != nil {
		out.Profile = &v1.AdminUserProfile{Nickname: item.Profile.Nickname, Gender: item.Profile.Gender, Birthday: item.Profile.Birthday, Bio: item.Profile.Bio, Location: item.Profile.Location, Occupation: item.Profile.Occupation, Industry: item.Profile.Industry, Language: item.Profile.Language, Timezone: item.Profile.Timezone}
	}
	return out
}

func toAdminMoodTagDTO(item *biz.MoodTag) *v1.AdminMoodTag {
	return &v1.AdminMoodTag{Id: item.ID, Name: item.Name, Color: item.Color, Icon: item.Icon, Sort: item.Sort, System: item.System, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminDiaryDTO(item *biz.AdminDiary) *v1.AdminDiary {
	tags := make([]*v1.AdminMoodTag, 0, len(item.Tags))
	for _, tag := range item.Tags {
		tags = append(tags, toAdminMoodTagDTO(tag))
	}
	return &v1.AdminDiary{Id: item.ID, UserId: item.UserID, Username: item.Username, Title: item.Title, Content: item.Content, Mood: item.Mood, MoodScore: item.MoodScore, Weather: item.Weather, Location: item.Location, OccurredOn: item.OccurredOn, Visibility: item.Visibility, Tags: tags, AttachmentUrls: item.AttachmentURLs, AnalysisId: item.AnalysisID, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminEmotionDTO(item *biz.AdminEmotionAnalysis) *v1.AdminEmotionAnalysis {
	dimensions := make([]*v1.EmotionDimensionScore, 0, len(item.Dimensions))
	for _, dim := range item.Dimensions {
		dimensions = append(dimensions, &v1.EmotionDimensionScore{Dimension: dim.Dimension, Score: dim.Score})
	}
	return &v1.AdminEmotionAnalysis{Id: item.ID, UserId: item.UserID, Username: item.Username, SourceType: item.SourceType, SourceId: item.SourceID, PrimaryEmotion: item.PrimaryEmotion, Sentiment: item.Sentiment, SentimentScore: item.SentimentScore, StressScore: item.StressScore, AnxietyScore: item.AnxietyScore, DepressionRiskScore: item.DepressionRiskScore, EnergyScore: item.EnergyScore, Confidence: item.Confidence, Summary: item.Summary, Advice: item.Advice, RiskLevel: item.RiskLevel, Model: item.Model, Dimensions: dimensions, CreatedAt: adminUnix(item.CreatedAt)}
}

func toAdminChatSessionDTO(item *biz.AdminChatSession) *v1.AdminChatSession {
	return &v1.AdminChatSession{Id: item.ID, UserId: item.UserID, Username: item.Username, Title: item.Title, Scenario: item.Scenario, Status: item.Status, Summary: item.Summary, MessageCount: item.MessageCount, LastMessageAt: adminUnix(item.LastMessageAt), CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminChatMessageDTO(item *biz.ChatMessage) *v1.AdminChatMessage {
	return &v1.AdminChatMessage{Id: item.ID, SessionId: item.SessionID, UserId: item.UserID, Role: item.Role, Content: item.Content, ContentType: item.ContentType, Status: item.Status, CreatedAt: adminUnix(item.CreatedAt)}
}

func toAdminConfigDTO(item *biz.AdminSystemConfig) *v1.AdminSystemConfig {
	return &v1.AdminSystemConfig{Id: item.ID, Key: item.Key, ValueJson: item.ValueJSON, Description: item.Description, IsPublic: item.IsPublic, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminAnnouncementDTO(item *biz.AdminAnnouncement) *v1.AdminAnnouncement {
	return &v1.AdminAnnouncement{Id: item.ID, Title: item.Title, Content: item.Content, TargetPlatform: item.TargetPlatform, StartAt: adminUnix(item.StartAt), EndAt: adminUnix(item.EndAt), Status: item.Status, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminVersionDTO(item *biz.AdminAppVersion) *v1.AdminAppVersion {
	return &v1.AdminAppVersion{Id: item.ID, Platform: item.Platform, Version: item.Version, BuildNo: item.BuildNo, ForceUpdate: item.ForceUpdate, DownloadUrl: item.DownloadURL, Changelog: item.Changelog, MinSupportedVersion: item.MinSupportedVersion, PublishedAt: adminUnix(item.PublishedAt), CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func toAdminFileDTO(item *biz.AdminFile) *v1.AdminFile {
	return &v1.AdminFile{Id: item.ID, OwnerUserId: item.OwnerUserID, Username: item.Username, BizType: item.BizType, StorageProvider: item.StorageProvider, Bucket: item.Bucket, ObjectKey: item.ObjectKey, Url: item.URL, MimeType: item.MimeType, SizeBytes: item.SizeBytes, Checksum: item.Checksum, Status: item.Status, CreatedAt: adminUnix(item.CreatedAt), UpdatedAt: adminUnix(item.UpdatedAt)}
}

func adminTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.Unix(value, 0)
}

func adminUnix(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}
