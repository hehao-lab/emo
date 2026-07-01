package service

import (
	"context"

	v1 "emo-ai-service/api/system/v1"
	"emo-ai-service/internal/biz"
)

type SystemService struct {
	uc *biz.SystemUsecase
}

func NewSystemService(uc *biz.SystemUsecase) *SystemService {
	return &SystemService{uc: uc}
}

var _ v1.SystemServiceHTTPServer = (*SystemService)(nil)

// GetAbout 实现关于我们接口：返回应用名称、公司信息、协议链接和联系方式。
func (s *SystemService) GetAbout(ctx context.Context, req *v1.GetAboutRequest) (*v1.AboutInfo, error) {
	info, err := s.uc.GetAbout(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.AboutInfo{AppName: info.AppName, Company: info.Company, Description: info.Description, PrivacyUrl: info.PrivacyURL, TermsUrl: info.TermsURL, ContactEmail: info.ContactEmail, Website: info.Website}, nil
}

// ListPublicConfigs 实现公开系统配置接口：返回前端可读取的公共配置项。
func (s *SystemService) ListPublicConfigs(ctx context.Context, req *v1.ListPublicConfigsRequest) (*v1.ListPublicConfigsResponse, error) {
	items, err := s.uc.ListPublicConfigs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.PublicConfig, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.PublicConfig{Key: item.Key, ValueJson: item.ValueJSON, Description: item.Description})
	}
	return &v1.ListPublicConfigsResponse{Configs: out}, nil
}

// GetLatestVersion 实现版本检查接口：按平台返回最新版本、更新说明和强制更新标记。
func (s *SystemService) GetLatestVersion(ctx context.Context, req *v1.GetLatestVersionRequest) (*v1.AppVersion, error) {
	version, err := s.uc.GetLatestVersion(ctx, req.GetPlatform())
	if err != nil {
		return nil, err
	}
	if version == nil {
		return &v1.AppVersion{}, nil
	}
	return &v1.AppVersion{Id: version.ID, Platform: version.Platform, Version: version.Version, BuildNo: version.BuildNo, ForceUpdate: version.ForceUpdate, DownloadUrl: version.DownloadURL, Changelog: version.Changelog, MinSupportedVersion: version.MinSupportedVersion, PublishedAt: version.PublishedAt.Unix()}, nil
}

// ListAnnouncements 实现公告列表接口：返回当前平台有效期内的系统公告。
func (s *SystemService) ListAnnouncements(ctx context.Context, req *v1.ListAnnouncementsRequest) (*v1.ListAnnouncementsResponse, error) {
	items, err := s.uc.ListAnnouncements(ctx, req.GetPlatform())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Announcement, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.Announcement{Id: item.ID, Title: item.Title, Content: item.Content, TargetPlatform: item.TargetPlatform, StartAt: item.StartAt.Unix(), EndAt: item.EndAt.Unix()})
	}
	return &v1.ListAnnouncementsResponse{Announcements: out}, nil
}
