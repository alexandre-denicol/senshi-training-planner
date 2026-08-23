package schedule

import (
	"context"
	"errors"
	"testing"
	"time"
)

const (
	entryID           = "11111111-1111-1111-1111-111111111111"
	missingEntryID    = "22222222-2222-2222-2222-222222222222"
	activeWorkoutID   = "33333333-3333-3333-3333-333333333333"
	otherWorkoutID    = "44444444-4444-4444-4444-444444444444"
	inactiveWorkoutID = "55555555-5555-5555-5555-555555555555"
)

func TestServiceCreateScheduleEntry(t *testing.T) {
	store := &fakeScheduleStore{}
	service := NewService(store)
	service.newUUID = func() (string, error) { return entryID, nil }

	entry, err := service.Create(context.Background(), CreateInput{WorkoutID: activeWorkoutID, ScheduledDate: "2026-08-24"})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if entry.ID != entryID {
		t.Fatalf("expected generated id, got %s", entry.ID)
	}
	if store.created.WorkoutID != activeWorkoutID || store.created.ScheduledDate != "2026-08-24" {
		t.Fatalf("expected create input to be preserved, got %#v", store.created)
	}
}

func TestServiceRejectsInvalidInputs(t *testing.T) {
	service := NewService(&fakeScheduleStore{})

	tests := []struct {
		name  string
		input CreateInput
		err   error
	}{
		{name: "invalid uuid", input: CreateInput{WorkoutID: "not-a-uuid", ScheduledDate: "2026-08-24"}, err: ErrInvalidWorkout},
		{name: "missing date", input: CreateInput{WorkoutID: activeWorkoutID}, err: ErrInvalidRequest},
		{name: "invalid date", input: CreateInput{WorkoutID: activeWorkoutID, ScheduledDate: "24/08/2026"}, err: ErrInvalidRequest},
		{name: "impossible date", input: CreateInput{WorkoutID: activeWorkoutID, ScheduledDate: "2026-02-30"}, err: ErrInvalidRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected %v, got %v", tt.err, err)
			}
		})
	}
}

func TestServiceStoreErrorMappingAndSameDateRules(t *testing.T) {
	store := &fakeScheduleStore{createErr: ErrInvalidWorkout}
	service := NewService(store)
	service.newUUID = func() (string, error) { return entryID, nil }

	_, err := service.Create(context.Background(), CreateInput{WorkoutID: activeWorkoutID, ScheduledDate: "2026-08-24"})
	if !errors.Is(err, ErrInvalidWorkout) {
		t.Fatalf("expected missing/inactive workout error, got %v", err)
	}

	store.createErr = ErrDuplicate
	_, err = service.Create(context.Background(), CreateInput{WorkoutID: activeWorkoutID, ScheduledDate: "2026-08-24"})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected duplicate workout/date, got %v", err)
	}

	store.createErr = nil
	if _, err := service.Create(context.Background(), CreateInput{WorkoutID: otherWorkoutID, ScheduledDate: "2026-08-24"}); err != nil {
		t.Fatalf("expected different workout on same date to be allowed, got %v", err)
	}
}

func TestServiceUpdateDeleteNotFoundAndInactiveWorkoutRules(t *testing.T) {
	store := &fakeScheduleStore{}
	service := NewService(store)

	updated, err := service.Update(context.Background(), entryID, UpdateInput{WorkoutID: otherWorkoutID, ScheduledDate: "2026-08-25"})
	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if updated.ScheduledDate != "2026-08-25" || store.updateWorkoutID != otherWorkoutID {
		t.Fatalf("expected updated date/workout, got %#v", updated)
	}

	store.updateErr = ErrInvalidWorkout
	_, err = service.Update(context.Background(), entryID, UpdateInput{WorkoutID: inactiveWorkoutID, ScheduledDate: "2026-08-25"})
	if !errors.Is(err, ErrInvalidWorkout) {
		t.Fatalf("expected inactive changed workout rejection, got %v", err)
	}
	store.updateErr = nil

	store.updateErr = ErrCompleted
	_, err = service.Update(context.Background(), entryID, UpdateInput{WorkoutID: activeWorkoutID, ScheduledDate: "2026-08-24"})
	if !errors.Is(err, ErrCompleted) {
		t.Fatalf("expected completed update conflict, got %v", err)
	}
	store.updateErr = nil

	if err := service.Delete(context.Background(), entryID); err != nil {
		t.Fatalf("expected delete success, got %v", err)
	}
	if store.deletedID != entryID {
		t.Fatalf("expected deleted id %s, got %s", entryID, store.deletedID)
	}

	store.deleteErr = ErrCompleted
	if err := service.Delete(context.Background(), entryID); !errors.Is(err, ErrCompleted) {
		t.Fatalf("expected completed delete conflict, got %v", err)
	}
	store.deleteErr = nil

	_, err = service.Update(context.Background(), missingEntryID, UpdateInput{WorkoutID: activeWorkoutID, ScheduledDate: "2026-08-24"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing update not found, got %v", err)
	}
	if err := service.Delete(context.Background(), missingEntryID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing delete not found, got %v", err)
	}
}

func TestServiceRangeValidation(t *testing.T) {
	service := NewService(&fakeScheduleStore{})

	if _, err := service.List(context.Background(), "2026-08-01", "2026-08-31"); err != nil {
		t.Fatalf("expected valid range, got %v", err)
	}

	tests := []struct {
		name string
		from string
		to   string
	}{
		{name: "missing from", from: "", to: "2026-08-31"},
		{name: "invalid from", from: "2026/08/01", to: "2026-08-31"},
		{name: "from after to", from: "2026-09-01", to: "2026-08-31"},
		{name: "range too large", from: "2026-01-01", to: "2026-04-04"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.List(context.Background(), tt.from, tt.to)
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("expected invalid request, got %v", err)
			}
		})
	}
}

type fakeScheduleStore struct {
	created         NewEntry
	updateWorkoutID string
	deletedID       string
	createErr       error
	updateErr       error
	deleteErr       error
}

func (s *fakeScheduleStore) ListEntries(context.Context, string, string) ([]Entry, error) {
	return nil, nil
}

func (s *fakeScheduleStore) CreateEntry(_ context.Context, entry NewEntry) (Entry, error) {
	s.created = entry
	if s.createErr != nil {
		return Entry{}, s.createErr
	}
	return entryFixture(entry.ID, entry.WorkoutID, entry.ScheduledDate, true), nil
}

func (s *fakeScheduleStore) UpdateEntry(_ context.Context, id string, workoutID string, scheduledDate string) (Entry, error) {
	if id == missingEntryID {
		return Entry{}, ErrNotFound
	}
	if s.updateErr != nil {
		return Entry{}, s.updateErr
	}
	s.updateWorkoutID = workoutID
	return entryFixture(id, workoutID, scheduledDate, workoutID != inactiveWorkoutID), nil
}

func (s *fakeScheduleStore) DeleteEntry(_ context.Context, id string) error {
	if id == missingEntryID {
		return ErrNotFound
	}
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedID = id
	return nil
}

func entryFixture(id string, workoutID string, scheduledDate string, workoutActive bool) Entry {
	now := time.Now().UTC()
	return Entry{
		ID:            id,
		ScheduledDate: scheduledDate,
		Workout:       WorkoutRef{ID: workoutID, Name: "Treino", Active: workoutActive},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
