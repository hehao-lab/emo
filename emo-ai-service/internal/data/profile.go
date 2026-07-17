package data

import (
	"context"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

type PersonalProfileModel struct {
	ID                 int64          `gorm:"primaryKey;autoIncrement;comment:个人信息ID"`
	UserID             int64          `gorm:"uniqueIndex;not null;comment:用户ID"`
	Age                int32          `gorm:"default:0;comment:年龄"`
	Gender             string         `gorm:"type:varchar(32);default:'';comment:性别"`
	MBTI               string         `gorm:"column:mbti;type:varchar(16);default:'';comment:MBTI人格"`
	RelationshipStatus string         `gorm:"type:varchar(128);default:'';comment:关系说明"`
	PersonalitySummary string         `gorm:"type:text;comment:性格评价"`
	CreatedAt          time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt          time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt          gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (PersonalProfileModel) TableName() string { return "personal_profiles" }

type TargetProfileModel struct {
	ID                   int64          `gorm:"primaryKey;autoIncrement;comment:目标信息ID"`
	UserID               int64          `gorm:"index;not null;comment:用户ID"`
	PersonalProfileID    int64          `gorm:"index;not null;comment:个人信息ID"`
	Name                 string         `gorm:"type:varchar(128);default:'';comment:对方称呼"`
	Age                  int32          `gorm:"default:0;comment:对方年龄"`
	Gender               string         `gorm:"type:varchar(32);default:'';comment:对方性别"`
	MBTI                 string         `gorm:"column:mbti;type:varchar(16);default:'';comment:MBTI人格"`
	CurrentRelationship  string         `gorm:"type:varchar(128);default:'';comment:当前关系"`
	InteractionFrequency string         `gorm:"type:varchar(128);default:'';comment:互动频率"`
	RelationshipGoal     string         `gorm:"type:varchar(256);default:'';comment:关系目标"`
	PersonalityTraits    string         `gorm:"type:text;comment:对方性格与相处特点"`
	RecentInteraction    string         `gorm:"type:text;comment:最近一次关键互动"`
	CreatedAt            time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt            time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt            gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (TargetProfileModel) TableName() string { return "target_profiles" }

type ImportantRecordModel struct {
	ID                int64          `gorm:"primaryKey;autoIncrement;comment:重要记录ID"`
	UserID            int64          `gorm:"index;not null;comment:用户ID"`
	PersonalProfileID int64          `gorm:"index;not null;comment:个人信息ID"`
	TargetProfileID   int64          `gorm:"index;not null;comment:目标信息ID"`
	Title             string         `gorm:"type:varchar(160);not null;comment:标题"`
	RecordTime        string         `gorm:"type:varchar(32);default:'';comment:记录时间"`
	EventDescription  string         `gorm:"type:text;comment:事件描述"`
	Resolution        string         `gorm:"type:text;comment:矛盾解决方式"`
	ConcernPoint      string         `gorm:"type:text;comment:在意的点"`
	Satisfaction      string         `gorm:"type:varchar(32);default:'';comment:满意度"`
	CreatedAt         time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt         gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (ImportantRecordModel) TableName() string { return "important_records" }

func (r *userRepoImpl) FindPersonalProfile(ctx context.Context, userID int64) (*biz.PersonalProfile, error) {
	var model PersonalProfileModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizPersonalProfile(&model), nil
}

func (r *userRepoImpl) UpsertPersonalProfile(ctx context.Context, profile *biz.PersonalProfile) (*biz.PersonalProfile, error) {
	var model PersonalProfileModel
	err := r.db.WithContext(ctx).Where("user_id = ?", profile.UserID).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		model = PersonalProfileModel{UserID: profile.UserID}
	} else if err != nil {
		return nil, err
	}
	model.Age = profile.Age
	model.Gender = profile.Gender
	model.MBTI = profile.MBTI
	model.RelationshipStatus = profile.RelationshipStatus
	model.PersonalitySummary = profile.PersonalitySummary
	if model.ID == 0 {
		if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
			return nil, err
		}
	} else if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}
	return toBizPersonalProfile(&model), nil
}

func (r *userRepoImpl) ListTargetProfiles(ctx context.Context, userID int64) ([]*biz.TargetProfile, error) {
	var models []TargetProfileModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at desc, id desc").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.TargetProfile, 0, len(models))
	for i := range models {
		out = append(out, toBizTargetProfile(&models[i]))
	}
	return out, nil
}

func (r *userRepoImpl) GetTargetProfile(ctx context.Context, userID, targetID int64) (*biz.TargetProfile, error) {
	var model TargetProfileModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, targetID).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizTargetProfile(&model), nil
}

