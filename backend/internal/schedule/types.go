package schedule

import (
	"context"
	"errors"
	"time"
)

const MaxRangeDays = 93

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidWorkout = errors.New("invalid workout")
	ErrDuplicate      = errors.New("workout already scheduled for date")
	ErrNotFound       = errors.New("schedule entry not found")
)

type WorkoutRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type Entry struct {
	ID            string     `json:"id"`
	ScheduledDate string     `json:"scheduledDate"`
	Workout       WorkoutRef `json:"workout"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type CreateInput struct {
	WorkoutID     string `json:"workoutId"`
	ScheduledDate string `json:"scheduledDate"`
}

type UpdateInput struct {
	WorkoutID     string `json:"workoutId"`
	ScheduledDate string `json:"scheduledDate"`
}

type NewEntry struct {
	ID            string
	WorkoutID     string
	ScheduledDate string
}

type Store interface {
	ListEntries(ctx context.Context, from string, to string) ([]Entry, error)
	CreateEntry(ctx context.Context, entry NewEntry) (Entry, error)
	UpdateEntry(ctx context.Context, id string, workoutID string, scheduledDate string) (Entry, error)
	DeleteEntry(ctx context.Context, id string) error
}

type ServiceAPI interface {
	List(ctx context.Context, from string, to string) ([]Entry, error)
	Create(ctx context.Context, input CreateInput) (Entry, error)
	Update(ctx context.Context, id string, input UpdateInput) (Entry, error)
	Delete(ctx context.Context, id string) error
}
