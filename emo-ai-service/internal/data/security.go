package data

import (
	"context"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

type AuthRefreshTokenModel struct {
	ID           int64          `gorm:"primaryKey;autoIncrement;comment:刷新令牌记录ID"`
	UserID       int64          `gorm:"index;not null;comment:用户ID"`
	TokenID      string         `gorm:"type:varchar(64);uniqueIndex;not null;comment:刷新令牌唯一ID"`
	TokenHash    string         `gorm:"type:varchar(255);uniqueIndex;not null;comment:刷新令牌哈希值"`
	DeviceID     string         `gorm:"type:varchar(128);default:'';comment:设备ID"`
	DeviceName   string         `gorm:"type:varchar(128);default:'';comment:设备名称"`
	IP           string         `gorm:"type:varchar(64);default:'';comment:登录IP"`
	UserAgent    string         `gorm:"type:varchar(512);default:'';comment:客户端User-Agent"`
	ExpiresAt    time.Time      `gorm:"index;not null;comment:过期时间"`
	RevokedAt    *time.Time     `gorm:"index;comment:撤销时间"`
	RevokeReason string         `gorm:"type:varchar(128);default:'';comment:撤销原因"`
	CreatedAt    time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt    gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (AuthRefreshTokenModel) TableName() string { return "auth_refresh_tokens" }

type LoginLogModel struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;comment:登录日志ID"`
	UserID     int64     `gorm:"index;comment:用户ID"`
	Username   string    `gorm:"type:varchar(64);index;comment:登录用户名"`
	LoginType  string    `gorm:"type:varchar(32);default:'password';comment:登录类型"`
	Success    bool      `gorm:"not null;comment:是否登录成功"`
	FailReason string    `gorm:"type:varchar(128);default:'';comment:失败原因"`
	IP         string    `gorm:"type:varchar(64);default:'';comment:登录IP"`
	UserAgent  string    `gorm:"type:varchar(512);default:'';comment:客户端User-Agent"`
	DeviceID   string    `gorm:"type:varchar(128);default:'';comment:设备ID"`
	Location   string    `gorm:"type:varchar(128);default:'';comment:登录地理位置"`
	CreatedAt  time.Time `gorm:"autoCreateTime;index;comment:创建时间"`
}

func (LoginLogModel) TableName() string { return "login_logs" }

type SecurityEventModel struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;comment:安全事件ID"`
	UserID       int64     `gorm:"index;not null;comment:用户ID"`
	EventType    string    `gorm:"type:varchar(64);not null;comment:事件类型"`
	RiskLevel    string    `gorm:"type:varchar(16);default:'low';comment:风险等级"`
	IP           string    `gorm:"type:varchar(64);default:'';comment:操作IP"`
	UserAgent    string    `gorm:"type:varchar(512);default:'';comment:客户端User-Agent"`
	MetadataJSON string    `gorm:"type:json;comment:事件扩展信息JSON"`
	CreatedAt    time.Time `gorm:"autoCreateTime;index;comment:创建时间"`
}

func (SecurityEventModel) TableName() string { return "security_events" }

type securityRepoImpl struct {
	db *gorm.DB
}

func NewSecurityRepo(d *Data) biz.SecurityRepo {
	return &securityRepoImpl{db: d.db}
}

func (r *securityRepoImpl) CreateRefreshToken(ctx context.Context, token *biz.AuthToken) error {
	return r.db.WithContext(ctx).Create(&AuthRefreshTokenModel{
		UserID:       token.UserID,
		TokenID:      token.TokenID,
		TokenHash:    token.TokenHash,
		DeviceID:     token.DeviceID,
		DeviceName:   token.DeviceName,
		IP:           token.IP,
		UserAgent:    token.UserAgent,
		ExpiresAt:    token.ExpiresAt,
		RevokeReason: token.RevokeReason,
	}).Error
}

func (r *securityRepoImpl) FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*biz.AuthToken, error) {
	var model AuthRefreshTokenModel
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizAuthToken(&model), nil
}

func (r *securityRepoImpl) ListRefreshTokens(ctx context.Context, userID int64) ([]*biz.AuthToken, error) {
	var models []AuthRefreshTokenModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.AuthToken, 0, len(models))
	for i := range models {
		out = append(out, toBizAuthToken(&models[i]))
	}
	return out, nil
}

