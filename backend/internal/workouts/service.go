package workouts

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

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

func (s *Service) List(ctx context.Context) ([]WorkoutListItem, error) {
	return s.store.ListWorkouts(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (WorkoutDetail, error) {
	if !validID(id) {
		return WorkoutDetail{}, ErrNotFound
	}

	workout, err := s.store.GetWorkout(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return WorkoutDetail{}, ErrNotFound
	}
	if err != nil {
		return WorkoutDetail{}, err
	}

	return workout, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (WorkoutDetail, error) {
	name, blockIDs, err := validateWorkoutInput(input.Name, input.BlockIDs)
	if err != nil {
		return WorkoutDetail{}, err
	}
	id, err := s.newUUID()
	if err != nil {
		return WorkoutDetail{}, err
	}

	workout, err := s.store.CreateWorkout(ctx, NewWorkout{ID: id, Name: name, BlockIDs: blockIDs})
	return mapWorkoutDetailResult(workout, err)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (WorkoutDetail, error) {
	if !validID(id) {
		return WorkoutDetail{}, ErrNotFound
	}
	name, blockIDs, err := validateWorkoutInput(input.Name, input.BlockIDs)
	if err != nil {
		return WorkoutDetail{}, err
	}

	workout, err := s.store.UpdateWorkout(ctx, id, name, blockIDs)
	return mapWorkoutDetailResult(workout, err)
}

func (s *Service) SetStatus(ctx context.Context, id string, input StatusInput) (WorkoutListItem, error) {
	if !validID(id) {
		return WorkoutListItem{}, ErrNotFound
	}
	if input.Active == nil {
		return WorkoutListItem{}, ErrInvalidRequest
	}

	workout, err := s.store.SetWorkoutStatus(ctx, id, *input.Active)
	if errors.Is(err, ErrNotFound) {
		return WorkoutListItem{}, ErrNotFound
	}
	if err != nil {
		return WorkoutListItem{}, err
	}

	return workout, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}

	err := s.store.DeleteWorkout(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, ErrInUse) {
		return ErrInUse
	}
	return err
}

func mapWorkoutDetailResult(workout WorkoutDetail, err error) (WorkoutDetail, error) {
	if errors.Is(err, ErrNameExists) {
		return WorkoutDetail{}, ErrNameExists
	}
	if errors.Is(err, ErrInvalidBlocks) {
		return WorkoutDetail{}, ErrInvalidBlocks
	}
	if errors.Is(err, ErrNotFound) {
		return WorkoutDetail{}, ErrNotFound
	}
	if err != nil {
		return WorkoutDetail{}, err
	}

	return workout, nil
}

func validateWorkoutInput(name string, blockIDs []string) (string, []string, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return "", nil, ErrInvalidRequest
	}
	if utf8.RuneCountInString(trimmedName) > MaxNameRunes {
		return "", nil, ErrInvalidRequest
	}
	if len(blockIDs) == 0 {
		return "", nil, ErrInvalidBlocks
	}

	seen := map[string]struct{}{}
	cleanIDs := make([]string, 0, len(blockIDs))
	for _, id := range blockIDs {
		if !validID(id) {
			return "", nil, ErrInvalidBlocks
		}
		if _, ok := seen[id]; ok {
			return "", nil, ErrInvalidBlocks
		}
		seen[id] = struct{}{}
		cleanIDs = append(cleanIDs, id)
	}

	return trimmedName, cleanIDs, nil
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
