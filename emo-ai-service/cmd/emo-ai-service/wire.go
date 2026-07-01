//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"emo-ai-service/internal/auth"
	"emo-ai-service/internal/biz"
	"emo-ai-service/internal/conf"
	"emo-ai-service/internal/data"
	"emo-ai-service/internal/server"
	"emo-ai-service/internal/service"
	"log/slog"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.Auth, *conf.AIService, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		data.ProviderSet,
		auth.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		server.ProviderSet,
		newApp,
	))
}
