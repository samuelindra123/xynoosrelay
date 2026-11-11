package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var MigrationFiles embed.FS

// Up applies all pending goose migrations to the specified database URL.
func Up(dbURL string, log *slog.Logger) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open sql connection for migration: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(MigrationFiles)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	log.Info("executing database migrations (goose up)")
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("failed to run goose up: %w", err)
	}

	log.Info("database migrations applied successfully")
	return nil
}

// Status returns current migration status.
func Status(dbURL string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open sql connection: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(MigrationFiles)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.Status(db, ".")
}
