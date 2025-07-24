package main

import (
	"context"

	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/api"
	"github.com/rryowa/medods_dvortsov/internal/controller"
	"github.com/rryowa/medods_dvortsov/internal/migrations"
	"github.com/rryowa/medods_dvortsov/internal/service"
	"github.com/rryowa/medods_dvortsov/internal/storage/postgres"
	"github.com/rryowa/medods_dvortsov/internal/storage/redis"
	"github.com/rryowa/medods_dvortsov/internal/util"

	_ "github.com/lib/pq"
)

func main() {
	ctx := context.Background()
	logger := util.NewZapLogger()

	db, dbCleanup, err := util.NewDBConnection(logger)
	if err != nil {
		logger.Fatal(zap.Error(err))
	}
	if err := migrations.RunMigrations(db, logger, "./internal/migrations"); err != nil {
		logger.Fatal(zap.Error(err))
	}

	redisClient, redisCleanup, err := util.NewRedisClient(logger, util.NewRedisConfig())
	if err != nil {
		logger.Fatal(zap.Error(err))
	}

	apiKeyService := service.NewAPIKeyService(redisClient, logger)
	if err := apiKeyService.SyncAPIKey(ctx); err != nil {
		logger.Fatal(zap.Error(err))
	}

	storage := postgres.NewStorage(db)
	cleanupFuncs := []func(){dbCleanup, redisCleanup}

	tokenStorage := redis.NewTokenStorage(redisClient)
	tokenService := service.NewTokenService(util.NewTokenConfig(), tokenStorage)
	webhookService := service.NewWebhookService(logger, util.GetWebhookURL())
	authService := service.NewAuthService(tokenService, storage, webhookService, logger)

	controller := controller.NewController(authService, logger)

	apiServer := api.NewAPI(controller, authService, redisClient, apiKeyService, util.NewServerConfig(), logger, cleanupFuncs)
	apiServer.Run(ctx)
}
