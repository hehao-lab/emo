package service

import (
	"context"
	"errors"

	v1 "emo-ai-service/api/security/v1"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SecurityService struct {
	uc *biz.SecurityUsecase
}

func NewSecurityService(uc *biz.SecurityUsecase) *SecurityService {
	return &SecurityService{uc: uc}
}

var _ v1.SecurityServiceHTTPServer = (*SecurityService)(nil)

// RefreshToken 实现 token 刷新接口：校验 refresh token，轮换旧 token 并签发新的 JWT。
func (s *SecurityService) RefreshToken(ctx context.Context, req *v1.RefreshTokenRequest) (*v1.RefreshTokenResponse, error) {
	result, err := s.uc.RefreshToken(ctx, req.GetRefreshToken(), requestMeta(ctx))
	if err != nil {
		if errors.Is(err, biz.ErrTokenInvalid) {
			return nil, kerrors.Unauthorized("INVALID_REFRESH_TOKEN", "refresh token is invalid")
		}
		if errors.Is(err, biz.ErrUserDisabled) {
			return nil, kerrors.New(403, "USER_DISABLED", "user is disabled")
		}
		return nil, err
	}
	return &v1.RefreshTokenResponse{AccessToken: result.AccessToken, RefreshToken: result.RefreshToken, ExpiresAt: result.ExpiresAt}, nil
}

// Logout 实现退出登录接口：撤销客户端提交的 refresh token。
func (s *SecurityService) Logout(ctx context.Context, req *v1.LogoutRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.uc.Logout(ctx, req.GetRefreshToken())
}

// ChangePassword 实现密码修改接口：校验旧密码，更新密码哈希，并撤销该用户所有 refresh token。
func (s *SecurityService) ChangePassword(ctx context.Context, req *v1.ChangePasswordRequest) (*emptypb.Empty, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.ChangePassword(ctx, userID, req.GetOldPassword(), req.GetNewPassword(), requestMeta(ctx)); err != nil {
		if errors.Is(err, biz.ErrPasswordMismatch) {
			return nil, kerrors.BadRequest("PASSWORD_MISMATCH", "old password is invalid")
		}
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ListLoginLogs 实现登录日志接口：分页返回当前用户的登录成功和失败记录。
func (s *SecurityService) ListLoginLogs(ctx context.Context, req *v1.ListLoginLogsRequest) (*v1.ListLoginLogsResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListLoginLogs(ctx, userID, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.LoginLog, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.LoginLog{Id: item.ID, Username: item.Username, LoginType: item.LoginType, Success: item.Success, FailReason: item.FailReason, Ip: item.IP, UserAgent: item.UserAgent, DeviceId: item.DeviceID, Location: item.Location, CreatedAt: item.CreatedAt.Unix()})
	}
	return &v1.ListLoginLogsResponse{Logs: out, Total: total}, nil
}

// ListTokens 实现登录设备/token 管理接口：返回当前用户所有 refresh token 会话。
func (s *SecurityService) ListTokens(ctx context.Context, req *v1.ListTokensRequest) (*v1.ListTokensResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.uc.ListTokens(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.AuthToken, 0, len(items))
	for _, item := range items {
		var revokedAt int64
		if item.RevokedAt != nil {
			revokedAt = item.RevokedAt.Unix()
		}
		out = append(out, &v1.AuthToken{TokenId: item.TokenID, DeviceId: item.DeviceID, DeviceName: item.DeviceName, Ip: item.IP, UserAgent: item.UserAgent, ExpiresAt: item.ExpiresAt.Unix(), RevokedAt: revokedAt, CreatedAt: item.CreatedAt.Unix()})
	}
	return &v1.ListTokensResponse{Tokens: out}, nil
}

// RevokeToken 实现单设备下线接口：撤销当前用户指定 token_id 的 refresh token。
func (s *SecurityService) RevokeToken(ctx context.Context, req *v1.RevokeTokenRequest) (*emptypb.Empty, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.RevokeToken(ctx, userID, req.GetTokenId())
}

// RevokeAllTokens 实现全部设备下线接口：撤销当前用户所有 refresh token。
func (s *SecurityService) RevokeAllTokens(ctx context.Context, req *v1.RevokeAllTokensRequest) (*emptypb.Empty, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.RevokeAllTokens(ctx, userID)
}

// ListSecurityEvents 实现安全事件接口：分页返回密码修改、token 撤销等安全事件。
func (s *SecurityService) ListSecurityEvents(ctx context.Context, req *v1.ListSecurityEventsRequest) (*v1.ListSecurityEventsResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListSecurityEvents(ctx, userID, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.SecurityEvent, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.SecurityEvent{Id: item.ID, EventType: item.EventType, RiskLevel: item.RiskLevel, Ip: item.IP, UserAgent: item.UserAgent, MetadataJson: item.MetadataJSON, CreatedAt: item.CreatedAt.Unix()})
	}
	return &v1.ListSecurityEventsResponse{Events: out, Total: total}, nil
}
