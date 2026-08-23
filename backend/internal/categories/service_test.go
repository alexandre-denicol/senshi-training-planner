package categories

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	categoryID = "11111111-1111-1111-1111-111111111111"
	missingID  = "22222222-2222-2222-2222-222222222222"
)

func TestServiceCreateCategory(t *testing.T) {
	store := &fakeCategoryStore{}
	service := NewService(store)
	service.newUUID = func() (string, error) { return categoryID, nil }

	category, err := service.Create(context.Background(), CreateInput{Name: "  Técnica  "})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if category.ID != categoryID {
		t.Fatalf("expected generated id, got %s", category.ID)
	}
	if store.created.Name != "Técnica" {
		t.Fatalf("expected trimmed name, got %q", store.created.Name)
	}
	if !category.Active {
		t.Fatal("expected new category to be active")
	}
}

func TestServiceRejectsInvalidNames(t *testing.T) {
	service := NewService(&fakeCategoryStore{})

	tests := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "whitespace", input: "   \t\n"},
		{name: "too long", input: strings.Repeat("á", MaxNameRunes+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), CreateInput{Name: tt.input})
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("expected invalid request, got %v", err)
			}
		})
	}
}

func TestServiceDuplicateCategoryHandling(t *testing.T) {
	store := &fakeCategoryStore{createErr: ErrNameExists}
	service := NewService(store)
	service.newUUID = func() (string, error) { return categoryID, nil }

	_, err := service.Create(context.Background(), CreateInput{Name: "Técnica"})
	if !errors.Is(err, ErrNameExists) {
		t.Fatalf("expected duplicate name error, got %v", err)
	}
}

func TestServiceUpdateStatusDeleteAndNotFound(t *testing.T) {
	store := &fakeCategoryStore{}
	service := NewService(store)

	updated, err := service.Update(context.Background(), categoryID, UpdateInput{Name: " Mobilidade "})
	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if updated.Name != "Mobilidade" {
		t.Fatalf("expected trimmed update name, got %q", updated.Name)
	}

	active := false
	inactive, err := service.SetStatus(context.Background(), categoryID, StatusInput{Active: &active})
	if err != nil {
		t.Fatalf("expected deactivate success, got %v", err)
	}
	if inactive.Active {
		t.Fatal("expected inactive category")
	}

	active = true
	reactivated, err := service.SetStatus(context.Background(), categoryID, StatusInput{Active: &active})
	if err != nil {
		t.Fatalf("expected reactivate success, got %v", err)
	}
	if !reactivated.Active {
		t.Fatal("expected active category")
	}

	if err := service.Delete(context.Background(), categoryID); err != nil {
		t.Fatalf("expected delete success, got %v", err)
	}
	if store.deletedID != categoryID {
		t.Fatalf("expected deleted id %s, got %s", categoryID, store.deletedID)
	}

	_, err = service.Update(context.Background(), missingID, UpdateInput{Name: "Nome"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing update not found, got %v", err)
	}
	if err := service.Delete(context.Background(), missingID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing delete not found, got %v", err)
	}
	store.deleteErr = ErrInUse
	if err := service.Delete(context.Background(), categoryID); !errors.Is(err, ErrInUse) {
		t.Fatalf("expected in-use delete error, got %v", err)
	}
	_, err = service.SetStatus(context.Background(), categoryID, StatusInput{})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected missing active invalid request, got %v", err)
	}
}

type fakeCategoryStore struct {
	created   NewCategory
	createErr error
	deleteErr error
	deletedID string
}

func (s *fakeCategoryStore) ListCategories(context.Context) ([]Category, error) {
	return nil, nil
}

func (s *fakeCategoryStore) CreateCategory(_ context.Context, category NewCategory) (Category, error) {
	s.created = category
	if s.createErr != nil {
		return Category{}, s.createErr
	}
	return Category{
		ID:        category.ID,
		Name:      category.Name,
		Active:    true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (s *fakeCategoryStore) UpdateCategory(_ context.Context, id string, name string) (Category, error) {
	if id == missingID {
		return Category{}, ErrNotFound
	}
	return Category{ID: id, Name: name, Active: true}, nil
}

func (s *fakeCategoryStore) SetCategoryStatus(_ context.Context, id string, active bool) (Category, error) {
	if id == missingID {
		return Category{}, ErrNotFound
	}
	return Category{ID: id, Name: "Categoria", Active: active}, nil
}

func (s *fakeCategoryStore) DeleteCategory(_ context.Context, id string) error {
	if id == missingID {
		return ErrNotFound
	}
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedID = id
	return nil
}
