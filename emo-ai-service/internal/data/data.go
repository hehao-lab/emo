package data

import (
	"emo-ai-service/internal/conf"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewUserRepo,
	NewProfileRepo,
	NewUserAccountRepo,
	NewSecurityRepo,
	NewDiaryRepo,
	NewChatRepo,
	NewAIClient,
	NewEmotionRepo,
	NewEmotionAnalyzer,
	NewSystemRepo,
	NewFileRepo,
)

type Data struct {
	db *gorm.DB
}

func NewData(c *conf.Data) (*Data, func(), error) {
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	if err := db.AutoMigrate(
		&UserModel{},
		&UserProfileModel{},
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
	if err := applyTableComments(db); err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		log.Info("closing the data resources")
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
	return &Data{db: db}, cleanup, nil
}
