package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenStorage struct {
	client *redis.Client
}

func NewTokenStorage(client *redis.Client) *TokenStorage {
	return &TokenStorage{client: client}
}

func (s *TokenStorage) InvalidateToken(ctx context.Context, token string, expiration time.Duration) error {
	if err := s.client.Set(ctx, token, "invalidated", expiration).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// IsTokenInvalidated проверяет наличие токена в Redis
func (s *TokenStorage) IsTokenInvalidated(ctx context.Context, token string) (bool, error) {
	result, err := s.client.Get(ctx, token).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("redis get result: %w", err)
	}
	return result == "invalidated", nil
}
