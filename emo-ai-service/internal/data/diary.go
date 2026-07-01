package data

import (
	"context"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

type MoodDiaryModel struct {
	ID         int64          `gorm:"primaryKey;autoIncrement;comment:心情日记ID"`
	UserID     int64          `gorm:"index:idx_user_day;not null;comment:用户ID"`
	Title      string         `gorm:"type:varchar(128);default:'';comment:日记标题"`
	Content    string         `gorm:"type:text;not null;comment:日记正文"`
	Mood       string         `gorm:"type:varchar(32);index;default:'';comment:心情类型"`
	MoodScore  int32          `gorm:"default:0;comment:心情分数 1到10"`
	Weather    string         `gorm:"type:varchar(32);default:'';comment:天气"`
	Location   string         `gorm:"type:varchar(128);default:'';comment:记录地点"`
	OccurredOn string         `gorm:"type:date;index:idx_user_day;not null;comment:日记发生日期"`
	Visibility string         `gorm:"type:varchar(16);default:'private';comment:可见性"`
	AnalysisID int64          `gorm:"index;comment:关联情绪分析ID"`
	CreatedAt  time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt  gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (MoodDiaryModel) TableName() string { return "mood_diaries" }

type MoodTagModel struct {
	ID        int64          `gorm:"primaryKey;autoIncrement;comment:心情标签ID"`
	UserID    int64          `gorm:"index;default:0;comment:用户ID 0表示系统标签"`
	Name      string         `gorm:"type:varchar(32);not null;comment:标签名称"`
	Color     string         `gorm:"type:varchar(16);default:'';comment:标签颜色"`
	Icon      string         `gorm:"type:varchar(64);default:'';comment:标签图标"`
	Sort      int32          `gorm:"default:0;comment:排序值"`
	CreatedAt time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (MoodTagModel) TableName() string { return "mood_tags" }

type MoodDiaryTagModel struct {
	DiaryID int64 `gorm:"primaryKey;comment:心情日记ID"`
	TagID   int64 `gorm:"primaryKey;comment:心情标签ID"`
}

func (MoodDiaryTagModel) TableName() string { return "mood_diary_tags" }

type MoodDiaryAttachmentModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;comment:日记附件ID"`
	DiaryID   int64     `gorm:"index;not null;comment:心情日记ID"`
	FileID    int64     `gorm:"index;comment:文件资源ID"`
	URL       string    `gorm:"type:varchar(1024);not null;comment:附件访问地址"`
	Sort      int32     `gorm:"default:0;comment:排序值"`
	CreatedAt time.Time `gorm:"autoCreateTime;comment:创建时间"`
}

func (MoodDiaryAttachmentModel) TableName() string { return "mood_diary_attachments" }

type diaryRepoImpl struct {
	db *gorm.DB
}

func NewDiaryRepo(d *Data) biz.DiaryRepo {
	return &diaryRepoImpl{db: d.db}
}

func (r *diaryRepoImpl) CreateDiary(ctx context.Context, diary *biz.MoodDiary, tagIDs []int64) (*biz.MoodDiary, error) {
	model := &MoodDiaryModel{
		UserID:     diary.UserID,
		Title:      diary.Title,
		Content:    diary.Content,
		Mood:       diary.Mood,
		MoodScore:  diary.MoodScore,
		Weather:    diary.Weather,
		Location:   diary.Location,
		OccurredOn: diary.OccurredOn,
		Visibility: diary.Visibility,
		AnalysisID: diary.AnalysisID,
	}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(model).Error; err != nil {
			return err
		}
		if err := replaceDiaryTags(tx, model.ID, tagIDs); err != nil {
			return err
		}
		return replaceDiaryAttachments(tx, model.ID, diary.AttachmentURLs)
	}); err != nil {
		return nil, err
	}
	return r.GetDiary(ctx, diary.UserID, model.ID)
}

