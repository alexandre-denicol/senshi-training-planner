package students

import (
	"context"
	"errors"
	"strings"
	"time"
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

func (s *Service) List(ctx context.Context) ([]Student, error) {
	return s.store.ListStudents(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Student, error) {
	clean, err := validateStudentInput(input.Name, input.BirthDate, input.Phone, input.GuardianName, input.GuardianPhone, input.Notes)
	if err != nil {
		return Student{}, err
	}
	id, err := s.newUUID()
	if err != nil {
		return Student{}, err
	}

	student, err := s.store.CreateStudent(ctx, NewStudent{
		ID:            id,
		Name:          clean.Name,
		BirthDate:     clean.BirthDate,
		Phone:         clean.Phone,
		GuardianName:  clean.GuardianName,
		GuardianPhone: clean.GuardianPhone,
		Notes:         clean.Notes,
	})
	if err != nil {
		return Student{}, err
	}

	return student, nil
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Student, error) {
	if !validID(id) {
		return Student{}, ErrNotFound
	}
	clean, err := validateStudentInput(input.Name, input.BirthDate, input.Phone, input.GuardianName, input.GuardianPhone, input.Notes)
	if err != nil {
		return Student{}, err
	}

	student, err := s.store.UpdateStudent(ctx, id, NewStudent{
		ID:            id,
		Name:          clean.Name,
		BirthDate:     clean.BirthDate,
		Phone:         clean.Phone,
		GuardianName:  clean.GuardianName,
		GuardianPhone: clean.GuardianPhone,
		Notes:         clean.Notes,
	})
	if errors.Is(err, ErrNotFound) {
		return Student{}, ErrNotFound
	}
	if err != nil {
		return Student{}, err
	}

	return student, nil
}

func (s *Service) SetStatus(ctx context.Context, id string, input StatusInput) (Student, error) {
	if !validID(id) {
		return Student{}, ErrNotFound
	}
	if input.Active == nil {
		return Student{}, ErrInvalidRequest
	}

	student, err := s.store.SetStudentStatus(ctx, id, *input.Active)
	if errors.Is(err, ErrNotFound) {
		return Student{}, ErrNotFound
	}
	if err != nil {
		return Student{}, err
	}

	return student, nil
}

type validatedStudentInput struct {
	Name          string
	BirthDate     *string
	Phone         *string
	GuardianName  *string
	GuardianPhone *string
	Notes         *string
}

func validateStudentInput(name string, birthDate *string, phone *string, guardianName *string, guardianPhone *string, notes *string) (validatedStudentInput, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return validatedStudentInput{}, ErrInvalidRequest
	}
	if utf8.RuneCountInString(trimmedName) > MaxNameRunes {
		return validatedStudentInput{}, ErrInvalidRequest
	}

	normalizedBirthDate, err := normalizeOptionalDate(birthDate)
	if err != nil {
		return validatedStudentInput{}, err
	}

	normalizedPhone := normalizeOptionalText(phone)
	if normalizedPhone != nil && utf8.RuneCountInString(*normalizedPhone) > MaxPhoneRunes {
		return validatedStudentInput{}, ErrInvalidRequest
	}

	normalizedGuardianName := normalizeOptionalText(guardianName)
	if normalizedGuardianName != nil && utf8.RuneCountInString(*normalizedGuardianName) > MaxNameRunes {
		return validatedStudentInput{}, ErrInvalidRequest
	}

	normalizedGuardianPhone := normalizeOptionalText(guardianPhone)
	if normalizedGuardianPhone != nil && utf8.RuneCountInString(*normalizedGuardianPhone) > MaxPhoneRunes {
		return validatedStudentInput{}, ErrInvalidRequest
	}

	normalizedNotes := normalizeOptionalText(notes)
	if normalizedNotes != nil && utf8.RuneCountInString(*normalizedNotes) > MaxNotesRunes {
		return validatedStudentInput{}, ErrInvalidRequest
	}

	return validatedStudentInput{
		Name:          trimmedName,
		BirthDate:     normalizedBirthDate,
		Phone:         normalizedPhone,
		GuardianName:  normalizedGuardianName,
		GuardianPhone: normalizedGuardianPhone,
		Notes:         normalizedNotes,
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

func normalizeOptionalDate(value *string) (*string, error) {
	trimmed := normalizeOptionalText(value)
	if trimmed == nil {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", *trimmed)
	if err != nil || parsed.Format("2006-01-02") != *trimmed {
		return nil, ErrInvalidRequest
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
