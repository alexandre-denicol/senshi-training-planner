package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/config"
)

func TestReadJSONValidation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid JSON",
			body:    `{"email":"admin@example.com","password":"uma senha longa e segura"}   `,
			wantErr: false,
		},
		{
			name:    "unknown field",
			body:    `{"email":"admin@example.com","password":"uma senha longa e segura","extra":true}`,
			wantErr: true,
		},
		{
			name:    "malformed JSON",
			body:    `{"email":"admin@example.com","password":`,
			wantErr: true,
		},
		{
			name:    "second JSON object",
			body:    `{"email":"admin@example.com","password":"uma senha longa e segura"} {"email":"other@example.com"}`,
			wantErr: true,
		},
		{
			name:    "non-object JSON",
			body:    `null`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(tt.body))
			response := httptest.NewRecorder()
			var destination loginRequest

			err := readJSON(response, request, &destination)
			if tt.wantErr && err == nil {
				t.Fatal("expected readJSON error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected valid JSON, got %v", err)
			}
		})
	}
}

func TestReadJSONRejectsOversizedBody(t *testing.T) {
	body := `{"email":"admin@example.com","password":"` + strings.Repeat("a", maxAuthBodyBytes) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	response := httptest.NewRecorder()
	var destination loginRequest

	if err := readJSON(response, request, &destination); err == nil {
		t.Fatal("expected oversized body error")
	}
}

func TestAuthenticationMiddleware(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{
		session: Session{
			ID:                "session-id",
			TokenHash:         HashSessionToken("raw-token"),
			LastSeenAt:        now,
			ExpiresAt:         now.Add(IdleTimeout),
			AbsoluteExpiresAt: now.Add(AbsoluteTimeout),
		},
		sessionUser: PublicUser{ID: "user-id", Name: "Admin", Email: "admin@example.com", Role: RoleAdmin},
	}
	service := NewService(store)
	service.now = func() time.Time { return now }
	handler := NewHandler(service, NewLoginLimiter(), config.AppEnvDevelopment)

	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	request.AddCookie(&http.Cookie{Name: CookieName, Value: "raw-token"})
	response := httptest.NewRecorder()

	handler.Authenticate(http.HandlerFunc(handler.Me)).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "password") {
		t.Fatal("expected response to omit password fields")
	}
}

func TestSecureCookieBehaviorInProduction(t *testing.T) {
	response := login(t, config.AppEnvProduction)
	cookie := response.Result().Cookies()[0]

	if !cookie.HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}
	if !cookie.Secure {
		t.Fatal("expected Secure cookie in production")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("expected SameSite=Lax cookie")
	}
	if cookie.Domain != "" {
		t.Fatal("expected host-only cookie without Domain")
	}
}

func TestDevelopmentCookieBehavior(t *testing.T) {
	response := login(t, config.AppEnvDevelopment)
	cookie := response.Result().Cookies()[0]

	if cookie.Secure {
		t.Fatal("expected non-secure cookie in development")
	}
}

func TestLogoutCookieExpiration(t *testing.T) {
	handler := NewHandler(NewService(&fakeStore{}), NewLoginLimiter(), config.AppEnvProduction)
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: CookieName, Value: "raw-token"})
	response := httptest.NewRecorder()

	handler.Logout(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.Code)
	}
	cookie := response.Result().Cookies()[0]
	if cookie.MaxAge != -1 {
		t.Fatalf("expected expired cookie, got MaxAge %d", cookie.MaxAge)
	}
	if !cookie.HttpOnly || !cookie.Secure {
		t.Fatal("expected expired production cookie to preserve security flags")
	}
}

func login(t *testing.T, appEnv config.AppEnv) *httptest.ResponseRecorder {
	t.Helper()

	passwordHash, err := hashPassword("uma senha longa e segura", testArgonParams)
	if err != nil {
		t.Fatalf("expected password hash, got %v", err)
	}
	store := &fakeStore{
		user: User{
			ID:           "user-id",
			Name:         "Admin",
			Email:        "admin@example.com",
			PasswordHash: passwordHash,
			Role:         RoleAdmin,
			Active:       true,
		},
	}
	handler := NewHandler(NewService(store), NewLoginLimiter(), appEnv)
	body := strings.NewReader(`{"email":"admin@example.com","password":"uma senha longa e segura"}`)
	request := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	response := httptest.NewRecorder()

	handler.Login(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	return response
}
