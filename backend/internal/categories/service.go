package categories

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

func (s *Service) List(ctx context.Context) ([]Category, error) {
	return s.store.ListCategories(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Category, error) {
	name, err := validateName(input.Name)
	if err != nil {
		return Category{}, err
	}
	id, err := s.newUUID()
	if err != nil {
		return Category{}, err
	}

	category, err := s.store.CreateCategory(ctx, NewCategory{ID: id, Name: name})
	if errors.Is(err, ErrNameExists) {
		return Category{}, ErrNameExists
	}
	if err != nil {
		return Category{}, err
	}

	return category, nil
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Category, error) {
	if !validID(id) {
		return Category{}, ErrNotFound
	}
	name, err := validateName(input.Name)
	if err != nil {
		return Category{}, err
	}

	category, err := s.store.UpdateCategory(ctx, id, name)
	if errors.Is(err, ErrNameExists) {
		return Category{}, ErrNameExists
	}
	if errors.Is(err, ErrNotFound) {
		return Category{}, ErrNotFound
	}
	if err != nil {
		return Category{}, err
	}

	return category, nil
}

func (s *Service) SetStatus(ctx context.Context, id string, input StatusInput) (Category, error) {
	if !validID(id) {
		return Category{}, ErrNotFound
	}
	if input.Active == nil {
		return Category{}, ErrInvalidRequest
	}

	category, err := s.store.SetCategoryStatus(ctx, id, *input.Active)
	if errors.Is(err, ErrNotFound) {
		return Category{}, ErrNotFound
	}
	if err != nil {
		return Category{}, err
	}

	return category, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}

	err := s.store.DeleteCategory(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, ErrInUse) {
		return ErrInUse
	}
	return err
}

func validateName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ErrInvalidRequest
	}
	if utf8.RuneCountInString(trimmed) > MaxNameRunes {
		return "", ErrInvalidRequest
	}

	return trimmed, nil
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
