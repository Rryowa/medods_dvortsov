package migrations

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

func RunMigrations(db *sql.DB, logger *zap.SugaredLogger, migrationsDir string) error {
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	logger.Info("Database migrations applied successfully!")
	return nil
}
