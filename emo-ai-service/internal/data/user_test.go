package data

import (
	"encoding/json"
	"testing"
)

func TestNewUserProfileModelInitializesValidJSONExtra(t *testing.T) {
	profile := newUserProfileModel(2)

	if profile.UserID != 2 {
		t.Fatalf("UserID = %d, want 2", profile.UserID)
	}
	if profile.Language != "zh-CN" {
		t.Fatalf("Language = %q, want zh-CN", profile.Language)
	}
	if !json.Valid([]byte(profile.Extra)) {
		t.Fatalf("Extra = %q, want valid JSON", profile.Extra)
	}
	if profile.Extra == "" {
		t.Fatal("Extra must not be empty because MySQL JSON columns reject empty documents")
	}
}

func TestNormalizeUserProfileModelRepairsEmptyJSONExtra(t *testing.T) {
	profile := UserProfileModel{}

	normalizeUserProfileModel(&profile)

	if profile.Language != "zh-CN" {
		t.Fatalf("Language = %q, want zh-CN", profile.Language)
	}
	if profile.Extra != "{}" {
		t.Fatalf("Extra = %q, want {}", profile.Extra)
	}
	if !json.Valid([]byte(profile.Extra)) {
		t.Fatalf("Extra = %q, want valid JSON", profile.Extra)
	}
}
