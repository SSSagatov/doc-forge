package db

import (
	"testing"
)

// TestDB checks a real PostgreSQL connection using the configured DB_* variables.
// Connect performs Ping with a timeout; the pool is closed after the check.
func TestDB(t *testing.T) {
	// Go runs this test from db/; the application expects .env in the project root.
	t.Chdir("..")

	pool, err := Connect(t.Context())
	if err != nil {
		t.Fatalf("db: connection test failed: %v", err)
	}
	t.Cleanup(pool.Close)

	t.Log("database connection passed successfully")
}
