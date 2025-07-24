package redis

import (
	"context"
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
	return s.client.Set(ctx, token, "invalidated", expiration).Err()
}

// IsTokenInvalidated проверяет наличие токена в Redis.
func (s *TokenStorage) IsTokenInvalidated(ctx context.Context, token string) (bool, error) {
	result, err := s.client.Get(ctx, token).Result()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return result == "invalidated", nil
}
