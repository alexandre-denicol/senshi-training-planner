package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/alexandre/senshi-training-planner/backend/internal/config"
)

const maxAuthBodyBytes = 16 * 1024

type Handler struct {
	service *Service
	limiter *LoginLimiter
	appEnv  config.AppEnv
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewHandler(service *Service, limiter *LoginLimiter, appEnv config.AppEnv) *Handler {
	return &Handler{
		service: service,
		limiter: limiter,
		appEnv:  appEnv,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/auth/login", h.Login)
	mux.HandleFunc("/auth/logout", h.Logout)
	mux.Handle("/auth/me", requireMethod(http.MethodGet, h.Authenticate(http.HandlerFunc(h.Me))))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var request loginRequest
	if err := readJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	email, err := NormalizeEmail(request.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	limitKey := loginLimitKey(r, email)
	if !h.limiter.Allow(limitKey) {
		writeError(w, http.StatusTooManyRequests, "too many authentication attempts")
		return
	}

	result, err := h.service.Login(r.Context(), email, request.Password)
	if err != nil {
		h.limiter.RecordFailure(limitKey)
		if errors.Is(err, ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	h.limiter.Reset(limitKey)

	http.SetCookie(w, h.sessionCookie(result.SessionToken))
	writeJSON(w, http.StatusOK, result.User)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie(CookieName)
	if err == nil {
		h.service.Logout(r.Context(), cookie.Value)
	}

	http.SetCookie(w, h.expiredSessionCookie())
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) sessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.appEnv == config.AppEnvProduction,
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *Handler) expiredSessionCookie() *http.Cookie {
	cookie := h.sessionCookie("")
	cookie.MaxAge = -1
	return cookie
}

func readJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)
	decoder := json.NewDecoder(r.Body)

	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON value")
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("request body must contain a JSON object")
	}

	objectDecoder := json.NewDecoder(bytes.NewReader(raw))
	objectDecoder.DisallowUnknownFields()
	return objectDecoder.Decode(destination)
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func loginLimitKey(r *http.Request, email string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	return strings.Join([]string{host, email}, "|")
}

func requireMethod(method string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		next.ServeHTTP(w, r)
	})
}
