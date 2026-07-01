package service

import (
	"context"
	"strings"

	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
)

// currentUserID 从 JWT 鉴权中间件写入的 context 中读取当前登录用户 ID。
func currentUserID(ctx context.Context) (int64, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "login required")
	}
	return userID, nil
}

// requestMeta 提取 IP、User-Agent 和设备信息，用于登录日志和安全事件记录。
func requestMeta(ctx context.Context) biz.LoginMeta {
	req, ok := khttp.RequestFromServerContext(ctx)
	if !ok || req == nil {
		return biz.LoginMeta{}
	}
	ip := req.Header.Get("X-Forwarded-For")
	if ip != "" {
		ip = strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	if ip == "" {
		ip = req.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = req.RemoteAddr
	}
	return biz.LoginMeta{
		IP:         ip,
		UserAgent:  req.UserAgent(),
		DeviceID:   req.Header.Get("X-Device-ID"),
		DeviceName: req.Header.Get("X-Device-Name"),
	}
}

func unixTime(t interface{ Unix() int64 }) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}
