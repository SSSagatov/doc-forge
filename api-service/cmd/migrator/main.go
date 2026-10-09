package main

import (
	"cloud-native-platform/api-service/database/postgres"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/stdlib"
)

func main() {
	path := flag.String("path", "api-service/migrations", "Directory containing SQL migrations")
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatal("unexpected arguments; use --help for usage")
	}

	if err := run(*path); err != nil {
		log.Fatal(err)
	}
}

func run(path string) error {
	source, err := iofs.New(os.DirFS(path), ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	defer source.Close()
	if _, err := source.First(); err != nil {
		return fmt.Errorf("no migrations found in %q: %w", path, err)
	}

	pool, err := postgres.Connect(context.Background())
	if err != nil {
		return err
	}
	defer pool.Close()

	// Reuse the existing DB_* configuration and pgx connection pool.
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	driver, err := pgxmigrate.WithInstance(sqlDB, &pgxmigrate.Config{
		StatementTimeout: 60 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("initialize migration driver: %w", err)
	}
	defer driver.Close()

	migrator, err := migrate.NewWithInstance("iofs", source, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("initialize migrator: %w", err)
	}

	if err := migrator.Up(); errors.Is(err, migrate.ErrNoChange) {
		log.Println("Database is up to date; no migrations to apply")
		return nil
	} else if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	log.Println("Database migrations applied successfully")
	return nil
}
