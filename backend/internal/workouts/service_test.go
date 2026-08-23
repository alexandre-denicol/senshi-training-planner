package workouts

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	workoutID       = "11111111-1111-1111-1111-111111111111"
	missingID       = "22222222-2222-2222-2222-222222222222"
	blockIDOne      = "33333333-3333-3333-3333-333333333333"
	blockIDTwo      = "44444444-4444-4444-4444-444444444444"
	inactiveBlockID = "55555555-5555-5555-5555-555555555555"
)

func TestServiceCreateWorkout(t *testing.T) {
	store := &fakeWorkoutStore{}
	service := NewService(store)
	service.newUUID = func() (string, error) { return workoutID, nil }

	workout, err := service.Create(context.Background(), CreateInput{Name: "  Treino Base  ", BlockIDs: []string{blockIDOne, blockIDTwo}})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if workout.ID != workoutID {
		t.Fatalf("expected generated id, got %s", workout.ID)
	}
	if store.created.Name != "Treino Base" {
		t.Fatalf("expected trimmed name, got %q", store.created.Name)
	}
	if got := strings.Join(store.created.BlockIDs, ","); got != blockIDOne+","+blockIDTwo {
		t.Fatalf("expected order to be preserved, got %s", got)
	}
}

func TestServiceRejectsInvalidWorkoutInput(t *testing.T) {
	service := NewService(&fakeWorkoutStore{})

	tests := []struct {
		name  string
		input CreateInput
		err   error
	}{
		{name: "empty name", input: CreateInput{Name: "", BlockIDs: []string{blockIDOne}}, err: ErrInvalidRequest},
		{name: "whitespace name", input: CreateInput{Name: "   \n\t", BlockIDs: []string{blockIDOne}}, err: ErrInvalidRequest},
		{name: "too long name", input: CreateInput{Name: strings.Repeat("á", MaxNameRunes+1), BlockIDs: []string{blockIDOne}}, err: ErrInvalidRequest},
		{name: "empty blocks", input: CreateInput{Name: "Treino", BlockIDs: nil}, err: ErrInvalidBlocks},
		{name: "invalid block uuid", input: CreateInput{Name: "Treino", BlockIDs: []string{"not-a-uuid"}}, err: ErrInvalidBlocks},
		{name: "duplicate block", input: CreateInput{Name: "Treino", BlockIDs: []string{blockIDOne, blockIDOne}}, err: ErrInvalidBlocks},
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

func TestServiceMapsStoreErrors(t *testing.T) {
	store := &fakeWorkoutStore{createErr: ErrInvalidBlocks}
	service := NewService(store)
	service.newUUID = func() (string, error) { return workoutID, nil }

	_, err := service.Create(context.Background(), CreateInput{Name: "Treino", BlockIDs: []string{blockIDOne}})
	if !errors.Is(err, ErrInvalidBlocks) {
		t.Fatalf("expected invalid blocks for nonexistent/inactive block, got %v", err)
	}

	store.createErr = ErrNameExists
	_, err = service.Create(context.Background(), CreateInput{Name: "Treino", BlockIDs: []string{blockIDOne}})
	if !errors.Is(err, ErrNameExists) {
		t.Fatalf("expected duplicate name, got %v", err)
	}
}

func TestServiceUpdateReorderStatusDeleteAndNotFound(t *testing.T) {
	store := &fakeWorkoutStore{}
	service := NewService(store)

	updated, err := service.Update(context.Background(), workoutID, UpdateInput{Name: "  Treino Editado  ", BlockIDs: []string{blockIDTwo, blockIDOne}})
	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if updated.Name != "Treino Editado" {
		t.Fatalf("expected trimmed update name, got %q", updated.Name)
	}
	if got := strings.Join(store.updatedBlockIDs, ","); got != blockIDTwo+","+blockIDOne {
		t.Fatalf("expected reordered block ids, got %s", got)
	}

	store.updateErr = ErrInvalidBlocks
	_, err = service.Update(context.Background(), workoutID, UpdateInput{Name: "Treino", BlockIDs: []string{inactiveBlockID}})
	if !errors.Is(err, ErrInvalidBlocks) {
		t.Fatalf("expected inactive block rejection, got %v", err)
	}
	store.updateErr = nil

	active := false
	inactive, err := service.SetStatus(context.Background(), workoutID, StatusInput{Active: &active})
	if err != nil {
		t.Fatalf("expected deactivate success, got %v", err)
	}
	if inactive.Active {
		t.Fatal("expected inactive workout")
	}

	active = true
	reactivated, err := service.SetStatus(context.Background(), workoutID, StatusInput{Active: &active})
	if err != nil {
		t.Fatalf("expected reactivate success, got %v", err)
	}
	if !reactivated.Active {
		t.Fatal("expected active workout")
	}

	if err := service.Delete(context.Background(), workoutID); err != nil {
		t.Fatalf("expected delete success, got %v", err)
	}
	if store.deletedID != workoutID {
		t.Fatalf("expected deleted id %s, got %s", workoutID, store.deletedID)
	}

	_, err = service.Get(context.Background(), missingID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing get not found, got %v", err)
	}
	_, err = service.Update(context.Background(), missingID, UpdateInput{Name: "Treino", BlockIDs: []string{blockIDOne}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing update not found, got %v", err)
	}
	if err := service.Delete(context.Background(), missingID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing delete not found, got %v", err)
	}
	_, err = service.SetStatus(context.Background(), workoutID, StatusInput{})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected missing active invalid request, got %v", err)
	}
}

type fakeWorkoutStore struct {
	created         NewWorkout
	updatedName     string
	updatedBlockIDs []string
	deletedID       string
	createErr       error
	updateErr       error
}

func (s *fakeWorkoutStore) ListWorkouts(context.Context) ([]WorkoutListItem, error) {
	return nil, nil
}

func (s *fakeWorkoutStore) GetWorkout(_ context.Context, id string) (WorkoutDetail, error) {
	if id == missingID {
		return WorkoutDetail{}, ErrNotFound
	}
	return workoutFixture(id, "Treino", []string{blockIDOne, blockIDTwo}), nil
}

func (s *fakeWorkoutStore) CreateWorkout(_ context.Context, workout NewWorkout) (WorkoutDetail, error) {
	s.created = workout
	if s.createErr != nil {
		return WorkoutDetail{}, s.createErr
	}
	return workoutFixture(workout.ID, workout.Name, workout.BlockIDs), nil
}

func (s *fakeWorkoutStore) UpdateWorkout(_ context.Context, id string, name string, blockIDs []string) (WorkoutDetail, error) {
	if id == missingID {
		return WorkoutDetail{}, ErrNotFound
	}
	if s.updateErr != nil {
		return WorkoutDetail{}, s.updateErr
	}
	s.updatedName = name
	s.updatedBlockIDs = blockIDs
	return workoutFixture(id, name, blockIDs), nil
}

func (s *fakeWorkoutStore) SetWorkoutStatus(_ context.Context, id string, active bool) (WorkoutListItem, error) {
	if id == missingID {
		return WorkoutListItem{}, ErrNotFound
	}
	now := time.Now().UTC()
	return WorkoutListItem{ID: id, Name: "Treino", Active: active, BlockCount: 2, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *fakeWorkoutStore) DeleteWorkout(_ context.Context, id string) error {
	if id == missingID {
		return ErrNotFound
	}
	s.deletedID = id
	return nil
}

func workoutFixture(id string, name string, blockIDs []string) WorkoutDetail {
	now := time.Now().UTC()
	blocks := make([]WorkoutBlock, 0, len(blockIDs))
	for index, blockID := range blockIDs {
		blocks = append(blocks, WorkoutBlock{
			ID:       blockID,
			Name:     "Bloco",
			Active:   true,
			Position: index + 1,
			Category: CategoryRef{ID: "66666666-6666-6666-6666-666666666666", Name: "Categoria"},
		})
	}
	return WorkoutDetail{ID: id, Name: name, Active: true, Blocks: blocks, CreatedAt: now, UpdatedAt: now}
}
