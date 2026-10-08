package postgres

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config contains PostgreSQL connection settings.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func loadConfig() (Config, error) {
	envPath, err := filepath.Abs("../../.env")
	if err != nil {
		return Config{}, fmt.Errorf("resolve .env path: %w", err)
	}

	if _, err := os.Stat(envPath); err != nil {
		return Config{}, fmt.Errorf("cannot access .env at %q: %w", envPath, err)
	}

	if err := godotenv.Load(envPath); err != nil {
		// Parsing errors may contain secrets from the file.
		return Config{}, errors.New("failed to load .env: check that the file exists, is readable, and has valid syntax")
	}

	return Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Name:     os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}, nil
}
