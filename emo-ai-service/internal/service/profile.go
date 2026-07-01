package service

import (
	"context"

	v1 "emo-ai-service/api/profile/v1"
	"emo-ai-service/internal/biz"
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
