package database

import (
	"context"
	"errors"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMissingDatabaseURL = errors.New("DATABASE_URL is required to connect to PostgreSQL")
	ErrInvalidDatabaseURL = errors.New("DATABASE_URL must be a valid PostgreSQL URL")
	ErrTLSRequired        = errors.New("DATABASE_URL must require TLS with sslmode=require, verify-ca, or verify-full")
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if err := ValidateDatabaseURL(databaseURL); err != nil {
		return nil, err
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, ErrInvalidDatabaseURL
	}

	return pgxpool.NewWithConfig(ctx, cfg)
}

func ValidateDatabaseURL(databaseURL string) error {
	if databaseURL == "" {
		return ErrMissingDatabaseURL
	}

	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidDatabaseURL
	}

	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return ErrInvalidDatabaseURL
	}

	switch parsed.Query().Get("sslmode") {
	case "require", "verify-ca", "verify-full":
		return nil
	default:
		return ErrTLSRequired
	}
}
