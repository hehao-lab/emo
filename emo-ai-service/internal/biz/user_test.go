package biz

import (
	"context"
	"testing"
)

type mockUserRepo struct{}

func (m *mockUserRepo) Create(ctx context.Context, u *User) (*User, error) {
	return &User{
		ID:       1,
		Username: u.Username,
		Password: u.Password,
		Phone:    u.Phone,
	}, nil
}
func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*User, error) {
	return nil, nil // 模拟不存在用户（注册场景）
}

func TestUserUsecase_Register(t *testing.T) {
	type fields struct {
		repo UserRepo
	}
	type args struct {
		ctx      context.Context
		username string
		password string
		phone    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				repo: &mockUserRepo{},
			},
			args: args{
				ctx:      context.Background(),
				username: "test",
				password: "<PASSWORD>",
				phone:    "1234567890",
			},
			want:    1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &UserUsecase{
				repo: tt.fields.repo,
			}
			got, err := uc.Register(tt.args.ctx, tt.args.username, tt.args.password, tt.args.phone)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Register() got = %v, want %v", got, tt.want)
			}
		})
	}
}
