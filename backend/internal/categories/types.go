package categories

import (
	"context"
	"errors"
	"time"
)

const MaxNameRunes = 120

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrNameExists     = errors.New("category name already exists")
	ErrNotFound       = errors.New("category not found")
)

type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Name string `json:"name"`
}

type UpdateInput struct {
	Name string `json:"name"`
}

type StatusInput struct {
	Active *bool `json:"active"`
}

type NewCategory struct {
	ID   string
	Name string
}

type Store interface {
	ListCategories(ctx context.Context) ([]Category, error)
	CreateCategory(ctx context.Context, category NewCategory) (Category, error)
	UpdateCategory(ctx context.Context, id string, name string) (Category, error)
	SetCategoryStatus(ctx context.Context, id string, active bool) (Category, error)
	DeleteCategory(ctx context.Context, id string) error
}

type ServiceAPI interface {
	List(ctx context.Context) ([]Category, error)
	Create(ctx context.Context, input CreateInput) (Category, error)
	Update(ctx context.Context, id string, input UpdateInput) (Category, error)
	SetStatus(ctx context.Context, id string, input StatusInput) (Category, error)
	Delete(ctx context.Context, id string) error
}
