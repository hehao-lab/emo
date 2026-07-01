package biz

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"emo-ai-service/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameExists   = errors.New("username already exists")
	ErrPhoneExists      = errors.New("phone already exists")
	ErrEmailExists      = errors.New("email already exists")
	ErrUserNotFound     = errors.New("user not found")
	ErrPasswordMismatch = errors.New("password mismatch")
	ErrTokenInvalid     = errors.New("token invalid")
	ErrPermissionDenied = errors.New("permission denied")
	ErrCodeExpired      = errors.New("verification code expired")
	ErrCodeMismatch     = errors.New("verification code mismatch")
)

const (
	VerificationSceneRegisterEmail = "register_email"
	registerEmailCodeTTL           = 5 * time.Minute
)

type User struct {
	ID           int64
	Username     string
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
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
}

type VerificationCodeRepo interface {
	Save(ctx context.Context, scene, target, code string, ttl time.Duration) error
	Get(ctx context.Context, scene, target string) (string, error)
	Delete(ctx context.Context, scene, target string) error
}

type EmailSender interface {
	SendVerificationCode(ctx context.Context, email, code string, ttl time.Duration) error
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
	repo              UserRepo
	security          SecurityRepo
	verificationCodes VerificationCodeRepo
	emailSender       EmailSender
	tokenManger       *auth.TokenManager
}

func NewUserUsecase(repo UserRepo, security SecurityRepo, verificationCodes VerificationCodeRepo, emailSender EmailSender, tokenManager *auth.TokenManager) *UserUsecase {
	return &UserUsecase{
		repo:              repo,
		security:          security,
		verificationCodes: verificationCodes,
		emailSender:       emailSender,
		tokenManger:       tokenManager,
	}
}

func (uc *UserUsecase) SendRegisterEmailCode(ctx context.Context, email string) (time.Duration, error) {
	email = normalizeEmail(email)
	existing, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, ErrEmailExists
	}
	code, err := generateVerificationCode(6)
	if err != nil {
		return 0, err
	}
	if err := uc.verificationCodes.Save(ctx, VerificationSceneRegisterEmail, email, code, registerEmailCodeTTL); err != nil {
		return 0, err
	}
	if uc.emailSender != nil {
		if err := uc.emailSender.SendVerificationCode(ctx, email, code, registerEmailCodeTTL); err != nil {
			_ = uc.verificationCodes.Delete(ctx, VerificationSceneRegisterEmail, email)
			return 0, err
		}
	}
	return registerEmailCodeTTL, nil
}

func (uc *UserUsecase) Register(ctx context.Context, username, password, phone, email, verificationCode string) (int64, error) {
	username = strings.TrimSpace(username)
	phone = strings.TrimSpace(phone)
	email = normalizeEmail(email)
	verificationCode = strings.TrimSpace(verificationCode)
	existing, err := uc.repo.FindByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, ErrUsernameExists
	}
	existing, err = uc.repo.FindByPhone(ctx, phone)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, ErrPhoneExists
	}
	existing, err = uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, ErrEmailExists
	}
	savedCode, err := uc.verificationCodes.Get(ctx, VerificationSceneRegisterEmail, email)
	if err != nil {
		return 0, err
	}
	if savedCode == "" {
		return 0, ErrCodeExpired
	}
	if savedCode != verificationCode {
		return 0, ErrCodeMismatch
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	u := &User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Phone:        phone,
		Email:        email,
		Roles:        []string{"user"},
		Status:       1,
	}
	created, err := uc.repo.Create(ctx, u)
	if err != nil {
		return 0, err
	}
	_ = uc.verificationCodes.Delete(ctx, VerificationSceneRegisterEmail, email)
	return created.ID, nil
}

func (uc *UserUsecase) Login(ctx context.Context, phone, password string, meta LoginMeta) (*LoginResult, error) {
	phone = strings.TrimSpace(phone)
	u, err := uc.repo.FindByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	if u == nil {
		uc.recordLogin(ctx, 0, phone, "password", false, "user_not_found", meta)
		return nil, ErrUserNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		uc.recordLogin(ctx, u.ID, phone, "password", false, "password_mismatch", meta)
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
	uc.recordLogin(ctx, u.ID, phone, "password", true, "", meta)
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

func (uc *UserUsecase) GetUserInfo(ctx context.Context, userID int64) (*User, error) {
	u, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (uc *UserUsecase) recordLogin(ctx context.Context, userID int64, account, loginType string, success bool, failReason string, meta LoginMeta) {
	if uc.security == nil {
		return
	}
	_ = uc.security.CreateLoginLog(ctx, &LoginLog{
		UserID:     userID,
		Username:   account,
		LoginType:  loginType,
		Success:    success,
		FailReason: failReason,
		IP:         meta.IP,
		UserAgent:  meta.UserAgent,
		DeviceID:   meta.DeviceID,
	})
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func generateVerificationCode(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	var out strings.Builder
	out.Grow(length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		out.WriteString(fmt.Sprintf("%d", n.Int64()))
	}
	return out.String(), nil
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
