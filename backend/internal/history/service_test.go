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
)

func TestServiceCompleteSnapshotsScheduleAndUserData(t *testing.T) {
	store := &fakeHistoryStore{detail: detailFixture()}
	service := NewService(store)
	completedAt := time.Date(2026, 8, 23, 15, 32, 0, 0, time.UTC)
	service.newUUID = func() (string, error) { return historyID, nil }
	service.now = func() time.Time { return completedAt }
	user := auth.PublicUser{ID: userID, Name: "Professor X", Email: "professor@example.com", Role: auth.RoleProfessor}

	detail, err := service.Complete(context.Background(), scheduleEntryID, user, CompletionDetails{
		ParticipantCount: intPtr(12),
		ParticipantNames: []string{" João ", "Maria"},
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
	if store.details.ParticipantCount == nil || *store.details.ParticipantCount != 12 {
		t.Fatalf("expected participant count to reach store, got %#v", store.details.ParticipantCount)
	}
	if len(store.details.ParticipantNames) != 2 || store.details.ParticipantNames[0] != "João" || store.details.ParticipantNames[1] != "Maria" {
		t.Fatalf("expected trimmed participant names to reach store, got %#v", store.details.ParticipantNames)
	}
	if detail.WorkoutName != "Treino Snapshot" || detail.Blocks[0].BlockName != "Bloco A" || detail.Blocks[0].CategoryName != "Categoria A" {
		t.Fatalf("expected snapshot values in detail, got %#v", detail)
	}
}

func TestServiceCompleteParticipantDetailsValidation(t *testing.T) {
	tests := []struct {
		name      string
		details   CompletionDetails
		wantErr   bool
		wantCount *int
		wantNames []string
	}{
		{name: "no details", details: CompletionDetails{}, wantNames: []string{}},
		{name: "empty object equivalent", details: CompletionDetails{ParticipantNames: nil}, wantNames: []string{}},
		{name: "count only", details: CompletionDetails{ParticipantCount: intPtr(12)}, wantCount: intPtr(12), wantNames: []string{}},
		{name: "names only", details: CompletionDetails{ParticipantNames: []string{" João ", "Maria"}}, wantNames: []string{"João", "Maria"}},
		{name: "count and names", details: CompletionDetails{ParticipantCount: intPtr(2), ParticipantNames: []string{"João", "Maria"}}, wantCount: intPtr(2), wantNames: []string{"João", "Maria"}},
		{name: "zero preserved", details: CompletionDetails{ParticipantCount: intPtr(0)}, wantCount: intPtr(0), wantNames: []string{}},
		{name: "count can be smaller than names", details: CompletionDetails{ParticipantCount: intPtr(1), ParticipantNames: []string{"João", "Maria"}}, wantCount: intPtr(1), wantNames: []string{"João", "Maria"}},
		{name: "duplicate names accepted", details: CompletionDetails{ParticipantNames: []string{"João", "João"}}, wantNames: []string{"João", "João"}},
		{name: "negative count", details: CompletionDetails{ParticipantCount: intPtr(-1)}, wantErr: true},
		{name: "too large count", details: CompletionDetails{ParticipantCount: intPtr(501)}, wantErr: true},
		{name: "blank name", details: CompletionDetails{ParticipantNames: []string{""}}, wantErr: true},
		{name: "whitespace name", details: CompletionDetails{ParticipantNames: []string{"   "}}, wantErr: true},
		{name: "oversized name", details: CompletionDetails{ParticipantNames: []string{strings.Repeat("ã", MaxParticipantNameChars+1)}}, wantErr: true},
		{name: "too many names", details: CompletionDetails{ParticipantNames: repeatNames(MaxParticipantNames + 1)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeHistoryStore{detail: detailFixture()}
			service := NewService(store)
			service.newUUID = func() (string, error) { return historyID, nil }
			_, err := service.Complete(context.Background(), scheduleEntryID, auth.PublicUser{ID: userID, Name: "User", Role: auth.RoleAdmin}, tt.details)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidRequest) {
					t.Fatalf("expected invalid request, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected success, got %v", err)
			}
			if (tt.wantCount == nil) != (store.details.ParticipantCount == nil) {
				t.Fatalf("expected count %#v, got %#v", tt.wantCount, store.details.ParticipantCount)
			}
			if tt.wantCount != nil && *tt.wantCount != *store.details.ParticipantCount {
				t.Fatalf("expected count %d, got %d", *tt.wantCount, *store.details.ParticipantCount)
			}
			if len(store.details.ParticipantNames) != len(tt.wantNames) {
				t.Fatalf("expected names %#v, got %#v", tt.wantNames, store.details.ParticipantNames)
			}
			for index, want := range tt.wantNames {
				if store.details.ParticipantNames[index] != want {
					t.Fatalf("expected names %#v, got %#v", tt.wantNames, store.details.ParticipantNames)
				}
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
			{Position: 1, BlockName: "Bloco A", CategoryName: "Categoria A"},
			{Position: 2, BlockName: "Bloco B", CategoryName: "Categoria B"},
		},
	}
}

func detailWithParticipantsFixture() Detail {
	detail := detailFixture()
	detail.ParticipantCount = intPtr(12)
	detail.ParticipantNames = []string{"João", "Maria"}
	return detail
}

func intPtr(value int) *int {
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
