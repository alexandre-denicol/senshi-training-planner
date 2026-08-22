package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	healthHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	contentType := response.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected application/json content type, got %q", contentType)
	}

	body := strings.TrimSpace(response.Body.String())
	if body != `{"status":"ok"}` {
		t.Fatalf("expected health response body, got %q", body)
	}
}
