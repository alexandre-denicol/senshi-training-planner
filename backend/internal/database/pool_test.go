package database

import (
	"errors"
	"testing"
)

func TestValidateDatabaseURLRequiresValue(t *testing.T) {
	err := ValidateDatabaseURL("")

	if !errors.Is(err, ErrMissingDatabaseURL) {
		t.Fatalf("expected missing database URL error, got %v", err)
	}
}

func TestValidateDatabaseURLRequiresPostgresScheme(t *testing.T) {
	err := ValidateDatabaseURL("mysql://app@example.com/app?sslmode=require")

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
			err := ValidateDatabaseURL(databaseURL)
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
			if err := ValidateDatabaseURL(databaseURL); err != nil {
				t.Fatalf("expected valid database URL, got %v", err)
			}
		})
	}
}
