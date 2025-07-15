package main

import (
	"context"
)

func main() {
	ctx := context.Background()
	dbConfig := util.NewDBConfig()
	zapLogger := util.NewZapLogger()
	storage := postgres.NewPostgresRepository(ctx, dbConfig, zapLogger)
	authService := service.NewAuthService(storage)
	controller := controller.NewController(zapLogger, authService)

	app := api.NewAPI(controller, zapLogger, util.NewServerConfig())

	app.Run(ctx)
}