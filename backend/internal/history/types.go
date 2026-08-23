package history

import (
	"context"
	"errors"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
)

const MaxRangeDays = 93

var (
	ErrInvalidRequest      = errors.New("invalid request")
	ErrNotFound            = errors.New("history record not found")
	ErrScheduleNotFound    = errors.New("schedule entry not found")
	ErrAlreadyCompleted    = errors.New("training already completed")
	ErrSnapshotUnavailable = errors.New("training snapshot unavailable")
)

type ListItem struct {
	ID              string    `json:"id"`
	TrainingDate    string    `json:"trainingDate"`
	WorkoutName     string    `json:"workoutName"`
	BlockCount      int       `json:"blockCount"`
	CompletedByName string    `json:"completedByName"`
	CompletedAt     time.Time `json:"completedAt"`
	ScheduleEntryID string    `json:"scheduleEntryId"`
}

type Detail struct {
	ID              string    `json:"id"`
	TrainingDate    string    `json:"trainingDate"`
	WorkoutName     string    `json:"workoutName"`
	CompletedByName string    `json:"completedByName"`
	CompletedAt     time.Time `json:"completedAt"`
	Blocks          []Block   `json:"blocks"`
}

type Block struct {
	Position     int    `json:"position"`
	BlockName    string `json:"blockName"`
	CategoryName string `json:"categoryName"`
}

type Store interface {
	ListHistory(ctx context.Context, from string, to string) ([]ListItem, error)
	GetHistory(ctx context.Context, id string) (Detail, error)
	CompleteScheduleEntry(ctx context.Context, historyID string, scheduleEntryID string, completedBy auth.PublicUser, completedAt time.Time) (Detail, error)
}

type ServiceAPI interface {
	List(ctx context.Context, from string, to string) ([]ListItem, error)
	Get(ctx context.Context, id string) (Detail, error)
	Complete(ctx context.Context, scheduleEntryID string, completedBy auth.PublicUser) (Detail, error)
}
