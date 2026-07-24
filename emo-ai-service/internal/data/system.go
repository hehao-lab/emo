package data

import (
	"context"
	"encoding/json"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

type SystemConfigModel struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;comment:系统配置ID"`
	ConfigKey   string    `gorm:"type:varchar(128);uniqueIndex;not null;comment:配置键"`
	ConfigValue string    `gorm:"type:json;not null;comment:配置值JSON"`
	Description string    `gorm:"type:varchar(255);default:'';comment:配置说明"`
	IsPublic    bool      `gorm:"default:false;comment:是否公开给前端读取"`
	CreatedAt   time.Time `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;comment:更新时间"`
}

func (SystemConfigModel) TableName() string { return "system_configs" }

type AppVersionModel struct {
	ID                  int64      `gorm:"primaryKey;autoIncrement;comment:应用版本ID"`
	Platform            string     `gorm:"type:varchar(16);index;not null;comment:平台 ios android web"`
	Version             string     `gorm:"type:varchar(32);not null;comment:版本号"`
	BuildNo             int32      `gorm:"index;not null;comment:构建号"`
	ForceUpdate         bool       `gorm:"default:false;comment:是否强制更新"`
	DownloadURL         string     `gorm:"type:varchar(1024);default:'';comment:下载地址"`
	Changelog           string     `gorm:"type:text;comment:更新说明"`
	MinSupportedVersion string     `gorm:"type:varchar(32);default:'';comment:最低支持版本"`
	PublishedAt         *time.Time `gorm:"index;comment:发布时间"`
	CreatedAt           time.Time  `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt           time.Time  `gorm:"autoUpdateTime;comment:更新时间"`
}

func (AppVersionModel) TableName() string { return "app_versions" }

type SystemAnnouncementModel struct {
	ID             int64      `gorm:"primaryKey;autoIncrement;comment:系统公告ID"`
	Title          string     `gorm:"type:varchar(128);not null;comment:公告标题"`
	Content        string     `gorm:"type:text;not null;comment:公告内容"`
	TargetPlatform string     `gorm:"type:varchar(32);index;default:'all';comment:目标平台"`
	StartAt        *time.Time `gorm:"index;comment:生效开始时间"`
	EndAt          *time.Time `gorm:"index;comment:生效结束时间"`
	Status         int32      `gorm:"default:1;comment:公告状态 1启用 0停用"`
	CreatedAt      time.Time  `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime;comment:更新时间"`
}

func (SystemAnnouncementModel) TableName() string { return "system_announcements" }

type systemRepoImpl struct {
	db *gorm.DB
}

func NewSystemRepo(d *Data) biz.SystemRepo {
	return &systemRepoImpl{db: d.db}
}

func (r *systemRepoImpl) GetAbout(ctx context.Context) (*biz.AboutInfo, error) {
	var rows []SystemConfigModel
	if err := r.db.WithContext(ctx).Where("config_key IN ?", []string{"about.app_name", "about.company", "about.description", "about.privacy_url", "about.terms_url", "about.contact_email", "about.website"}).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	values := map[string]string{}
	for _, row := range rows {
		values[row.ConfigKey] = decodeConfigString(row.ConfigValue)
	}
	return &biz.AboutInfo{
		AppName:      values["about.app_name"],
		Company:      values["about.company"],
		Description:  values["about.description"],
		PrivacyURL:   values["about.privacy_url"],
		TermsURL:     values["about.terms_url"],
		ContactEmail: values["about.contact_email"],
		Website:      values["about.website"],
	}, nil
}

func (r *systemRepoImpl) ListPublicConfigs(ctx context.Context) ([]*biz.PublicConfig, error) {
	var models []SystemConfigModel
	if err := r.db.WithContext(ctx).Where("is_public = ?", true).Order("config_key asc").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.PublicConfig, 0, len(models))
	for _, model := range models {
		out = append(out, &biz.PublicConfig{Key: model.ConfigKey, ValueJSON: model.ConfigValue, Description: model.Description})
	}
	return out, nil
}

func (r *systemRepoImpl) GetLatestVersion(ctx context.Context, platform string) (*biz.AppVersion, error) {
	if platform == "" {
		platform = "web"
	}
	var model AppVersionModel
	err := r.db.WithContext(ctx).Where("platform = ?", platform).Order("build_no desc").First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var publishedAt time.Time
	if model.PublishedAt != nil {
		publishedAt = *model.PublishedAt
	}
	return &biz.AppVersion{ID: model.ID, Platform: model.Platform, Version: model.Version, BuildNo: model.BuildNo, ForceUpdate: model.ForceUpdate, DownloadURL: model.DownloadURL, Changelog: model.Changelog, MinSupportedVersion: model.MinSupportedVersion, PublishedAt: publishedAt}, nil
}

func (r *systemRepoImpl) ListAnnouncements(ctx context.Context, platform string) ([]*biz.Announcement, error) {
	now := time.Now()
	q := r.db.WithContext(ctx).Where("status = ?", 1).
		Where("(start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at >= ?)", now, now)
	if platform != "" {
		q = q.Where("target_platform IN ?", []string{"all", platform})
	}
	var models []SystemAnnouncementModel
	if err := q.Order("created_at desc").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Announcement, 0, len(models))
	for _, model := range models {
		var startAt, endAt time.Time
		if model.StartAt != nil {
			startAt = *model.StartAt
		}
		if model.EndAt != nil {
			endAt = *model.EndAt
		}
		out = append(out, &biz.Announcement{ID: model.ID, Title: model.Title, Content: model.Content, TargetPlatform: model.TargetPlatform, StartAt: startAt, EndAt: endAt})
	}
	return out, nil
}

func trimJSONQuote(value string) string {
	var decoded string
	if err := json.Unmarshal([]byte(value), &decoded); err == nil {
		return decoded
	}
	return value
}
