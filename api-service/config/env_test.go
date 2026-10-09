package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFromParent(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "nested")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DOC_FORGE_CONFIG_TEST=file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(child)
	t.Setenv("ENV_FILE", "")
	t.Setenv("DOC_FORGE_CONFIG_TEST", "process")
	if err := LoadEnv(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DOC_FORGE_CONFIG_TEST") != "process" {
		t.Fatal("process variable overwritten")
	}
	if err := os.Unsetenv("DOC_FORGE_CONFIG_TEST"); err != nil {
		t.Fatal(err)
	}
	if err := LoadEnv(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DOC_FORGE_CONFIG_TEST") != "file" {
		t.Fatal("parent .env not loaded")
	}
}

func TestExplicitEnvMissing(t *testing.T) {
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "missing"))
	if LoadEnv() == nil {
		t.Fatal("missing explicit file accepted")
	}
}
