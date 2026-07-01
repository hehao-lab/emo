package service

import (
	"context"

	v1 "emo-ai-service/api/file/v1"
	"emo-ai-service/internal/biz"

	"google.golang.org/protobuf/types/known/emptypb"
)

type FileService struct {
	uc *biz.FileUsecase
}

func NewFileService(uc *biz.FileUsecase) *FileService {
	return &FileService{uc: uc}
}

var _ v1.FileServiceHTTPServer = (*FileService)(nil)

// CreateUploadToken 实现上传凭证接口：生成本地/对象存储上传地址和最终访问地址。
func (s *FileService) CreateUploadToken(ctx context.Context, req *v1.CreateUploadTokenRequest) (*v1.UploadToken, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	token, err := s.uc.CreateUploadToken(ctx, userID, req.GetBizType(), req.GetFilename(), req.GetMimeType(), req.GetSizeBytes())
	if err != nil {
		return nil, err
	}
	return &v1.UploadToken{Provider: token.Provider, UploadUrl: token.UploadURL, ObjectKey: token.ObjectKey, PublicUrl: token.PublicURL, ExpiresAt: token.ExpiresAt.Unix()}, nil
}

// CreateFile 实现文件登记接口：保存头像、日记附件等文件元数据。
func (s *FileService) CreateFile(ctx context.Context, req *v1.CreateFileRequest) (*v1.FileAsset, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	file, err := s.uc.CreateFile(ctx, &biz.FileAsset{OwnerUserID: userID, BizType: req.GetBizType(), StorageProvider: req.GetStorageProvider(), Bucket: req.GetBucket(), ObjectKey: req.GetObjectKey(), URL: req.GetUrl(), MimeType: req.GetMimeType(), SizeBytes: req.GetSizeBytes(), Checksum: req.GetChecksum()})
	if err != nil {
		return nil, err
	}
	return toFileDTO(file), nil
}

// GetFile 实现文件详情接口：查询当前用户名下的文件资源信息。
func (s *FileService) GetFile(ctx context.Context, req *v1.GetFileRequest) (*v1.FileAsset, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	file, err := s.uc.GetFile(ctx, userID, req.GetId())
	if err != nil {
		return nil, err
	}
	return toFileDTO(file), nil
}

// DeleteFile 实现文件删除接口：软删除当前用户名下的文件资源记录。
func (s *FileService) DeleteFile(ctx context.Context, req *v1.DeleteFileRequest) (*emptypb.Empty, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.DeleteFile(ctx, userID, req.GetId())
}

func toFileDTO(file *biz.FileAsset) *v1.FileAsset {
	if file == nil {
		return &v1.FileAsset{}
	}
	return &v1.FileAsset{Id: file.ID, OwnerUserId: file.OwnerUserID, BizType: file.BizType, StorageProvider: file.StorageProvider, Bucket: file.Bucket, ObjectKey: file.ObjectKey, Url: file.URL, MimeType: file.MimeType, SizeBytes: file.SizeBytes, Checksum: file.Checksum, Status: file.Status, CreatedAt: file.CreatedAt.Unix(), UpdatedAt: file.UpdatedAt.Unix()}
}
