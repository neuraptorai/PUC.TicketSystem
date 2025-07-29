// File: internal/database/postgres.g
package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Driver para ler migrations de arquivos
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}
	return pool, nil
}

func RunMigrations(dsn string, log *slog.Logger) error {

	migrateDSN := strings.Replace(dsn, "postgres://", "pgx5://", 1)

	m, err := migrate.New("file://sql/migrations", migrateDSN)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		// Se não houver migrações, a versão pode dar erro. Tratamos isso.
		if errors.Is(err, migrate.ErrNilVersion) {
			log.Info("Database schema is at base level (no migrations applied yet).")
			return nil
		}
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	log.Info("Database migration check complete", "version", version, "dirty", dirty)
	return nil
}
