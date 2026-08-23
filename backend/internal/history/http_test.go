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
	if !strings.Contains(list.Body.String(), `"participantCount":1`) || strings.Contains(list.Body.String(), "participantNames") || strings.Contains(list.Body.String(), "notes") {
		t.Fatalf("expected compact participant count without names or notes in list, got %s", list.Body.String())
	}

	service.detail = detailWithNotesFixture()
	detail := performHistoryRequest(t, auth.RoleProfessor, service, http.MethodGet, "/history/"+historyID)
	if detail.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d", detail.Code)
	}
	if !strings.Contains(detail.Body.String(), `"blockName":"Bloco A"`) || !strings.Contains(detail.Body.String(), `"categoryName":"Categoria A"`) {
		t.Fatalf("expected snapshot blocks, got %s", detail.Body.String())
	}
	if !strings.Contains(detail.Body.String(), `"participantCount":12`) || !strings.Contains(detail.Body.String(), `"participantNames":["João","Maria"]`) {
		t.Fatalf("expected participant details, got %s", detail.Body.String())
	}
	if !strings.Contains(detail.Body.String(), `"notes":"Turma respondeu bem.\nReduzimos a intensidade."`) {
		t.Fatalf("expected notes in detail, got %s", detail.Body.String())
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

func TestHistoryHTTPCompletionRequestBody(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCode  int
		wantCount *int
		wantNames []string
		wantNotes *string
	}{
		{name: "no body remains valid", body: "", wantCode: http.StatusCreated, wantNames: []string{}},
		{name: "empty object valid", body: `{}`, wantCode: http.StatusCreated, wantNames: []string{}},
		{name: "count only", body: `{"participantCount":12}`, wantCode: http.StatusCreated, wantCount: intPtr(12), wantNames: []string{}},
		{name: "names only", body: `{"participantNames":["João","Maria"]}`, wantCode: http.StatusCreated, wantNames: []string{"João", "Maria"}},
		{name: "count names and notes", body: `{"participantCount":1,"participantNames":["João","Maria"],"notes":"  Boa resposta.\nLinha 2  "}`, wantCode: http.StatusCreated, wantCount: intPtr(1), wantNames: []string{"João", "Maria"}, wantNotes: stringPtr("Boa resposta.\nLinha 2")},
		{name: "zero preserved", body: `{"participantCount":0}`, wantCode: http.StatusCreated, wantCount: intPtr(0), wantNames: []string{}},
		{name: "whitespace notes normalize to null", body: `{"notes":"   \n\t  "}`, wantCode: http.StatusCreated, wantNames: []string{}},
		{name: "exactly max notes accepted", body: `{"notes":"` + strings.Repeat("a", MaxNotesChars) + `"}`, wantCode: http.StatusCreated, wantNames: []string{}, wantNotes: stringPtr(strings.Repeat("a", MaxNotesChars))},
		{name: "oversized notes rejected", body: `{"notes":"` + strings.Repeat("a", MaxNotesChars+1) + `"}`, wantCode: http.StatusBadRequest},
		{name: "decimal count rejected", body: `{"participantCount":2.5}`, wantCode: http.StatusBadRequest},
		{name: "negative count rejected", body: `{"participantCount":-1}`, wantCode: http.StatusBadRequest},
		{name: "blank name rejected", body: `{"participantNames":["   "]}`, wantCode: http.StatusBadRequest},
		{name: "unknown field rejected", body: `{"participantCount":1,"completedAt":"2026-08-23T12:00:00Z"}`, wantCode: http.StatusBadRequest},
		{name: "malformed json rejected", body: `{"participantCount":`, wantCode: http.StatusBadRequest},
		{name: "trailing json rejected", body: `{"participantCount":1} {"participantCount":2}`, wantCode: http.StatusBadRequest},
		{name: "non object rejected", body: `[]`, wantCode: http.StatusBadRequest},
		{name: "oversized body rejected", body: `{"participantNames":["` + strings.Repeat("a", 20*1024) + `"]}`, wantCode: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeHistoryService{detail: detailFixture()}
			response := performCompleteRequestWithBody(t, auth.RoleAdmin, service, http.MethodPost, "/schedule/"+scheduleEntryID+"/complete", tt.body)
			if response.Code != tt.wantCode {
				t.Fatalf("expected status %d, got %d with body %s", tt.wantCode, response.Code, response.Body.String())
			}
			if tt.wantCode != http.StatusCreated {
				return
			}
			if (tt.wantCount == nil) != (service.details.ParticipantCount == nil) {
				t.Fatalf("expected count %#v, got %#v", tt.wantCount, service.details.ParticipantCount)
			}
			if tt.wantCount != nil && *tt.wantCount != *service.details.ParticipantCount {
				t.Fatalf("expected count %d, got %d", *tt.wantCount, *service.details.ParticipantCount)
			}
			if len(service.details.ParticipantNames) != len(tt.wantNames) {
				t.Fatalf("expected names %#v, got %#v", tt.wantNames, service.details.ParticipantNames)
			}
			for index, want := range tt.wantNames {
				if service.details.ParticipantNames[index] != want {
					t.Fatalf("expected names %#v, got %#v", tt.wantNames, service.details.ParticipantNames)
				}
			}
			if (tt.wantNotes == nil) != (service.details.Notes == nil) {
				t.Fatalf("expected notes %#v, got %#v", tt.wantNotes, service.details.Notes)
			}
			if tt.wantNotes != nil && *tt.wantNotes != *service.details.Notes {
				t.Fatalf("expected notes %q, got %q", *tt.wantNotes, *service.details.Notes)
			}
		})
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
	return performCompleteAuthenticatedRequest(t, &user, service, method, path, "")
}

func performCompleteRequestWithBody(t *testing.T, role auth.Role, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	user := auth.PublicUser{ID: userID, Name: "User", Email: "user@example.com", Role: role}
	return performCompleteAuthenticatedRequest(t, &user, service, method, path, body)
}

func performUnauthenticatedCompleteRequest(t *testing.T, service ServiceAPI, method string, path string) *httptest.ResponseRecorder {
	t.Helper()
	return performCompleteAuthenticatedRequest(t, nil, service, method, path, "")
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

func performCompleteAuthenticatedRequest(t *testing.T, user *auth.PublicUser, service ServiceAPI, method string, path string, body string) *httptest.ResponseRecorder {
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

	var requestBody *strings.Reader
	if body == "" {
		requestBody = strings.NewReader("")
	} else {
		requestBody = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, requestBody)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
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
	details     CompletionDetails
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

func (s *fakeHistoryService) Complete(_ context.Context, _ string, completedBy auth.PublicUser, details CompletionDetails) (Detail, error) {
	normalizedDetails, err := validateCompletionDetails(details)
	if err != nil {
		return Detail{}, err
	}
	s.completedBy = completedBy
	s.details = normalizedDetails
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
