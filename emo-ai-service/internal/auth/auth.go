package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"emo-ai-service/internal/conf"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/google/wire"
	"google.golang.org/protobuf/types/known/durationpb"
)

var ProviderSet = wire.NewSet(NewTokenManager)

type contextKey string

const userIDKey contextKey = "auth_user_id"

type Claims struct {
	UserID int64    `json:"uid"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	AccessJTI    string
	RefreshJTI   string
	ExpiresAt    time.Time
}

type TokenManager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenManager(c *conf.Auth) *TokenManager {
	if c == nil {
		c = &conf.Auth{}
	}
	secret := c.GetJwtSecret()
	if secret == "" {
		secret = "please-change-this-secret-in-production"
	}
	issuer := c.GetIssuer()
	if issuer == "" {
		issuer = "emo-ai-service"
	}
	return &TokenManager{
		secret:     []byte(secret),
		issuer:     issuer,
		accessTTL:  durationOrDefault(c.GetAccessTokenTtl(), 2*time.Hour),
		refreshTTL: durationOrDefault(c.GetRefreshTokenTtl(), 30*24*time.Hour),
	}
}

func durationOrDefault(v *durationpb.Duration, fallback time.Duration) time.Duration {
	if v == nil {
		return fallback
	}
	d := v.AsDuration()
	if d <= 0 {
		return fallback
	}
	return d
}

func (m *TokenManager) RefreshTTL() time.Duration {
	return m.refreshTTL
}

func (m *TokenManager) IssuePair(userID int64, roles []string) (*TokenPair, error) {
	now := time.Now()
	accessJTI := uuid.NewString()
	refreshJTI := uuid.NewString()
	expiresAt := now.Add(m.accessTTL)
	claims := Claims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        accessJTI,
			Issuer:    m.issuer,
			Subject:   stringFromInt64(userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: uuid.NewString() + "." + uuid.NewString(),
		AccessJTI:    accessJTI,
		RefreshJTI:   refreshJTI,
		ExpiresAt:    expiresAt,
	}, nil
}

func (m *TokenManager) Parse(accessToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(accessToken, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected jwt signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok && userID > 0
}

func MustUserID(ctx context.Context) (int64, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "login required")
	}
	return userID, nil
}

func ServerMiddleware(tm *TokenManager, publicOperations map[string]bool) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if tm == nil {
				return next(ctx, req)
			}
			if tr, ok := transport.FromServerContext(ctx); ok {
				if publicOperations[tr.Operation()] {
					return next(ctx, req)
				}
				authorization := tr.RequestHeader().Get("Authorization")
				token := bearerToken(authorization)
				if token == "" {
					return nil, kerrors.Unauthorized("UNAUTHORIZED", "missing access token")
				}
				claims, err := tm.Parse(token)
				if err != nil {
					return nil, kerrors.Unauthorized("UNAUTHORIZED", "invalid access token")
				}
				ctx = WithUserID(ctx, claims.UserID)
			}
			return next(ctx, req)
		}
	}
}

func bearerToken(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Fields(value)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func stringFromInt64(v int64) string {
	return strconvFormatInt(v)
}
