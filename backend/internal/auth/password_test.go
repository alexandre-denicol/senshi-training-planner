package auth

import (
	"errors"
	"strings"
	"testing"
)

var testArgonParams = argonParams{
	memory:      64,
	iterations:  1,
	parallelism: 1,
	saltBytes:   16,
	keyBytes:    32,
}

func TestPasswordHashingAndVerification(t *testing.T) {
	hash, err := hashPassword("uma senha longa e segura", testArgonParams)
	if err != nil {
		t.Fatalf("expected password hash, got %v", err)
	}

	ok, err := VerifyPassword("uma senha longa e segura", hash)
	if err != nil {
		t.Fatalf("expected password verification, got %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}
}

func TestWrongPasswordRejected(t *testing.T) {
	hash, err := hashPassword("uma senha longa e segura", testArgonParams)
	if err != nil {
		t.Fatalf("expected password hash, got %v", err)
	}

	ok, err := VerifyPassword("outra senha longa", hash)
	if err != nil {
		t.Fatalf("expected password verification, got %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to be rejected")
	}
}

func TestPasswordPolicyBoundaries(t *testing.T) {
	if err := ValidatePassword(strings.Repeat("a", MinPasswordRunes-1)); !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected short password error, got %v", err)
	}

	if err := ValidatePassword(strings.Repeat("a", MinPasswordRunes)); err != nil {
		t.Fatalf("expected minimum length password to pass, got %v", err)
	}

	if err := ValidatePassword(strings.Repeat("a", MaxPasswordBytes+1)); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("expected long password error, got %v", err)
	}

	passphrase := " senha com espaços e ç "
	if err := ValidatePassword(passphrase); err != nil {
		t.Fatalf("expected unicode passphrase with whitespace to pass, got %v", err)
	}
}
