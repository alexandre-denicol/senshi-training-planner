package students

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	studentID = "11111111-1111-1111-1111-111111111111"
	missingID = "22222222-2222-2222-2222-222222222222"
)

func TestServiceCreateStudent(t *testing.T) {
	store := &fakeStudentStore{}
	service := NewService(store)
	service.newUUID = func() (string, error) { return studentID, nil }

	student, err := service.Create(context.Background(), CreateInput{Name: "  Ana Silva  "})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if student.ID != studentID {
		t.Fatalf("expected generated id, got %s", student.ID)
	}
	if store.created.Name != "Ana Silva" {
		t.Fatalf("expected trimmed name, got %q", store.created.Name)
	}
	if !student.Active {
		t.Fatal("expected new student to be active")
	}
}

func TestServiceCreateStudentWithOptionalFields(t *testing.T) {
	store := &fakeStudentStore{}
	service := NewService(store)
	service.newUUID = func() (string, error) { return studentID, nil }

	birthDate := "2010-05-20"
	phone := "  (48) 99999-0000  "
	guardianName := "  Maria Silva  "
	guardianPhone := "  (48) 98888-0000  "
	notes := "  Alergia a poeira.  "

	student, err := service.Create(context.Background(), CreateInput{
		Name:          "Ana Silva",
		BirthDate:     &birthDate,
		Phone:         &phone,
		GuardianName:  &guardianName,
		GuardianPhone: &guardianPhone,
		Notes:         &notes,
	})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}

	if store.created.BirthDate == nil || *store.created.BirthDate != "2010-05-20" {
		t.Fatalf("expected birth date to persist, got %v", store.created.BirthDate)
	}
	if store.created.Phone == nil || *store.created.Phone != "(48) 99999-0000" {
		t.Fatalf("expected trimmed phone to persist, got %v", store.created.Phone)
	}
	if store.created.GuardianName == nil || *store.created.GuardianName != "Maria Silva" {
		t.Fatalf("expected trimmed guardian name to persist, got %v", store.created.GuardianName)
	}
	if store.created.GuardianPhone == nil || *store.created.GuardianPhone != "(48) 98888-0000" {
		t.Fatalf("expected trimmed guardian phone to persist, got %v", store.created.GuardianPhone)
	}
	if store.created.Notes == nil || *store.created.Notes != "Alergia a poeira." {
		t.Fatalf("expected trimmed notes to persist, got %v", store.created.Notes)
	}
	if student.ID != studentID {
		t.Fatalf("expected generated id, got %s", student.ID)
	}
}

func TestServiceCreateStudentNormalizesBlankOptionalFields(t *testing.T) {
	store := &fakeStudentStore{}
	service := NewService(store)
	service.newUUID = func() (string, error) { return studentID, nil }

	blank := "   "
	_, err := service.Create(context.Background(), CreateInput{
		Name:          "Ana Silva",
		Phone:         &blank,
		GuardianName:  &blank,
		GuardianPhone: &blank,
		Notes:         &blank,
	})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}

	if store.created.Phone != nil {
		t.Fatalf("expected blank phone to normalize to nil, got %v", store.created.Phone)
	}
	if store.created.GuardianName != nil {
		t.Fatalf("expected blank guardian name to normalize to nil, got %v", store.created.GuardianName)
	}
	if store.created.GuardianPhone != nil {
		t.Fatalf("expected blank guardian phone to normalize to nil, got %v", store.created.GuardianPhone)
	}
	if store.created.Notes != nil {
		t.Fatalf("expected blank notes to normalize to nil, got %v", store.created.Notes)
	}
}

func TestServiceRejectsInvalidNames(t *testing.T) {
	service := NewService(&fakeStudentStore{})

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

func TestServiceRejectsInvalidBirthDate(t *testing.T) {
	service := NewService(&fakeStudentStore{})

	tests := []struct {
		name  string
		value string
	}{
		{name: "not a date", value: "not-a-date"},
		{name: "wrong format", value: "20/05/2010"},
		{name: "invalid calendar date", value: "2010-02-30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := tt.value
			_, err := service.Create(context.Background(), CreateInput{Name: "Ana Silva", BirthDate: &value})
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("expected invalid request, got %v", err)
			}
		})
	}
}

