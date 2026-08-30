package students

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

func TestStudentHTTPAuthorization(t *testing.T) {
	if response := performUnauthenticatedStudentRequest(t, &fakeStudentService{}, http.MethodGet, "/students", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated status 401, got %d", response.Code)
	}
	if response := performStudentRequest(t, auth.RoleProfessor, &fakeStudentService{}, http.MethodGet, "/students", ""); response.Code != http.StatusOK {
		t.Fatalf("expected professor list status 200, got %d", response.Code)
	}
	if response := performStudentRequest(t, auth.RoleAdmin, &fakeStudentService{}, http.MethodGet, "/students", ""); response.Code != http.StatusOK {
		t.Fatalf("expected admin list status 200, got %d", response.Code)
	}

	service := &fakeStudentService{createdStudent: Student{ID: studentID, Name: "Ana Silva", Active: true}}
	if response := performStudentRequest(t, auth.RoleProfessor, service, http.MethodPost, "/students", `{"name":"Ana Silva"}`); response.Code != http.StatusCreated {
		t.Fatalf("expected professor create status 201, got %d", response.Code)
	}
	if response := performStudentRequest(t, auth.RoleProfessor, service, http.MethodPut, "/students/"+studentID, `{"name":"Ana Souza"}`); response.Code != http.StatusOK {
		t.Fatalf("expected professor update status 200, got %d", response.Code)
	}
	if response := performStudentRequest(t, auth.RoleProfessor, service, http.MethodPatch, "/students/"+studentID+"/status", `{"active":false}`); response.Code != http.StatusOK {
		t.Fatalf("expected professor status update 200, got %d", response.Code)
	}

	if response := performStudentRequest(t, auth.RoleAdmin, service, http.MethodPost, "/students", `{"name":"Ana Silva"}`); response.Code != http.StatusCreated {
		t.Fatalf("expected admin create status 201, got %d", response.Code)
	}
	if response := performStudentRequest(t, auth.RoleAdmin, service, http.MethodPut, "/students/"+studentID, `{"name":"Ana Souza"}`); response.Code != http.StatusOK {
		t.Fatalf("expected admin update status 200, got %d", response.Code)
	}
	if response := performStudentRequest(t, auth.RoleAdmin, service, http.MethodPatch, "/students/"+studentID+"/status", `{"active":false}`); response.Code != http.StatusOK {
		t.Fatalf("expected admin status update 200, got %d", response.Code)
	}
}

func TestStudentHTTPCreateUpdateStatusAndOptionalFields(t *testing.T) {
	birthDate := "2010-05-20"
	phone := "(48) 99999-0000"
	service := &fakeStudentService{
		createdStudent: Student{ID: studentID, Name: "Ana Silva", Active: true, BirthDate: &birthDate, Phone: &phone},
		updatedStudent: Student{ID: studentID, Name: "Ana Souza", Active: true},
		statusStudent:  Student{ID: studentID, Name: "Ana Souza", Active: false},
	}

	create := performStudentRequest(t, auth.RoleAdmin, service, http.MethodPost, "/students",
		`{"name":"Ana Silva","birthDate":"2010-05-20","phone":"(48) 99999-0000"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", create.Code)
	}
	if service.createInput.Name != "Ana Silva" {
		t.Fatal("expected create input to reach service")
	}
	if service.createInput.BirthDate == nil || *service.createInput.BirthDate != "2010-05-20" {
		t.Fatal("expected birth date to reach service")
	}
	if !strings.Contains(create.Body.String(), `"birthDate":"2010-05-20"`) {
		t.Fatalf("expected birth date in response, got %s", create.Body.String())
	}

	update := performStudentRequest(t, auth.RoleAdmin, service, http.MethodPut, "/students/"+studentID, `{"name":"Ana Souza"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d", update.Code)
	}
	if service.updateID != studentID {
		t.Fatal("expected update route to pass id")
	}

	status := performStudentRequest(t, auth.RoleAdmin, service, http.MethodPatch, "/students/"+studentID+"/status", `{"active":false}`)
	if status.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status.Code)
	}
	if service.statusInput.Active == nil || *service.statusInput.Active {
		t.Fatal("expected status input to deactivate")
	}

	service.updateErr = ErrNotFound
	notFound := performStudentRequest(t, auth.RoleAdmin, service, http.MethodPut, "/students/"+studentID, `{"name":"Outra"}`)
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected not found 404, got %d", notFound.Code)
	}

	service.createErr = ErrInvalidRequest
	invalid := performStudentRequest(t, auth.RoleAdmin, service, http.MethodPost, "/students", `{"name":""}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid request 400, got %d", invalid.Code)
	}
}

func TestStudentHTTPJSONValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"name":`},
		{name: "unknown field", body: `{"name":"Ana Silva","unexpected":"x"}`},
		{name: "trailing json", body: `{"name":"Ana Silva"} {"name":"Outra"}`},
		{name: "oversized", body: `{"name":"` + strings.Repeat("a", httpapi.MaxJSONBodyBytes) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performStudentRequest(t, auth.RoleAdmin, &fakeStudentService{}, http.MethodPost, "/students", tt.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", response.Code)
			}
		})
	}
}

func TestStudentHTTPResourceNotFound(t *testing.T) {
	response := performStudentRequest(t, auth.RoleAdmin, &fakeStudentService{}, http.MethodGet, "/students/", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected not found 404, got %d", response.Code)
	}
}

func performStudentRequest(t *testing.T, role auth.Role, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	user := auth.PublicUser{ID: "user-id", Name: "User", Email: "user@example.com", Role: role}
	return performRequest(t, &user, service, method, path, body)
}

func performUnauthenticatedStudentRequest(t *testing.T, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	return performRequest(t, nil, service, method, path, body)
}

func performRequest(t *testing.T, user *auth.PublicUser, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	authStore := &fakeAuthStore{user: user}
	authService := auth.NewService(authStore)
	authHandler := auth.NewHandler(authService, auth.NewLoginLimiter(), config.AppEnvDevelopment)
	studentHandler := NewHandler(service)

	mux := http.NewServeMux()
	mux.Handle("/students", authHandler.Authenticate(http.HandlerFunc(studentHandler.Collection)))
	mux.Handle("/students/", authHandler.Authenticate(http.HandlerFunc(studentHandler.Resource)))

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != nil {
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "raw-token"})
	}
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	return response
}

type fakeStudentService struct {
	students       []Student
	createdStudent Student
	updatedStudent Student
	statusStudent  Student
	createInput    CreateInput
	updateID       string
	statusInput    StatusInput
	createErr      error
	updateErr      error
}

func (s *fakeStudentService) List(context.Context) ([]Student, error) {
	return s.students, nil
}

func (s *fakeStudentService) Create(_ context.Context, input CreateInput) (Student, error) {
	s.createInput = input
	if s.createErr != nil {
		return Student{}, s.createErr
	}
	return s.createdStudent, nil
}

func (s *fakeStudentService) Update(_ context.Context, id string, input UpdateInput) (Student, error) {
	s.updateID = id
	if s.updateErr != nil {
		return Student{}, s.updateErr
	}
	if s.updatedStudent.ID == "" {
		return Student{ID: id, Name: input.Name, Active: true}, nil
	}
	return s.updatedStudent, nil
}

func (s *fakeStudentService) SetStatus(_ context.Context, id string, input StatusInput) (Student, error) {
	s.statusInput = input
	if s.statusStudent.ID == "" {
		active := false
		if input.Active != nil {
			active = *input.Active
		}
		return Student{ID: id, Name: "Aluno", Active: active}, nil
	}
	return s.statusStudent, nil
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
