package db

import (
	"errors"
	"fmt"
	"os"
	"strings"

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
	if err := godotenv.Load(".env"); err != nil {
		// Parsing errors may contain secrets from the file.
		return Config{}, errors.New("failed to load .env: check that the file exists, is readable, and has valid syntax")
	}

	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_NAME", "DB_SSLMODE"} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			return Config{}, fmt.Errorf("db: %s is required", key)
		}
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
