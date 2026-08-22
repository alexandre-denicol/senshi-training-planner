package auth

import (
	"encoding/base64"
	"testing"
)

func TestSessionTokenGeneration(t *testing.T) {
	token, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("expected session token, got %v", err)
	}

	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("expected base64url token, got %v", err)
	}
	if len(decoded) != SessionTokenBytes {
		t.Fatalf("expected %d random bytes, got %d", SessionTokenBytes, len(decoded))
	}
}

func TestTokenHashing(t *testing.T) {
	first := HashSessionToken("token")
	second := HashSessionToken("token")
	other := HashSessionToken("other")

	if first != second {
		t.Fatal("expected deterministic token hash")
	}
	if first == other {
		t.Fatal("expected different tokens to hash differently")
	}
	if len(first) != 64 {
		t.Fatalf("expected SHA-256 hex hash length 64, got %d", len(first))
	}
}
