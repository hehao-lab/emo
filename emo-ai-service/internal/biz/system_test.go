package biz

import (
	"context"
	"testing"
)

type systemRepoStub struct {
	about *AboutInfo
	err   error
}

func (r *systemRepoStub) GetAbout(context.Context) (*AboutInfo, error) {
	return r.about, r.err
}

func (r *systemRepoStub) ListPublicConfigs(context.Context) ([]*PublicConfig, error) {
	return nil, nil
}

func (r *systemRepoStub) GetLatestVersion(context.Context, string) (*AppVersion, error) {
	return nil, nil
}

func (r *systemRepoStub) ListAnnouncements(context.Context, string) ([]*Announcement, error) {
	return nil, nil
}

func TestSystemUsecaseGetAboutDoesNotInventData(t *testing.T) {
	uc := NewSystemUsecase(&systemRepoStub{})

	info, err := uc.GetAbout(context.Background())
	if err != nil {
		t.Fatalf("GetAbout returned error: %v", err)
	}
	if info == nil {
		t.Fatal("GetAbout returned nil info")
	}
	if info.AppName != "" || info.Company != "" || info.Website != "" || info.ContactEmail != "" {
		t.Fatalf("GetAbout returned invented data: %+v", info)
	}
}

func TestSystemUsecaseGetAboutReturnsRepositoryData(t *testing.T) {
	expected := &AboutInfo{AppName: "Stored app", Company: "Stored company"}
	uc := NewSystemUsecase(&systemRepoStub{about: expected})

	info, err := uc.GetAbout(context.Background())
	if err != nil {
		t.Fatalf("GetAbout returned error: %v", err)
	}
	if info != expected {
		t.Fatalf("GetAbout = %+v, want repository value %+v", info, expected)
	}
}
