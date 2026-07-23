package service

import (
	"context"
	"testing"

	v1 "emo-ai-service/api/admin/v1"
	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type adminRepoStub struct {
	biz.AdminRepo
	users []*biz.AdminUser
}

func (s *adminRepoStub) ListUsers(context.Context, biz.AdminUserListOption) ([]*biz.AdminUser, int64, error) {
	return s.users, int64(len(s.users)), nil
}

func TestAdminServiceRejectsNonAdmin(t *testing.T) {
	svc := NewAdminService(biz.NewAdminUsecase(&adminRepoStub{}))
	ctx := auth.WithUserID(context.Background(), 7)
	ctx = auth.WithRoles(ctx, []string{"user"})

	_, err := svc.ListUsers(ctx, &v1.ListUsersRequest{})
	if got := kerrors.Code(err); got != 403 {
		t.Fatalf("ListUsers() code = %d, want 403", got)
	}
}

func TestAdminServiceAllowsAdminAndReturnsGlobalUsers(t *testing.T) {
	repo := &adminRepoStub{users: []*biz.AdminUser{
		{ID: 1, Username: "first"},
		{ID: 2, Username: "second"},
	}}
	svc := NewAdminService(biz.NewAdminUsecase(repo))
	ctx := auth.WithUserID(context.Background(), 1)
	ctx = auth.WithRoles(ctx, []string{"admin"})

	resp, err := svc.ListUsers(ctx, &v1.ListUsersRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if resp.GetTotal() != 2 || len(resp.GetUsers()) != 2 {
		t.Fatalf("ListUsers() = total %d, users %d; want 2 global users", resp.GetTotal(), len(resp.GetUsers()))
	}
}
