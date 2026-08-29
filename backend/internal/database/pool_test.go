package database

import (
	"errors"
	"testing"
)

func TestValidateDatabaseURLRequiresValue(t *testing.T) {
	err := ValidateDatabaseURL("", "production")

	if !errors.Is(err, ErrMissingDatabaseURL) {
		t.Fatalf("expected missing database URL error, got %v", err)
	}
}

func TestValidateDatabaseURLRequiresPostgresScheme(t *testing.T) {
	err := ValidateDatabaseURL("mysql://app@example.com/app?sslmode=require", "production")

	if !errors.Is(err, ErrInvalidDatabaseURL) {
		t.Fatalf("expected invalid database URL error, got %v", err)
	}
}

func TestValidateDatabaseURLRequiresSecureSSLMode(t *testing.T) {
	tests := []string{
		"postgres://app@example.com/app",
		"postgres://app@example.com/app?sslmode=disable",
		"postgres://app@example.com/app?sslmode=prefer",
	}

	for _, databaseURL := range tests {
		t.Run(databaseURL, func(t *testing.T) {
			err := ValidateDatabaseURL(databaseURL, "production")
			if !errors.Is(err, ErrTLSRequired) {
				t.Fatalf("expected TLS required error, got %v", err)
			}
		})
	}
}

func TestValidateDatabaseURLAcceptsTLSModes(t *testing.T) {
	tests := []string{
		"postgres://app@example.com/app?sslmode=require",
		"postgresql://app@example.com/app?sslmode=verify-ca",
		"postgres://app@example.com/app?sslmode=verify-full",
	}

	for _, databaseURL := range tests {
		t.Run(databaseURL, func(t *testing.T) {
			if err := ValidateDatabaseURL(databaseURL, "production"); err != nil {
				t.Fatalf("expected valid database URL, got %v", err)
			}
		})
	}
}

func TestValidateDatabaseURLAllowsLocalDisableOnlyInDevelopment(t *testing.T) {
	tests := []string{
		"postgres://app@localhost/app?sslmode=disable",
		"postgres://app@127.0.0.1/app?sslmode=disable",
		"postgres://app@[::1]/app?sslmode=disable",
	}

	for _, databaseURL := range tests {
		t.Run(databaseURL, func(t *testing.T) {
			if err := ValidateDatabaseURL(databaseURL, "development"); err != nil {
				t.Fatalf("expected local development database URL to be valid, got %v", err)
			}
		})
	}
}

func TestValidateDatabaseURLRejectsDisableForProductionOrRemoteHosts(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		appEnv      string
	}{
		{
			name:        "production localhost",
			databaseURL: "postgres://app@localhost/app?sslmode=disable",
			appEnv:      "production",
		},
		{
			name:        "development remote",
			databaseURL: "postgres://app@db.example.com/app?sslmode=disable",
			appEnv:      "development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDatabaseURL(tt.databaseURL, tt.appEnv)
			if !errors.Is(err, ErrTLSRequired) {
				t.Fatalf("expected TLS required error, got %v", err)
			}
		})
	}
}
