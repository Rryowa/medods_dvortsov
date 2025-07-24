package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	CurrentAPIKeyRedisKey      = "apikey:current"
	OldAPIKeyRedisKey          = "apikey:old"
	APIKeyRotationTimeRedisKey = "apikey:rotation_time"
)

type APIKeyService struct {
	rdb *redis.Client
	log *zap.SugaredLogger
}

func NewAPIKeyService(rdb *redis.Client, log *zap.SugaredLogger) *APIKeyService {
	return &APIKeyService{rdb: rdb, log: log}
}

func (s *APIKeyService) SyncAPIKey(ctx context.Context) error {
	newKey := os.Getenv("AUTH_SERVICE_API_KEY")
	if newKey == "" {
		return fmt.Errorf("AUTH_SERVICE_API_KEY is empty during sync attempt")
	}

	hashedNewKey := s.hashAPIKey(newKey)

	currentHashedKey, err := s.rdb.Get(ctx, CurrentAPIKeyRedisKey).Result()
	if err != nil {
		if err == redis.Nil {
			s.log.Warn("Current API key not found during sync; re-initializing.")
			return s.setInitialAPIKey(ctx)
		}
		return fmt.Errorf("failed to get current API key from Redis: %w", err)
	}

	if len(hashedNewKey) == len(currentHashedKey) && subtle.ConstantTimeCompare([]byte(hashedNewKey), []byte(currentHashedKey)) == 1 {
		s.log.Info("Skipping key sync: new key is the same as the current one.")
		return nil
	}

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, OldAPIKeyRedisKey, currentHashedKey, 24*time.Hour)
	pipe.Set(ctx, CurrentAPIKeyRedisKey, hashedNewKey, 0)
	pipe.Set(ctx, APIKeyRotationTimeRedisKey, time.Now().UTC().Format(time.RFC3339), 0)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync API key in Redis: %w", err)
	}

	s.log.Info("API Key synced successfully.")
	return nil
}

// IsValidAPIKey rotation is 24-hour
func (s *APIKeyService) IsValidAPIKey(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, nil
	}

	hashedKey := s.hashAPIKey(key)

	currentHashedKey, err := s.rdb.Get(ctx, CurrentAPIKeyRedisKey).Result()
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("failed to get current API key from Redis: %w", err)
	}

	if len(hashedKey) == len(currentHashedKey) && subtle.ConstantTimeCompare([]byte(hashedKey), []byte(currentHashedKey)) == 1 {
		return true, nil
	}

	oldHashedKey, err := s.rdb.Get(ctx, OldAPIKeyRedisKey).Result()
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("failed to get old API key from Redis: %w", err)
	}

	if oldHashedKey != "" && len(hashedKey) == len(oldHashedKey) && subtle.ConstantTimeCompare([]byte(hashedKey), []byte(oldHashedKey)) == 1 {
		rotationTimeStr, err := s.rdb.Get(ctx, APIKeyRotationTimeRedisKey).Result()
		if err != nil {
			return false, fmt.Errorf("failed to get key rotation time from Redis: %w", err)
		}
		rotationTime, err := time.Parse(time.RFC3339, rotationTimeStr)
		if err != nil {
			return false, fmt.Errorf("failed to parse key rotation time: %w", err)
		}

		if time.Since(rotationTime) <= 24*time.Hour {
			return true, nil
		}
	}

	return false, nil
}

func (s *APIKeyService) setInitialAPIKey(ctx context.Context) error {
	apiKey := os.Getenv("AUTH_SERVICE_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("AUTH_SERVICE_API_KEY environment variable not set")
	}

	hashedKey := s.hashAPIKey(apiKey)

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, CurrentAPIKeyRedisKey, hashedKey, 0)
	pipe.Set(ctx, APIKeyRotationTimeRedisKey, time.Now().UTC().Format(time.RFC3339), 0)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("init API key: %w", err)
	}
	s.log.Info("API Key initialized in Redis.")
	return nil
}

func (s *APIKeyService) hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
