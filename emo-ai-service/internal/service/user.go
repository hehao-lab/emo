package service

import (
	"context"
	"errors"

	v1 "emo-ai-service/api/user/v1"
	"emo-ai-service/internal/biz"
)

type UserService struct {
	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase) *UserService {
	return &UserService{uc: uc}
}

var _ v1.UserServiceHTTPServer = (*UserService)(nil)

// Register 注册
func (s *UserService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	userID, err := s.uc.Register(ctx, req.GetUsername(), req.GetPassword(), req.GetPhone())
	if err != nil {
		if errors.Is(err, biz.ErrUsernameExists) {
			return nil, errors.New("用户已经存在了")
		}
		return nil, errors.New("注册失败")
	}

	return &v1.RegisterResponse{
		UserId: userID,
	}, nil
}

// Login 登录
func (s *UserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	result, err := s.uc.Login(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		if errors.Is(err, biz.ErrUserNotFound) || errors.Is(err, biz.ErrPasswordMismatch) {
			return nil, errors.New("用户名或密码无效")
		}
		return nil, errors.New("登录失败")
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
