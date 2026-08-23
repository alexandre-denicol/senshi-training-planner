package professors

import (
	"context"
	"errors"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
)

const (
	MaxNameRunes  = 120
	MaxEmailBytes = 254
)

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrEmailExists    = errors.New("email already exists")
	ErrNotFound       = errors.New("professor not found")
)

type Professor struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type StatusInput struct {
	Active *bool `json:"active"`
}

type PasswordInput struct {
	Password string `json:"password"`
}

type NewProfessor struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         auth.Role
}

type Store interface {
	ListProfessors(ctx context.Context) ([]Professor, error)
	CreateProfessor(ctx context.Context, professor NewProfessor) (Professor, error)
	UpdateProfessor(ctx context.Context, id string, name string, email string) (Professor, error)
	SetProfessorStatus(ctx context.Context, id string, active bool) (Professor, error)
	SetProfessorPassword(ctx context.Context, id string, passwordHash string) error
	DeleteProfessor(ctx context.Context, id string) error
}

type ServiceAPI interface {
	List(ctx context.Context) ([]Professor, error)
	Create(ctx context.Context, input CreateInput) (Professor, error)
	Update(ctx context.Context, id string, input UpdateInput) (Professor, error)
	SetStatus(ctx context.Context, id string, input StatusInput) (Professor, error)
	SetPassword(ctx context.Context, id string, input PasswordInput) error
	Delete(ctx context.Context, id string) error
}
