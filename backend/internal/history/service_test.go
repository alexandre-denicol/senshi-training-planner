package history

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
)

const (
	historyID       = "11111111-1111-1111-1111-111111111111"
	scheduleEntryID = "22222222-2222-2222-2222-222222222222"
	userID          = "33333333-3333-3333-3333-333333333333"
	studentIDOne    = "44444444-4444-4444-4444-444444444444"
	studentIDTwo    = "55555555-5555-5555-5555-555555555555"
)

func TestServiceCompleteSnapshotsScheduleAndUserData(t *testing.T) {
	store := &fakeHistoryStore{detail: detailFixture()}
	service := NewService(store)
	completedAt := time.Date(2026, 8, 23, 15, 32, 0, 0, time.UTC)
	service.newUUID = func() (string, error) { return historyID, nil }
	service.now = func() time.Time { return completedAt }
	user := auth.PublicUser{ID: userID, Name: "Professor X", Email: "professor@example.com", Role: auth.RoleProfessor}

	detail, err := service.Complete(context.Background(), scheduleEntryID, user, CompletionDetails{
		ParticipantStudentIDs: []string{studentIDOne, studentIDTwo},
		Notes:                 stringPtr("  Turma respondeu bem.\nReduzimos a intensidade.  "),
	})
	if err != nil {
		t.Fatalf("expected completion success, got %v", err)
	}

	if detail.TrainingDate != "2026-08-20" {
		t.Fatalf("expected scheduled date to become training date, got %s", detail.TrainingDate)
	}
	if store.completedHistoryID != historyID || store.completedScheduleID != scheduleEntryID {
		t.Fatal("expected generated history id and schedule id to reach store")
	}
	if store.completedBy.ID != userID || store.completedBy.Name != "Professor X" {
		t.Fatalf("expected authenticated user snapshot input, got %#v", store.completedBy)
	}
	if !store.completedAt.Equal(completedAt) {
		t.Fatalf("expected server completion timestamp, got %s", store.completedAt)
	}
	if len(store.details.ParticipantStudentIDs) != 2 || store.details.ParticipantStudentIDs[0] != studentIDOne || store.details.ParticipantStudentIDs[1] != studentIDTwo {
		t.Fatalf("expected participant student ids to reach store, got %#v", store.details.ParticipantStudentIDs)
	}
	if store.details.Notes == nil || *store.details.Notes != "Turma respondeu bem.\nReduzimos a intensidade." {
		t.Fatalf("expected trimmed notes with line breaks to reach store, got %#v", store.details.Notes)
	}
	if detail.WorkoutName != "Treino Snapshot" || detail.Blocks[0].BlockName != "Bloco A" || detail.Blocks[0].CategoryName != "Categoria A" {
		t.Fatalf("expected snapshot values in detail, got %#v", detail)
	}
	if detail.Blocks[0].Description == nil || *detail.Blocks[0].Description != "Descrição snapshot" {
		t.Fatalf("expected block description snapshot, got %#v", detail.Blocks[0].Description)
	}
	if len(detail.Blocks[0].Sequence) != 2 || detail.Blocks[0].Sequence[0].Text != "Jab" || detail.Blocks[0].Sequence[1].Text != "Direto" {
		t.Fatalf("expected ordered block sequence snapshot, got %#v", detail.Blocks[0].Sequence)
	}
}

