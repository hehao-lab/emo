package biz

import (
	"context"
	"time"
)

type AboutInfo struct {
	AppName      string
	Company      string
	Description  string
	PrivacyURL   string
	TermsURL     string
	ContactEmail string
	Website      string
}

type PublicConfig struct {
	Key         string
	ValueJSON   string
	Description string
}

type AppVersion struct {
	ID                  int64
	Platform            string
	Version             string
	BuildNo             int32
	ForceUpdate         bool
	DownloadURL         string
	Changelog           string
	MinSupportedVersion string
	PublishedAt         time.Time
}

type Announcement struct {
	ID             int64
	Title          string
	Content        string
	TargetPlatform string
	StartAt        time.Time
	EndAt          time.Time
}

type SystemRepo interface {
	GetAbout(ctx context.Context) (*AboutInfo, error)
	ListPublicConfigs(ctx context.Context) ([]*PublicConfig, error)
	GetLatestVersion(ctx context.Context, platform string) (*AppVersion, error)
	ListAnnouncements(ctx context.Context, platform string) ([]*Announcement, error)
}

type SystemUsecase struct {
	repo SystemRepo
}

func NewSystemUsecase(repo SystemRepo) *SystemUsecase {
	return &SystemUsecase{repo: repo}
}

func (uc *SystemUsecase) GetAbout(ctx context.Context) (*AboutInfo, error) {
	info, err := uc.repo.GetAbout(ctx)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return &AboutInfo{}, nil
	}
	return info, nil
}

func (uc *SystemUsecase) ListPublicConfigs(ctx context.Context) ([]*PublicConfig, error) {
	return uc.repo.ListPublicConfigs(ctx)
}

func (uc *SystemUsecase) GetLatestVersion(ctx context.Context, platform string) (*AppVersion, error) {
	return uc.repo.GetLatestVersion(ctx, platform)
}

func (uc *SystemUsecase) ListAnnouncements(ctx context.Context, platform string) ([]*Announcement, error) {
	return uc.repo.ListAnnouncements(ctx, platform)
}
