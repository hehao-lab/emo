package biz

import (
	"context"
	"testing"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type adminUsecaseRepoStub struct {
	AdminRepo
	createdTag *MoodTag
}

func (s *adminUsecaseRepoStub) CreateMoodTag(_ context.Context, tag *MoodTag) (*MoodTag, error) {
	copy := *tag
	s.createdTag = &copy
	return &copy, nil
}

func TestAdminMoodTagIsAlwaysSystemOwned(t *testing.T) {
	repo := &adminUsecaseRepoStub{}
	uc := NewAdminUsecase(repo)

	created, err := uc.CreateMoodTag(context.Background(), &MoodTag{UserID: 99, Name: " Calm "})
	if err != nil {
		t.Fatalf("CreateMoodTag() error = %v", err)
	}
	if repo.createdTag.UserID != 0 || !created.System {
		t.Fatalf("CreateMoodTag() owner = %d, system = %v; want system owner 0", repo.createdTag.UserID, created.System)
	}
	if created.Name != "Calm" {
		t.Fatalf("CreateMoodTag() name = %q, want trimmed name", created.Name)
	}
}

func TestAdminCannotFreezeOrDemoteSelf(t *testing.T) {
	uc := NewAdminUsecase(&adminUsecaseRepoStub{})

	if _, err := uc.UpdateUserStatus(context.Background(), 7, 7, 2, ""); kerrors.Code(err) != 403 {
		t.Fatalf("UpdateUserStatus(self) code = %d, want 403", kerrors.Code(err))
	}
	if _, err := uc.UpdateUserRoles(context.Background(), 7, 7, []string{"user"}); kerrors.Code(err) != 403 {
		t.Fatalf("UpdateUserRoles(self) code = %d, want 403", kerrors.Code(err))
	}
}
