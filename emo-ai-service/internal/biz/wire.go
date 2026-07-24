package biz

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewUserUsecase,
	NewProfileUsecase,
	NewSecurityUsecase,
	NewDiaryUsecase,
	NewChatUsecase,
	NewAIChatUsecase,
	NewEmotionUsecase,
	NewSystemUsecase,
	NewFileUsecase,
	NewAdminUsecase,
)
