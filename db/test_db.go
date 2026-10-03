package db

import (
	"context"
	"fmt"
)

// TestDB checks a real PostgreSQL connection using the configured DB_* variables.
// Connect performs Ping with a timeout; the pool is closed after the check.
func TestDB(ctx context.Context) error {
	pool, err := Connect(ctx)
	if err != nil {
		return fmt.Errorf("db: connection test failed: %w", err)
	}
	defer pool.Close()

	return nil
}
