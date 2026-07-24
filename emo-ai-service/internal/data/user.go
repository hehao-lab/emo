package data

import (
	"context"
	"encoding/json"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

type UserModel struct {
	ID           int64          `gorm:"primaryKey;autoIncrement;comment:用户ID"`
	Username     string         `gorm:"type:varchar(64);uniqueIndex;not null;comment:登录用户名"`
	PasswordHash string         `gorm:"type:varchar(255);not null;comment:密码哈希值"`
	Phone        string         `gorm:"type:varchar(20);uniqueIndex;comment:手机号"`
	Email        string         `gorm:"type:varchar(128);uniqueIndex;comment:邮箱"`
	Avatar       string         `gorm:"type:varchar(512);default:'';comment:头像地址"`
	Roles        string         `gorm:"type:json;not null;comment:角色列表JSON"`
	Status       int32          `gorm:"not null;default:1;comment:账号状态 1正常 2冻结 3注销"`
	LastLoginAt  *time.Time     `gorm:"comment:最后登录时间"`
	CreatedAt    time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt    gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (UserModel) TableName() string { return "users" }

func migrateLegacyUserPasswordColumn(db *gorm.DB) error {
	if !db.Migrator().HasColumn(&UserModel{}, "password") {
		return nil
	}
	return db.Migrator().DropColumn(&UserModel{}, "password")
}

type UserProfileModel struct {
	ID         int64          `gorm:"primaryKey;autoIncrement;comment:资料ID"`
	UserID     int64          `gorm:"uniqueIndex;not null;comment:用户ID"`
	Nickname   string         `gorm:"type:varchar(64);default:'';comment:昵称"`
	AvatarURL  string         `gorm:"type:varchar(512);default:'';comment:头像地址"`
	Gender     string         `gorm:"type:varchar(16);default:'';comment:性别"`
	Birthday   string         `gorm:"type:varchar(16);default:'';comment:生日"`
	Bio        string         `gorm:"type:varchar(512);default:'';comment:个人简介"`
	Location   string         `gorm:"type:varchar(128);default:'';comment:所在地区"`
	Occupation string         `gorm:"type:varchar(128);default:'';comment:职业"`
	Industry   string         `gorm:"type:varchar(128);default:'';comment:行业"`
	Language   string         `gorm:"type:varchar(32);default:'zh-CN';comment:语言偏好"`
	Timezone   string         `gorm:"type:varchar(64);default:'';comment:时区"`
	Extra      string         `gorm:"type:json;comment:扩展资料JSON"`
	CreatedAt  time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt  gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (UserProfileModel) TableName() string { return "user_profiles" }

func newUserProfileModel(userID int64) UserProfileModel {
	profile := UserProfileModel{UserID: userID}
	normalizeUserProfileModel(&profile)
	return profile
}

func normalizeUserProfileModel(profile *UserProfileModel) {
	if profile == nil {
		return
	}
	if profile.Language == "" {
		profile.Language = "zh-CN"
	}
	if profile.Extra == "" {
		profile.Extra = "{}"
	}
}

type userRepoImpl struct {
	db      *gorm.DB
	storage *minioStorage
}

func NewUserRepo(d *Data) biz.UserRepo {
	return &userRepoImpl{db: d.db}
}

func NewProfileRepo(d *Data) biz.ProfileRepo {
	return &userRepoImpl{db: d.db, storage: newMinioStorage()}
}

func NewUserAccountRepo(d *Data) biz.UserAccountRepo {
	return &userRepoImpl{db: d.db}
}

func (r *userRepoImpl) Create(ctx context.Context, u *biz.User) (*biz.User, error) {
	rolesJSON, _ := json.Marshal(u.Roles)
	model := &UserModel{
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Phone:        u.Phone,
		Email:        u.Email,
		Avatar:       u.Avatar,
		Roles:        string(rolesJSON),
		Status:       u.Status,
	}
	if model.Roles == "" {
		model.Roles = `["user"]`
	}
	if model.Status == 0 {
		model.Status = 1
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toBizUser(model), nil
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
	return toBizUser(&model), nil
}

func (r *userRepoImpl) FindByPhone(ctx context.Context, phone string) (*biz.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizUser(&model), nil
}

func (r *userRepoImpl) FindByEmail(ctx context.Context, email string) (*biz.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizUser(&model), nil
}

func (r *userRepoImpl) FindByID(ctx context.Context, id int64) (*biz.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).First(&model, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizUser(&model), nil
}

func (r *userRepoImpl) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", userID).Update("password_hash", passwordHash).Error
}

func (r *userRepoImpl) FindProfile(ctx context.Context, userID int64) (*biz.UserProfile, error) {
	var user UserModel
	err := r.db.WithContext(ctx).First(&user, userID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var profile UserProfileModel
	err = r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	if err == gorm.ErrRecordNotFound {
		return &biz.UserProfile{
			UserID:    user.ID,
			Username:  user.Username,
			Phone:     user.Phone,
			Email:     user.Email,
			AvatarURL: user.Avatar,
			Roles:     rolesFromJSON(user.Roles),
			Language:  "zh-CN",
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizProfile(&user, &profile), nil
}

func (r *userRepoImpl) UpsertProfile(ctx context.Context, p *biz.UserProfile) (*biz.UserProfile, error) {
	var profile UserProfileModel
	err := r.db.WithContext(ctx).Where("user_id = ?", p.UserID).First(&profile).Error
	if err == gorm.ErrRecordNotFound {
		profile = newUserProfileModel(p.UserID)
	} else if err != nil {
		return nil, err
	}
	if p.Nickname != "" {
		profile.Nickname = p.Nickname
	}
	if p.AvatarURL != "" {
		profile.AvatarURL = p.AvatarURL
	}
	if p.Email != "" {
		if err := r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", p.UserID).Update("email", p.Email).Error; err != nil {
			return nil, err
		}
	}
	if p.Gender != "" {
		profile.Gender = p.Gender
	}
	if p.Birthday != "" {
		profile.Birthday = p.Birthday
	}
	if p.Bio != "" {
		profile.Bio = p.Bio
	}
	if p.Location != "" {
		profile.Location = p.Location
	}
	if p.Occupation != "" {
		profile.Occupation = p.Occupation
	}
	if p.Industry != "" {
		profile.Industry = p.Industry
	}
	if p.Language != "" {
		profile.Language = p.Language
	}
	if p.Timezone != "" {
		profile.Timezone = p.Timezone
	}
	normalizeUserProfileModel(&profile)
	if profile.ID == 0 {
		if err := r.db.WithContext(ctx).Create(&profile).Error; err != nil {
			return nil, err
		}
	} else if err := r.db.WithContext(ctx).Save(&profile).Error; err != nil {
		return nil, err
	}
	return r.FindProfile(ctx, p.UserID)
}

func (r *userRepoImpl) UpdateAvatar(ctx context.Context, userID int64, avatarURL string) (*biz.UserProfile, error) {
	if r.storage == nil || !r.storage.ownsAvatarURL(userID, avatarURL) {
		return nil, biz.ErrInvalidAvatar
	}
	return r.UpsertProfile(ctx, &biz.UserProfile{UserID: userID, AvatarURL: avatarURL})
}

func toBizUser(model *UserModel) *biz.User {
	return &biz.User{
		ID:           model.ID,
		Username:     model.Username,
		PasswordHash: model.PasswordHash,
		Phone:        model.Phone,
		Email:        model.Email,
		Avatar:       model.Avatar,
		Roles:        rolesFromJSON(model.Roles),
		Status:       model.Status,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

func toBizProfile(user *UserModel, profile *UserProfileModel) *biz.UserProfile {
	avatar := profile.AvatarURL
	if avatar == "" {
		avatar = user.Avatar
	}
	return &biz.UserProfile{
		UserID:     user.ID,
		Username:   user.Username,
		Phone:      user.Phone,
		Email:      user.Email,
		Nickname:   profile.Nickname,
		AvatarURL:  avatar,
		Roles:      rolesFromJSON(user.Roles),
		Gender:     profile.Gender,
		Birthday:   profile.Birthday,
		Bio:        profile.Bio,
		Location:   profile.Location,
		Occupation: profile.Occupation,
		Industry:   profile.Industry,
		Language:   profile.Language,
		Timezone:   profile.Timezone,
		CreatedAt:  profile.CreatedAt,
		UpdatedAt:  profile.UpdatedAt,
	}
}

func rolesFromJSON(raw string) []string {
	var roles []string
	_ = json.Unmarshal([]byte(raw), &roles)
	if len(roles) == 0 {
		return []string{"user"}
	}
	return roles
}
