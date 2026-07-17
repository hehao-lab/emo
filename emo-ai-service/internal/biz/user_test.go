package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/conf"

	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	usernameUser *User
	phoneUser    *User
	emailUser    *User
	created      *User
	updatedHash  string
}

func (m *mockUserRepo) Create(ctx context.Context, u *User) (*User, error) {
	m.created = u
	return &User{
		ID:           1,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Phone:        u.Phone,
		Email:        u.Email,
		Roles:        u.Roles,
		Status:       u.Status,
	}, nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id int64) (*User, error) {
	return nil, nil
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*User, error) {
	return m.usernameUser, nil
}

func (m *mockUserRepo) FindByPhone(ctx context.Context, phone string) (*User, error) {
	return m.phoneUser, nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*User, error) {
	return m.emailUser, nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	m.updatedHash = passwordHash
	return nil
}

type mockVerificationCodeRepo struct {
	codes   map[string]string
	deleted string
}

func (m *mockVerificationCodeRepo) Save(ctx context.Context, scene, target, code string, ttl time.Duration) error {
	if m.codes == nil {
		m.codes = map[string]string{}
	}
	m.codes[verificationCodeTestKey(scene, target)] = code
	return nil
}

func (m *mockVerificationCodeRepo) Get(ctx context.Context, scene, target string) (string, error) {
	return m.codes[verificationCodeTestKey(scene, target)], nil
}

func (m *mockVerificationCodeRepo) Delete(ctx context.Context, scene, target string) error {
	m.deleted = verificationCodeTestKey(scene, target)
	delete(m.codes, m.deleted)
	return nil
}

func TestUserUsecase_Register(t *testing.T) {
	ctx := context.Background()
	codes := &mockVerificationCodeRepo{
		codes: map[string]string{
			verificationCodeTestKey(VerificationSceneRegisterEmail, "test@example.com"): "123456",
		},
	}
	repo := &mockUserRepo{}
	uc := &UserUsecase{repo: repo, verificationCodes: codes}

	got, err := uc.Register(ctx, " test ", "password123", " 13800138000 ", " Test@Example.COM ", " 123456 ")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got != 1 {
		t.Fatalf("Register() got = %v, want 1", got)
	}
	if repo.created == nil {
		t.Fatal("Register() did not create user")
	}
	if repo.created.Username != "test" {
		t.Fatalf("created username = %q, want test", repo.created.Username)
	}
	if repo.created.Phone != "13800138000" {
		t.Fatalf("created phone = %q, want 13800138000", repo.created.Phone)
	}
	if repo.created.Email != "test@example.com" {
		t.Fatalf("created email = %q, want test@example.com", repo.created.Email)
	}
	if repo.created.PasswordHash == "" || repo.created.PasswordHash == "password123" {
		t.Fatal("created password hash was not generated")
	}
	if codes.deleted != verificationCodeTestKey(VerificationSceneRegisterEmail, "test@example.com") {
		t.Fatalf("deleted code key = %q, want register email key", codes.deleted)
	}
}

func TestUserUsecase_RegisterCodeMismatch(t *testing.T) {
	ctx := context.Background()
	uc := &UserUsecase{
		repo: &mockUserRepo{},
		verificationCodes: &mockVerificationCodeRepo{
			codes: map[string]string{
				verificationCodeTestKey(VerificationSceneRegisterEmail, "test@example.com"): "123456",
			},
		},
	}

	_, err := uc.Register(ctx, "test", "password123", "13800138000", "test@example.com", "654321")
	if !errors.Is(err, ErrCodeMismatch) {
		t.Fatalf("Register() error = %v, want ErrCodeMismatch", err)
	}
}

func TestUserUsecase_RegisterCodeExpired(t *testing.T) {
	ctx := context.Background()
	uc := &UserUsecase{
		repo:              &mockUserRepo{},
		verificationCodes: &mockVerificationCodeRepo{codes: map[string]string{}},
	}

	_, err := uc.Register(ctx, "test", "password123", "13800138000", "test@example.com", "123456")
	if !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("Register() error = %v, want ErrCodeExpired", err)
	}
}

func TestUserUsecase_LoginAcceptsUsernameAccount(t *testing.T) {
	ctx := context.Background()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}
	repo := &mockUserRepo{
		usernameUser: &User{
			ID:           7,
			Username:     "testuser",
			PasswordHash: string(passwordHash),
			Phone:        "13800138000",
			Email:        "test@example.com",
			Roles:        []string{"user"},
		},
	}
	uc := &UserUsecase{
		repo:        repo,
		tokenManger: auth.NewTokenManager(&conf.Auth{}),
	}

	result, err := uc.Login(ctx, " testuser ", "123456", LoginMeta{})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.UserID != 7 {
		t.Fatalf("Login() userID = %d, want 7", result.UserID)
	}
}

func TestUserUsecase_LoginAcceptsLegacyPlainPasswordAndUpgradesHash(t *testing.T) {
	ctx := context.Background()
	repo := &mockUserRepo{
		phoneUser: &User{
			ID:           8,
			Username:     "legacy",
			PasswordHash: "123456",
			Phone:        "13800138001",
			Email:        "legacy@example.com",
			Roles:        []string{"user"},
		},
	}
	uc := &UserUsecase{
		repo:        repo,
		tokenManger: auth.NewTokenManager(&conf.Auth{}),
	}

	result, err := uc.Login(ctx, "13800138001", "123456", LoginMeta{})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.UserID != 8 {
		t.Fatalf("Login() userID = %d, want 8", result.UserID)
	}
	if repo.updatedHash == "" || repo.updatedHash == "123456" {
		t.Fatalf("updated password hash = %q, want a bcrypt hash", repo.updatedHash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.updatedHash), []byte("123456")); err != nil {
		t.Fatalf("updated password hash does not verify: %v", err)
	}
}

func verificationCodeTestKey(scene, target string) string {
	return scene + ":" + target
}
