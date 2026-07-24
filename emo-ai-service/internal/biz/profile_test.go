package biz

import (
	"context"
	"testing"
	"time"
)

type mockProfileRepo struct {
	nextPersonalID int64
	nextTargetID   int64
	nextRecordID   int64
	personal       map[int64]*PersonalProfile
	targets        map[int64]*TargetProfile
	records        map[int64]*ImportantRecord
}

func newMockProfileRepo() *mockProfileRepo {
	return &mockProfileRepo{
		nextPersonalID: 11,
		nextTargetID:   21,
		nextRecordID:   31,
		personal:       map[int64]*PersonalProfile{},
		targets:        map[int64]*TargetProfile{},
		records:        map[int64]*ImportantRecord{},
	}
}

func (m *mockProfileRepo) FindByID(ctx context.Context, id int64) (*User, error) {
	return &User{ID: id, Username: "user"}, nil
}

func (m *mockProfileRepo) FindProfile(ctx context.Context, userID int64) (*UserProfile, error) {
	return &UserProfile{UserID: userID, Username: "user"}, nil
}

func (m *mockProfileRepo) UpsertProfile(ctx context.Context, profile *UserProfile) (*UserProfile, error) {
	return profile, nil
}

func (m *mockProfileRepo) UpdateAvatar(ctx context.Context, userID int64, avatarURL string) (*UserProfile, error) {
	return &UserProfile{UserID: userID, AvatarURL: avatarURL}, nil
}

func (m *mockProfileRepo) FindPersonalProfile(ctx context.Context, userID int64) (*PersonalProfile, error) {
	if profile := m.personal[userID]; profile != nil {
		out := *profile
		return &out, nil
	}
	return nil, nil
}

func (m *mockProfileRepo) UpsertPersonalProfile(ctx context.Context, profile *PersonalProfile) (*PersonalProfile, error) {
	out := *profile
	if existing := m.personal[profile.UserID]; existing != nil {
		out.ID = existing.ID
		out.CreatedAt = existing.CreatedAt
	} else {
		out.ID = m.nextPersonalID
		m.nextPersonalID++
		out.CreatedAt = time.Now()
	}
	out.UpdatedAt = time.Now()
	m.personal[out.UserID] = &out
	return &out, nil
}

func (m *mockProfileRepo) ListTargetProfiles(ctx context.Context, userID int64) ([]*TargetProfile, error) {
	var out []*TargetProfile
	for _, target := range m.targets {
		if target.UserID == userID {
			next := *target
			out = append(out, &next)
		}
	}
	return out, nil
}

func (m *mockProfileRepo) GetTargetProfile(ctx context.Context, userID, targetID int64) (*TargetProfile, error) {
	target := m.targets[targetID]
	if target == nil || target.UserID != userID {
		return nil, nil
	}
	out := *target
	return &out, nil
}

func (m *mockProfileRepo) UpsertTargetProfile(ctx context.Context, target *TargetProfile) (*TargetProfile, error) {
	out := *target
	if out.ID == 0 {
		out.ID = m.nextTargetID
		m.nextTargetID++
		out.CreatedAt = time.Now()
	} else if existing := m.targets[out.ID]; existing != nil {
		out.CreatedAt = existing.CreatedAt
	}
	out.UpdatedAt = time.Now()
	m.targets[out.ID] = &out
	return &out, nil
}

func (m *mockProfileRepo) ListImportantRecords(ctx context.Context, userID, targetID int64) ([]*ImportantRecord, error) {
	var out []*ImportantRecord
	for _, record := range m.records {
		if record.UserID == userID && (targetID == 0 || record.TargetProfileID == targetID) {
			next := *record
			out = append(out, &next)
		}
	}
	return out, nil
}

func (m *mockProfileRepo) UpsertImportantRecord(ctx context.Context, record *ImportantRecord) (*ImportantRecord, error) {
	out := *record
	if out.ID == 0 {
		out.ID = m.nextRecordID
		m.nextRecordID++
		out.CreatedAt = time.Now()
	} else if existing := m.records[out.ID]; existing != nil {
		out.CreatedAt = existing.CreatedAt
	}
	out.UpdatedAt = time.Now()
	m.records[out.ID] = &out
	return &out, nil
}

func (m *mockProfileRepo) DeleteImportantRecord(ctx context.Context, userID, recordID int64) error {
	record := m.records[recordID]
	if record != nil && record.UserID == userID {
		delete(m.records, recordID)
	}
	return nil
}

func TestProfileUsecaseBindsPersonalTargetAndImportantRecords(t *testing.T) {
	ctx := context.Background()
	repo := newMockProfileRepo()
	uc := NewProfileUsecase(repo)

	personal, err := uc.SavePersonalProfile(ctx, 7, &PersonalProfile{
		Age:                26,
		Gender:             "female",
		MBTI:               "INFP",
		RelationshipStatus: "dating",
		PersonalitySummary: "sensitive but direct",
	})
	if err != nil {
		t.Fatalf("SavePersonalProfile() error = %v", err)
	}

	target, err := uc.SaveTargetProfile(ctx, 7, &TargetProfile{
		Name:                 "Alex",
		Age:                  28,
		Gender:               "male",
		MBTI:                 " INFJ ",
		CurrentRelationship:  "dating",
		InteractionFrequency: "daily",
		RelationshipGoal:     "repair conflict",
		PersonalityTraits:    "reserved",
		RecentInteraction:    "argument last night",
	})
	if err != nil {
		t.Fatalf("SaveTargetProfile() error = %v", err)
	}

	record, err := uc.SaveImportantRecord(ctx, 7, &ImportantRecord{
		TargetProfileID:  target.ID,
		Title:            "late-night conflict",
		RecordTime:       "2026-07-03",
		EventDescription: "we argued about response time",
		Resolution:       "paused and talked later",
		ConcernPoint:     "being ignored",
		Satisfaction:     "normal",
	})
	if err != nil {
		t.Fatalf("SaveImportantRecord() error = %v", err)
	}

	if personal.UserID != 7 {
		t.Fatalf("personal user id = %d, want 7", personal.UserID)
	}
	if target.UserID != 7 || target.PersonalProfileID != personal.ID {
		t.Fatalf("target binding = (user %d, personal %d), want (7, %d)", target.UserID, target.PersonalProfileID, personal.ID)
	}
	if target.MBTI != "INFJ" {
		t.Fatalf("target mbti = %q, want INFJ", target.MBTI)
	}
	if record.UserID != 7 || record.PersonalProfileID != personal.ID || record.TargetProfileID != target.ID {
		t.Fatalf("record binding = (user %d, personal %d, target %d), want (7, %d, %d)", record.UserID, record.PersonalProfileID, record.TargetProfileID, personal.ID, target.ID)
	}
}

func TestProfileUsecaseRequiresTargetForImportantRecord(t *testing.T) {
	uc := NewProfileUsecase(newMockProfileRepo())

	_, err := uc.SaveImportantRecord(context.Background(), 7, &ImportantRecord{
		Title:            "missing target",
		EventDescription: "no target selected",
	})
	if err != ErrTargetProfileRequired {
		t.Fatalf("SaveImportantRecord() error = %v, want ErrTargetProfileRequired", err)
	}
}
