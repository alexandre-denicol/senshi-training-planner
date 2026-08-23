package professors

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
)

const (
	professorID = "11111111-1111-1111-1111-111111111111"
	adminID     = "22222222-2222-2222-2222-222222222222"
)

func TestServiceCreateProfessor(t *testing.T) {
	store := &fakeProfessorStore{}
	service := NewService(store)
	service.hashPassword = func(password string) (string, error) {
		return "hash:" + password, nil
	}
	service.newUUID = func() (string, error) {
		return professorID, nil
	}

	professor, err := service.Create(context.Background(), CreateInput{
		Name:     "  Professor Um  ",
		Email:    "  PROFESSOR@example.COM  ",
		Password: "uma senha longa e segura",
	})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if professor.ID != professorID {
		t.Fatalf("expected generated id, got %s", professor.ID)
	}
	if store.created.Name != "Professor Um" {
		t.Fatalf("expected trimmed name, got %q", store.created.Name)
	}
	if store.created.Email != "professor@example.com" {
		t.Fatalf("expected normalized email, got %q", store.created.Email)
	}
	if store.created.Role != auth.RoleProfessor {
		t.Fatalf("expected PROFESSOR role, got %s", store.created.Role)
	}
	if store.created.PasswordHash != "hash:uma senha longa e segura" {
		t.Fatal("expected password to be hashed before storage")
	}
}

func TestServiceRejectsDuplicateNormalizedEmail(t *testing.T) {
	store := &fakeProfessorStore{createErr: ErrEmailExists}
	service := NewService(store)
	service.hashPassword = func(string) (string, error) { return "hash", nil }
	service.newUUID = func() (string, error) { return professorID, nil }

	_, err := service.Create(context.Background(), CreateInput{
		Name:     "Professor",
		Email:    "PROFESSOR@example.com",
		Password: "uma senha longa e segura",
	})
	if !errors.Is(err, ErrEmailExists) {
		t.Fatalf("expected duplicate email error, got %v", err)
	}
	if store.created.Email != "professor@example.com" {
		t.Fatalf("expected duplicate check to receive normalized email, got %q", store.created.Email)
	}
}

func TestServiceUpdateProfessor(t *testing.T) {
	store := &fakeProfessorStore{}
	service := NewService(store)

	professor, err := service.Update(context.Background(), professorID, UpdateInput{
		Name:  "  Novo Nome ",
		Email: " NOVO@example.com ",
	})
	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if professor.Name != "Novo Nome" || professor.Email != "novo@example.com" {
		t.Fatalf("expected normalized update, got %#v", professor)
	}
}

