package service

import (
	"context"

	v1 "emo-ai-service/api/diary/v1"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DiaryService struct {
	uc *biz.DiaryUsecase
}

func NewDiaryService(uc *biz.DiaryUsecase) *DiaryService {
	return &DiaryService{uc: uc}
}

var _ v1.DiaryServiceHTTPServer = (*DiaryService)(nil)

// CreateDiary 实现心情日记新增接口：校验内容，保存日记正文、心情分数、标签和附件。
func (s *DiaryService) CreateDiary(ctx context.Context, req *v1.CreateDiaryRequest) (*v1.MoodDiary, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetContent() == "" {
		return nil, kerrors.BadRequest("INVALID_ARGUMENT", "content is required")
	}
	diary, err := s.uc.CreateDiary(ctx, &biz.MoodDiary{
		UserID:         userID,
		Title:          req.GetTitle(),
		Content:        req.GetContent(),
		Mood:           req.GetMood(),
		MoodScore:      req.GetMoodScore(),
		Weather:        req.GetWeather(),
		Location:       req.GetLocation(),
		OccurredOn:     req.GetOccurredOn(),
		Visibility:     req.GetVisibility(),
		AttachmentURLs: req.GetAttachmentUrls(),
	}, req.GetTagIds())
	if err != nil {
		return nil, err
	}
	return toDiaryDTO(diary), nil
}

// ListDiaries 实现心情日记列表接口：按用户、标签、心情、日期范围分页查询日记。
func (s *DiaryService) ListDiaries(ctx context.Context, req *v1.ListDiariesRequest) (*v1.ListDiariesResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.uc.ListDiaries(ctx, userID, biz.DiaryListOption{Page: req.GetPage(), PageSize: req.GetPageSize(), TagID: req.GetTagId(), Mood: req.GetMood(), StartDate: req.GetStartDate(), EndDate: req.GetEndDate()})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.MoodDiary, 0, len(items))
	for _, item := range items {
		out = append(out, toDiaryDTO(item))
	}
	return &v1.ListDiariesResponse{Diaries: out, Total: total}, nil
}

// GetDiary 实现心情日记详情接口：只允许当前登录用户读取自己的日记详情。
func (s *DiaryService) GetDiary(ctx context.Context, req *v1.GetDiaryRequest) (*v1.MoodDiary, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	diary, err := s.uc.GetDiary(ctx, userID, req.GetId())
	if err != nil {
		return nil, err
	}
	if diary == nil {
		return nil, kerrors.NotFound("DIARY_NOT_FOUND", "diary not found")
	}
	return toDiaryDTO(diary), nil
}

// UpdateDiary 实现心情日记编辑接口：更新日记内容、情绪信息、标签关系和附件列表。
func (s *DiaryService) UpdateDiary(ctx context.Context, req *v1.UpdateDiaryRequest) (*v1.MoodDiary, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	diary, err := s.uc.UpdateDiary(ctx, &biz.MoodDiary{
		ID:             req.GetId(),
		UserID:         userID,
		Title:          req.GetTitle(),
		Content:        req.GetContent(),
		Mood:           req.GetMood(),
		MoodScore:      req.GetMoodScore(),
		Weather:        req.GetWeather(),
		Location:       req.GetLocation(),
		OccurredOn:     req.GetOccurredOn(),
		Visibility:     req.GetVisibility(),
		AttachmentURLs: req.GetAttachmentUrls(),
	}, req.GetTagIds())
	if err != nil {
		return nil, err
	}
	return toDiaryDTO(diary), nil
}

// DeleteDiary 实现心情日记删除接口：按当前用户和日记 ID 软删除日记。
func (s *DiaryService) DeleteDiary(ctx context.Context, req *v1.DeleteDiaryRequest) (*emptypb.Empty, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.DeleteDiary(ctx, userID, req.GetId())
}

// ListTags 实现心情标签列表接口：返回系统标签和当前用户自定义标签。
func (s *DiaryService) ListTags(ctx context.Context, req *v1.ListTagsRequest) (*v1.ListTagsResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	tags, err := s.uc.ListTags(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.MoodTag, 0, len(tags))
	for _, tag := range tags {
		out = append(out, toTagDTO(tag))
	}
	return &v1.ListTagsResponse{Tags: out}, nil
}

// CreateTag 实现心情标签新增接口：为当前用户创建自定义标签。
func (s *DiaryService) CreateTag(ctx context.Context, req *v1.CreateTagRequest) (*v1.MoodTag, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	tag, err := s.uc.CreateTag(ctx, &biz.MoodTag{UserID: userID, Name: req.GetName(), Color: req.GetColor(), Icon: req.GetIcon(), Sort: req.GetSort()})
	if err != nil {
		return nil, err
	}
	return toTagDTO(tag), nil
}

// UpdateTag 实现心情标签编辑接口：更新当前用户自定义标签的名称、颜色、图标和排序。
func (s *DiaryService) UpdateTag(ctx context.Context, req *v1.UpdateTagRequest) (*v1.MoodTag, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	tag, err := s.uc.UpdateTag(ctx, &biz.MoodTag{ID: req.GetId(), UserID: userID, Name: req.GetName(), Color: req.GetColor(), Icon: req.GetIcon(), Sort: req.GetSort()})
	if err != nil {
		return nil, err
	}
	return toTagDTO(tag), nil
}

// DeleteTag 实现心情标签删除接口：只删除当前用户自己的自定义标签。
func (s *DiaryService) DeleteTag(ctx context.Context, req *v1.DeleteTagRequest) (*emptypb.Empty, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.uc.DeleteTag(ctx, userID, req.GetId())
}

func toDiaryDTO(diary *biz.MoodDiary) *v1.MoodDiary {
	if diary == nil {
		return &v1.MoodDiary{}
	}
	tags := make([]*v1.MoodTag, 0, len(diary.Tags))
	for _, tag := range diary.Tags {
		tags = append(tags, toTagDTO(tag))
	}
	return &v1.MoodDiary{Id: diary.ID, UserId: diary.UserID, Title: diary.Title, Content: diary.Content, Mood: diary.Mood, MoodScore: diary.MoodScore, Weather: diary.Weather, Location: diary.Location, OccurredOn: diary.OccurredOn, Visibility: diary.Visibility, Tags: tags, AttachmentUrls: diary.AttachmentURLs, AnalysisId: diary.AnalysisID, CreatedAt: diary.CreatedAt.Unix(), UpdatedAt: diary.UpdatedAt.Unix()}
}

func toTagDTO(tag *biz.MoodTag) *v1.MoodTag {
	if tag == nil {
		return &v1.MoodTag{}
	}
	return &v1.MoodTag{Id: tag.ID, Name: tag.Name, Color: tag.Color, Icon: tag.Icon, Sort: tag.Sort, System: tag.System, CreatedAt: tag.CreatedAt.Unix(), UpdatedAt: tag.UpdatedAt.Unix()}
}
