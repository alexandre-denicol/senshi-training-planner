package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/alexandre/senshi-training-planner/backend/internal/config"
	"github.com/alexandre/senshi-training-planner/backend/internal/database"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const migrationsSourceURL = "file://migrations"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: go run ./cmd/migrate [up|down|status]")
	}

	command := args[0]
	if command != "up" && command != "down" && command != "status" {
		return errors.New("usage: go run ./cmd/migrate [up|down|status]")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := database.ValidateDatabaseURL(cfg.DatabaseURL); err != nil {
		return err
	}

	m, err := migrate.New(migrationsSourceURL, cfg.DatabaseURL)
	if err != nil {
		return errors.New("could not initialize migrations")
	}
	defer m.Close()

	switch command {
	case "up":
		return migrateUp(m)
	case "down":
		return migrateDown(m)
	case "status":
		return migrateStatus(m)
	}

	return nil
}

func migrateUp(m *migrate.Migrate) error {
	err := m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("no pending migrations")
		return nil
	}
	if err != nil {
		return fmt.Errorf("migration up failed: %w", err)
	}

	fmt.Println("migrations applied")
	return nil
}

func migrateDown(m *migrate.Migrate) error {
	err := m.Steps(-1)
	if errors.Is(err, migrate.ErrNoChange) || errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("no migrations to roll back")
		return nil
	}
	if err != nil {
		return fmt.Errorf("migration down failed: %w", err)
	}

	fmt.Println("rolled back one migration")
	return nil
}

func migrateStatus(m *migrate.Migrate) error {
	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("status: no migrations applied")
		return nil
	}
	if err != nil {
		return fmt.Errorf("migration status failed: %w", err)
	}

	fmt.Printf("version: %d dirty: %t\n", version, dirty)
	return nil
}
