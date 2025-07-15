package service

import (
	"context"
	"time"

	"github.com/rryowa/medods_dvortsov/internal/models"
)

type RefreshTokenManager interface {
	CreateSession(ctx context.Context, session models.RefreshSession, ttl time.Duration) error
	FindSession(ctx context.Context, refreshToken string) (*models.RefreshSession, error)
	DeleteSession(ctx context.Context, refreshToken string) error
	DeleteAllUserSessions(ctx context.Context, userID string) error
}
