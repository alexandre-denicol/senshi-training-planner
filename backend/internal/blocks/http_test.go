package blocks

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

func TestBlockHTTPAuthorization(t *testing.T) {
	if response := performUnauthenticatedBlockRequest(t, &fakeBlockService{}, http.MethodGet, "/blocks", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated status 401, got %d", response.Code)
	}
	if response := performBlockRequest(t, auth.RoleProfessor, &fakeBlockService{}, http.MethodGet, "/blocks", ""); response.Code != http.StatusOK {
		t.Fatalf("expected professor list status 200, got %d", response.Code)
	}
	if response := performBlockRequest(t, auth.RoleAdmin, &fakeBlockService{}, http.MethodGet, "/blocks", ""); response.Code != http.StatusOK {
		t.Fatalf("expected admin status 200, got %d", response.Code)
	}
	service := &fakeBlockService{createdBlock: blockFixture(blockID, "Base", activeCategoryID, true)}
	if response := performBlockRequest(t, auth.RoleProfessor, service, http.MethodPost, "/blocks", `{"name":"Base","categoryId":"`+activeCategoryID+`"}`); response.Code != http.StatusCreated {
		t.Fatalf("expected professor create status 201, got %d", response.Code)
	}
	if response := performBlockRequest(t, auth.RoleProfessor, service, http.MethodPut, "/blocks/"+blockID, `{"name":"Avançado","categoryId":"`+otherCategoryID+`"}`); response.Code != http.StatusOK {
		t.Fatalf("expected professor update status 200, got %d", response.Code)
	}
	if response := performBlockRequest(t, auth.RoleProfessor, service, http.MethodPatch, "/blocks/"+blockID+"/status", `{"active":false}`); response.Code != http.StatusOK {
		t.Fatalf("expected professor status update 200, got %d", response.Code)
	}
	if response := performBlockRequest(t, auth.RoleProfessor, service, http.MethodDelete, "/blocks/"+blockID, ""); response.Code != http.StatusNoContent {
		t.Fatalf("expected professor delete status 204, got %d", response.Code)
	}
}

func TestBlockHTTPCreateDuplicateUpdateStatusDeleteAndNotFound(t *testing.T) {
	service := &fakeBlockService{
		createdBlock: blockFixture(blockID, "Base", activeCategoryID, true),
		updatedBlock: blockFixture(blockID, "Avançado", otherCategoryID, true),
		statusBlock:  blockFixture(blockID, "Avançado", otherCategoryID, false),
	}

	create := performBlockRequest(t, auth.RoleAdmin, service, http.MethodPost, "/blocks", `{"name":"Base","categoryId":"`+activeCategoryID+`"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", create.Code)
	}
	if service.createInput.Name != "Base" || service.createInput.CategoryID != activeCategoryID {
		t.Fatal("expected create input to reach service")
	}
	if strings.Contains(create.Body.String(), "password_hash") {
		t.Fatal("block response leaked unexpected sensitive field")
	}

	service.createErr = ErrNameExists
	duplicate := performBlockRequest(t, auth.RoleAdmin, service, http.MethodPost, "/blocks", `{"name":"base","categoryId":"`+activeCategoryID+`"}`)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("expected duplicate 409, got %d", duplicate.Code)
	}
	service.createErr = nil

	update := performBlockRequest(t, auth.RoleAdmin, service, http.MethodPut, "/blocks/"+blockID, `{"name":"Avançado","categoryId":"`+otherCategoryID+`"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d", update.Code)
	}
	if service.updateID != blockID {
		t.Fatal("expected update route to pass id")
	}

	status := performBlockRequest(t, auth.RoleAdmin, service, http.MethodPatch, "/blocks/"+blockID+"/status", `{"active":false}`)
	if status.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status.Code)
	}
	if service.statusInput.Active == nil || *service.statusInput.Active {
		t.Fatal("expected status input to deactivate")
	}

	remove := performBlockRequest(t, auth.RoleAdmin, service, http.MethodDelete, "/blocks/"+blockID, "")
	if remove.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", remove.Code)
	}
	if service.deleteID != blockID {
		t.Fatal("expected delete route to pass id")
	}

	service.updateErr = ErrNotFound
	notFound := performBlockRequest(t, auth.RoleAdmin, service, http.MethodPut, "/blocks/"+blockID, `{"name":"Outra","categoryId":"`+activeCategoryID+`"}`)
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected not found 404, got %d", notFound.Code)
	}

	service.deleteErr = ErrInUse
	inUse := performBlockRequest(t, auth.RoleAdmin, service, http.MethodDelete, "/blocks/"+blockID, "")
	if inUse.Code != http.StatusConflict {
		t.Fatalf("expected in-use delete 409, got %d", inUse.Code)
	}
	if strings.Contains(inUse.Body.String(), "workout_blocks_block_id_fkey") {
		t.Fatal("expected FK details to stay hidden")
	}
}

func TestBlockHTTPJSONValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"name":`},
		{name: "unknown field", body: `{"name":"Base","categoryId":"` + activeCategoryID + `","unexpected":"x"}`},
		{name: "trailing json", body: `{"name":"Base","categoryId":"` + activeCategoryID + `"} {"name":"Outra"}`},
		{name: "oversized", body: `{"name":"` + strings.Repeat("a", httpapi.MaxJSONBodyBytes) + `","categoryId":"` + activeCategoryID + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performBlockRequest(t, auth.RoleAdmin, &fakeBlockService{}, http.MethodPost, "/blocks", tt.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", response.Code)
			}
		})
	}
}

func performBlockRequest(t *testing.T, role auth.Role, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	user := auth.PublicUser{ID: "user-id", Name: "User", Email: "user@example.com", Role: role}
	return performRequest(t, &user, service, method, path, body)
}

func performUnauthenticatedBlockRequest(t *testing.T, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	return performRequest(t, nil, service, method, path, body)
}

func performRequest(t *testing.T, user *auth.PublicUser, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()

	authStore := &fakeAuthStore{user: user}
	authService := auth.NewService(authStore)
	authHandler := auth.NewHandler(authService, auth.NewLoginLimiter(), config.AppEnvDevelopment)
	blockHandler := NewHandler(service)

	mux := http.NewServeMux()
	mux.Handle("/blocks", authHandler.Authenticate(http.HandlerFunc(blockHandler.Collection)))
	mux.Handle("/blocks/", authHandler.Authenticate(http.HandlerFunc(blockHandler.Resource)))

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != nil {
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "raw-token"})
	}
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	return response
}

type fakeBlockService struct {
	blocks       []Block
	createdBlock Block
	updatedBlock Block
	statusBlock  Block
	createInput  CreateInput
	updateID     string
	statusInput  StatusInput
	deleteID     string
	createErr    error
	updateErr    error
	deleteErr    error
}

func (s *fakeBlockService) List(context.Context) ([]Block, error) {
	return s.blocks, nil
}

func (s *fakeBlockService) Create(_ context.Context, input CreateInput) (Block, error) {
	s.createInput = input
	if s.createErr != nil {
		return Block{}, s.createErr
	}
	return s.createdBlock, nil
}

func (s *fakeBlockService) Update(_ context.Context, id string, input UpdateInput) (Block, error) {
	s.updateID = id
	if s.updateErr != nil {
		return Block{}, s.updateErr
	}
	if s.updatedBlock.ID == "" {
		return blockFixture(id, input.Name, input.CategoryID, true), nil
	}
	return s.updatedBlock, nil
}

func (s *fakeBlockService) SetStatus(_ context.Context, id string, input StatusInput) (Block, error) {
	s.statusInput = input
	if s.statusBlock.ID == "" {
		active := false
		if input.Active != nil {
			active = *input.Active
		}
		return blockFixture(id, "Base", activeCategoryID, active), nil
	}
	return s.statusBlock, nil
}

func (s *fakeBlockService) Delete(_ context.Context, id string) error {
	s.deleteID = id
	if s.deleteErr != nil {
		return s.deleteErr
	}
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
