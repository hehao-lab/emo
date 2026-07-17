package biz

import (
	"context"
	"time"

	"emo-ai-service/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

type AuthToken struct {
	ID           int64
	UserID       int64
	TokenID      string
	TokenHash    string
	DeviceID     string
	DeviceName   string
	IP           string
	UserAgent    string
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	RevokeReason string
	CreatedAt    time.Time
}

type LoginLog struct {
	ID         int64
	UserID     int64
	Username   string
	LoginType  string
	Success    bool
	FailReason string
	IP         string
	UserAgent  string
	DeviceID   string
	Location   string
	CreatedAt  time.Time
}

type SecurityEvent struct {
	ID           int64
	UserID       int64
	EventType    string
	RiskLevel    string
	IP           string
	UserAgent    string
	MetadataJSON string
	CreatedAt    time.Time
}

type SecurityRepo interface {
	CreateRefreshToken(ctx context.Context, token *AuthToken) error
	FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*AuthToken, error)
	ListRefreshTokens(ctx context.Context, userID int64) ([]*AuthToken, error)
	RevokeRefreshToken(ctx context.Context, userID int64, tokenID, reason string) error
	RevokeRefreshTokenByHash(ctx context.Context, tokenHash, reason string) error
	RevokeAllRefreshTokens(ctx context.Context, userID int64, reason string) error
	CreateLoginLog(ctx context.Context, log *LoginLog) error
	ListLoginLogs(ctx context.Context, userID int64, page, pageSize int32) ([]*LoginLog, int64, error)
	CreateSecurityEvent(ctx context.Context, event *SecurityEvent) error
	ListSecurityEvents(ctx context.Context, userID int64, page, pageSize int32) ([]*SecurityEvent, int64, error)
}

type SecurityUsecase struct {
	users        UserAccountRepo
	repo         SecurityRepo
	tokenManager *auth.TokenManager
}

func NewSecurityUsecase(users UserAccountRepo, repo SecurityRepo, tokenManager *auth.TokenManager) *SecurityUsecase {
	return &SecurityUsecase{users: users, repo: repo, tokenManager: tokenManager}
}

func (uc *SecurityUsecase) RefreshToken(ctx context.Context, refreshToken string, meta LoginMeta) (*LoginResult, error) {
	token, err := uc.repo.FindRefreshTokenByHash(ctx, auth.HashToken(refreshToken))
	if err != nil {
		return nil, err
	}
	if token == nil || token.RevokedAt != nil || time.Now().After(token.ExpiresAt) {
		return nil, ErrTokenInvalid
	}
	u, err := uc.users.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	if u.Status != 1 {
		return nil, ErrUserDisabled
	}
	if err := uc.repo.RevokeRefreshToken(ctx, token.UserID, token.TokenID, "rotated"); err != nil {
		return nil, err
	}
	pair, err := uc.tokenManager.IssuePair(u.ID, u.Roles)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.CreateRefreshToken(ctx, &AuthToken{
		UserID:     u.ID,
		TokenID:    pair.RefreshJTI,
		TokenHash:  auth.HashToken(pair.RefreshToken),
		DeviceID:   meta.DeviceID,
		DeviceName: meta.DeviceName,
		IP:         meta.IP,
		UserAgent:  meta.UserAgent,
		ExpiresAt:  time.Now().Add(uc.tokenManager.RefreshTTL()),
	}); err != nil {
		return nil, err
	}
	return &LoginResult{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresAt: pair.ExpiresAt.Unix(), UserID: u.ID, Username: u.Username, Avatar: u.Avatar, Roles: u.Roles}, nil
}

func (uc *SecurityUsecase) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return uc.repo.RevokeRefreshTokenByHash(ctx, auth.HashToken(refreshToken), "logout")
}

func (uc *SecurityUsecase) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string, meta LoginMeta) error {
	u, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrPasswordMismatch
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := uc.users.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return err
	}
	if err := uc.repo.RevokeAllRefreshTokens(ctx, userID, "password_changed"); err != nil {
		return err
	}
	return uc.repo.CreateSecurityEvent(ctx, &SecurityEvent{
		UserID:    userID,
		EventType: "password_changed",
		RiskLevel: "medium",
		IP:        meta.IP,
		UserAgent: meta.UserAgent,
	})
}

func (uc *SecurityUsecase) ListLoginLogs(ctx context.Context, userID int64, page, pageSize int32) ([]*LoginLog, int64, error) {
	return uc.repo.ListLoginLogs(ctx, userID, page, pageSize)
}

func (uc *SecurityUsecase) ListTokens(ctx context.Context, userID int64) ([]*AuthToken, error) {
	return uc.repo.ListRefreshTokens(ctx, userID)
}

func (uc *SecurityUsecase) RevokeToken(ctx context.Context, userID int64, tokenID string) error {
	return uc.repo.RevokeRefreshToken(ctx, userID, tokenID, "manual_revoke")
}

func (uc *SecurityUsecase) RevokeAllTokens(ctx context.Context, userID int64) error {
	return uc.repo.RevokeAllRefreshTokens(ctx, userID, "manual_revoke_all")
}

func (uc *SecurityUsecase) ListSecurityEvents(ctx context.Context, userID int64, page, pageSize int32) ([]*SecurityEvent, int64, error) {
	return uc.repo.ListSecurityEvents(ctx, userID, page, pageSize)
}
