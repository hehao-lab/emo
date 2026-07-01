package biz

import (
	"context"
	"errors"
	"time"

	"emo-ai-service/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameExists   = errors.New("username already exists")
	ErrUserNotFound     = errors.New("user not found")
	ErrPasswordMismatch = errors.New("password mismatch")
	ErrTokenInvalid     = errors.New("token invalid")
	ErrPermissionDenied = errors.New("permission denied")
)

type User struct {
	ID           int64
	Username     string
	Password     string
	PasswordHash string
	Phone        string
	Email        string
	Avatar       string
	Roles        []string
	Status       int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserProfile struct {
	UserID     int64
	Username   string
	Phone      string
	Email      string
	Nickname   string
	AvatarURL  string
	Roles      []string
	Gender     string
	Birthday   string
	Bio        string
	Location   string
	Occupation string
	Industry   string
	Language   string
	Timezone   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type LoginMeta struct {
	IP         string
	UserAgent  string
	DeviceID   string
	DeviceName string
}

type UserRepo interface {
	Create(ctx context.Context, u *User) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
}

type UserAccountRepo interface {
	FindByID(ctx context.Context, id int64) (*User, error)
	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
}

type ProfileRepo interface {
	FindByID(ctx context.Context, id int64) (*User, error)
	FindProfile(ctx context.Context, userID int64) (*UserProfile, error)
	UpsertProfile(ctx context.Context, profile *UserProfile) (*UserProfile, error)
}

type UserUsecase struct {
	repo        UserRepo
	security    SecurityRepo
	tokenManger *auth.TokenManager
}

func NewUserUsecase(repo UserRepo, security SecurityRepo, tokenManager *auth.TokenManager) *UserUsecase {
	return &UserUsecase{repo: repo, security: security, tokenManger: tokenManager}
}

func (uc *UserUsecase) Register(ctx context.Context, username, password, phone string) (int64, error) {
	existing, err := uc.repo.FindByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, ErrUsernameExists
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	u := &User{
		Username:     username,
		Password:     string(hashedPassword),
		PasswordHash: string(hashedPassword),
		Phone:        phone,
		Roles:        []string{"user"},
		Status:       1,
	}
	created, err := uc.repo.Create(ctx, u)
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}

func (uc *UserUsecase) Login(ctx context.Context, username, password string, meta LoginMeta) (*LoginResult, error) {
	u, err := uc.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		uc.recordLogin(ctx, 0, username, false, "user_not_found", meta)
		return nil, ErrUserNotFound
	}
	passwordHash := u.PasswordHash
	if passwordHash == "" {
		passwordHash = u.Password
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		uc.recordLogin(ctx, u.ID, username, false, "password_mismatch", meta)
		return nil, ErrPasswordMismatch
	}
	pair, err := uc.tokenManger.IssuePair(u.ID, u.Roles)
	if err != nil {
		return nil, err
	}
	if uc.security != nil {
		err = uc.security.CreateRefreshToken(ctx, &AuthToken{
			UserID:     u.ID,
			TokenID:    pair.RefreshJTI,
			TokenHash:  auth.HashToken(pair.RefreshToken),
			DeviceID:   meta.DeviceID,
			DeviceName: meta.DeviceName,
			IP:         meta.IP,
			UserAgent:  meta.UserAgent,
			ExpiresAt:  time.Now().Add(uc.tokenManger.RefreshTTL()),
		})
		if err != nil {
			return nil, err
		}
	}
	uc.recordLogin(ctx, u.ID, username, true, "", meta)
	return &LoginResult{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt.Unix(),
		UserID:       u.ID,
		Username:     u.Username,
		Avatar:       u.Avatar,
		Roles:        u.Roles,
	}, nil
}

func (uc *UserUsecase) recordLogin(ctx context.Context, userID int64, username string, success bool, failReason string, meta LoginMeta) {
	if uc.security == nil {
		return
	}
	_ = uc.security.CreateLoginLog(ctx, &LoginLog{
		UserID:     userID,
		Username:   username,
		LoginType:  "password",
		Success:    success,
		FailReason: failReason,
		IP:         meta.IP,
		UserAgent:  meta.UserAgent,
		DeviceID:   meta.DeviceID,
	})
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	UserID       int64
	Username     string
	Avatar       string
	Roles        []string
}

type ProfileUsecase struct {
	repo ProfileRepo
}

func NewProfileUsecase(repo ProfileRepo) *ProfileUsecase {
	return &ProfileUsecase{repo: repo}
}

func (uc *ProfileUsecase) GetMe(ctx context.Context, userID int64) (*UserProfile, error) {
	profile, err := uc.repo.FindProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile != nil {
		return profile, nil
	}
	u, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	return &UserProfile{
		UserID:    u.ID,
		Username:  u.Username,
		Phone:     u.Phone,
		Email:     u.Email,
		AvatarURL: u.Avatar,
		Roles:     u.Roles,
		Language:  "zh-CN",
	}, nil
}

func (uc *ProfileUsecase) UpdateProfile(ctx context.Context, p *UserProfile) (*UserProfile, error) {
	return uc.repo.UpsertProfile(ctx, p)
}

func (uc *ProfileUsecase) UpdateAvatar(ctx context.Context, userID int64, avatarURL string) (*UserProfile, error) {
	return uc.repo.UpsertProfile(ctx, &UserProfile{UserID: userID, AvatarURL: avatarURL})
}
