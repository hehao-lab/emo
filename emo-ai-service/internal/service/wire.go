package service

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewUserService,
	NewProfileService,
	NewSecurityService,
	NewDiaryService,
	NewChatService,
	NewEmotionService,
	NewSystemService,
	NewFileService,
)