func TestServiceCompleteParticipantDetailsValidation(t *testing.T) {
	tests := []struct {
		name      string
		details   CompletionDetails
		wantErr   error
		wantIDs   []string
		wantNotes *string
	}{
		{name: "no details", details: CompletionDetails{}, wantIDs: []string{}},
		{name: "empty ids equivalent", details: CompletionDetails{ParticipantStudentIDs: nil}, wantIDs: []string{}},
		{name: "single student", details: CompletionDetails{ParticipantStudentIDs: []string{studentIDOne}}, wantIDs: []string{studentIDOne}},
		{name: "multiple students preserve order", details: CompletionDetails{ParticipantStudentIDs: []string{studentIDTwo, studentIDOne}}, wantIDs: []string{studentIDTwo, studentIDOne}},
		{name: "ids are trimmed", details: CompletionDetails{ParticipantStudentIDs: []string{" " + studentIDOne + " "}}, wantIDs: []string{studentIDOne}},
		{name: "notes trimmed", details: CompletionDetails{Notes: stringPtr("  Boa resposta da turma.  ")}, wantIDs: []string{}, wantNotes: stringPtr("Boa resposta da turma.")},
		{name: "notes line breaks preserved", details: CompletionDetails{Notes: stringPtr("Linha 1\nLinha 2")}, wantIDs: []string{}, wantNotes: stringPtr("Linha 1\nLinha 2")},
		{name: "whitespace notes become null", details: CompletionDetails{Notes: stringPtr("   \n\t  ")}, wantIDs: []string{}},
		{name: "exactly max notes accepted", details: CompletionDetails{Notes: stringPtr(strings.Repeat("ã", MaxNotesChars))}, wantIDs: []string{}, wantNotes: stringPtr(strings.Repeat("ã", MaxNotesChars))},
		{name: "malformed student id", details: CompletionDetails{ParticipantStudentIDs: []string{"not-a-uuid"}}, wantErr: ErrInvalidParticipants},
		{name: "duplicate student id rejected", details: CompletionDetails{ParticipantStudentIDs: []string{studentIDOne, studentIDOne}}, wantErr: ErrInvalidParticipants},
		{name: "too many student ids", details: CompletionDetails{ParticipantStudentIDs: repeatNames(MaxParticipantStudentIDs + 1)}, wantErr: ErrInvalidParticipants},
		{name: "oversized notes", details: CompletionDetails{Notes: stringPtr(strings.Repeat("ã", MaxNotesChars+1))}, wantErr: ErrInvalidRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeHistoryStore{detail: detailFixture()}
			service := NewService(store)
			service.newUUID = func() (string, error) { return historyID, nil }
			_, err := service.Complete(context.Background(), scheduleEntryID, auth.PublicUser{ID: userID, Name: "User", Role: auth.RoleAdmin}, tt.details)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected success, got %v", err)
			}
			if len(store.details.ParticipantStudentIDs) != len(tt.wantIDs) {
				t.Fatalf("expected ids %#v, got %#v", tt.wantIDs, store.details.ParticipantStudentIDs)
			}
			for index, want := range tt.wantIDs {
				if store.details.ParticipantStudentIDs[index] != want {
					t.Fatalf("expected ids %#v, got %#v", tt.wantIDs, store.details.ParticipantStudentIDs)
				}
			}
			if (tt.wantNotes == nil) != (store.details.Notes == nil) {
				t.Fatalf("expected notes %#v, got %#v", tt.wantNotes, store.details.Notes)
			}
			if tt.wantNotes != nil && *tt.wantNotes != *store.details.Notes {
				t.Fatalf("expected notes %q, got %q", *tt.wantNotes, *store.details.Notes)
			}
		})
	}
}

func TestServiceCompleteMapsErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "missing schedule", err: ErrScheduleNotFound, want: ErrScheduleNotFound},
		{name: "already completed", err: ErrAlreadyCompleted, want: ErrAlreadyCompleted},
		{name: "empty composition", err: ErrSnapshotUnavailable, want: ErrSnapshotUnavailable},
		{name: "nonexistent or inactive student", err: ErrInvalidParticipants, want: ErrInvalidParticipants},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeHistoryStore{completeErr: tt.err}
			service := NewService(store)
			service.newUUID = func() (string, error) { return historyID, nil }
			_, err := service.Complete(context.Background(), scheduleEntryID, auth.PublicUser{ID: userID, Name: "User", Role: auth.RoleAdmin}, CompletionDetails{})
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestServiceHistoryRangeAndDetailValidation(t *testing.T) {
	service := NewService(&fakeHistoryStore{detail: detailFixture()})

	if _, err := service.List(context.Background(), "2026-08-01", "2026-08-31"); err != nil {
		t.Fatalf("expected valid range, got %v", err)
	}
	if _, err := service.Get(context.Background(), historyID); err != nil {
		t.Fatalf("expected valid detail, got %v", err)
	}

	tests := []struct {
		name string
		from string
		to   string
	}{
		{name: "invalid date", from: "2026/08/01", to: "2026-08-31"},
		{name: "impossible date", from: "2026-02-30", to: "2026-08-31"},
		{name: "from after to", from: "2026-09-01", to: "2026-08-31"},
		{name: "too large", from: "2026-01-01", to: "2026-04-04"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := service.List(context.Background(), tt.from, tt.to); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("expected invalid range, got %v", err)
			}
		})
	}

	if _, err := service.Get(context.Background(), "not-a-uuid"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected invalid id to be not found, got %v", err)
	}
}

type fakeHistoryStore struct {
	detail              Detail
	completedHistoryID  string
	completedScheduleID string
	completedBy         auth.PublicUser
	completedAt         time.Time
	details             CompletionDetails
	completeErr         error
	listErr             error
	getErr              error
}

func (s *fakeHistoryStore) ListHistory(context.Context, string, string) ([]ListItem, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []ListItem{listItemFixture()}, nil
}

func (s *fakeHistoryStore) GetHistory(context.Context, string) (Detail, error) {
	if s.getErr != nil {
		return Detail{}, s.getErr
	}
	return s.detail, nil
}

func (s *fakeHistoryStore) CompleteScheduleEntry(_ context.Context, historyID string, scheduleID string, completedBy auth.PublicUser, completedAt time.Time, details CompletionDetails) (Detail, error) {
	s.completedHistoryID = historyID
	s.completedScheduleID = scheduleID
	s.completedBy = completedBy
	s.completedAt = completedAt
	s.details = details
	if s.completeErr != nil {
		return Detail{}, s.completeErr
	}
	return s.detail, nil
}

func detailFixture() Detail {
	return Detail{
		ID:               historyID,
		TrainingDate:     "2026-08-20",
		WorkoutName:      "Treino Snapshot",
		CompletedByName:  "Professor X",
		CompletedAt:      time.Date(2026, 8, 23, 15, 32, 0, 0, time.UTC),
		ParticipantNames: []string{},
		Blocks: []Block{
			{
				Position:     1,
				BlockName:    "Bloco A",
				CategoryName: "Categoria A",
				Description:  stringPtr("Descrição snapshot"),
				Sequence: []SequenceItem{
					{Position: 1, Text: "Jab"},
					{Position: 2, Text: "Direto"},
				},
			},
			{Position: 2, BlockName: "Bloco B", CategoryName: "Categoria B", Sequence: []SequenceItem{}},
		},
	}
}

func detailWithParticipantsFixture() Detail {
	detail := detailFixture()
	detail.ParticipantCount = intPtr(12)
	detail.ParticipantNames = []string{"João", "Maria"}
	return detail
}

func detailWithNotesFixture() Detail {
	detail := detailWithParticipantsFixture()
	detail.Notes = stringPtr("Turma respondeu bem.\nReduzimos a intensidade.")
	return detail
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func repeatNames(total int) []string {
	names := make([]string, total)
	for index := range names {
		names[index] = "Nome"
	}
	return names
}

func listItemFixture() ListItem {
	return ListItem{
		ID:               historyID,
		TrainingDate:     "2026-08-20",
		WorkoutName:      "Treino Snapshot",
		BlockCount:       2,
		ParticipantCount: intPtr(1),
		CompletedByName:  "Professor X",
		CompletedAt:      time.Date(2026, 8, 23, 15, 32, 0, 0, time.UTC),
		ScheduleEntryID:  scheduleEntryID,
	}
}
