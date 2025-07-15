package main

import (
	"context"
)

func main() {
	ctx := context.Background()
	dbConfig := util.NewDBConfig()
	serviceConfig := util.NewServiceConfig()
	zapLogger := util.NewZapLogger()
	storage := postgres.NewPostgresRepository(ctx, dbConfig, zapLogger)
	authService := service.NewAuthService(storage)
	controller := controller.NewController(zapLogger, authService)

	app := api.NewAPI(controller, zapLogger, serviceConfig)

	app.Run(ctx)
}
