package biz

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewUserUsecase,
	NewProfileUsecase,
	NewSecurityUsecase,
	NewDiaryUsecase,
	NewChatUsecase,
	NewEmotionUsecase,
	NewSystemUsecase,
	NewFileUsecase,
)
