package history

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

func TestHistoryHTTPAuthorization(t *testing.T) {
	if response := performUnauthenticatedHistoryRequest(t, &fakeHistoryService{}, http.MethodGet, "/history?from=2026-08-01&to=2026-08-31"); response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated history 401, got %d", response.Code)
	}
	if response := performHistoryRequest(t, auth.RoleProfessor, &fakeHistoryService{items: []ListItem{listItemFixture()}}, http.MethodGet, "/history?from=2026-08-01&to=2026-08-31"); response.Code != http.StatusOK {
		t.Fatalf("expected professor history 200, got %d", response.Code)
	}
	if response := performHistoryRequest(t, auth.RoleAdmin, &fakeHistoryService{items: []ListItem{listItemFixture()}}, http.MethodGet, "/history?from=2026-08-01&to=2026-08-31"); response.Code != http.StatusOK {
		t.Fatalf("expected admin history 200, got %d", response.Code)
	}
	if response := performUnauthenticatedCompleteRequest(t, &fakeHistoryService{}, http.MethodPost, "/schedule/"+scheduleEntryID+"/complete"); response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated complete 401, got %d", response.Code)
	}
	if response := performCompleteRequest(t, auth.RoleProfessor, &fakeHistoryService{detail: detailFixture()}, http.MethodPost, "/schedule/"+scheduleEntryID+"/complete"); response.Code != http.StatusCreated {
		t.Fatalf("expected professor complete 201, got %d", response.Code)
	}
	if response := performCompleteRequest(t, auth.RoleAdmin, &fakeHistoryService{detail: detailFixture()}, http.MethodPost, "/schedule/"+scheduleEntryID+"/complete"); response.Code != http.StatusCreated {
		t.Fatalf("expected admin complete 201, got %d", response.Code)
	}
}

func TestHistoryHTTPListDetailAndCompletionErrors(t *testing.T) {
	service := &fakeHistoryService{
		items:  []ListItem{listItemFixture()},
		detail: detailFixture(),
	}

	list := performHistoryRequest(t, auth.RoleAdmin, service, http.MethodGet, "/history?from=2026-08-01&to=2026-08-31")
	if list.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", list.Code)
	}
	if service.from != "2026-08-01" || service.to != "2026-08-31" {
		t.Fatalf("expected range to reach service, got %s %s", service.from, service.to)
	}
	if !strings.Contains(list.Body.String(), `"blockCount":2`) {
		t.Fatalf("expected compact history list, got %s", list.Body.String())
	}

	detail := performHistoryRequest(t, auth.RoleProfessor, service, http.MethodGet, "/history/"+historyID)
	if detail.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d", detail.Code)
	}
	if !strings.Contains(detail.Body.String(), `"blockName":"Bloco A"`) || !strings.Contains(detail.Body.String(), `"categoryName":"Categoria A"`) {
		t.Fatalf("expected snapshot blocks, got %s", detail.Body.String())
	}

	service.listErr = ErrInvalidRequest
	invalidRange := performHistoryRequest(t, auth.RoleAdmin, service, http.MethodGet, "/history?from=2026-09-01&to=2026-08-31")
	if invalidRange.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid range 400, got %d", invalidRange.Code)
	}
	service.listErr = nil

	service.getErr = ErrNotFound
	notFound := performHistoryRequest(t, auth.RoleAdmin, service, http.MethodGet, "/history/"+historyID)
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected missing history 404, got %d", notFound.Code)
	}
	service.getErr = nil

	service.completeErr = ErrScheduleNotFound
	missingSchedule := performCompleteRequest(t, auth.RoleAdmin, service, http.MethodPost, "/schedule/"+scheduleEntryID+"/complete")
	if missingSchedule.Code != http.StatusNotFound {
		t.Fatalf("expected missing schedule 404, got %d", missingSchedule.Code)
	}

	service.completeErr = ErrAlreadyCompleted
	duplicate := performCompleteRequest(t, auth.RoleProfessor, service, http.MethodPost, "/schedule/"+scheduleEntryID+"/complete")
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("expected duplicate completion 409, got %d", duplicate.Code)
	}
}

func performHistoryRequest(t *testing.T, role auth.Role, service ServiceAPI, method string, path string) *httptest.ResponseRecorder {
	t.Helper()
	user := auth.PublicUser{ID: userID, Name: "User", Email: "user@example.com", Role: role}
	return performAuthenticatedRequest(t, &user, service, method, path)
}

func performUnauthenticatedHistoryRequest(t *testing.T, service ServiceAPI, method string, path string) *httptest.ResponseRecorder {
	t.Helper()
	return performAuthenticatedRequest(t, nil, service, method, path)
}

func performCompleteRequest(t *testing.T, role auth.Role, service ServiceAPI, method string, path string) *httptest.ResponseRecorder {
	t.Helper()
	user := auth.PublicUser{ID: userID, Name: "User", Email: "user@example.com", Role: role}
	return performCompleteAuthenticatedRequest(t, &user, service, method, path)
}

func performUnauthenticatedCompleteRequest(t *testing.T, service ServiceAPI, method string, path string) *httptest.ResponseRecorder {
	t.Helper()
	return performCompleteAuthenticatedRequest(t, nil, service, method, path)
}

func performAuthenticatedRequest(t *testing.T, user *auth.PublicUser, service ServiceAPI, method string, path string) *httptest.ResponseRecorder {
	t.Helper()

	authHandler, historyHandler := testHandlers(user, service)
	mux := http.NewServeMux()
	mux.Handle("/history", authHandler.Authenticate(http.HandlerFunc(historyHandler.Collection)))
	mux.Handle("/history/", authHandler.Authenticate(http.HandlerFunc(historyHandler.Resource)))

	request := httptest.NewRequest(method, path, nil)
	if user != nil {
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "raw-token"})
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func performCompleteAuthenticatedRequest(t *testing.T, user *auth.PublicUser, service ServiceAPI, method string, path string) *httptest.ResponseRecorder {
	t.Helper()

	authHandler, historyHandler := testHandlers(user, service)
	mux := http.NewServeMux()
	mux.Handle("/schedule/", authHandler.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		segments := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/schedule/"), "/"), "/")
		if len(segments) == 2 && segments[1] == "complete" {
			historyHandler.CompleteSchedule(w, r, segments[0])
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})))

	request := httptest.NewRequest(method, path, nil)
	if user != nil {
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "raw-token"})
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func testHandlers(user *auth.PublicUser, service ServiceAPI) (*auth.Handler, *Handler) {
	authStore := &fakeAuthStore{user: user}
	authService := auth.NewService(authStore)
	authHandler := auth.NewHandler(authService, auth.NewLoginLimiter(), config.AppEnvDevelopment)
	return authHandler, NewHandler(service)
}

type fakeHistoryService struct {
	items       []ListItem
	detail      Detail
	completedBy auth.PublicUser
	from        string
	to          string
	listErr     error
	getErr      error
	completeErr error
}

func (s *fakeHistoryService) List(_ context.Context, from string, to string) ([]ListItem, error) {
	s.from = from
	s.to = to
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.items, nil
}

func (s *fakeHistoryService) Get(context.Context, string) (Detail, error) {
	if s.getErr != nil {
		return Detail{}, s.getErr
	}
	return s.detail, nil
}

func (s *fakeHistoryService) Complete(_ context.Context, _ string, completedBy auth.PublicUser) (Detail, error) {
	s.completedBy = completedBy
	if s.completeErr != nil {
		return Detail{}, s.completeErr
	}
	return s.detail, nil
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
