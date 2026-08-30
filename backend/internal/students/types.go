package students

import (
	"context"
	"errors"
	"time"
)

const MaxNameRunes = 120
const MaxPhoneRunes = 30
const MaxNotesRunes = 2000

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrNotFound       = errors.New("student not found")
)

type Student struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Active        bool      `json:"active"`
	BirthDate     *string   `json:"birthDate"`
	Phone         *string   `json:"phone"`
	GuardianName  *string   `json:"guardianName"`
	GuardianPhone *string   `json:"guardianPhone"`
	Notes         *string   `json:"notes"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Name          string  `json:"name"`
	BirthDate     *string `json:"birthDate"`
	Phone         *string `json:"phone"`
	GuardianName  *string `json:"guardianName"`
	GuardianPhone *string `json:"guardianPhone"`
	Notes         *string `json:"notes"`
}

type UpdateInput struct {
	Name          string  `json:"name"`
	BirthDate     *string `json:"birthDate"`
	Phone         *string `json:"phone"`
	GuardianName  *string `json:"guardianName"`
	GuardianPhone *string `json:"guardianPhone"`
	Notes         *string `json:"notes"`
}

type StatusInput struct {
	Active *bool `json:"active"`
}

type NewStudent struct {
	ID            string
	Name          string
	BirthDate     *string
	Phone         *string
	GuardianName  *string
	GuardianPhone *string
	Notes         *string
}

type Store interface {
	ListStudents(ctx context.Context) ([]Student, error)
	CreateStudent(ctx context.Context, student NewStudent) (Student, error)
	UpdateStudent(ctx context.Context, id string, student NewStudent) (Student, error)
	SetStudentStatus(ctx context.Context, id string, active bool) (Student, error)
}

type ServiceAPI interface {
	List(ctx context.Context) ([]Student, error)
	Create(ctx context.Context, input CreateInput) (Student, error)
	Update(ctx context.Context, id string, input UpdateInput) (Student, error)
	SetStatus(ctx context.Context, id string, input StatusInput) (Student, error)
}
