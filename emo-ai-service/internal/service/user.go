package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	v1 "emo-ai-service/api/user/v1"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type UserService struct {
	v1.UnimplementedUserServiceServer

	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase) *UserService {
	return &UserService{uc: uc}
}

var _ v1.UserServiceHTTPServer = (*UserService)(nil)

func (s *UserService) SendRegisterEmailCode(ctx context.Context, req *v1.SendRegisterEmailCodeRequest) (*v1.SendRegisterEmailCodeResponse, error) {
	email := strings.TrimSpace(req.GetEmail())
	if !validEmail(email) {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "valid email is required")
	}
	ttl, err := s.uc.SendRegisterEmailCode(ctx, email)
	if err != nil {
		if errors.Is(err, biz.ErrEmailExists) {
			return nil, kerrors.BadRequest("EMAIL_EXISTS", "email already exists")
		}
		return nil, err
	}
	return &v1.SendRegisterEmailCodeResponse{ExpiresIn: int64(ttl.Seconds())}, nil
}

func (s *UserService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	if strings.TrimSpace(req.GetUsername()) == "" ||
		strings.TrimSpace(req.GetPassword()) == "" ||
		strings.TrimSpace(req.GetPhone()) == "" ||
		!validEmail(req.GetEmail()) ||
		strings.TrimSpace(req.GetVerificationCode()) == "" {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "username, password, phone, email and verification_code are required")
	}
	userID, err := s.uc.Register(ctx, req.GetUsername(), req.GetPassword(), req.GetPhone(), req.GetEmail(), req.GetVerificationCode())
	if err != nil {
		switch {
		case errors.Is(err, biz.ErrUsernameExists):
			return nil, kerrors.BadRequest("USERNAME_EXISTS", "username already exists")
		case errors.Is(err, biz.ErrPhoneExists):
			return nil, kerrors.BadRequest("PHONE_EXISTS", "phone already exists")
		case errors.Is(err, biz.ErrEmailExists):
			return nil, kerrors.BadRequest("EMAIL_EXISTS", "email already exists")
		case errors.Is(err, biz.ErrCodeExpired):
			return nil, kerrors.BadRequest("VERIFICATION_CODE_EXPIRED", "verification code expired, please request a new one")
		case errors.Is(err, biz.ErrCodeMismatch):
			return nil, kerrors.BadRequest("VERIFICATION_CODE_MISMATCH", "verification code is invalid")
		default:
			return nil, err
		}
	}
	return &v1.RegisterResponse{UserId: userID}, nil
}

func (s *UserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	if strings.TrimSpace(req.GetPhone()) == "" || strings.TrimSpace(req.GetPassword()) == "" {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "phone and password are required")
	}
	result, err := s.uc.Login(ctx, req.GetPhone(), req.GetPassword(), requestMeta(ctx))
	if err != nil {
		if errors.Is(err, biz.ErrUserNotFound) || errors.Is(err, biz.ErrPasswordMismatch) {
			return nil, kerrors.Unauthorized("INVALID_CREDENTIALS", "phone or password is invalid")
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

func (s *UserService) GetUserInfo(ctx context.Context, req *v1.GetUserInfoRequest) (*v1.GetUserInfoResponse, error) {
	if req.GetUserId() <= 0 {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "user_id is required")
	}
	u, err := s.uc.GetUserInfo(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, biz.ErrUserNotFound) {
			return nil, kerrors.NotFound("USER_NOT_FOUND", "user not found")
		}
		return nil, err
	}
	return &v1.GetUserInfoResponse{
		UserId:   u.ID,
		Username: u.Username,
		Avatar:   u.Avatar,
		Roles:    u.Roles,
		Phone:    u.Phone,
		Email:    u.Email,
		Language: "zh-CN",
	}, nil
}

func validEmail(email string) bool {
	email = strings.TrimSpace(email)
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}
