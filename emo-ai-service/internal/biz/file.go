package biz

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

const (
	AvatarBizType        = "avatar"
	MaxAvatarSize        = 5 * 1024 * 1024
	MaxKnowledgeFileSize = 20 * 1024 * 1024
)

var (
	ErrUnsupportedUpload    = errors.New("unsupported upload type")
	ErrInvalidAvatar        = errors.New("avatar must be a JPEG, PNG, WebP, or GIF image no larger than 5 MB")
	ErrFileStorageMissing   = errors.New("file storage is not configured")
	ErrInvalidKnowledgeFile = errors.New("knowledge file must be TXT, Markdown, PDF, or DOCX and no larger than 20 MB")
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

type KnowledgeFile struct {
	ObjectReference string
	ObjectKey       string
	Name            string
	SizeBytes       int64
	LastModified    time.Time
}

type FileRepo interface {
	CreateFile(ctx context.Context, file *FileAsset) (*FileAsset, error)
	GetFile(ctx context.Context, userID, id int64) (*FileAsset, error)
	DeleteFile(ctx context.Context, userID, id int64) error
	UploadAvatar(ctx context.Context, objectKey, mimeType string, content []byte) (string, error)
	UploadKnowledge(ctx context.Context, objectKey, mimeType string, content []byte) (string, error)
	ListKnowledgeFiles(ctx context.Context, userID int64) ([]*KnowledgeFile, error)
}

func (uc *FileUsecase) UploadKnowledge(ctx context.Context, userID int64, filename, mimeType string, content []byte) (string, error) {
	if userID <= 0 || len(content) == 0 || len(content) > MaxKnowledgeFileSize {
		return "", ErrInvalidKnowledgeFile
	}
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	allowed := map[string]bool{"txt": true, "md": true, "markdown": true, "pdf": true, "docx": true}
	if !allowed[extension] {
		return "", ErrInvalidKnowledgeFile
	}
	if strings.TrimSpace(mimeType) == "" {
		mimeType = "application/octet-stream"
	}
	objectKey := fmt.Sprintf(
		"knowledge/%d/%s/%s/%s",
		userID,
		time.Now().Format("20060102"),
		uuid.NewString(),
		knowledgeFilename(filename, extension),
	)
	return uc.repo.UploadKnowledge(ctx, objectKey, mimeType, bytes.Clone(content))
}

func (uc *FileUsecase) ListKnowledgeFiles(ctx context.Context, userID int64) ([]*KnowledgeFile, error) {
	if userID <= 0 {
		return nil, ErrInvalidKnowledgeFile
	}
	return uc.repo.ListKnowledgeFiles(ctx, userID)
}

func knowledgeFilename(filename, extension string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(filename), "\\", "/")
	name := strings.TrimSpace(path.Base(normalized))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." || name == ".." {
		return "document." + extension
	}
	return name
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
