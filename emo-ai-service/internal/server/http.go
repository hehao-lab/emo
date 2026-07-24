package server

import (
	adminv1 "emo-ai-service/api/admin/v1"
	aichatv1 "emo-ai-service/api/aichat/v1"
	chatv1 "emo-ai-service/api/chat/v1"
	diaryv1 "emo-ai-service/api/diary/v1"
	emotionv1 "emo-ai-service/api/emotion/v1"
	filev1 "emo-ai-service/api/file/v1"
	profilev1 "emo-ai-service/api/profile/v1"
	securityv1 "emo-ai-service/api/security/v1"
	systemv1 "emo-ai-service/api/system/v1"
	userv1 "emo-ai-service/api/user/v1"
	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/conf"
	"emo-ai-service/internal/service"
	nethttp "net/http"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
)

func NewHTTPServer(
	c *conf.Server,
	tokenManager *auth.TokenManager,
	userSvc *service.UserService,
	profileSvc *service.ProfileService,
	securitySvc *service.SecurityService,
	diarySvc *service.DiaryService,
	chatSvc *service.ChatService,
	aiChatSvc *service.AIChatService,
	emotionSvc *service.EmotionService,
	systemSvc *service.SystemService,
	fileSvc *service.FileService,
	adminSvc *service.AdminService,
) *http.Server {
	publicOperations := map[string]bool{
		userv1.OperationUserServiceSendRegisterEmailCode: true,
		userv1.OperationUserServiceRegister:              true,
		userv1.OperationUserServiceLogin:                 true,
		securityv1.OperationSecurityServiceRefreshToken:  true,
		securityv1.OperationSecurityServiceLogout:        true,
		systemv1.OperationSystemServiceGetAbout:          true,
		systemv1.OperationSystemServiceListPublicConfigs: true,
		systemv1.OperationSystemServiceGetLatestVersion:  true,
		systemv1.OperationSystemServiceListAnnouncements: true,
		aichatv1.OperationAIChatServiceHealth:            true,
	}
	var opts = []http.ServerOption{
		http.Filter(corsFilter),
		http.ResponseEncoder(contractResponseEncoder),
		http.Middleware(
			recovery.Recovery(),
			auth.ServerMiddleware(tokenManager, publicOperations),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	userv1.RegisterUserServiceHTTPServer(srv, userSvc)
	profilev1.RegisterProfileServiceHTTPServer(srv, profileSvc)
	securityv1.RegisterSecurityServiceHTTPServer(srv, securitySvc)
	diaryv1.RegisterDiaryServiceHTTPServer(srv, diarySvc)
	chatv1.RegisterChatServiceHTTPServer(srv, chatSvc)
	aichatv1.RegisterAIChatServiceHTTPServer(srv, aiChatSvc)
	srv.HandleFunc("/api/v1/chat/stream", aiChatSvc.StreamChatHTTP)
	emotionv1.RegisterEmotionServiceHTTPServer(srv, emotionSvc)
	srv.HandleFunc("/v1/emotion/reports/relationship-health", emotionSvc.RelationshipHealthReportHTTP)
	systemv1.RegisterSystemServiceHTTPServer(srv, systemSvc)
	filev1.RegisterFileServiceHTTPServer(srv, fileSvc)
	srv.HandleFunc("/v1/files/avatar", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		fileSvc.UploadAvatarHTTP(w, r, tokenManager)
	})
	srv.HandleFunc("/v1/files/knowledge", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		fileSvc.UploadKnowledgeHTTP(w, r, tokenManager)
	})
	srv.HandleFunc("/v1/knowledge/files", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		fileSvc.ListKnowledgeHTTP(w, r, tokenManager)
	})
	adminv1.RegisterAdminServiceHTTPServer(srv, adminSvc)
	return srv
}

func contractResponseEncoder(w nethttp.ResponseWriter, r *nethttp.Request, v any) error {
	switch {
	case r.Method == nethttp.MethodPost && r.URL.Path == "/api/v1/conversations":
		w.WriteHeader(nethttp.StatusCreated)
	case r.Method == nethttp.MethodPost && r.URL.Path == "/api/v1/chat":
		w.WriteHeader(nethttp.StatusCreated)
	case r.Method == nethttp.MethodPost && r.URL.Path == "/api/v1/knowledge/documents":
		w.WriteHeader(nethttp.StatusAccepted)
	case r.Method == nethttp.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1/knowledge/documents/") && strings.HasSuffix(r.URL.Path, ":reindex"):
		w.WriteHeader(nethttp.StatusAccepted)
	case r.Method == nethttp.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/knowledge/documents/"):
		w.WriteHeader(nethttp.StatusNoContent)
		_, err := w.Write(nil)
		return err
	}
	return http.DefaultResponseEncoder(w, r, v)
}

func corsFilter(next nethttp.Handler) nethttp.Handler {
	allowedOrigins := make(map[string]struct{})
	for _, value := range strings.Split(os.Getenv("EMO_CORS_ALLOWED_ORIGINS"), ",") {
		if origin := strings.TrimSpace(value); origin != "" {
			allowedOrigins[origin] = struct{}{}
		}
	}
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			_, allowed := allowedOrigins[origin]
			if !allowed && os.Getenv("EMO_ENV") == "production" {
				if r.Method == nethttp.MethodOptions {
					w.WriteHeader(nethttp.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			if !allowed && len(allowedOrigins) > 0 {
				if r.Method == nethttp.MethodOptions {
					w.WriteHeader(nethttp.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,Accept,Idempotency-Key,traceparent,X-Device-ID,X-Device-Name")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length,Content-Type,Idempotency-Replayed,traceparent")
		if r.Method == nethttp.MethodOptions {
			w.WriteHeader(nethttp.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
