package auth

import (
	"errors"
	"net/mail"
	"strings"
)

var ErrInvalidEmail = errors.New("email is invalid")

func NormalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", ErrInvalidEmail
	}

	parsed, err := mail.ParseAddress(normalized)
	if err != nil || parsed.Address != normalized {
		return "", ErrInvalidEmail
	}

	return normalized, nil
}
