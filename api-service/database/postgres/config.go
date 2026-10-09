package postgres

import (
	"cloud-native-platform/api-service/config"
	"os"
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
	if err := config.LoadEnv(); err != nil {
		return Config{}, err
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
