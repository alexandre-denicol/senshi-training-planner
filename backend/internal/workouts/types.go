package workouts

import (
	"context"
	"errors"
	"time"
)

const MaxNameRunes = 120

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidBlocks  = errors.New("invalid blocks")
	ErrNameExists     = errors.New("workout name already exists")
	ErrNotFound       = errors.New("workout not found")
	ErrInUse          = errors.New("workout is in use")
)

type CategoryRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WorkoutListItem struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Active     bool      `json:"active"`
	BlockCount int       `json:"blockCount"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type WorkoutBlock struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description *string             `json:"description"`
	Sequence    []BlockSequenceItem `json:"sequence"`
	Active      bool                `json:"active"`
	Position    int                 `json:"position"`
	Category    CategoryRef         `json:"category"`
}

type BlockSequenceItem struct {
	Position int    `json:"position"`
	Text     string `json:"text"`
}

type WorkoutDetail struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Active    bool           `json:"active"`
	Blocks    []WorkoutBlock `json:"blocks"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type CreateInput struct {
	Name     string   `json:"name"`
	BlockIDs []string `json:"blockIds"`
}

type UpdateInput struct {
	Name     string   `json:"name"`
	BlockIDs []string `json:"blockIds"`
}

type StatusInput struct {
	Active *bool `json:"active"`
}

type NewWorkout struct {
	ID       string
	Name     string
	BlockIDs []string
}

type Store interface {
	ListWorkouts(ctx context.Context) ([]WorkoutListItem, error)
	GetWorkout(ctx context.Context, id string) (WorkoutDetail, error)
	CreateWorkout(ctx context.Context, workout NewWorkout) (WorkoutDetail, error)
	UpdateWorkout(ctx context.Context, id string, name string, blockIDs []string) (WorkoutDetail, error)
	SetWorkoutStatus(ctx context.Context, id string, active bool) (WorkoutListItem, error)
	DeleteWorkout(ctx context.Context, id string) error
}

type ServiceAPI interface {
	List(ctx context.Context) ([]WorkoutListItem, error)
	Get(ctx context.Context, id string) (WorkoutDetail, error)
	Create(ctx context.Context, input CreateInput) (WorkoutDetail, error)
	Update(ctx context.Context, id string, input UpdateInput) (WorkoutDetail, error)
	SetStatus(ctx context.Context, id string, input StatusInput) (WorkoutListItem, error)
	Delete(ctx context.Context, id string) error
}
