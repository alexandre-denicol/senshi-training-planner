package config

import (
	"errors"
	"testing"
)

func TestLoadUsesDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_ENV", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}

	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %q", cfg.Port)
	}

	if cfg.DatabaseURL != "" {
		t.Fatal("expected empty database URL when DATABASE_URL is not set")
	}

	if cfg.AppEnv != AppEnvProduction {
		t.Fatalf("expected production as safe default, got %q", cfg.AppEnv)
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("APP_ENV", "development")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}

	if cfg.Port != "9000" {
		t.Fatalf("expected port from environment, got %q", cfg.Port)
	}

	if cfg.DatabaseURL != "postgres://example" {
		t.Fatal("expected database URL from environment")
	}

	if cfg.AppEnv != AppEnvDevelopment {
		t.Fatalf("expected development app env, got %q", cfg.AppEnv)
	}
}

func TestLoadRejectsInvalidAppEnv(t *testing.T) {
	t.Setenv("APP_ENV", "local")

	_, err := Load()
	if !errors.Is(err, ErrInvalidAppEnv) {
		t.Fatalf("expected invalid app env error, got %v", err)
	}
}
