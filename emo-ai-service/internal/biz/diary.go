package biz

import (
	"context"
	"time"
)

type MoodDiary struct {
	ID             int64
	UserID         int64
	Title          string
	Content        string
	Mood           string
	MoodScore      int32
	Weather        string
	Location       string
	OccurredOn     string
	Visibility     string
	Tags           []*MoodTag
	AttachmentURLs []string
	AnalysisID     int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MoodTag struct {
	ID        int64
	UserID    int64
	Name      string
	Color     string
	Icon      string
	Sort      int32
	System    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DiaryListOption struct {
	Page      int32
	PageSize  int32
	TagID     int64
	Mood      string
	StartDate string
	EndDate   string
}

type DiaryRepo interface {
	CreateDiary(ctx context.Context, diary *MoodDiary, tagIDs []int64) (*MoodDiary, error)
	ListDiaries(ctx context.Context, userID int64, opt DiaryListOption) ([]*MoodDiary, int64, error)
	GetDiary(ctx context.Context, userID, id int64) (*MoodDiary, error)
	UpdateDiary(ctx context.Context, diary *MoodDiary, tagIDs []int64) (*MoodDiary, error)
	DeleteDiary(ctx context.Context, userID, id int64) error
	ListTags(ctx context.Context, userID int64) ([]*MoodTag, error)
	CreateTag(ctx context.Context, tag *MoodTag) (*MoodTag, error)
	UpdateTag(ctx context.Context, tag *MoodTag) (*MoodTag, error)
	DeleteTag(ctx context.Context, userID, id int64) error
}

type DiaryUsecase struct {
	repo DiaryRepo
}

func NewDiaryUsecase(repo DiaryRepo) *DiaryUsecase {
	return &DiaryUsecase{repo: repo}
}

func (uc *DiaryUsecase) CreateDiary(ctx context.Context, diary *MoodDiary, tagIDs []int64) (*MoodDiary, error) {
	if diary.Visibility == "" {
		diary.Visibility = "private"
	}
	if diary.OccurredOn == "" {
		diary.OccurredOn = time.Now().Format("2006-01-02")
	}
	return uc.repo.CreateDiary(ctx, diary, tagIDs)
}

func (uc *DiaryUsecase) ListDiaries(ctx context.Context, userID int64, opt DiaryListOption) ([]*MoodDiary, int64, error) {
	return uc.repo.ListDiaries(ctx, userID, opt)
}

func (uc *DiaryUsecase) GetDiary(ctx context.Context, userID, id int64) (*MoodDiary, error) {
	return uc.repo.GetDiary(ctx, userID, id)
}

func (uc *DiaryUsecase) UpdateDiary(ctx context.Context, diary *MoodDiary, tagIDs []int64) (*MoodDiary, error) {
	return uc.repo.UpdateDiary(ctx, diary, tagIDs)
}

func (uc *DiaryUsecase) DeleteDiary(ctx context.Context, userID, id int64) error {
	return uc.repo.DeleteDiary(ctx, userID, id)
}

func (uc *DiaryUsecase) ListTags(ctx context.Context, userID int64) ([]*MoodTag, error) {
	return uc.repo.ListTags(ctx, userID)
}

func (uc *DiaryUsecase) CreateTag(ctx context.Context, tag *MoodTag) (*MoodTag, error) {
	return uc.repo.CreateTag(ctx, tag)
}

func (uc *DiaryUsecase) UpdateTag(ctx context.Context, tag *MoodTag) (*MoodTag, error) {
	return uc.repo.UpdateTag(ctx, tag)
}

func (uc *DiaryUsecase) DeleteTag(ctx context.Context, userID, id int64) error {
	return uc.repo.DeleteTag(ctx, userID, id)
}
