package professors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
	"github.com/alexandre/senshi-training-planner/backend/internal/config"
)

func TestAdminCanListProfessors(t *testing.T) {
	service := &fakeProfessorService{
		professors: []Professor{{
			ID:        professorID,
			Name:      "Professor",
			Email:     "professor@example.com",
			Active:    true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}},
	}
	response := performProfessorRequest(t, auth.RoleAdmin, service, http.MethodGet, "/professors", "")

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "professor@example.com") {
		t.Fatalf("expected professor response, got %s", body)
	}
	if strings.Contains(body, "password_hash") || strings.Contains(body, "PasswordHash") || strings.Contains(body, "passwordHash") {
		t.Fatal("expected response not to serialize password_hash")
	}
}

func TestProfessorReceivesForbidden(t *testing.T) {
	response := performProfessorRequest(t, auth.RoleProfessor, &fakeProfessorService{}, http.MethodGet, "/professors", "")

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", response.Code)
	}
}

func TestUnauthenticatedReceivesUnauthorized(t *testing.T) {
	response := performUnauthenticatedProfessorRequest(t, &fakeProfessorService{}, http.MethodGet, "/professors", "")

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
}

func TestAdminCanCreateProfessor(t *testing.T) {
	service := &fakeProfessorService{
		createdProfessor: Professor{ID: professorID, Name: "Professor", Email: "professor@example.com", Active: true},
	}
	response := performProfessorRequest(
		t,
		auth.RoleAdmin,
		service,
		http.MethodPost,
		"/professors",
		`{"name":"Professor","email":"professor@example.com","password":"uma senha longa e segura"}`,
	)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	if service.createInput.Password != "uma senha longa e segura" {
		t.Fatal("expected handler to pass password to service without logging or returning it")
	}
	if strings.Contains(response.Body.String(), "password") {
		t.Fatal("expected create response not to serialize password fields")
	}
}

func TestDuplicateEmailReturnsConflict(t *testing.T) {
	service := &fakeProfessorService{createErr: ErrEmailExists}
	response := performProfessorRequest(
		t,
		auth.RoleAdmin,
		service,
		http.MethodPost,
		"/professors",
		`{"name":"Professor","email":"professor@example.com","password":"uma senha longa e segura"}`,
	)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", response.Code)
	}
}

func TestAdminCanEditProfessor(t *testing.T) {
	service := &fakeProfessorService{
		updatedProfessor: Professor{ID: professorID, Name: "Novo", Email: "novo@example.com", Active: true},
	}
	response := performProfessorRequest(
		t,
		auth.RoleAdmin,
		service,
		http.MethodPut,
		"/professors/"+professorID,
		`{"name":"Novo","email":"novo@example.com"}`,
	)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if service.updateID != professorID {
		t.Fatalf("expected update id %s, got %s", professorID, service.updateID)
	}
}

func TestUnknownProfessorReturnsNotFound(t *testing.T) {
	service := &fakeProfessorService{updateErr: ErrNotFound}
	response := performProfessorRequest(
		t,
		auth.RoleAdmin,
		service,
		http.MethodPut,
		"/professors/"+professorID,
		`{"name":"Novo","email":"novo@example.com"}`,
	)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}

func TestStatusAndPasswordAndDeleteRoutes(t *testing.T) {
	service := &fakeProfessorService{
		statusProfessor: Professor{ID: professorID, Name: "Professor", Email: "professor@example.com", Active: false},
	}

	statusResponse := performProfessorRequest(
		t,
		auth.RoleAdmin,
		service,
		http.MethodPatch,
		"/professors/"+professorID+"/status",
		`{"active":false}`,
	)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("expected status update 200, got %d", statusResponse.Code)
	}
	if service.statusID != professorID || service.statusInput.Active == nil || *service.statusInput.Active {
		t.Fatal("expected status input to deactivate professor")
	}

	passwordResponse := performProfessorRequest(
		t,
		auth.RoleAdmin,
		service,
		http.MethodPut,
		"/professors/"+professorID+"/password",
		`{"password":"uma senha longa e segura"}`,
	)
	if passwordResponse.Code != http.StatusNoContent {
		t.Fatalf("expected password reset 204, got %d", passwordResponse.Code)
	}
	if service.passwordID != professorID {
		t.Fatal("expected password route to call service")
	}

	deleteResponse := performProfessorRequest(t, auth.RoleAdmin, service, http.MethodDelete, "/professors/"+professorID, "")
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", deleteResponse.Code)
	}
	if service.deleteID != professorID {
		t.Fatal("expected delete route to call service")
	}
}

func TestProfessorJSONValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"name":"Professor","email":`},
		{name: "unknown field", body: `{"name":"Professor","email":"professor@example.com","password":"uma senha longa e segura","role":"ADMIN"}`},
		{name: "trailing json", body: `{"name":"Professor","email":"professor@example.com","password":"uma senha longa e segura"} {"name":"Outro"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performProfessorRequest(t, auth.RoleAdmin, &fakeProfessorService{}, http.MethodPost, "/professors", tt.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", response.Code)
			}
		})
	}
}

func performProfessorRequest(t *testing.T, role auth.Role, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	user := auth.PublicUser{ID: "user-id", Name: "User", Email: "user@example.com", Role: role}
	return performRequest(t, &user, service, method, path, body)
}

func performUnauthenticatedProfessorRequest(t *testing.T, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	return performRequest(t, nil, service, method, path, body)
}

func performRequest(t *testing.T, user *auth.PublicUser, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	authStore := &fakeAuthStore{user: user}
	authService := auth.NewService(authStore)
	authHandler := auth.NewHandler(authService, auth.NewLoginLimiter(), config.AppEnvDevelopment)
	professorHandler := NewHandler(service)

	mux := http.NewServeMux()
	adminOnly := func(next http.Handler) http.Handler {
		return authHandler.Authenticate(auth.RequireAdmin(next))
	}
	mux.Handle("/professors", adminOnly(http.HandlerFunc(professorHandler.Collection)))
	mux.Handle("/professors/", adminOnly(http.HandlerFunc(professorHandler.Resource)))

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != nil {
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "raw-token"})
	}
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	return response
}

type fakeProfessorService struct {
	professors       []Professor
	createdProfessor Professor
	updatedProfessor Professor
	statusProfessor  Professor
	createInput      CreateInput
	updateID         string
	statusID         string
	statusInput      StatusInput
	passwordID       string
	deleteID         string
	createErr        error
	updateErr        error
}

func (s *fakeProfessorService) List(context.Context) ([]Professor, error) {
	return s.professors, nil
}

func (s *fakeProfessorService) Create(_ context.Context, input CreateInput) (Professor, error) {
	s.createInput = input
	if s.createErr != nil {
		return Professor{}, s.createErr
	}
	return s.createdProfessor, nil
}

func (s *fakeProfessorService) Update(_ context.Context, id string, input UpdateInput) (Professor, error) {
	s.updateID = id
	if s.updateErr != nil {
		return Professor{}, s.updateErr
	}
	if s.updatedProfessor.ID == "" {
		return Professor{ID: id, Name: input.Name, Email: input.Email, Active: true}, nil
	}
	return s.updatedProfessor, nil
}

func (s *fakeProfessorService) SetStatus(_ context.Context, id string, input StatusInput) (Professor, error) {
	s.statusID = id
	s.statusInput = input
	if s.statusProfessor.ID == "" {
		active := false
		if input.Active != nil {
			active = *input.Active
		}
		return Professor{ID: id, Active: active}, nil
	}
	return s.statusProfessor, nil
}

func (s *fakeProfessorService) SetPassword(_ context.Context, id string, _ PasswordInput) error {
	s.passwordID = id
	return nil
}

func (s *fakeProfessorService) Delete(_ context.Context, id string) error {
	s.deleteID = id
	return nil
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
