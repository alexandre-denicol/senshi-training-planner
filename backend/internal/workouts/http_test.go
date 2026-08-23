package workouts

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

func TestWorkoutHTTPAuthorization(t *testing.T) {
	if response := performUnauthenticatedWorkoutRequest(t, &fakeWorkoutService{}, http.MethodGet, "/workouts", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated status 401, got %d", response.Code)
	}
	if response := performUnauthenticatedWorkoutRequest(t, &fakeWorkoutService{}, http.MethodGet, "/workouts/"+workoutID, ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated detail status 401, got %d", response.Code)
	}
	if response := performWorkoutRequest(t, auth.RoleProfessor, &fakeWorkoutService{}, http.MethodGet, "/workouts", ""); response.Code != http.StatusOK {
		t.Fatalf("expected professor list status 200, got %d", response.Code)
	}
	if response := performWorkoutRequest(t, auth.RoleAdmin, &fakeWorkoutService{}, http.MethodGet, "/workouts", ""); response.Code != http.StatusOK {
		t.Fatalf("expected admin status 200, got %d", response.Code)
	}
	service := &fakeWorkoutService{detailWorkout: workoutFixture(workoutID, "Treino", []string{blockIDOne, blockIDTwo})}
	if response := performWorkoutRequest(t, auth.RoleAdmin, service, http.MethodGet, "/workouts/"+workoutID, ""); response.Code != http.StatusOK {
		t.Fatalf("expected admin detail status 200, got %d", response.Code)
	}
	if response := performWorkoutRequest(t, auth.RoleProfessor, service, http.MethodGet, "/workouts/"+workoutID, ""); response.Code != http.StatusOK {
		t.Fatalf("expected professor detail status 200, got %d", response.Code)
	}
	if response := performWorkoutRequest(t, auth.RoleProfessor, service, http.MethodPost, "/workouts", `{"name":"Novo","blockIds":["`+blockIDOne+`"]}`); response.Code != http.StatusCreated {
		t.Fatalf("expected professor create status 201, got %d", response.Code)
	}
	if response := performWorkoutRequest(t, auth.RoleProfessor, service, http.MethodPut, "/workouts/"+workoutID, `{"name":"Editado","blockIds":["`+blockIDOne+`"]}`); response.Code != http.StatusOK {
		t.Fatalf("expected professor update status 200, got %d", response.Code)
	}
	if response := performWorkoutRequest(t, auth.RoleProfessor, service, http.MethodPatch, "/workouts/"+workoutID+"/status", `{"active":false}`); response.Code != http.StatusOK {
		t.Fatalf("expected professor status mutation 200, got %d", response.Code)
	}
	if response := performWorkoutRequest(t, auth.RoleProfessor, service, http.MethodDelete, "/workouts/"+workoutID, ""); response.Code != http.StatusNoContent {
		t.Fatalf("expected professor delete status 204, got %d", response.Code)
	}
}

