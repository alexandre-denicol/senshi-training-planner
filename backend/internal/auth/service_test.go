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
	changedPasswordUser string
	changedPasswordHash string
	usersByID           map[string]User
	lookedUpEmail       string
	lookedUpUserID      string
}

func (s *fakeStore) FindUserByEmail(_ context.Context, email string) (User, error) {
	s.lookedUpEmail = email
	if s.findUserErr != nil {
		return User{}, s.findUserErr
	}
	if s.user.Email == "" || email != s.user.Email {
		return User{}, ErrInvalidCredentials
	}
	return s.user, nil
}

func (s *fakeStore) FindUserByID(_ context.Context, userID string) (User, error) {
	s.lookedUpUserID = userID
	if user, ok := s.usersByID[userID]; ok {
		return user, nil
	}
	if s.user.ID == userID {
		return s.user, nil
	}
	return User{}, ErrUnauthenticated
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

func (s *fakeStore) ChangePassword(_ context.Context, userID string, passwordHash string) error {
	s.changedPasswordUser = userID
	s.changedPasswordHash = passwordHash
	return nil
}

func TestChangePasswordForBothRolesInvalidatesThroughStore(t *testing.T) {
	for _, role := range []Role{RoleAdmin, RoleProfessor} {
		t.Run(string(role), func(t *testing.T) {
			oldHash, err := hashPassword("senha antiga segura", testArgonParams)
			if err != nil {
				t.Fatal(err)
			}
			store := &fakeStore{usersByID: map[string]User{
				"user-id": {ID: "user-id", Email: "User@Example.com", PasswordHash: oldHash, Role: role, Active: true},
			}}
			service := NewService(store)
			err = service.ChangePassword(context.Background(), PublicUser{ID: "user-id", Email: "USER@example.com", Role: role}, "senha antiga segura", "nova1234")
			if err != nil {
				t.Fatalf("expected password change, got %v", err)
			}
			if store.changedPasswordUser != "user-id" || store.changedPasswordHash == "" {
				t.Fatal("expected authenticated user hash update")
			}
			if store.lookedUpUserID != "user-id" {
				t.Fatalf("expected lookup by authenticated user ID, got %q", store.lookedUpUserID)
			}
			if store.lookedUpEmail != "" {
				t.Fatalf("expected password change not to resolve by email, got %q", store.lookedUpEmail)
			}
			if ok, verifyErr := VerifyPassword("nova1234", store.changedPasswordHash); verifyErr != nil || !ok {
				t.Fatal("expected new password to verify")
			}
			if ok, verifyErr := VerifyPassword("senha antiga segura", store.changedPasswordHash); verifyErr != nil || ok {
				t.Fatal("expected old password to fail")
			}
		})
	}
}

func TestChangePasswordRejectsWrongCurrentAndShortNewPassword(t *testing.T) {
	oldHash, err := hashPassword("senha antiga segura", testArgonParams)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{usersByID: map[string]User{
		"user-id": {ID: "user-id", Email: "User@Example.com", PasswordHash: oldHash, Active: true},
	}}
	service := NewService(store)
	user := PublicUser{ID: "user-id", Email: "USER@example.com"}
	if err := service.ChangePassword(context.Background(), user, "senha incorreta", "nova1234"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected generic credential error, got %v", err)
	}
	if err := service.ChangePassword(context.Background(), user, "senha antiga segura", "curta"); !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected short-password error, got %v", err)
	}
	if store.changedPasswordUser != "" {
		t.Fatal("expected rejected password changes not to update any account")
	}
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
