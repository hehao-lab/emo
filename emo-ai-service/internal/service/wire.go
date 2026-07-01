package service

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewUserService,
	NewProfileService,
	NewSecurityService,
	NewDiaryService,
	NewChatService,
	NewAIChatService,
	NewEmotionService,
	NewSystemService,
	NewFileService,
)
