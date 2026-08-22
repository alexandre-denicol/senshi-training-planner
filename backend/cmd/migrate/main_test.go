package main

import (
	"errors"
	"testing"

	"github.com/alexandre/senshi-training-planner/backend/internal/database"
)

func TestRunRequiresCommand(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if err := run(nil); err == nil {
		t.Fatal("expected usage error")
	}
}

func TestRunRejectsUnknownCommandBeforeDatabaseValidation(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if err := run([]string{"unknown"}); err == nil {
		t.Fatal("expected usage error")
	}
}

func TestRunRequiresDatabaseURLForKnownCommand(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	err := run([]string{"status"})
	if !errors.Is(err, database.ErrMissingDatabaseURL) {
		t.Fatalf("expected missing database URL error, got %v", err)
	}
}
