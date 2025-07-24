package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rryowa/medods_dvortsov/internal/models"
	"github.com/rryowa/medods_dvortsov/internal/storage"
)

type Storage struct {
	db *sql.DB
	*UserRepository
	*SessionRepository
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{
		db:                db,
		UserRepository:    NewUserRepository(db),
		SessionRepository: NewSessionRepository(db),
	}
}

// IssueTokensTx выполняет транзакцию по выпуску токенов.
// Логика работы "get or create"
// У нового пользователя будет свой GUID, а не из запроса.
func (s *Storage) IssueTokensTx(ctx context.Context, guid string, session models.RefreshSession) (*models.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	userRepoTx := NewUserRepository(tx)
	sessionRepoTx := NewSessionRepository(tx)

	user, err := userRepoTx.GetUserByGUID(ctx, guid)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			user, err = userRepoTx.CreateUser(ctx, guid)
			if err != nil {
				return nil, fmt.Errorf("failed to create user in tx: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user by guid in tx: %w", err)
		}
	}

	session.UserID = user.ID
	_, err = sessionRepoTx.CreateSession(ctx, session, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create session in tx: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return user, nil
}

// RotateTokensTx выполняет транзакцию по ротации refresh-токенов.
// Старая сессия помечается как 'used', создается новая.
func (s *Storage) RotateTokensTx(ctx context.Context, oldSelector string, newSession models.RefreshSession, userID int64) (*models.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	sessionRepoTx := NewSessionRepository(tx)
	userRepoTx := NewUserRepository(tx)

	if err := sessionRepoTx.MarkSessionAsUsed(ctx, oldSelector); err != nil {
		return nil, fmt.Errorf("failed to mark session as used in tx: %w", err)
	}

	newSession.UserID = userID
	if _, err := sessionRepoTx.CreateSession(ctx, newSession, 0); err != nil {
		return nil, fmt.Errorf("failed to create new session in tx: %w", err)
	}

	user, err := userRepoTx.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id in tx: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return user, nil
}