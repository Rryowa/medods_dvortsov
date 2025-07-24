package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rryowa/medods_dvortsov/internal/models"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrUserNotFound    = errors.New("user not found")
)

type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type Storage interface {
	SessionRepository
	UserRepository
	IssueTokensTx(ctx context.Context, guid string, session models.RefreshSession) (*models.User, error)
	RotateTokensTx(
		ctx context.Context,
		oldSelector string,
		newSession models.RefreshSession,
		userID int64,
	) (*models.User, error)
}

type UserRepository interface {
	CreateUser(ctx context.Context, guid string) (*models.User, error)
	GetUserByGUID(ctx context.Context, guid string) (*models.User, error)
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session models.RefreshSession) (int64, error)
	GetActiveSessionBySelector(ctx context.Context, selector string) (*models.RefreshSession, error)
	FindSessionBySelector(ctx context.Context, selector string) (*models.RefreshSession, error)
	MarkSessionAsUsed(ctx context.Context, selector string) error
	DeleteSession(ctx context.Context, selector string) error
	DeleteAllUserSessions(ctx context.Context, userID int64) error
}

type TokenStorage interface {
	InvalidateToken(ctx context.Context, token string, expiration time.Duration) error
	IsTokenInvalidated(ctx context.Context, token string) (bool, error)
}
