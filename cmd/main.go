package main

import (
	"context"

	"github.com/rryowa/medods_dvortsov/internal/api"
	"github.com/rryowa/medods_dvortsov/internal/controller"
	"github.com/rryowa/medods_dvortsov/internal/service"
	"github.com/rryowa/medods_dvortsov/internal/util"
)

func main() {
	ctx := context.Background()

	logger := util.NewZapLogger()

	manager := service.NewInMemoryRefreshTokenManager()
	authService := service.NewAuthService(manager, util.JWTSecret())

	ctrl := controller.NewController(logger, authService)
	apiServer := api.NewAPI(ctrl, logger, util.NewServerConfig())

	apiServer.Run(ctx)
}
