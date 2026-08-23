package schedule

import (
	"context"
	"errors"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
)

type Service struct {
	store   Store
	newUUID func() (string, error)
}

func NewService(store Store) *Service {
	return &Service{
		store:   store,
		newUUID: auth.NewUUID,
	}
}

func (s *Service) List(ctx context.Context, from string, to string) ([]Entry, error) {
	fromDate, err := parseDate(from)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	toDate, err := parseDate(to)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	if fromDate.After(toDate) {
		return nil, ErrInvalidRequest
	}
	if int(toDate.Sub(fromDate).Hours()/24)+1 > MaxRangeDays {
		return nil, ErrInvalidRequest
	}

	return s.store.ListEntries(ctx, from, to)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Entry, error) {
	workoutID, scheduledDate, err := validateEntryInput(input.WorkoutID, input.ScheduledDate)
	if err != nil {
		return Entry{}, err
	}
	id, err := s.newUUID()
	if err != nil {
		return Entry{}, err
	}

	entry, err := s.store.CreateEntry(ctx, NewEntry{ID: id, WorkoutID: workoutID, ScheduledDate: scheduledDate})
	return mapEntryResult(entry, err)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Entry, error) {
	if !validID(id) {
		return Entry{}, ErrNotFound
	}
	workoutID, scheduledDate, err := validateEntryInput(input.WorkoutID, input.ScheduledDate)
	if err != nil {
		return Entry{}, err
	}

	entry, err := s.store.UpdateEntry(ctx, id, workoutID, scheduledDate)
	return mapEntryResult(entry, err)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}

	err := s.store.DeleteEntry(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func mapEntryResult(entry Entry, err error) (Entry, error) {
	if errors.Is(err, ErrDuplicate) {
		return Entry{}, ErrDuplicate
	}
	if errors.Is(err, ErrInvalidWorkout) {
		return Entry{}, ErrInvalidWorkout
	}
	if errors.Is(err, ErrNotFound) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, err
	}

	return entry, nil
}

func validateEntryInput(workoutID string, scheduledDate string) (string, string, error) {
	if !validID(workoutID) {
		return "", "", ErrInvalidWorkout
	}
	if _, err := parseDate(scheduledDate); err != nil {
		return "", "", ErrInvalidRequest
	}

	return workoutID, scheduledDate, nil
}

func parseDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	if parsed.Format("2006-01-02") != value {
		return time.Time{}, ErrInvalidRequest
	}

	return parsed, nil
}

func validID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for index, value := range id {
		switch index {
		case 8, 13, 18, 23:
			if value != '-' {
				return false
			}
		default:
			if (value < '0' || value > '9') &&
				(value < 'a' || value > 'f') &&
				(value < 'A' || value > 'F') {
				return false
			}
		}
	}

	return true
}
