package categories

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

func TestCategoryHTTPAuthorization(t *testing.T) {
	if response := performUnauthenticatedCategoryRequest(t, &fakeCategoryService{}, http.MethodGet, "/categories", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated status 401, got %d", response.Code)
	}
	if response := performCategoryRequest(t, auth.RoleProfessor, &fakeCategoryService{}, http.MethodGet, "/categories", ""); response.Code != http.StatusForbidden {
		t.Fatalf("expected professor status 403, got %d", response.Code)
	}
	if response := performCategoryRequest(t, auth.RoleAdmin, &fakeCategoryService{}, http.MethodGet, "/categories", ""); response.Code != http.StatusOK {
		t.Fatalf("expected admin status 200, got %d", response.Code)
	}
}

func TestCategoryHTTPCreateDuplicateUpdateStatusDeleteAndNotFound(t *testing.T) {
	service := &fakeCategoryService{
		createdCategory: Category{ID: categoryID, Name: "Técnica", Active: true},
		updatedCategory: Category{ID: categoryID, Name: "Mobilidade", Active: true},
		statusCategory:  Category{ID: categoryID, Name: "Mobilidade", Active: false},
	}

	create := performCategoryRequest(t, auth.RoleAdmin, service, http.MethodPost, "/categories", `{"name":"Técnica"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", create.Code)
	}
	if service.createInput.Name != "Técnica" {
		t.Fatal("expected create input to reach service")
	}

	service.createErr = ErrNameExists
	duplicate := performCategoryRequest(t, auth.RoleAdmin, service, http.MethodPost, "/categories", `{"name":"técnica"}`)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("expected duplicate 409, got %d", duplicate.Code)
	}
	service.createErr = nil

	update := performCategoryRequest(t, auth.RoleAdmin, service, http.MethodPut, "/categories/"+categoryID, `{"name":"Mobilidade"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d", update.Code)
	}
	if service.updateID != categoryID {
		t.Fatal("expected update route to pass id")
	}

	status := performCategoryRequest(t, auth.RoleAdmin, service, http.MethodPatch, "/categories/"+categoryID+"/status", `{"active":false}`)
	if status.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status.Code)
	}
	if service.statusInput.Active == nil || *service.statusInput.Active {
		t.Fatal("expected status input to deactivate")
	}

	remove := performCategoryRequest(t, auth.RoleAdmin, service, http.MethodDelete, "/categories/"+categoryID, "")
	if remove.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", remove.Code)
	}
	if service.deleteID != categoryID {
		t.Fatal("expected delete route to pass id")
	}

	service.updateErr = ErrNotFound
	notFound := performCategoryRequest(t, auth.RoleAdmin, service, http.MethodPut, "/categories/"+categoryID, `{"name":"Outra"}`)
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected not found 404, got %d", notFound.Code)
	}
}

func TestCategoryHTTPJSONValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"name":`},
		{name: "unknown field", body: `{"name":"Técnica","description":"x"}`},
		{name: "trailing json", body: `{"name":"Técnica"} {"name":"Outra"}`},
		{name: "oversized", body: `{"name":"` + strings.Repeat("a", httpapi.MaxJSONBodyBytes) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performCategoryRequest(t, auth.RoleAdmin, &fakeCategoryService{}, http.MethodPost, "/categories", tt.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", response.Code)
			}
		})
	}
}

func performCategoryRequest(t *testing.T, role auth.Role, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	user := auth.PublicUser{ID: "user-id", Name: "User", Email: "user@example.com", Role: role}
	return performRequest(t, &user, service, method, path, body)
}

func performUnauthenticatedCategoryRequest(t *testing.T, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	return performRequest(t, nil, service, method, path, body)
}

func performRequest(t *testing.T, user *auth.PublicUser, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	authStore := &fakeAuthStore{user: user}
	authService := auth.NewService(authStore)
	authHandler := auth.NewHandler(authService, auth.NewLoginLimiter(), config.AppEnvDevelopment)
	categoryHandler := NewHandler(service)

	mux := http.NewServeMux()
	adminOnly := func(next http.Handler) http.Handler {
		return authHandler.Authenticate(auth.RequireAdmin(next))
	}
	mux.Handle("/categories", adminOnly(http.HandlerFunc(categoryHandler.Collection)))
	mux.Handle("/categories/", adminOnly(http.HandlerFunc(categoryHandler.Resource)))

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != nil {
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "raw-token"})
	}
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	return response
}

type fakeCategoryService struct {
	categories      []Category
	createdCategory Category
	updatedCategory Category
	statusCategory  Category
	createInput     CreateInput
	updateID        string
	statusInput     StatusInput
	deleteID        string
	createErr       error
	updateErr       error
}

func (s *fakeCategoryService) List(context.Context) ([]Category, error) {
	return s.categories, nil
}

func (s *fakeCategoryService) Create(_ context.Context, input CreateInput) (Category, error) {
	s.createInput = input
	if s.createErr != nil {
		return Category{}, s.createErr
	}
	return s.createdCategory, nil
}

func (s *fakeCategoryService) Update(_ context.Context, id string, input UpdateInput) (Category, error) {
	s.updateID = id
	if s.updateErr != nil {
		return Category{}, s.updateErr
	}
	if s.updatedCategory.ID == "" {
		return Category{ID: id, Name: input.Name, Active: true}, nil
	}
	return s.updatedCategory, nil
}

func (s *fakeCategoryService) SetStatus(_ context.Context, id string, input StatusInput) (Category, error) {
	s.statusInput = input
	if s.statusCategory.ID == "" {
		active := false
		if input.Active != nil {
			active = *input.Active
		}
		return Category{ID: id, Name: "Categoria", Active: active}, nil
	}
	return s.statusCategory, nil
}

func (s *fakeCategoryService) Delete(_ context.Context, id string) error {
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
