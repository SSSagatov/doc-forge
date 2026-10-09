package config

import (
	"errors"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
)

// LoadEnv respects process variables. ENV_FILE overrides automatic discovery.
// Without ENV_FILE, search from the working directory up to the filesystem root.
func LoadEnv() error {
	if path := os.Getenv("ENV_FILE"); path != "" {
		return load(path)
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			return load(path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return errors.New("cannot access .env")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}
func load(path string) error {
	if err := godotenv.Load(path); err != nil {
		return errors.New("cannot load .env: check path, permissions and syntax")
	}
	return nil
}
