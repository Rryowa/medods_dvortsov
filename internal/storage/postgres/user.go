package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rryowa/medods_dvortsov/internal/models"
	"github.com/rryowa/medods_dvortsov/internal/storage"
)

type UserRepository struct {
	db storage.DBTX
}

func NewUserRepository(db storage.DBTX) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, guid string) (*models.User, error) {
	var user models.User
	query := `INSERT INTO users (guid) VALUES ($1) RETURNING id, guid`
	err := r.db.QueryRowContext(ctx, query, guid).Scan(&user.ID, &user.GUID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) GetUserByGUID(ctx context.Context, guid string) (*models.User, error) {
	var user models.User
	query := `SELECT id, guid FROM users WHERE guid = $1`
	err := r.db.QueryRowContext(ctx, query, guid).Scan(&user.ID, &user.GUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by guid: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	var user models.User
	query := `SELECT id, guid FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.GUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}
