package data

import (
	"emo-ai-service/internal/conf"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewUserRepo)

// Data .
type Data struct {
	db *gorm.DB
}

// NewData .
func NewData(c *conf.Data) (*Data, func(), error) {
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	// 自动建表
	if err := db.AutoMigrate(&UserModel{}); err != nil {
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
