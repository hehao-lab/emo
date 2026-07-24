package data

import (
	"context"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

type FileAssetModel struct {
	ID              int64          `gorm:"primaryKey;autoIncrement;comment:文件资源ID"`
	OwnerUserID     int64          `gorm:"index;comment:所属用户ID"`
	BizType         string         `gorm:"type:varchar(32);index;comment:业务类型 avatar diary system"`
	StorageProvider string         `gorm:"type:varchar(32);default:'local';comment:存储服务商"`
	Bucket          string         `gorm:"type:varchar(128);default:'';comment:存储桶"`
	ObjectKey       string         `gorm:"type:varchar(512);not null;comment:对象存储Key"`
	URL             string         `gorm:"type:varchar(1024);not null;comment:文件访问地址"`
	MimeType        string         `gorm:"type:varchar(128);default:'';comment:文件MIME类型"`
	SizeBytes       int64          `gorm:"default:0;comment:文件大小字节数"`
	Checksum        string         `gorm:"type:varchar(128);default:'';comment:文件校验值"`
	Status          int32          `gorm:"default:1;comment:文件状态"`
	CreatedAt       time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt       gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (FileAssetModel) TableName() string { return "file_assets" }

type fileRepoImpl struct {
	db               *gorm.DB
	storage          *minioStorage
	knowledgeStorage *minioStorage
}

func NewFileRepo(d *Data) biz.FileRepo {
	return &fileRepoImpl{db: d.db, storage: newMinioStorage(), knowledgeStorage: newKnowledgeMinioStorage()}
}

func (r *fileRepoImpl) UploadKnowledge(ctx context.Context, objectKey, mimeType string, content []byte) (string, error) {
	if r.knowledgeStorage == nil || !r.knowledgeStorage.configured() {
		return "", biz.ErrFileStorageMissing
	}
	return r.knowledgeStorage.uploadKnowledge(ctx, objectKey, mimeType, content)
}

func (r *fileRepoImpl) ListKnowledgeFiles(ctx context.Context, userID int64) ([]*biz.KnowledgeFile, error) {
	if r.knowledgeStorage == nil || !r.knowledgeStorage.configured() {
		return nil, biz.ErrFileStorageMissing
	}
	objects, err := r.knowledgeStorage.listKnowledgeObjects(ctx, userID)
	if err != nil {
		return nil, err
	}
	files := make([]*biz.KnowledgeFile, 0, len(objects))
	for _, object := range objects {
		files = append(files, &biz.KnowledgeFile{
			ObjectReference: object.ObjectReference,
			ObjectKey:       object.ObjectKey,
			Name:            object.Name,
			SizeBytes:       object.SizeBytes,
			LastModified:    object.LastModified,
		})
	}
	return files, nil
}

func (r *fileRepoImpl) UploadAvatar(ctx context.Context, objectKey, mimeType string, content []byte) (string, error) {
	if r.storage == nil || !r.storage.configured() {
		return "", biz.ErrFileStorageMissing
	}
	return r.storage.uploadAvatar(ctx, objectKey, mimeType, content)
}

func (r *fileRepoImpl) CreateFile(ctx context.Context, file *biz.FileAsset) (*biz.FileAsset, error) {
	model := &FileAssetModel{
		OwnerUserID:     file.OwnerUserID,
		BizType:         file.BizType,
		StorageProvider: file.StorageProvider,
		Bucket:          file.Bucket,
		ObjectKey:       file.ObjectKey,
		URL:             file.URL,
		MimeType:        file.MimeType,
		SizeBytes:       file.SizeBytes,
		Checksum:        file.Checksum,
		Status:          file.Status,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toBizFile(model), nil
}

func (r *fileRepoImpl) GetFile(ctx context.Context, userID, id int64) (*biz.FileAsset, error) {
	var model FileAssetModel
	err := r.db.WithContext(ctx).Where("owner_user_id = ? AND id = ?", userID, id).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizFile(&model), nil
}

func (r *fileRepoImpl) DeleteFile(ctx context.Context, userID, id int64) error {
	return r.db.WithContext(ctx).Where("owner_user_id = ? AND id = ?", userID, id).Delete(&FileAssetModel{}).Error
}

func toBizFile(model *FileAssetModel) *biz.FileAsset {
	return &biz.FileAsset{
		ID:              model.ID,
		OwnerUserID:     model.OwnerUserID,
		BizType:         model.BizType,
		StorageProvider: model.StorageProvider,
		Bucket:          model.Bucket,
		ObjectKey:       model.ObjectKey,
		URL:             model.URL,
		MimeType:        model.MimeType,
		SizeBytes:       model.SizeBytes,
		Checksum:        model.Checksum,
		Status:          model.Status,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}
