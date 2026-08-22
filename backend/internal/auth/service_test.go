package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	user                User
	findUserErr         error
	session             Session
	sessionUser         PublicUser
	findSessionErr      error
	createdSession      Session
	deletedTokenHash    string
	refreshedSessionID  string
	refreshedLastSeenAt time.Time
	refreshedExpiresAt  time.Time
	emailExists         bool
	adminExists         bool
	createdAdmin        User
}

func (s *fakeStore) FindUserByEmail(context.Context, string) (User, error) {
	if s.findUserErr != nil {
		return User{}, s.findUserErr
	}
	return s.user, nil
}

func (s *fakeStore) CreateSession(_ context.Context, session Session) error {
	s.createdSession = session
	return nil
}

func (s *fakeStore) FindSessionByTokenHash(context.Context, string) (Session, PublicUser, error) {
	if s.findSessionErr != nil {
		return Session{}, PublicUser{}, s.findSessionErr
	}
	return s.session, s.sessionUser, nil
}

func (s *fakeStore) RefreshSession(_ context.Context, sessionID string, lastSeenAt time.Time, expiresAt time.Time) error {
	s.refreshedSessionID = sessionID
	s.refreshedLastSeenAt = lastSeenAt
	s.refreshedExpiresAt = expiresAt
	return nil
}

func (s *fakeStore) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	s.deletedTokenHash = tokenHash
	return nil
}

func (s *fakeStore) EmailExists(context.Context, string) (bool, error) {
	return s.emailExists, nil
}

func (s *fakeStore) AdminExists(context.Context) (bool, error) {
	return s.adminExists, nil
}

func (s *fakeStore) CreateAdmin(_ context.Context, user User) error {
	s.createdAdmin = user
	return nil
}

func TestExpiredSessionBehavior(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{
		session: Session{
			ID:                "session-id",
			TokenHash:         HashSessionToken("raw-token"),
			LastSeenAt:        now.Add(-time.Hour),
			ExpiresAt:         now.Add(-time.Minute),
			AbsoluteExpiresAt: now.Add(time.Hour),
		},
		sessionUser: PublicUser{ID: "user-id"},
	}
	service := NewService(store)
	service.now = func() time.Time { return now }

	_, err := service.CurrentUser(context.Background(), "raw-token")
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected unauthenticated error, got %v", err)
	}

	if store.deletedTokenHash != HashSessionToken("raw-token") {
		t.Fatal("expected expired session to be deleted")
	}
}

func TestLoginFailureIsGeneric(t *testing.T) {
	passwordHash, err := hashPassword("uma senha longa e segura", testArgonParams)
	if err != nil {
		t.Fatalf("expected password hash, got %v", err)
	}

	tests := []fakeStore{
		{findUserErr: ErrInvalidCredentials},
		{user: User{Active: true, PasswordHash: passwordHash}},
		{user: User{Active: false, PasswordHash: passwordHash}},
	}

	for _, store := range tests {
		service := NewService(&store)
		_, err := service.Login(context.Background(), "admin@example.com", "outra senha longa")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected generic invalid credentials error, got %v", err)
		}
	}
}
