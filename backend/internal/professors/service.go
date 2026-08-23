package professors

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
)

type Service struct {
	store        Store
	hashPassword func(string) (string, error)
	newUUID      func() (string, error)
}

func NewService(store Store) *Service {
	return &Service{
		store:        store,
		hashPassword: auth.HashPassword,
		newUUID:      auth.NewUUID,
	}
}

func (s *Service) List(ctx context.Context) ([]Professor, error) {
	return s.store.ListProfessors(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Professor, error) {
	name, err := validateName(input.Name)
	if err != nil {
		return Professor{}, err
	}
	email, err := validateEmail(input.Email)
	if err != nil {
		return Professor{}, err
	}
	if err := auth.ValidatePassword(input.Password); err != nil {
		return Professor{}, ErrInvalidRequest
	}

	passwordHash, err := s.hashPassword(input.Password)
	if err != nil {
		return Professor{}, err
	}
	id, err := s.newUUID()
	if err != nil {
		return Professor{}, err
	}

	professor, err := s.store.CreateProfessor(ctx, NewProfessor{
		ID:           id,
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         auth.RoleProfessor,
	})
	if errors.Is(err, ErrEmailExists) {
		return Professor{}, ErrEmailExists
	}
	if err != nil {
		return Professor{}, err
	}

	return professor, nil
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Professor, error) {
	if !validID(id) {
		return Professor{}, ErrNotFound
	}
	name, err := validateName(input.Name)
	if err != nil {
		return Professor{}, err
	}
	email, err := validateEmail(input.Email)
	if err != nil {
		return Professor{}, err
	}

	professor, err := s.store.UpdateProfessor(ctx, id, name, email)
	if errors.Is(err, ErrEmailExists) {
		return Professor{}, ErrEmailExists
	}
	if errors.Is(err, ErrNotFound) {
		return Professor{}, ErrNotFound
	}
	if err != nil {
		return Professor{}, err
	}

	return professor, nil
}

func (s *Service) SetStatus(ctx context.Context, id string, input StatusInput) (Professor, error) {
	if !validID(id) {
		return Professor{}, ErrNotFound
	}
	if input.Active == nil {
		return Professor{}, ErrInvalidRequest
	}

	professor, err := s.store.SetProfessorStatus(ctx, id, *input.Active)
	if errors.Is(err, ErrNotFound) {
		return Professor{}, ErrNotFound
	}
	if err != nil {
		return Professor{}, err
	}

	return professor, nil
}

func (s *Service) SetPassword(ctx context.Context, id string, input PasswordInput) error {
	if !validID(id) {
		return ErrNotFound
	}
	if err := auth.ValidatePassword(input.Password); err != nil {
		return ErrInvalidRequest
	}

	passwordHash, err := s.hashPassword(input.Password)
	if err != nil {
		return err
	}

	err = s.store.SetProfessorPassword(ctx, id, passwordHash)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}

	err := s.store.DeleteProfessor(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
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

func validateEmail(email string) (string, error) {
	normalized, err := auth.NormalizeEmail(email)
	if err != nil {
		return "", ErrInvalidRequest
	}
	if len([]byte(normalized)) > MaxEmailBytes {
		return "", ErrInvalidRequest
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
