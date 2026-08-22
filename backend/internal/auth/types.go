package auth

import (
	"context"
	"errors"
	"time"
)

const (
	CookieName             = "session"
	IdleTimeout            = 30 * time.Minute
	AbsoluteTimeout        = 8 * time.Hour
	SessionRefreshInterval = 5 * time.Minute
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthenticated    = errors.New("authentication required")
)

type Role string

const (
	RoleAdmin     Role = "ADMIN"
	RoleProfessor Role = "PROFESSOR"
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         Role
	Active       bool
}

type PublicUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  Role   `json:"role"`
}

type Session struct {
	ID                string
	UserID            string
	TokenHash         string
	CreatedAt         time.Time
	LastSeenAt        time.Time
	ExpiresAt         time.Time
	AbsoluteExpiresAt time.Time
}

type Store interface {
	FindUserByEmail(ctx context.Context, email string) (User, error)
	CreateSession(ctx context.Context, session Session) error
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (Session, PublicUser, error)
	RefreshSession(ctx context.Context, sessionID string, lastSeenAt time.Time, expiresAt time.Time) error
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
	EmailExists(ctx context.Context, email string) (bool, error)
	AdminExists(ctx context.Context) (bool, error)
	CreateAdmin(ctx context.Context, user User) error
}

func publicUser(user User) PublicUser {
	return PublicUser{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Role:  user.Role,
	}
}
