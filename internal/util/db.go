package util

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
)

type DBConfig struct {
	DSN string
}

func NewDBConfig() *DBConfig {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	return &DBConfig{
		DSN: dsn,
	}
}

type RedisConfig struct {
	Addr string
}

func NewRedisConfig() *RedisConfig {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		log.Fatal("REDIS_ADDR is not set")
	}

	return &RedisConfig{
		Addr: addr,
	}
}

func NewDBConnection(ctx context.Context, logger *zap.SugaredLogger) (*sql.DB, func(), error) {
	dbConfig := NewDBConfig()
	db, err := sql.Open("postgres", dbConfig.DSN)
	if err != nil {
		return nil, nil, fmt.Errorf("sql open: %w", err)
	}

	if err = db.PingContext(ctx); err != nil {
		return nil, nil, fmt.Errorf("db ping context: %w", err)
	}

	logger.Info("Successfully connected to database!")

	cleanup := func() {
		if err := db.Close(); err != nil {
			logger.Errorf("Failed to close database connection: %v", err)
		} else {
			logger.Info("Database connection closed successfully.")
		}
	}

	return db, cleanup, nil
}

func NewRedisClient(logger *zap.SugaredLogger, cfg *RedisConfig) (*redis.Client, func(), error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: "",
		DB:       0,
	})

	logger.Info("Successfully connected to Redis!")

	cleanup := func() {
		if err := redisClient.Close(); err != nil {
			logger.Errorf("Failed to close Redis connection: %v", err)
		} else {
			logger.Info("Redis connection closed successfully.")
		}
	}

	return redisClient, cleanup, nil
}
