package biz

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	AvatarBizType = "avatar"
	MaxAvatarSize = 5 * 1024 * 1024
)

var (
	ErrUnsupportedUpload  = errors.New("unsupported upload type")
	ErrInvalidAvatar      = errors.New("avatar must be a JPEG, PNG, WebP, or GIF image no larger than 5 MB")
	ErrFileStorageMissing = errors.New("file storage is not configured")
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
	UploadAvatar(ctx context.Context, objectKey, mimeType string, content []byte) (string, error)
}

type FileUsecase struct {
	repo FileRepo
}

func NewFileUsecase(repo FileRepo) *FileUsecase {
	return &FileUsecase{repo: repo}
}

func (uc *FileUsecase) CreateUploadToken(ctx context.Context, userID int64, bizType, filename, mimeType string, sizeBytes int64) (*UploadToken, error) {
	return nil, ErrUnsupportedUpload
}

// UploadAvatar stores a selected avatar in the configured object storage.
// The HTTP handler keeps multipart parsing at the transport boundary; the
// usecase owns the object key and image constraints.
func (uc *FileUsecase) UploadAvatar(ctx context.Context, userID int64, filename, mimeType string, content []byte) (*UploadToken, error) {
	if userID <= 0 || len(content) == 0 || len(content) > MaxAvatarSize {
		return nil, ErrInvalidAvatar
	}

	extension, ok := avatarExtension(mimeType)
	if !ok {
		return nil, ErrInvalidAvatar
	}
	if strings.TrimSpace(filename) == "" {
		return nil, ErrInvalidAvatar
	}

	objectKey := fmt.Sprintf("avatars/%d/%s/%s.%s", userID, time.Now().Format("20060102"), uuid.NewString(), extension)
	publicURL, err := uc.repo.UploadAvatar(ctx, objectKey, mimeType, bytes.Clone(content))
	if err != nil {
		return nil, err
	}
	return &UploadToken{
		Provider:  "minio",
		ObjectKey: objectKey,
		PublicURL: publicURL,
		ExpiresAt: time.Now(),
	}, nil
}

func avatarExtension(mimeType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0])) {
	case "image/jpeg":
		return "jpg", true
	case "image/png":
		return "png", true
	case "image/webp":
		return "webp", true
	case "image/gif":
		return "gif", true
	default:
		return "", false
	}
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
