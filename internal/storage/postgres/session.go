package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rryowa/medods_dvortsov/internal/models"
	"github.com/rryowa/medods_dvortsov/internal/storage"
)

type SessionRepository struct {
	db storage.DBTX
}

func NewSessionRepository(db storage.DBTX) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) CreateSession(ctx context.Context, session models.RefreshSession) (int64, error) {
	query := `INSERT INTO sessions (user_id, selector, verifier_hash, client_ip, user_agent, expires_at, created_at, access_token_jti) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	var id int64
	err := r.db.QueryRowContext(
		ctx,
		query,
		session.UserID,
		session.Selector,
		session.VerifierHash,
		session.IPAddress,
		session.UserAgent,
		session.ExpiresAt,
		session.CreatedAt,
		session.AccessTokenJTI,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert session: %w", err)
	}
	return id, nil
}

func (r *SessionRepository) GetActiveSessionBySelector(
	ctx context.Context,
	selector string,
) (*models.RefreshSession, error) {
	var session models.RefreshSession
	query := `SELECT id, user_id, selector, verifier_hash, client_ip, user_agent, expires_at, created_at, access_token_jti FROM sessions WHERE selector = $1 AND status = 'active'`
	err := r.db.QueryRowContext(ctx, query, selector).Scan(
		&session.ID,
		&session.UserID,
		&session.Selector,
		&session.VerifierHash,
		&session.IPAddress,
		&session.UserAgent,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.AccessTokenJTI,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session with selector %s not found: %w", selector, storage.ErrSessionNotFound)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

func (r *SessionRepository) FindSessionBySelector(
	ctx context.Context,
	selector string,
) (*models.RefreshSession, error) {
	var session models.RefreshSession
	query := `SELECT id, user_id, selector, verifier_hash, client_ip, user_agent, expires_at, created_at, access_token_jti FROM sessions WHERE selector = $1`
	err := r.db.QueryRowContext(ctx, query, selector).Scan(
		&session.ID,
		&session.UserID,
		&session.Selector,
		&session.VerifierHash,
		&session.IPAddress,
		&session.UserAgent,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.AccessTokenJTI,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session with selector %s not found: %w", selector, storage.ErrSessionNotFound)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

// MarkSessionAsUsed помечает сессию как использованную.
func (r *SessionRepository) MarkSessionAsUsed(ctx context.Context, selector string) error {
	query := `UPDATE sessions SET status = 'used' WHERE selector = $1`
	_, err := r.db.ExecContext(ctx, query, selector)
	if err != nil {
		return fmt.Errorf("failed to mark session as used: %w", err)
	}
	return nil
}

func (r *SessionRepository) DeleteSession(ctx context.Context, selector string) error {
	query := `DELETE FROM sessions WHERE selector = $1`
	_, err := r.db.ExecContext(ctx, query, selector)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (r *SessionRepository) DeleteAllUserSessions(ctx context.Context, userID int64) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}
	return nil
}
