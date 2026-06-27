package data

import (
	"context"
	"encoding/json"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

// UserModel 数据库模型
type UserModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Username  string    `gorm:"type:varchar(64);uniqueIndex;not null"`
	Password  string    `gorm:"type:varchar(128);not null"`
	Phone     string    `gorm:"type:varchar(20);default:''"`
	Avatar    string    `gorm:"type:varchar(255);default:''"`
	Roles     string    `gorm:"type:varchar(255);default:'[]';comment:JSON array"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (UserModel) TableName() string {
	return "users"
}

type userRepoImpl struct {
	db *gorm.DB
}

func NewUserRepo(d *Data) biz.UserRepo {
	return &userRepoImpl{db: d.db}
}

func (r *userRepoImpl) Create(ctx context.Context, u *biz.User) (*biz.User, error) {
	rolesJSON, _ := json.Marshal(u.Roles)
	model := &UserModel{
		Username: u.Username,
		Password: u.Password,
		Phone:    u.Phone,
		Avatar:   u.Avatar,
		Roles:    string(rolesJSON),
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}

	u.ID = model.ID
	return u, nil
}

func (r *userRepoImpl) FindByUsername(ctx context.Context, username string) (*biz.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var roles []string
	json.Unmarshal([]byte(model.Roles), &roles)

	return &biz.User{
		ID:       model.ID,
		Username: model.Username,
		Password: model.Password,
		Phone:    model.Phone,
		Avatar:   model.Avatar,
		Roles:    roles,
	}, nil
}