func (r *diaryRepoImpl) ListDiaries(ctx context.Context, userID int64, opt biz.DiaryListOption) ([]*biz.MoodDiary, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&MoodDiaryModel{}).Where("user_id = ?", userID)
	if opt.Mood != "" {
		q = q.Where("mood = ?", opt.Mood)
	}
	if opt.StartDate != "" {
		q = q.Where("occurred_on >= ?", opt.StartDate)
	}
	if opt.EndDate != "" {
		q = q.Where("occurred_on <= ?", opt.EndDate)
	}
	if opt.TagID > 0 {
		q = q.Joins("JOIN mood_diary_tags ON mood_diary_tags.diary_id = mood_diaries.id").Where("mood_diary_tags.tag_id = ?", opt.TagID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []MoodDiaryModel
	if err := q.Order("occurred_on desc, created_at desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.MoodDiary, 0, len(models))
	for i := range models {
		item, err := r.fillDiary(ctx, &models[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (r *diaryRepoImpl) GetDiary(ctx context.Context, userID, id int64) (*biz.MoodDiary, error) {
	var model MoodDiaryModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, id).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.fillDiary(ctx, &model)
}

func (r *diaryRepoImpl) UpdateDiary(ctx context.Context, diary *biz.MoodDiary, tagIDs []int64) (*biz.MoodDiary, error) {
	var model MoodDiaryModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", diary.UserID, diary.ID).First(&model).Error
	if err != nil {
		return nil, err
	}
	if diary.Title != "" {
		model.Title = diary.Title
	}
	if diary.Content != "" {
		model.Content = diary.Content
	}
	if diary.Mood != "" {
		model.Mood = diary.Mood
	}
	if diary.MoodScore > 0 {
		model.MoodScore = diary.MoodScore
	}
	if diary.Weather != "" {
		model.Weather = diary.Weather
	}
	if diary.Location != "" {
		model.Location = diary.Location
	}
	if diary.OccurredOn != "" {
		model.OccurredOn = diary.OccurredOn
	}
	if diary.Visibility != "" {
		model.Visibility = diary.Visibility
	}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&model).Error; err != nil {
			return err
		}
		if tagIDs != nil {
			if err := replaceDiaryTags(tx, model.ID, tagIDs); err != nil {
				return err
			}
		}
		if diary.AttachmentURLs != nil {
			if err := replaceDiaryAttachments(tx, model.ID, diary.AttachmentURLs); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetDiary(ctx, diary.UserID, diary.ID)
}

func (r *diaryRepoImpl) DeleteDiary(ctx context.Context, userID, id int64) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&MoodDiaryModel{}).Error
}

func (r *diaryRepoImpl) ListTags(ctx context.Context, userID int64) ([]*biz.MoodTag, error) {
	var models []MoodTagModel
	if err := r.db.WithContext(ctx).Where("user_id = 0 OR user_id = ?", userID).Order("sort asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.MoodTag, 0, len(models))
	for i := range models {
		out = append(out, toBizTag(&models[i]))
	}
	return out, nil
}

func (r *diaryRepoImpl) CreateTag(ctx context.Context, tag *biz.MoodTag) (*biz.MoodTag, error) {
	model := &MoodTagModel{UserID: tag.UserID, Name: tag.Name, Color: tag.Color, Icon: tag.Icon, Sort: tag.Sort}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toBizTag(model), nil
}

func (r *diaryRepoImpl) UpdateTag(ctx context.Context, tag *biz.MoodTag) (*biz.MoodTag, error) {
	var model MoodTagModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", tag.UserID, tag.ID).First(&model).Error
	if err != nil {
		return nil, err
	}
	if tag.Name != "" {
		model.Name = tag.Name
	}
	if tag.Color != "" {
		model.Color = tag.Color
	}
	if tag.Icon != "" {
		model.Icon = tag.Icon
	}
	if tag.Sort != 0 {
		model.Sort = tag.Sort
	}
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}
	return toBizTag(&model), nil
}

func (r *diaryRepoImpl) DeleteTag(ctx context.Context, userID, id int64) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&MoodTagModel{}).Error
}

func (r *diaryRepoImpl) fillDiary(ctx context.Context, model *MoodDiaryModel) (*biz.MoodDiary, error) {
	var links []MoodDiaryTagModel
	if err := r.db.WithContext(ctx).Where("diary_id = ?", model.ID).Find(&links).Error; err != nil {
		return nil, err
	}
	tagIDs := make([]int64, 0, len(links))
	for _, link := range links {
		tagIDs = append(tagIDs, link.TagID)
	}
	tags := make([]*biz.MoodTag, 0, len(tagIDs))
	if len(tagIDs) > 0 {
		var tagModels []MoodTagModel
		if err := r.db.WithContext(ctx).Where("id IN ?", tagIDs).Find(&tagModels).Error; err != nil {
			return nil, err
		}
		for i := range tagModels {
			tags = append(tags, toBizTag(&tagModels[i]))
		}
	}
	var attachments []MoodDiaryAttachmentModel
	if err := r.db.WithContext(ctx).Where("diary_id = ?", model.ID).Order("sort asc").Find(&attachments).Error; err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(attachments))
	for _, item := range attachments {
		urls = append(urls, item.URL)
	}
	return &biz.MoodDiary{
		ID:             model.ID,
		UserID:         model.UserID,
		Title:          model.Title,
		Content:        model.Content,
		Mood:           model.Mood,
		MoodScore:      model.MoodScore,
		Weather:        model.Weather,
		Location:       model.Location,
		OccurredOn:     model.OccurredOn,
		Visibility:     model.Visibility,
		Tags:           tags,
		AttachmentURLs: urls,
		AnalysisID:     model.AnalysisID,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}, nil
}

func replaceDiaryTags(tx *gorm.DB, diaryID int64, tagIDs []int64) error {
	if err := tx.Where("diary_id = ?", diaryID).Delete(&MoodDiaryTagModel{}).Error; err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if tagID <= 0 {
			continue
		}
		if err := tx.Create(&MoodDiaryTagModel{DiaryID: diaryID, TagID: tagID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func replaceDiaryAttachments(tx *gorm.DB, diaryID int64, urls []string) error {
	if err := tx.Where("diary_id = ?", diaryID).Delete(&MoodDiaryAttachmentModel{}).Error; err != nil {
		return err
	}
	for i, url := range urls {
		if url == "" {
			continue
		}
		if err := tx.Create(&MoodDiaryAttachmentModel{DiaryID: diaryID, URL: url, Sort: int32(i)}).Error; err != nil {
			return err
		}
	}
	return nil
}

func toBizTag(model *MoodTagModel) *biz.MoodTag {
	return &biz.MoodTag{
		ID:        model.ID,
		UserID:    model.UserID,
		Name:      model.Name,
		Color:     model.Color,
		Icon:      model.Icon,
		Sort:      model.Sort,
		System:    model.UserID == 0,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}