func (r *securityRepoImpl) RevokeRefreshToken(ctx context.Context, userID int64, tokenID, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&AuthRefreshTokenModel{}).
		Where("user_id = ? AND token_id = ? AND revoked_at IS NULL", userID, tokenID).
		Updates(map[string]any{"revoked_at": &now, "revoke_reason": reason}).Error
}

func (r *securityRepoImpl) RevokeRefreshTokenByHash(ctx context.Context, tokenHash, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&AuthRefreshTokenModel{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Updates(map[string]any{"revoked_at": &now, "revoke_reason": reason}).Error
}

func (r *securityRepoImpl) RevokeAllRefreshTokens(ctx context.Context, userID int64, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&AuthRefreshTokenModel{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]any{"revoked_at": &now, "revoke_reason": reason}).Error
}

func (r *securityRepoImpl) CreateLoginLog(ctx context.Context, log *biz.LoginLog) error {
	return r.db.WithContext(ctx).Create(&LoginLogModel{
		UserID:     log.UserID,
		Username:   log.Username,
		LoginType:  log.LoginType,
		Success:    log.Success,
		FailReason: log.FailReason,
		IP:         log.IP,
		UserAgent:  log.UserAgent,
		DeviceID:   log.DeviceID,
		Location:   log.Location,
	}).Error
}

func (r *securityRepoImpl) ListLoginLogs(ctx context.Context, userID int64, page, pageSize int32) ([]*biz.LoginLog, int64, error) {
	p, size := normalizePage(page, pageSize)
	var total int64
	q := r.db.WithContext(ctx).Model(&LoginLogModel{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []LoginLogModel
	if err := q.Order("created_at desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.LoginLog, 0, len(models))
	for i := range models {
		out = append(out, toBizLoginLog(&models[i]))
	}
	return out, total, nil
}

func (r *securityRepoImpl) CreateSecurityEvent(ctx context.Context, event *biz.SecurityEvent) error {
	return r.db.WithContext(ctx).Create(&SecurityEventModel{
		UserID:       event.UserID,
		EventType:    event.EventType,
		RiskLevel:    event.RiskLevel,
		IP:           event.IP,
		UserAgent:    event.UserAgent,
		MetadataJSON: event.MetadataJSON,
	}).Error
}

func (r *securityRepoImpl) ListSecurityEvents(ctx context.Context, userID int64, page, pageSize int32) ([]*biz.SecurityEvent, int64, error) {
	p, size := normalizePage(page, pageSize)
	var total int64
	q := r.db.WithContext(ctx).Model(&SecurityEventModel{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []SecurityEventModel
	if err := q.Order("created_at desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.SecurityEvent, 0, len(models))
	for i := range models {
		out = append(out, &biz.SecurityEvent{
			ID:           models[i].ID,
			UserID:       models[i].UserID,
			EventType:    models[i].EventType,
			RiskLevel:    models[i].RiskLevel,
			IP:           models[i].IP,
			UserAgent:    models[i].UserAgent,
			MetadataJSON: models[i].MetadataJSON,
			CreatedAt:    models[i].CreatedAt,
		})
	}
	return out, total, nil
}

func toBizAuthToken(model *AuthRefreshTokenModel) *biz.AuthToken {
	return &biz.AuthToken{
		ID:           model.ID,
		UserID:       model.UserID,
		TokenID:      model.TokenID,
		TokenHash:    model.TokenHash,
		DeviceID:     model.DeviceID,
		DeviceName:   model.DeviceName,
		IP:           model.IP,
		UserAgent:    model.UserAgent,
		ExpiresAt:    model.ExpiresAt,
		RevokedAt:    model.RevokedAt,
		RevokeReason: model.RevokeReason,
		CreatedAt:    model.CreatedAt,
	}
}

func toBizLoginLog(model *LoginLogModel) *biz.LoginLog {
	return &biz.LoginLog{
		ID:         model.ID,
		UserID:     model.UserID,
		Username:   model.Username,
		LoginType:  model.LoginType,
		Success:    model.Success,
		FailReason: model.FailReason,
		IP:         model.IP,
		UserAgent:  model.UserAgent,
		DeviceID:   model.DeviceID,
		Location:   model.Location,
		CreatedAt:  model.CreatedAt,
	}
}