func TestServiceRejectsOversizedOptionalFields(t *testing.T) {
	service := NewService(&fakeStudentStore{})

	tests := []struct {
		name  string
		input CreateInput
	}{
		{name: "phone too long", input: CreateInput{Name: "Ana Silva", Phone: strPtr(strings.Repeat("1", MaxPhoneRunes+1))}},
		{name: "guardian name too long", input: CreateInput{Name: "Ana Silva", GuardianName: strPtr(strings.Repeat("a", MaxNameRunes+1))}},
		{name: "guardian phone too long", input: CreateInput{Name: "Ana Silva", GuardianPhone: strPtr(strings.Repeat("1", MaxPhoneRunes+1))}},
		{name: "notes too long", input: CreateInput{Name: "Ana Silva", Notes: strPtr(strings.Repeat("a", MaxNotesRunes+1))}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), tt.input)
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("expected invalid request, got %v", err)
			}
		})
	}
}

func TestServiceUpdateAndStatusAndNotFound(t *testing.T) {
	store := &fakeStudentStore{}
	service := NewService(store)

	updated, err := service.Update(context.Background(), studentID, UpdateInput{Name: " Ana Souza "})
	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if updated.Name != "Ana Souza" {
		t.Fatalf("expected trimmed update name, got %q", updated.Name)
	}

	active := false
	inactive, err := service.SetStatus(context.Background(), studentID, StatusInput{Active: &active})
	if err != nil {
		t.Fatalf("expected deactivate success, got %v", err)
	}
	if inactive.Active {
		t.Fatal("expected inactive student")
	}

	active = true
	reactivated, err := service.SetStatus(context.Background(), studentID, StatusInput{Active: &active})
	if err != nil {
		t.Fatalf("expected reactivate success, got %v", err)
	}
	if !reactivated.Active {
		t.Fatal("expected active student")
	}

	_, err = service.Update(context.Background(), missingID, UpdateInput{Name: "Nome"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing update not found, got %v", err)
	}

	_, err = service.SetStatus(context.Background(), missingID, StatusInput{Active: &active})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing status not found, got %v", err)
	}

	_, err = service.SetStatus(context.Background(), studentID, StatusInput{})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected missing active invalid request, got %v", err)
	}

	_, err = service.Update(context.Background(), "not-a-uuid", UpdateInput{Name: "Nome"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected invalid id not found, got %v", err)
	}
}

func strPtr(value string) *string {
	return &value
}

type fakeStudentStore struct {
	created NewStudent
}

func (s *fakeStudentStore) ListStudents(context.Context) ([]Student, error) {
	return nil, nil
}

func (s *fakeStudentStore) CreateStudent(_ context.Context, student NewStudent) (Student, error) {
	s.created = student
	return Student{
		ID:            student.ID,
		Name:          student.Name,
		Active:        true,
		BirthDate:     student.BirthDate,
		Phone:         student.Phone,
		GuardianName:  student.GuardianName,
		GuardianPhone: student.GuardianPhone,
		Notes:         student.Notes,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}, nil
}

func (s *fakeStudentStore) UpdateStudent(_ context.Context, id string, student NewStudent) (Student, error) {
	if id == missingID {
		return Student{}, ErrNotFound
	}
	return Student{ID: id, Name: student.Name, Active: true, BirthDate: student.BirthDate, Phone: student.Phone, GuardianName: student.GuardianName, GuardianPhone: student.GuardianPhone, Notes: student.Notes}, nil
}

func (s *fakeStudentStore) SetStudentStatus(_ context.Context, id string, active bool) (Student, error) {
	if id == missingID {
		return Student{}, ErrNotFound
	}
	return Student{ID: id, Name: "Aluno", Active: active}, nil
}
