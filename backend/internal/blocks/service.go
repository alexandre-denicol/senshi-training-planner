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
	clean, err := validateBlockInput(input.Name, input.CategoryID, input.Description, input.Sequence)
	if err != nil {
		return Block{}, err
	}
	id, err := s.newUUID()
	if err != nil {
		return Block{}, err
	}

	block, err := s.store.CreateBlock(ctx, NewBlock{
		ID:          id,
		Name:        clean.Name,
		CategoryID:  clean.CategoryID,
		Description: clean.Description,
		Sequence:    clean.Sequence,
	})
	return mapBlockResult(block, err)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Block, error) {
	if !validID(id) {
		return Block{}, ErrNotFound
	}
	clean, err := validateBlockInput(input.Name, input.CategoryID, input.Description, input.Sequence)
	if err != nil {
		return Block{}, err
	}

	block, err := s.store.UpdateBlock(ctx, id, clean.Name, clean.CategoryID, clean.Description, clean.Sequence)
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

type validatedBlockInput struct {
	Name        string
	CategoryID  string
	Description *string
	Sequence    []NewSequenceItem
}

func validateBlockInput(name string, categoryID string, description *string, sequence []SequenceItemInput) (validatedBlockInput, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return validatedBlockInput{}, ErrInvalidRequest
	}
	if utf8.RuneCountInString(trimmedName) > MaxNameRunes {
		return validatedBlockInput{}, ErrInvalidRequest
	}
	if !validID(categoryID) {
		return validatedBlockInput{}, ErrInvalidCategory
	}

	normalizedDescription := normalizeOptionalText(description)
	if normalizedDescription != nil && utf8.RuneCountInString(*normalizedDescription) > MaxDescriptionRunes {
		return validatedBlockInput{}, ErrInvalidRequest
	}

	normalizedSequence, err := normalizeSequence(sequence)
	if err != nil {
		return validatedBlockInput{}, err
	}

	return validatedBlockInput{
		Name:        trimmedName,
		CategoryID:  categoryID,
		Description: normalizedDescription,
		Sequence:    normalizedSequence,
	}, nil
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func normalizeSequence(sequence []SequenceItemInput) ([]NewSequenceItem, error) {
	if len(sequence) > MaxSequenceItems {
		return nil, ErrInvalidRequest
	}

	normalized := make([]NewSequenceItem, 0, len(sequence))
	for index, item := range sequence {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			return nil, ErrInvalidRequest
		}
		if utf8.RuneCountInString(text) > MaxSequenceTextRunes {
			return nil, ErrInvalidRequest
		}
		normalized = append(normalized, NewSequenceItem{
			Position: index + 1,
			Text:     text,
		})
	}

	return normalized, nil
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
