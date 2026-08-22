package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const SessionTokenBytes = 32

func GenerateSessionToken() (string, error) {
	token := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(token), nil
}

func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
