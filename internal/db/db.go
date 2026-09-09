package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/masoniclounge/masoniccore/migrations"
)

// DB wraps the pgx connection pool and the goose migration provider.
type DB struct {
	pool     *pgxpool.Pool
	provider *goose.Provider
}

// Open creates a pgx connection pool and prepares the goose migration provider.
// Migrations are not applied here; call Migrate explicitly.
func Open(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create goose provider: %w", err)
	}

	return &DB{pool: pool, provider: provider}, nil
}

// Migrate applies all pending schema migrations.
func (d *DB) Migrate(ctx context.Context) error {
	if _, err := d.provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// SchemaVersion returns the current goose schema version.
func (d *DB) SchemaVersion(ctx context.Context) (int64, error) {
	return d.provider.GetDBVersion(ctx)
}

// Ping verifies database connectivity.
func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

// Pool exposes the underlying pgx pool for store/services layers.
func (d *DB) Pool() *pgxpool.Pool {
	return d.pool
}

// Close releases the connection pool.
func (d *DB) Close() {
	d.pool.Close()
}
