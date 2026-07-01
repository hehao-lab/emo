package data

import (
	"context"
	"fmt"
	"time"

	"emo-ai-service/internal/biz"

	"github.com/redis/go-redis/v9"
)

type verificationCodeRepo struct {
	rdb *redis.Client
}

func NewVerificationCodeRepo(d *Data) biz.VerificationCodeRepo {
	return &verificationCodeRepo{rdb: d.rdb}
}

func (r *verificationCodeRepo) Save(ctx context.Context, scene, target, code string, ttl time.Duration) error {
	return r.rdb.Set(ctx, verificationCodeKey(scene, target), code, ttl).Err()
}

func (r *verificationCodeRepo) Get(ctx context.Context, scene, target string) (string, error) {
	code, err := r.rdb.Get(ctx, verificationCodeKey(scene, target)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return code, nil
}

func (r *verificationCodeRepo) Delete(ctx context.Context, scene, target string) error {
	return r.rdb.Del(ctx, verificationCodeKey(scene, target)).Err()
}

func verificationCodeKey(scene, target string) string {
	return fmt.Sprintf("verification:%s:%s", scene, target)
}