func TestWorkoutHTTPListDetailCreateDuplicateUpdateStatusDeleteAndNotFound(t *testing.T) {
	service := &fakeWorkoutService{
		workouts:       []WorkoutListItem{workoutListFixture(workoutID, "Treino", true, 2)},
		detailWorkout:  workoutFixture(workoutID, "Treino", []string{blockIDOne, blockIDTwo}),
		createdWorkout: workoutFixture(workoutID, "Novo", []string{blockIDOne, blockIDTwo}),
		updatedWorkout: workoutFixture(workoutID, "Editado", []string{blockIDTwo, blockIDOne}),
		statusWorkout:  workoutListFixture(workoutID, "Editado", false, 2),
	}

	list := performWorkoutRequest(t, auth.RoleAdmin, service, http.MethodGet, "/workouts", "")
	if list.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", list.Code)
	}
	if !strings.Contains(list.Body.String(), `"blockCount":2`) {
		t.Fatalf("expected compact list block count, got %s", list.Body.String())
	}

	detail := performWorkoutRequest(t, auth.RoleAdmin, service, http.MethodGet, "/workouts/"+workoutID, "")
	if detail.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d", detail.Code)
	}
	if service.detailID != workoutID {
		t.Fatalf("expected detail id %s, got %s", workoutID, service.detailID)
	}
	if !strings.Contains(detail.Body.String(), `"position":1`) || !strings.Contains(detail.Body.String(), `"position":2`) {
		t.Fatalf("expected ordered detail positions, got %s", detail.Body.String())
	}

	service.detailWorkout = workoutFixture(workoutID, "Treino Inativo", []string{blockIDOne, inactiveBlockID})
	service.detailWorkout.Active = false
	service.detailWorkout.Blocks[1].Active = false
	inactiveDetail := performWorkoutRequest(t, auth.RoleProfessor, service, http.MethodGet, "/workouts/"+workoutID, "")
	if inactiveDetail.Code != http.StatusOK {
		t.Fatalf("expected inactive detail 200, got %d", inactiveDetail.Code)
	}
	if !strings.Contains(inactiveDetail.Body.String(), `"active":false`) || !strings.Contains(inactiveDetail.Body.String(), inactiveBlockID) {
		t.Fatalf("expected inactive workout and block to remain readable, got %s", inactiveDetail.Body.String())
	}

	create := performWorkoutRequest(t, auth.RoleAdmin, service, http.MethodPost, "/workouts", `{"name":"Novo","blockIds":["`+blockIDOne+`","`+blockIDTwo+`"]}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", create.Code)
	}
	if service.createInput.Name != "Novo" || strings.Join(service.createInput.BlockIDs, ",") != blockIDOne+","+blockIDTwo {
		t.Fatal("expected create input order to reach service")
	}

	service.createErr = ErrNameExists
	duplicate := performWorkoutRequest(t, auth.RoleAdmin, service, http.MethodPost, "/workouts", `{"name":"novo","blockIds":["`+blockIDOne+`"]}`)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("expected duplicate 409, got %d", duplicate.Code)
	}
	service.createErr = nil

	update := performWorkoutRequest(t, auth.RoleAdmin, service, http.MethodPut, "/workouts/"+workoutID, `{"name":"Editado","blockIds":["`+blockIDTwo+`","`+blockIDOne+`"]}`)
	if update.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d", update.Code)
	}
	if service.updateID != workoutID || strings.Join(service.updateInput.BlockIDs, ",") != blockIDTwo+","+blockIDOne {
		t.Fatal("expected update route to preserve submitted order")
	}

	status := performWorkoutRequest(t, auth.RoleAdmin, service, http.MethodPatch, "/workouts/"+workoutID+"/status", `{"active":false}`)
	if status.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status.Code)
	}
	if service.statusInput.Active == nil || *service.statusInput.Active {
		t.Fatal("expected status input to deactivate")
	}

	remove := performWorkoutRequest(t, auth.RoleAdmin, service, http.MethodDelete, "/workouts/"+workoutID, "")
	if remove.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", remove.Code)
	}
	if service.deleteID != workoutID {
		t.Fatal("expected delete route to pass id")
	}

	service.deleteErr = ErrInUse
	inUse := performWorkoutRequest(t, auth.RoleAdmin, service, http.MethodDelete, "/workouts/"+workoutID, "")
	if inUse.Code != http.StatusConflict {
		t.Fatalf("expected scheduled workout delete 409, got %d", inUse.Code)
	}
	if strings.Contains(inUse.Body.String(), "schedule_entries_workout_id_fkey") {
		t.Fatalf("expected safe error body, got %s", inUse.Body.String())
	}
	service.deleteErr = nil

	service.detailErr = ErrNotFound
	notFound := performWorkoutRequest(t, auth.RoleAdmin, service, http.MethodGet, "/workouts/"+workoutID, "")
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected not found 404, got %d", notFound.Code)
	}
}

func TestWorkoutHTTPJSONValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"name":`},
		{name: "unknown field", body: `{"name":"Treino","blockIds":["` + blockIDOne + `"],"description":"x"}`},
		{name: "trailing json", body: `{"name":"Treino","blockIds":["` + blockIDOne + `"]} {"name":"Outro"}`},
		{name: "oversized", body: `{"name":"` + strings.Repeat("a", httpapi.MaxJSONBodyBytes) + `","blockIds":["` + blockIDOne + `"]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performWorkoutRequest(t, auth.RoleAdmin, &fakeWorkoutService{}, http.MethodPost, "/workouts", tt.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", response.Code)
			}
		})
	}
}

func performWorkoutRequest(t *testing.T, role auth.Role, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	user := auth.PublicUser{ID: "user-id", Name: "User", Email: "user@example.com", Role: role}
	return performRequest(t, &user, service, method, path, body)
}

func performUnauthenticatedWorkoutRequest(t *testing.T, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	return performRequest(t, nil, service, method, path, body)
}

func performRequest(t *testing.T, user *auth.PublicUser, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	authStore := &fakeAuthStore{user: user}
	authService := auth.NewService(authStore)
	authHandler := auth.NewHandler(authService, auth.NewLoginLimiter(), config.AppEnvDevelopment)
	workoutHandler := NewHandler(service)

	mux := http.NewServeMux()
	mux.Handle("/workouts", authHandler.Authenticate(http.HandlerFunc(workoutHandler.Collection)))
	mux.Handle("/workouts/", authHandler.Authenticate(http.HandlerFunc(workoutHandler.Resource)))

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != nil {
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "raw-token"})
	}
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	return response
}

type fakeWorkoutService struct {
	workouts       []WorkoutListItem
	detailWorkout  WorkoutDetail
	createdWorkout WorkoutDetail
	updatedWorkout WorkoutDetail
	statusWorkout  WorkoutListItem
	createInput    CreateInput
	updateInput    UpdateInput
	updateID       string
	statusInput    StatusInput
	deleteID       string
	detailID       string
	detailErr      error
	createErr      error
	updateErr      error
	deleteErr      error
}

func (s *fakeWorkoutService) List(context.Context) ([]WorkoutListItem, error) {
	return s.workouts, nil
}

func (s *fakeWorkoutService) Get(_ context.Context, id string) (WorkoutDetail, error) {
	s.detailID = id
	if s.detailErr != nil {
		return WorkoutDetail{}, s.detailErr
	}
	return s.detailWorkout, nil
}

func (s *fakeWorkoutService) Create(_ context.Context, input CreateInput) (WorkoutDetail, error) {
	s.createInput = input
	if s.createErr != nil {
		return WorkoutDetail{}, s.createErr
	}
	return s.createdWorkout, nil
}

func (s *fakeWorkoutService) Update(_ context.Context, id string, input UpdateInput) (WorkoutDetail, error) {
	s.updateID = id
	s.updateInput = input
	if s.updateErr != nil {
		return WorkoutDetail{}, s.updateErr
	}
	return s.updatedWorkout, nil
}

func (s *fakeWorkoutService) SetStatus(_ context.Context, _ string, input StatusInput) (WorkoutListItem, error) {
	s.statusInput = input
	return s.statusWorkout, nil
}

func (s *fakeWorkoutService) Delete(_ context.Context, id string) error {
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

func workoutListFixture(id string, name string, active bool, blockCount int) WorkoutListItem {
	now := time.Now().UTC()
	return WorkoutListItem{ID: id, Name: name, Active: active, BlockCount: blockCount, CreatedAt: now, UpdatedAt: now}
}
