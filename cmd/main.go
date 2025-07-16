package main

import (
	"context"

	"github.com/rryowa/medods_dvortsov/internal/api"
	"github.com/rryowa/medods_dvortsov/internal/controller"
	"github.com/rryowa/medods_dvortsov/internal/service"
	"github.com/rryowa/medods_dvortsov/internal/storage/memory"
	"github.com/rryowa/medods_dvortsov/internal/util"
)

func main() {
	ctx := context.Background()

	logger := util.NewZapLogger()

	tokenConfig := util.NewTokenConfig()
	tokenService := service.NewTokenService(tokenConfig)

	sessionRepository := memory.NewSessionRepository(logger)
	apiKeyRepository := memory.NewAPIKeyRepository()
	authService := service.NewAuthService(*tokenService, sessionRepository, logger)

	controller := controller.NewController(logger, authService, tokenService)
	apiServer := api.NewAPI(controller, logger, util.NewServerConfig(), apiKeyRepository)

	apiServer.Run(ctx)
}
