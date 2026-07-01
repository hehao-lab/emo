package service

import (
	"context"
	"errors"

	v1 "emo-ai-service/api/user/v1"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type UserService struct {
	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase) *UserService {
	return &UserService{uc: uc}
}

var _ v1.UserServiceHTTPServer = (*UserService)(nil)

// Register 实现用户注册接口：校验用户名、密码、手机号，创建用户并写入加密后的密码。
func (s *UserService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	if req.GetUsername() == "" || req.GetPassword() == "" || req.GetPhone() == "" {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "username, password and phone are required")
	}
	userID, err := s.uc.Register(ctx, req.GetUsername(), req.GetPassword(), req.GetPhone())
	if err != nil {
		if errors.Is(err, biz.ErrUsernameExists) {
			return nil, kerrors.BadRequest("USERNAME_EXISTS", "username already exists")
		}
		return nil, err
	}
	return &v1.RegisterResponse{UserId: userID}, nil
}

// Login 实现用户登录接口：校验账号密码，记录登录上下文，并返回 JWT access token 和 refresh token。
func (s *UserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	result, err := s.uc.Login(ctx, req.GetUsername(), req.GetPassword(), requestMeta(ctx))
	if err != nil {
		if errors.Is(err, biz.ErrUserNotFound) || errors.Is(err, biz.ErrPasswordMismatch) {
			return nil, kerrors.Unauthorized("INVALID_CREDENTIALS", "username or password is invalid")
		}
		return nil, err
	}
	return &v1.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt,
		UserId:       result.UserID,
		Username:     result.Username,
		Avatar:       result.Avatar,
		Roles:        result.Roles,
	}, nil
}
