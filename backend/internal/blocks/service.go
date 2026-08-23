package blocks

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

func (s *Service) List(ctx context.Context) ([]Block, error) {
	return s.store.ListBlocks(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Block, error) {
	name, categoryID, err := validateBlockInput(input.Name, input.CategoryID)
	if err != nil {
		return Block{}, err
	}
	id, err := s.newUUID()
	if err != nil {
		return Block{}, err
	}

	block, err := s.store.CreateBlock(ctx, NewBlock{ID: id, Name: name, CategoryID: categoryID})
	return mapBlockResult(block, err)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Block, error) {
	if !validID(id) {
		return Block{}, ErrNotFound
	}
	name, categoryID, err := validateBlockInput(input.Name, input.CategoryID)
	if err != nil {
		return Block{}, err
	}

	block, err := s.store.UpdateBlock(ctx, id, name, categoryID)
	return mapBlockResult(block, err)
}

func (s *Service) SetStatus(ctx context.Context, id string, input StatusInput) (Block, error) {
	if !validID(id) {
		return Block{}, ErrNotFound
	}
	if input.Active == nil {
		return Block{}, ErrInvalidRequest
	}

	block, err := s.store.SetBlockStatus(ctx, id, *input.Active)
	if errors.Is(err, ErrNotFound) {
		return Block{}, ErrNotFound
	}
	if err != nil {
		return Block{}, err
	}

	return block, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}

	err := s.store.DeleteBlock(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, ErrInUse) {
		return ErrInUse
	}
	return err
}

func mapBlockResult(block Block, err error) (Block, error) {
	if errors.Is(err, ErrNameExists) {
		return Block{}, ErrNameExists
	}
	if errors.Is(err, ErrInvalidCategory) {
		return Block{}, ErrInvalidCategory
	}
	if errors.Is(err, ErrNotFound) {
		return Block{}, ErrNotFound
	}
	if err != nil {
		return Block{}, err
	}

	return block, nil
}

func validateBlockInput(name string, categoryID string) (string, string, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return "", "", ErrInvalidRequest
	}
	if utf8.RuneCountInString(trimmedName) > MaxNameRunes {
		return "", "", ErrInvalidRequest
	}
	if !validID(categoryID) {
		return "", "", ErrInvalidCategory
	}

	return trimmedName, categoryID, nil
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
