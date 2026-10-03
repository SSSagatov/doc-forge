package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a PostgreSQL pool using DB_* environment variables and verifies connectivity.
// The caller must close the returned pool when it is no longer needed.
func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	settings, err := loadConfig()
	if err != nil {
		return nil, err
	}

	config, err := pgxpool.ParseConfig(dsn(settings))
	if err != nil {
		// Parse errors can contain the connection string, including credentials.
		return nil, errors.New("db: invalid PostgreSQL configuration; check DB_* variables")
	}
	if config.ConnConfig.ConnectTimeout == 0 {
		config.ConnConfig.ConnectTimeout = 5 * time.Second
	}

	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, config)
	if err != nil {
		return nil, fmt.Errorf("db: create PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping PostgreSQL: %w", err)
	}
	return pool, nil
}

// connection string for postgres
// it formatted by constructor in ./db/config
func dsn(config Config) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Name,
		config.SSLMode,
	)
}
