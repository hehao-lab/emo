package service

import (
	"context"
	"errors"

	v1 "emo-ai-service/api/profile/v1"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type ProfileService struct {
	uc *biz.ProfileUsecase
}

func NewProfileService(uc *biz.ProfileUsecase) *ProfileService {
	return &ProfileService{uc: uc}
}

var _ v1.ProfileServiceHTTPServer = (*ProfileService)(nil)

// GetMe 实现“我的页面”用户信息接口：根据 JWT 中的用户 ID 查询账号和个人资料。
func (s *ProfileService) GetMe(ctx context.Context, req *v1.GetMeRequest) (*v1.UserProfile, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.uc.GetMe(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toProfileDTO(profile), nil
}

// UpdateProfile 实现资料修改接口：更新昵称、邮箱、生日、简介、地区、行业等个人资料字段。
func (s *ProfileService) UpdateProfile(ctx context.Context, req *v1.UpdateProfileRequest) (*v1.UserProfile, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.uc.UpdateProfile(ctx, &biz.UserProfile{
		UserID:     userID,
		Nickname:   req.GetNickname(),
		Email:      req.GetEmail(),
		Gender:     req.GetGender(),
		Birthday:   req.GetBirthday(),
		Bio:        req.GetBio(),
		Location:   req.GetLocation(),
		Occupation: req.GetOccupation(),
		Industry:   req.GetIndustry(),
		Language:   req.GetLanguage(),
		Timezone:   req.GetTimezone(),
	})
	if err != nil {
		return nil, err
	}
	return toProfileDTO(profile), nil
}

// UpdateAvatar 实现头像修改接口：保存当前用户的新头像地址并返回最新个人资料。
func (s *ProfileService) UpdateAvatar(ctx context.Context, req *v1.UpdateAvatarRequest) (*v1.UserProfile, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.uc.UpdateAvatar(ctx, userID, req.GetAvatarUrl())
	if err != nil {
		return nil, err
	}
	return toProfileDTO(profile), nil
}

func (s *ProfileService) GetPersonalProfile(ctx context.Context, req *v1.GetPersonalProfileRequest) (*v1.PersonalProfile, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.uc.GetPersonalProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toPersonalProfileDTO(profile), nil
}

func (s *ProfileService) SavePersonalProfile(ctx context.Context, req *v1.SavePersonalProfileRequest) (*v1.PersonalProfile, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.uc.SavePersonalProfile(ctx, userID, &biz.PersonalProfile{
		Age:                req.GetAge(),
		Gender:             req.GetGender(),
		MBTI:               req.GetMbti(),
		RelationshipStatus: req.GetRelationshipStatus(),
		PersonalitySummary: req.GetPersonalitySummary(),
	})
	if err != nil {
		return nil, err
	}
	return toPersonalProfileDTO(profile), nil
}

func (s *ProfileService) ListTargetProfiles(ctx context.Context, req *v1.ListTargetProfilesRequest) (*v1.ListTargetProfilesResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	targets, err := s.uc.ListTargetProfiles(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.TargetProfile, 0, len(targets))
	for _, target := range targets {
		out = append(out, toTargetProfileDTO(target))
	}
	return &v1.ListTargetProfilesResponse{Targets: out}, nil
}

func (s *ProfileService) SaveTargetProfile(ctx context.Context, req *v1.SaveTargetProfileRequest) (*v1.TargetProfile, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	target, err := s.uc.SaveTargetProfile(ctx, userID, &biz.TargetProfile{
		ID:                   req.GetId(),
		Name:                 req.GetName(),
		Age:                  req.GetAge(),
		Gender:               req.GetGender(),
		MBTI:                 req.GetMbti(),
		CurrentRelationship:  req.GetCurrentRelationship(),
		InteractionFrequency: req.GetInteractionFrequency(),
		RelationshipGoal:     req.GetRelationshipGoal(),
		PersonalityTraits:    req.GetPersonalityTraits(),
		RecentInteraction:    req.GetRecentInteraction(),
	})
	if err != nil {
		return nil, profileError(err)
	}
	return toTargetProfileDTO(target), nil
}

func (s *ProfileService) ListImportantRecords(ctx context.Context, req *v1.ListImportantRecordsRequest) (*v1.ListImportantRecordsResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	records, err := s.uc.ListImportantRecords(ctx, userID, req.GetTargetId())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.ImportantRecord, 0, len(records))
	for _, record := range records {
		out = append(out, toImportantRecordDTO(record))
	}
	return &v1.ListImportantRecordsResponse{Records: out}, nil
}

func (s *ProfileService) SaveImportantRecord(ctx context.Context, req *v1.SaveImportantRecordRequest) (*v1.ImportantRecord, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	record, err := s.uc.SaveImportantRecord(ctx, userID, &biz.ImportantRecord{
		ID:               req.GetId(),
		TargetProfileID:  req.GetTargetProfileId(),
		Title:            req.GetTitle(),
		RecordTime:       req.GetRecordTime(),
		EventDescription: req.GetEventDescription(),
		Resolution:       req.GetResolution(),
		ConcernPoint:     req.GetConcernPoint(),
		Satisfaction:     req.GetSatisfaction(),
	})
	if err != nil {
		return nil, profileError(err)
	}
	return toImportantRecordDTO(record), nil
}

func (s *ProfileService) DeleteImportantRecord(ctx context.Context, req *v1.DeleteImportantRecordRequest) (*v1.DeleteImportantRecordResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.DeleteImportantRecord(ctx, userID, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteImportantRecordResponse{Ok: true}, nil
}

func toProfileDTO(profile *biz.UserProfile) *v1.UserProfile {
	if profile == nil {
		return &v1.UserProfile{}
	}
	return &v1.UserProfile{
		UserId:     profile.UserID,
		Username:   profile.Username,
		Phone:      profile.Phone,
		Email:      profile.Email,
		Nickname:   profile.Nickname,
		AvatarUrl:  profile.AvatarURL,
		Roles:      profile.Roles,
		Gender:     profile.Gender,
		Birthday:   profile.Birthday,
		Bio:        profile.Bio,
		Location:   profile.Location,
		Occupation: profile.Occupation,
		Industry:   profile.Industry,
		Language:   profile.Language,
		Timezone:   profile.Timezone,
		CreatedAt:  profile.CreatedAt.Unix(),
		UpdatedAt:  profile.UpdatedAt.Unix(),
	}
}

func toPersonalProfileDTO(profile *biz.PersonalProfile) *v1.PersonalProfile {
	if profile == nil {
		return &v1.PersonalProfile{}
	}
	return &v1.PersonalProfile{
		Id:                 profile.ID,
		UserId:             profile.UserID,
		Age:                profile.Age,
		Gender:             profile.Gender,
		Mbti:               profile.MBTI,
		RelationshipStatus: profile.RelationshipStatus,
		PersonalitySummary: profile.PersonalitySummary,
		CreatedAt:          unixOrZero(profile.CreatedAt),
		UpdatedAt:          unixOrZero(profile.UpdatedAt),
	}
}

func toTargetProfileDTO(target *biz.TargetProfile) *v1.TargetProfile {
	if target == nil {
		return &v1.TargetProfile{}
	}
	return &v1.TargetProfile{
		Id:                   target.ID,
		UserId:               target.UserID,
		PersonalProfileId:    target.PersonalProfileID,
		Name:                 target.Name,
		Age:                  target.Age,
		Gender:               target.Gender,
		Mbti:                 target.MBTI,
		CurrentRelationship:  target.CurrentRelationship,
		InteractionFrequency: target.InteractionFrequency,
		RelationshipGoal:     target.RelationshipGoal,
		PersonalityTraits:    target.PersonalityTraits,
		RecentInteraction:    target.RecentInteraction,
		CreatedAt:            unixOrZero(target.CreatedAt),
		UpdatedAt:            unixOrZero(target.UpdatedAt),
	}
}

func toImportantRecordDTO(record *biz.ImportantRecord) *v1.ImportantRecord {
	if record == nil {
		return &v1.ImportantRecord{}
	}
	return &v1.ImportantRecord{
		Id:                record.ID,
		UserId:            record.UserID,
		PersonalProfileId: record.PersonalProfileID,
		TargetProfileId:   record.TargetProfileID,
		Title:             record.Title,
		RecordTime:        record.RecordTime,
		EventDescription:  record.EventDescription,
		Resolution:        record.Resolution,
		ConcernPoint:      record.ConcernPoint,
		Satisfaction:      record.Satisfaction,
		CreatedAt:         unixOrZero(record.CreatedAt),
		UpdatedAt:         unixOrZero(record.UpdatedAt),
	}
}

func unixOrZero(value interface{ Unix() int64 }) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}

func profileError(err error) error {
	switch {
	case errors.Is(err, biz.ErrTargetProfileRequired):
		return kerrors.BadRequest("TARGET_PROFILE_REQUIRED", "target profile required")
	case errors.Is(err, biz.ErrTargetProfileNotFound):
		return kerrors.NotFound("TARGET_PROFILE_NOT_FOUND", "target profile not found")
	default:
		return err
	}
}
