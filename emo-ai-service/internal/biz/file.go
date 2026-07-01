package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type FileAsset struct {
	ID              int64
	OwnerUserID     int64
	BizType         string
	StorageProvider string
	Bucket          string
	ObjectKey       string
	URL             string
	MimeType        string
	SizeBytes       int64
	Checksum        string
	Status          int32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type UploadToken struct {
	Provider  string
	UploadURL string
	ObjectKey string
	PublicURL string
	ExpiresAt time.Time
}

type FileRepo interface {
	CreateFile(ctx context.Context, file *FileAsset) (*FileAsset, error)
	GetFile(ctx context.Context, userID, id int64) (*FileAsset, error)
	DeleteFile(ctx context.Context, userID, id int64) error
}

type FileUsecase struct {
	repo FileRepo
}

func NewFileUsecase(repo FileRepo) *FileUsecase {
	return &FileUsecase{repo: repo}
}

func (uc *FileUsecase) CreateUploadToken(ctx context.Context, userID int64, bizType, filename, mimeType string, sizeBytes int64) (*UploadToken, error) {
	objectKey := bizType + "/" + time.Now().Format("20060102") + "/" + uuid.NewString() + "-" + filename
	publicURL := "/uploads/" + objectKey
	return &UploadToken{
		Provider:  "local",
		UploadURL: publicURL,
		ObjectKey: objectKey,
		PublicURL: publicURL,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}, nil
}

func (uc *FileUsecase) CreateFile(ctx context.Context, file *FileAsset) (*FileAsset, error) {
	if file.StorageProvider == "" {
		file.StorageProvider = "local"
	}
	if file.Status == 0 {
		file.Status = 1
	}
	return uc.repo.CreateFile(ctx, file)
}

func (uc *FileUsecase) GetFile(ctx context.Context, userID, id int64) (*FileAsset, error) {
	return uc.repo.GetFile(ctx, userID, id)
}

func (uc *FileUsecase) DeleteFile(ctx context.Context, userID, id int64) error {
	return uc.repo.DeleteFile(ctx, userID, id)
}
