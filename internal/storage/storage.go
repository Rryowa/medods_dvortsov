package storage

import (
	"context"
	"errors"
	"time"

	"github.com/rryowa/medods_dvortsov/internal/models"
)

var ErrSessionNotFound = errors.New("session not found")

type Storage interface {
	SessionRepository
	APIKeyRepository
}

type APIKeyRepository interface {
	GetAPIKey(ctx context.Context, apiKey string) (*models.APIKey, error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session models.RefreshSession, ttl time.Duration) error
	GetCurrentSession(ctx context.Context, refreshToken string) (*models.RefreshSession, error)
	DeleteSession(ctx context.Context, refreshToken string) error
	DeleteAllUserSessions(ctx context.Context, userID string) error
}
