package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	v1 "emo-ai-service/api/file/v1"
	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type FileService struct {
	uc *biz.FileUsecase
}

func (s *FileService) ListKnowledgeHTTP(w http.ResponseWriter, r *http.Request, tokenManager *auth.TokenManager) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID, err := authenticatedHTTPUserID(tokenManager, r)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	files, err := s.uc.ListKnowledgeFiles(r.Context(), userID)
	if err != nil {
		writeJSONError(w, listKnowledgeError(err))
		return
	}
	items := make([]map[string]any, 0, len(files))
	for _, file := range files {
		items = append(items, map[string]any{
			"objectReference": file.ObjectReference,
			"objectKey":       file.ObjectKey,
			"name":            file.Name,
			"sizeBytes":       file.SizeBytes,
			"lastModified":    file.LastModified.UTC().Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "total": len(items)})
}

// UploadAvatarHTTP accepts multipart uploads from uni-app and stores the
// image in MinIO. It is a raw handler because protobuf JSON endpoints cannot
// decode multipart/form-data.
func (s *FileService) UploadAvatarHTTP(w http.ResponseWriter, r *http.Request, tokenManager *auth.TokenManager) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID, err := authenticatedHTTPUserID(tokenManager, r)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, biz.MaxAvatarSize+1024*1024)
	if err := r.ParseMultipartForm(biz.MaxAvatarSize + 1024*1024); err != nil {
		writeJSONError(w, kerrors.BadRequest("INVALID_AVATAR", "avatar upload must be a multipart file smaller than 5 MB"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, kerrors.BadRequest("INVALID_AVATAR", "missing avatar file"))
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, biz.MaxAvatarSize+1))
	if err != nil {
		writeJSONError(w, kerrors.BadRequest("INVALID_AVATAR", "could not read avatar file"))
		return
	}
	mimeType := http.DetectContentType(content)
	token, err := s.uc.UploadAvatar(r.Context(), userID, header.Filename, mimeType, content)
	if err != nil {
		writeJSONError(w, uploadAvatarError(err))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"provider":  token.Provider,
		"objectKey": token.ObjectKey,
		"publicUrl": token.PublicURL,
	})
}

// UploadKnowledgeHTTP stores a private knowledge file and returns only the
// opaque object reference that the BFF will pass to the AI service.
func (s *FileService) UploadKnowledgeHTTP(w http.ResponseWriter, r *http.Request, tokenManager *auth.TokenManager) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID, err := authenticatedHTTPUserID(tokenManager, r)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, biz.MaxKnowledgeFileSize+1024*1024)
	if err := r.ParseMultipartForm(biz.MaxKnowledgeFileSize + 1024*1024); err != nil {
		writeJSONError(w, kerrors.BadRequest("INVALID_KNOWLEDGE_FILE", biz.ErrInvalidKnowledgeFile.Error()))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, kerrors.BadRequest("INVALID_KNOWLEDGE_FILE", "missing knowledge file"))
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, biz.MaxKnowledgeFileSize+1))
	if err != nil {
		writeJSONError(w, kerrors.BadRequest("INVALID_KNOWLEDGE_FILE", "could not read knowledge file"))
		return
	}
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(content)
	}
	objectReference, err := s.uc.UploadKnowledge(r.Context(), userID, header.Filename, mimeType, content)
	if err != nil {
		writeJSONError(w, uploadKnowledgeError(err))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"objectReference": objectReference,
		"source":          header.Filename,
	})
}

func authenticatedHTTPUserID(tokenManager *auth.TokenManager, r *http.Request) (int64, error) {
	if tokenManager == nil {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "login required")
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "missing access token")
	}
	claims, err := tokenManager.Parse(parts[1])
	if err != nil || claims.UserID <= 0 {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "invalid access token")
	}
	return claims.UserID, nil
}

func uploadAvatarError(err error) error {
	switch err {
	case biz.ErrInvalidAvatar:
		return kerrors.BadRequest("INVALID_AVATAR", err.Error())
	case biz.ErrFileStorageMissing:
		return kerrors.InternalServer("FILE_STORAGE_UNAVAILABLE", "avatar storage is not configured")
	default:
		return err
	}
}

func uploadKnowledgeError(err error) error {
	switch err {
	case biz.ErrInvalidKnowledgeFile:
		return kerrors.BadRequest("INVALID_KNOWLEDGE_FILE", err.Error())
	case biz.ErrFileStorageMissing:
		return kerrors.InternalServer("FILE_STORAGE_UNAVAILABLE", "knowledge storage is not configured")
	default:
		return err
	}
}

func listKnowledgeError(err error) error {
	if err == biz.ErrFileStorageMissing {
		return kerrors.InternalServer("FILE_STORAGE_UNAVAILABLE", "knowledge storage is not configured")
	}
	return kerrors.InternalServer("FILE_STORAGE_ERROR", "could not list knowledge files")
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