func (r *userRepoImpl) UpsertTargetProfile(ctx context.Context, target *biz.TargetProfile) (*biz.TargetProfile, error) {
	var model TargetProfileModel
	if target.ID != 0 {
		err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", target.UserID, target.ID).First(&model).Error
		if err == gorm.ErrRecordNotFound {
			return nil, biz.ErrTargetProfileNotFound
		}
		if err != nil {
			return nil, err
		}
	} else {
		model = TargetProfileModel{UserID: target.UserID}
	}
	model.UserID = target.UserID
	model.PersonalProfileID = target.PersonalProfileID
	model.Name = target.Name
	model.Age = target.Age
	model.Gender = target.Gender
	model.MBTI = target.MBTI
	model.CurrentRelationship = target.CurrentRelationship
	model.InteractionFrequency = target.InteractionFrequency
	model.RelationshipGoal = target.RelationshipGoal
	model.PersonalityTraits = target.PersonalityTraits
	model.RecentInteraction = target.RecentInteraction
	if model.ID == 0 {
		if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
			return nil, err
		}
	} else if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}
	return toBizTargetProfile(&model), nil
}

func (r *userRepoImpl) ListImportantRecords(ctx context.Context, userID, targetID int64) ([]*biz.ImportantRecord, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if targetID != 0 {
		q = q.Where("target_profile_id = ?", targetID)
	}
	var models []ImportantRecordModel
	if err := q.Order("updated_at desc, id desc").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.ImportantRecord, 0, len(models))
	for i := range models {
		out = append(out, toBizImportantRecord(&models[i]))
	}
	return out, nil
}

func (r *userRepoImpl) UpsertImportantRecord(ctx context.Context, record *biz.ImportantRecord) (*biz.ImportantRecord, error) {
	var model ImportantRecordModel
	if record.ID != 0 {
		err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", record.UserID, record.ID).First(&model).Error
		if err == gorm.ErrRecordNotFound {
			model = ImportantRecordModel{UserID: record.UserID}
		} else if err != nil {
			return nil, err
		}
	} else {
		model = ImportantRecordModel{UserID: record.UserID}
	}
	model.UserID = record.UserID
	model.PersonalProfileID = record.PersonalProfileID
	model.TargetProfileID = record.TargetProfileID
	model.Title = record.Title
	model.RecordTime = record.RecordTime
	model.EventDescription = record.EventDescription
	model.Resolution = record.Resolution
	model.ConcernPoint = record.ConcernPoint
	model.Satisfaction = record.Satisfaction
	if model.ID == 0 {
		if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
			return nil, err
		}
	} else if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}
	return toBizImportantRecord(&model), nil
}

func (r *userRepoImpl) DeleteImportantRecord(ctx context.Context, userID, recordID int64) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, recordID).Delete(&ImportantRecordModel{}).Error
}

func toBizPersonalProfile(model *PersonalProfileModel) *biz.PersonalProfile {
	return &biz.PersonalProfile{
		ID:                 model.ID,
		UserID:             model.UserID,
		Age:                model.Age,
		Gender:             model.Gender,
		MBTI:               model.MBTI,
		RelationshipStatus: model.RelationshipStatus,
		PersonalitySummary: model.PersonalitySummary,
		CreatedAt:          model.CreatedAt,
		UpdatedAt:          model.UpdatedAt,
	}
}

func toBizTargetProfile(model *TargetProfileModel) *biz.TargetProfile {
	return &biz.TargetProfile{
		ID:                   model.ID,
		UserID:               model.UserID,
		PersonalProfileID:    model.PersonalProfileID,
		Name:                 model.Name,
		Age:                  model.Age,
		Gender:               model.Gender,
		MBTI:                 model.MBTI,
		CurrentRelationship:  model.CurrentRelationship,
		InteractionFrequency: model.InteractionFrequency,
		RelationshipGoal:     model.RelationshipGoal,
		PersonalityTraits:    model.PersonalityTraits,
		RecentInteraction:    model.RecentInteraction,
		CreatedAt:            model.CreatedAt,
		UpdatedAt:            model.UpdatedAt,
	}
}

func toBizImportantRecord(model *ImportantRecordModel) *biz.ImportantRecord {
	return &biz.ImportantRecord{
		ID:                model.ID,
		UserID:            model.UserID,
		PersonalProfileID: model.PersonalProfileID,
		TargetProfileID:   model.TargetProfileID,
		Title:             model.Title,
		RecordTime:        model.RecordTime,
		EventDescription:  model.EventDescription,
		Resolution:        model.Resolution,
		ConcernPoint:      model.ConcernPoint,
		Satisfaction:      model.Satisfaction,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
	}
}
