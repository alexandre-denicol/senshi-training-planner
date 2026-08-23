package schedule

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
	"github.com/alexandre/senshi-training-planner/backend/internal/config"
	"github.com/alexandre/senshi-training-planner/backend/internal/httpapi"
)

func TestScheduleHTTPAuthorization(t *testing.T) {
	if response := performUnauthenticatedScheduleRequest(t, &fakeScheduleService{}, http.MethodGet, "/schedule?from=2026-08-01&to=2026-08-31", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated GET 401, got %d", response.Code)
	}
	if response := performScheduleRequest(t, auth.RoleProfessor, &fakeScheduleService{}, http.MethodGet, "/schedule?from=2026-08-01&to=2026-08-31", ""); response.Code != http.StatusOK {
		t.Fatalf("expected professor GET 200, got %d", response.Code)
	}
	if response := performScheduleRequest(t, auth.RoleAdmin, &fakeScheduleService{}, http.MethodGet, "/schedule?from=2026-08-01&to=2026-08-31", ""); response.Code != http.StatusOK {
		t.Fatalf("expected admin GET 200, got %d", response.Code)
	}
	service := &fakeScheduleService{
		createdEntry: entryFixture(entryID, activeWorkoutID, "2026-08-24", true),
		updatedEntry: entryFixture(entryID, otherWorkoutID, "2026-08-25", true),
	}
	if response := performScheduleRequest(t, auth.RoleProfessor, service, http.MethodPost, "/schedule", `{"workoutId":"`+activeWorkoutID+`","scheduledDate":"2026-08-24"}`); response.Code != http.StatusCreated {
		t.Fatalf("expected professor POST 201, got %d", response.Code)
	}
	if response := performScheduleRequest(t, auth.RoleProfessor, service, http.MethodPut, "/schedule/"+entryID, `{"workoutId":"`+otherWorkoutID+`","scheduledDate":"2026-08-25"}`); response.Code != http.StatusOK {
		t.Fatalf("expected professor PUT 200, got %d", response.Code)
	}
	if response := performScheduleRequest(t, auth.RoleProfessor, service, http.MethodDelete, "/schedule/"+entryID, ""); response.Code != http.StatusNoContent {
		t.Fatalf("expected professor DELETE 204, got %d", response.Code)
	}
}

func TestScheduleHTTPCreateUpdateDeleteDuplicateNotFoundAndRange(t *testing.T) {
	completedEntry := entryFixture(entryID, activeWorkoutID, "2026-08-24", true)
	completedAt := time.Date(2026, 8, 23, 15, 32, 0, 0, time.UTC)
	completedEntry.CompletedAt = &completedAt
	service := &fakeScheduleService{
		entries:      []Entry{completedEntry},
		createdEntry: entryFixture(entryID, activeWorkoutID, "2026-08-24", true),
		updatedEntry: entryFixture(entryID, otherWorkoutID, "2026-08-25", true),
	}

	list := performScheduleRequest(t, auth.RoleAdmin, service, http.MethodGet, "/schedule?from=2026-08-01&to=2026-08-31", "")
	if list.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", list.Code)
	}
	if service.from != "2026-08-01" || service.to != "2026-08-31" {
		t.Fatalf("expected query range to reach service, got %s %s", service.from, service.to)
	}
	if !strings.Contains(list.Body.String(), `"completedAt"`) {
		t.Fatalf("expected completed entry state to be serialized when present, got %s", list.Body.String())
	}

	service.listErr = ErrInvalidRequest
	invalidRange := performScheduleRequest(t, auth.RoleAdmin, service, http.MethodGet, "/schedule?from=2026-09-01&to=2026-08-31", "")
	if invalidRange.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid range 400, got %d", invalidRange.Code)
	}
	service.listErr = nil

	create := performScheduleRequest(t, auth.RoleAdmin, service, http.MethodPost, "/schedule", `{"workoutId":"`+activeWorkoutID+`","scheduledDate":"2026-08-24"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", create.Code)
	}
	if service.createInput.WorkoutID != activeWorkoutID || service.createInput.ScheduledDate != "2026-08-24" {
		t.Fatal("expected create input to reach service")
	}

	service.createErr = ErrDuplicate
	duplicate := performScheduleRequest(t, auth.RoleAdmin, service, http.MethodPost, "/schedule", `{"workoutId":"`+activeWorkoutID+`","scheduledDate":"2026-08-24"}`)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("expected duplicate 409, got %d", duplicate.Code)
	}
	service.createErr = nil

	update := performScheduleRequest(t, auth.RoleAdmin, service, http.MethodPut, "/schedule/"+entryID, `{"workoutId":"`+otherWorkoutID+`","scheduledDate":"2026-08-25"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d", update.Code)
	}
	if service.updateID != entryID {
		t.Fatal("expected update route to pass id")
	}

	remove := performScheduleRequest(t, auth.RoleAdmin, service, http.MethodDelete, "/schedule/"+entryID, "")
	if remove.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", remove.Code)
	}

	service.updateErr = ErrCompleted
	completedUpdate := performScheduleRequest(t, auth.RoleAdmin, service, http.MethodPut, "/schedule/"+entryID, `{"workoutId":"`+activeWorkoutID+`","scheduledDate":"2026-08-24"}`)
	if completedUpdate.Code != http.StatusConflict {
		t.Fatalf("expected completed update 409, got %d", completedUpdate.Code)
	}
	service.updateErr = nil

	service.deleteErr = ErrCompleted
	completedDelete := performScheduleRequest(t, auth.RoleAdmin, service, http.MethodDelete, "/schedule/"+entryID, "")
	if completedDelete.Code != http.StatusConflict {
		t.Fatalf("expected completed delete 409, got %d", completedDelete.Code)
	}
	service.deleteErr = nil

	service.updateErr = ErrNotFound
	notFound := performScheduleRequest(t, auth.RoleAdmin, service, http.MethodPut, "/schedule/"+entryID, `{"workoutId":"`+activeWorkoutID+`","scheduledDate":"2026-08-24"}`)
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected not found 404, got %d", notFound.Code)
	}
}

func TestScheduleHTTPJSONValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"workoutId":`},
		{name: "unknown field", body: `{"workoutId":"` + activeWorkoutID + `","scheduledDate":"2026-08-24","notes":"x"}`},
		{name: "trailing json", body: `{"workoutId":"` + activeWorkoutID + `","scheduledDate":"2026-08-24"} {"workoutId":"` + otherWorkoutID + `"}`},
		{name: "oversized", body: `{"workoutId":"` + activeWorkoutID + `","scheduledDate":"` + strings.Repeat("a", httpapi.MaxJSONBodyBytes) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performScheduleRequest(t, auth.RoleAdmin, &fakeScheduleService{}, http.MethodPost, "/schedule", tt.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", response.Code)
			}
		})
	}
}

func performScheduleRequest(t *testing.T, role auth.Role, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	user := auth.PublicUser{ID: "user-id", Name: "User", Email: "user@example.com", Role: role}
	return performRequest(t, &user, service, method, path, body)
}

func performUnauthenticatedScheduleRequest(t *testing.T, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	return performRequest(t, nil, service, method, path, body)
}

func performRequest(t *testing.T, user *auth.PublicUser, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	authStore := &fakeAuthStore{user: user}
	authService := auth.NewService(authStore)
	authHandler := auth.NewHandler(authService, auth.NewLoginLimiter(), config.AppEnvDevelopment)
	scheduleHandler := NewHandler(service)

	mux := http.NewServeMux()
	mux.Handle("/schedule", authHandler.Authenticate(http.HandlerFunc(scheduleHandler.Collection)))
	mux.Handle("/schedule/", authHandler.Authenticate(http.HandlerFunc(scheduleHandler.Resource)))

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != nil {
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "raw-token"})
	}
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	return response
}

type fakeScheduleService struct {
	entries      []Entry
	createdEntry Entry
	updatedEntry Entry
	createInput  CreateInput
	updateInput  UpdateInput
	updateID     string
	deleteID     string
	from         string
	to           string
	listErr      error
	createErr    error
	updateErr    error
	deleteErr    error
}

func (s *fakeScheduleService) List(_ context.Context, from string, to string) ([]Entry, error) {
	s.from = from
	s.to = to
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.entries, nil
}

func (s *fakeScheduleService) Create(_ context.Context, input CreateInput) (Entry, error) {
	s.createInput = input
	if s.createErr != nil {
		return Entry{}, s.createErr
	}
	return s.createdEntry, nil
}

func (s *fakeScheduleService) Update(_ context.Context, id string, input UpdateInput) (Entry, error) {
	s.updateID = id
	s.updateInput = input
	if s.updateErr != nil {
		return Entry{}, s.updateErr
	}
	return s.updatedEntry, nil
}

func (s *fakeScheduleService) Delete(_ context.Context, id string) error {
	s.deleteID = id
	return s.deleteErr
}

type fakeAuthStore struct {
	user *auth.PublicUser
}

func (s *fakeAuthStore) FindUserByEmail(context.Context, string) (auth.User, error) {
	return auth.User{}, auth.ErrInvalidCredentials
}

func (s *fakeAuthStore) CreateSession(context.Context, auth.Session) error {
	return nil
}

func (s *fakeAuthStore) FindSessionByTokenHash(_ context.Context, tokenHash string) (auth.Session, auth.PublicUser, error) {
	if s.user == nil || tokenHash != auth.HashSessionToken("raw-token") {
		return auth.Session{}, auth.PublicUser{}, auth.ErrUnauthenticated
	}

	now := time.Now().UTC()
	return auth.Session{
		ID:                "session-id",
		UserID:            s.user.ID,
		TokenHash:         tokenHash,
		CreatedAt:         now,
		LastSeenAt:        now,
		ExpiresAt:         now.Add(auth.IdleTimeout),
		AbsoluteExpiresAt: now.Add(auth.AbsoluteTimeout),
	}, *s.user, nil
}

func (s *fakeAuthStore) RefreshSession(context.Context, string, time.Time, time.Time) error {
	return nil
}

func (s *fakeAuthStore) DeleteSessionByTokenHash(context.Context, string) error {
	return nil
}

func (s *fakeAuthStore) EmailExists(context.Context, string) (bool, error) {
	return false, nil
}

func (s *fakeAuthStore) AdminExists(context.Context) (bool, error) {
	return false, nil
}

func (s *fakeAuthStore) CreateAdmin(context.Context, auth.User) error {
	return nil
}
