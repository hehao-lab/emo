package data

import (
	"context"
	"os"
	"time"

	"emo-ai-service/internal/conf"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/durationpb"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewUserRepo,
	NewVerificationCodeRepo,
	NewEmailSender,
	NewProfileRepo,
	NewUserAccountRepo,
	NewSecurityRepo,
	NewDiaryRepo,
	NewChatRepo,
	NewAIClient,
	NewAIChatRepo,
	NewEmotionRepo,
	NewEmotionAnalyzer,
	NewSystemRepo,
	NewFileRepo,
	NewAdminRepo,
)

type Data struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewData(c *conf.Data) (*Data, func(), error) {
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	if err := db.AutoMigrate(
		&UserModel{},
		&UserProfileModel{},
		&PersonalProfileModel{},
		&TargetProfileModel{},
		&ImportantRecordModel{},
		&AuthRefreshTokenModel{},
		&LoginLogModel{},
		&SecurityEventModel{},
		&MoodDiaryModel{},
		&MoodTagModel{},
		&MoodDiaryTagModel{},
		&MoodDiaryAttachmentModel{},
		&ChatSessionModel{},
		&ChatMessageModel{},
		&ChatContextSummaryModel{},
		&ChatFeedbackModel{},
		&EmotionAnalysisModel{},
		&EmotionDimensionScoreModel{},
		&EmotionDailyStatModel{},
		&SystemConfigModel{},
		&AppVersionModel{},
		&SystemAnnouncementModel{},
		&FileAssetModel{},
	); err != nil {
		return nil, nil, err
	}
	if err := migrateLegacyUserPasswordColumn(db); err != nil {
		return nil, nil, err
	}
	if err := applyTableComments(db); err != nil {
		return nil, nil, err
	}

	rdb := newRedisClient(c.GetRedis())
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		log.Info("closing the data resources")
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		if rdb != nil {
			_ = rdb.Close()
		}
	}
	return &Data{db: db, rdb: rdb}, cleanup, nil
}

func newRedisClient(c *conf.Data_Redis) *redis.Client {
	if c == nil {
		c = &conf.Data_Redis{}
	}
	network := c.GetNetwork()
	if network == "" {
		network = "tcp"
	}
	addr := c.GetAddr()
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	readTimeout := protoDurationOrDefault(c.GetReadTimeout(), 3*time.Second)
	writeTimeout := protoDurationOrDefault(c.GetWriteTimeout(), 3*time.Second)
	return redis.NewClient(&redis.Options{
		Network:      network,
		Addr:         addr,
		Password:     os.Getenv("EMO_REDIS_PASSWORD"),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	})
}

func protoDurationOrDefault(d *durationpb.Duration, fallback time.Duration) time.Duration {
	if d == nil || d.AsDuration() <= 0 {
		return fallback
	}
	return d.AsDuration()
}
