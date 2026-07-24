package service

import (
	"context"
	"testing"

	diaryv1 "emo-ai-service/api/diary/v1"
	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/biz"
)

type diaryScopeRepoStub struct {
	biz.DiaryRepo
	listedUserID int64
}

func (s *diaryScopeRepoStub) ListDiaries(_ context.Context, userID int64, _ biz.DiaryListOption) ([]*biz.MoodDiary, int64, error) {
	s.listedUserID = userID
	return []*biz.MoodDiary{{ID: 1, UserID: userID}}, 1, nil
}

func TestAppDiaryListRemainsScopedToJWTUser(t *testing.T) {
	repo := &diaryScopeRepoStub{}
	svc := NewDiaryService(biz.NewDiaryUsecase(repo))
	ctx := auth.WithUserID(context.Background(), 42)

	resp, err := svc.ListDiaries(ctx, &diaryv1.ListDiariesRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListDiaries() error = %v", err)
	}
	if repo.listedUserID != 42 {
		t.Fatalf("repository userID = %d, want JWT user 42", repo.listedUserID)
	}
	if len(resp.GetDiaries()) != 1 || resp.GetDiaries()[0].GetUserId() != 42 {
		t.Fatalf("ListDiaries() returned data outside JWT user scope")
	}
}
