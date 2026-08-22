package auth

import (
	"context"
	"errors"
	"time"
)

type Service struct {
	store Store
	now   func() time.Time
}

type LoginResult struct {
	User         PublicUser
	SessionToken string
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   time.Now,
	}
}

func (s *Service) Login(ctx context.Context, email string, password string) (LoginResult, error) {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err := ValidatePassword(password); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	user, err := s.store.FindUserByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, err
	}
	if !user.Active {
		return LoginResult{}, ErrInvalidCredentials
	}

	ok, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !ok {
		return LoginResult{}, ErrInvalidCredentials
	}

	sessionToken, err := GenerateSessionToken()
	if err != nil {
		return LoginResult{}, err
	}
	sessionID, err := NewUUID()
	if err != nil {
		return LoginResult{}, err
	}

	now := s.now().UTC()
	session := Session{
		ID:                sessionID,
		UserID:            user.ID,
		TokenHash:         HashSessionToken(sessionToken),
		CreatedAt:         now,
		LastSeenAt:        now,
		ExpiresAt:         now.Add(IdleTimeout),
		AbsoluteExpiresAt: now.Add(AbsoluteTimeout),
	}
	if err := s.store.CreateSession(ctx, session); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		User:         publicUser(user),
		SessionToken: sessionToken,
	}, nil
}

func (s *Service) Logout(ctx context.Context, sessionToken string) {
	if sessionToken == "" {
		return
	}

	_ = s.store.DeleteSessionByTokenHash(ctx, HashSessionToken(sessionToken))
}

func (s *Service) CurrentUser(ctx context.Context, sessionToken string) (PublicUser, error) {
	if sessionToken == "" {
		return PublicUser{}, ErrUnauthenticated
	}

	tokenHash := HashSessionToken(sessionToken)
	session, user, err := s.store.FindSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return PublicUser{}, ErrUnauthenticated
	}

	now := s.now().UTC()
	if !now.Before(session.ExpiresAt) || !now.Before(session.AbsoluteExpiresAt) {
		_ = s.store.DeleteSessionByTokenHash(ctx, tokenHash)
		return PublicUser{}, ErrUnauthenticated
	}

	if now.Sub(session.LastSeenAt) >= SessionRefreshInterval {
		expiresAt := now.Add(IdleTimeout)
		if expiresAt.After(session.AbsoluteExpiresAt) {
			expiresAt = session.AbsoluteExpiresAt
		}
		if err := s.store.RefreshSession(ctx, session.ID, now, expiresAt); err != nil {
			return PublicUser{}, err
		}
	}

	return user, nil
}

func IsInvalidCredentials(err error) bool {
	return errors.Is(err, ErrInvalidCredentials)
}