func TestServiceCannotModifyAdminThroughProfessorOperations(t *testing.T) {
	store := &fakeProfessorStore{adminID: adminID}
	service := NewService(store)

	_, err := service.Update(context.Background(), adminID, UpdateInput{
		Name:  "Admin",
		Email: "admin@example.com",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected admin update to be hidden as not found, got %v", err)
	}

	active := false
	if _, err := service.SetStatus(context.Background(), adminID, StatusInput{Active: &active}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected admin status change to be hidden as not found, got %v", err)
	}

	if err := service.SetPassword(context.Background(), adminID, PasswordInput{Password: "uma senha longa e segura"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected admin password reset to be hidden as not found, got %v", err)
	}

	if err := service.Delete(context.Background(), adminID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected admin delete to be hidden as not found, got %v", err)
	}
}

func TestServiceStatusInvalidatesSessionsWhenDeactivating(t *testing.T) {
	store := &fakeProfessorStore{}
	service := NewService(store)
	active := false

	professor, err := service.SetStatus(context.Background(), professorID, StatusInput{Active: &active})
	if err != nil {
		t.Fatalf("expected deactivate success, got %v", err)
	}
	if professor.Active {
		t.Fatal("expected professor to be inactive")
	}
	if store.invalidatedSessionsFor != professorID {
		t.Fatal("expected deactivation to invalidate professor sessions")
	}
}

func TestServiceStatusReactivatesProfessor(t *testing.T) {
	store := &fakeProfessorStore{}
	service := NewService(store)
	active := true

	professor, err := service.SetStatus(context.Background(), professorID, StatusInput{Active: &active})
	if err != nil {
		t.Fatalf("expected reactivate success, got %v", err)
	}
	if !professor.Active {
		t.Fatal("expected professor to be active")
	}
	if store.invalidatedSessionsFor != "" {
		t.Fatal("expected reactivation not to invalidate sessions")
	}
}

func TestServiceRejectsMissingStatus(t *testing.T) {
	store := &fakeProfessorStore{}
	service := NewService(store)

	_, err := service.SetStatus(context.Background(), professorID, StatusInput{})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected missing active field to be invalid, got %v", err)
	}
}

func TestServicePasswordResetUsesPolicyAndInvalidatesSessions(t *testing.T) {
	store := &fakeProfessorStore{}
	service := NewService(store)

	err := service.SetPassword(context.Background(), professorID, PasswordInput{Password: "curta"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid password request, got %v", err)
	}
	if store.passwordHash != "" {
		t.Fatal("expected invalid password not to be stored")
	}

	err = service.SetPassword(context.Background(), professorID, PasswordInput{Password: "uma senha longa e segura"})
	if err != nil {
		t.Fatalf("expected password reset success, got %v", err)
	}
	if store.invalidatedSessionsFor != professorID {
		t.Fatal("expected password reset to invalidate professor sessions")
	}
	ok, err := auth.VerifyPassword("uma senha longa e segura", store.passwordHash)
	if err != nil || !ok {
		t.Fatal("expected stored password hash to verify with existing Argon2id implementation")
	}
}

func TestServiceDeleteProfessor(t *testing.T) {
	store := &fakeProfessorStore{}
	service := NewService(store)

	if err := service.Delete(context.Background(), professorID); err != nil {
		t.Fatalf("expected delete success, got %v", err)
	}
	if store.deletedID != professorID {
		t.Fatalf("expected deleted id %s, got %s", professorID, store.deletedID)
	}
}

func TestServiceValidationBoundaries(t *testing.T) {
	store := &fakeProfessorStore{}
	service := NewService(store)

	_, err := service.Create(context.Background(), CreateInput{
		Name:     strings.Repeat("a", MaxNameRunes+1),
		Email:    "professor@example.com",
		Password: "uma senha longa e segura",
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid name, got %v", err)
	}

	_, err = service.Create(context.Background(), CreateInput{
		Name:     "Professor",
		Email:    strings.Repeat("a", MaxEmailBytes) + "@example.com",
		Password: "uma senha longa e segura",
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid email, got %v", err)
	}
}

type fakeProfessorStore struct {
	professors             []Professor
	created                NewProfessor
	createErr              error
	adminID                string
	invalidatedSessionsFor string
	passwordHash           string
	deletedID              string
}

func (s *fakeProfessorStore) ListProfessors(context.Context) ([]Professor, error) {
	return s.professors, nil
}

func (s *fakeProfessorStore) CreateProfessor(_ context.Context, professor NewProfessor) (Professor, error) {
	s.created = professor
	if s.createErr != nil {
		return Professor{}, s.createErr
	}
	return Professor{
		ID:        professor.ID,
		Name:      professor.Name,
		Email:     professor.Email,
		Active:    true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (s *fakeProfessorStore) UpdateProfessor(_ context.Context, id string, name string, email string) (Professor, error) {
	if id == s.adminID {
		return Professor{}, ErrNotFound
	}
	return Professor{ID: id, Name: name, Email: email, Active: true}, nil
}

func (s *fakeProfessorStore) SetProfessorStatus(_ context.Context, id string, active bool) (Professor, error) {
	if id == s.adminID {
		return Professor{}, ErrNotFound
	}
	if !active {
		s.invalidatedSessionsFor = id
	}
	return Professor{ID: id, Name: "Professor", Email: "professor@example.com", Active: active}, nil
}

func (s *fakeProfessorStore) SetProfessorPassword(_ context.Context, id string, passwordHash string) error {
	if id == s.adminID {
		return ErrNotFound
	}
	s.passwordHash = passwordHash
	s.invalidatedSessionsFor = id
	return nil
}

func (s *fakeProfessorStore) DeleteProfessor(_ context.Context, id string) error {
	if id == s.adminID {
		return ErrNotFound
	}
	s.deletedID = id
	return nil
}
