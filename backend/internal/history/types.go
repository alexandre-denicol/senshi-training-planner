package history

import (
	"context"
	"errors"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
)

const MaxRangeDays = 93
const MaxParticipantStudentIDs = 100
const MaxNotesChars = 2000

var (
	ErrInvalidRequest      = errors.New("invalid request")
	ErrInvalidParticipants = errors.New("invalid participants")
	ErrNotFound            = errors.New("history record not found")
	ErrScheduleNotFound    = errors.New("schedule entry not found")
	ErrAlreadyCompleted    = errors.New("training already completed")
	ErrSnapshotUnavailable = errors.New("training snapshot unavailable")
)

type ListItem struct {
	ID               string    `json:"id"`
	TrainingDate     string    `json:"trainingDate"`
	WorkoutName      string    `json:"workoutName"`
	BlockCount       int       `json:"blockCount"`
	ParticipantCount *int      `json:"participantCount"`
	CompletedByName  string    `json:"completedByName"`
	CompletedAt      time.Time `json:"completedAt"`
	ScheduleEntryID  string    `json:"scheduleEntryId"`
}

type Detail struct {
	ID               string    `json:"id"`
	TrainingDate     string    `json:"trainingDate"`
	WorkoutName      string    `json:"workoutName"`
	ParticipantCount *int      `json:"participantCount"`
	ParticipantNames []string  `json:"participantNames"`
	Notes            *string   `json:"notes"`
	CompletedByName  string    `json:"completedByName"`
	CompletedAt      time.Time `json:"completedAt"`
	Blocks           []Block   `json:"blocks"`
}

type Block struct {
	Position     int            `json:"position"`
	BlockName    string         `json:"blockName"`
	CategoryName string         `json:"categoryName"`
	Description  *string        `json:"description"`
	Sequence     []SequenceItem `json:"sequence"`
}

type SequenceItem struct {
	Position int    `json:"position"`
	Text     string `json:"text"`
}

type Store interface {
	ListHistory(ctx context.Context, from string, to string) ([]ListItem, error)
	GetHistory(ctx context.Context, id string) (Detail, error)
	CompleteScheduleEntry(ctx context.Context, historyID string, scheduleEntryID string, completedBy auth.PublicUser, completedAt time.Time, details CompletionDetails) (Detail, error)
}

type ServiceAPI interface {
	List(ctx context.Context, from string, to string) ([]ListItem, error)
	Get(ctx context.Context, id string) (Detail, error)
	Complete(ctx context.Context, scheduleEntryID string, completedBy auth.PublicUser, details CompletionDetails) (Detail, error)
}

type CompletionDetails struct {
	ParticipantStudentIDs []string
	Notes                 *string
}
