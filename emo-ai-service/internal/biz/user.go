package biz

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameExists   = errors.New("username already exists")
	ErrUserNotFound     = errors.New("user not found")
	ErrPasswordMismatch = errors.New("password mismatch")
)

// User 业务实体
type User struct {
	ID       int64
	Username string
	Password string
	Phone    string
	Avatar   string
	Roles    []string
}

// UserRepo data 层需要实现的接口
type UserRepo interface {
	Create(ctx context.Context, u *User) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
}

// UserUsecase 用户业务用例
type UserUsecase struct {
	repo UserRepo
}

func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{repo: repo}
}

// Register 注册
func (uc *UserUsecase) Register(ctx context.Context, username, password, phone string) (int64, error) {
	// 检查用户名是否已存在
	existing, err := uc.repo.FindByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, ErrUsernameExists
	}
	// 2. 对明文密码进行加密处理
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	//构建新用户对象
	u := &User{
		Username: username,
		Password: string(hashedPassword),
		Phone:    phone,
		Roles:    []string{"user"},
	}

	//写入数据库中
	created, err := uc.repo.Create(ctx, u)
	if err != nil {
		return 0, err
	}

	return created.ID, nil
}

// Login 登录
func (uc *UserUsecase) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	u, err := uc.repo.FindByUsername(ctx, username)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}

	if u.Password != password {
		return nil, ErrPasswordMismatch
	}

	accessToken := uuid.New().String()
	refreshToken := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		UserID:       u.ID,
		Username:     u.Username,
		Avatar:       u.Avatar,
		Roles:        u.Roles,
	}, nil
}

// LoginResult 登录返回
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	UserID       int64
	Username     string
	Avatar       string
	Roles        []string
}
