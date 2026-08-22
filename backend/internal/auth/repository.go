package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) FindUserByEmail(ctx context.Context, email string) (User, error) {
	const query = `
		SELECT id::text, name, email, password_hash, role, active
		FROM users
		WHERE lower(email) = $1
		LIMIT 1`

	var user User
	err := s.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Active,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (s *PostgresStore) CreateSession(ctx context.Context, session Session) error {
	const query = `
		INSERT INTO sessions (id, user_id, token_hash, created_at, last_seen_at, expires_at, absolute_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := s.pool.Exec(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.CreatedAt,
		session.LastSeenAt,
		session.ExpiresAt,
		session.AbsoluteExpiresAt,
	)
	return err
}

func (s *PostgresStore) FindSessionByTokenHash(ctx context.Context, tokenHash string) (Session, PublicUser, error) {
	const query = `
		SELECT
			s.id::text,
			s.user_id::text,
			s.token_hash,
			s.created_at,
			s.last_seen_at,
			s.expires_at,
			s.absolute_expires_at,
			u.id::text,
			u.name,
			u.email,
			u.role
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
			AND u.active = true
		LIMIT 1`

	var session Session
	var user PublicUser
	err := s.pool.QueryRow(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.ExpiresAt,
		&session.AbsoluteExpiresAt,
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Role,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, PublicUser{}, ErrUnauthenticated
	}
	if err != nil {
		return Session{}, PublicUser{}, err
	}

	return session, user, nil
}

func (s *PostgresStore) RefreshSession(ctx context.Context, sessionID string, lastSeenAt time.Time, expiresAt time.Time) error {
	const query = `
		UPDATE sessions
		SET last_seen_at = $2,
			expires_at = $3
		WHERE id = $1`

	_, err := s.pool.Exec(ctx, query, sessionID, lastSeenAt, expiresAt)
	return err
}

func (s *PostgresStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	const query = `DELETE FROM sessions WHERE token_hash = $1`

	_, err := s.pool.Exec(ctx, query, tokenHash)
	return err
}

func (s *PostgresStore) EmailExists(ctx context.Context, email string) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM users WHERE lower(email) = $1)`

	var exists bool
	if err := s.pool.QueryRow(ctx, query, email).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (s *PostgresStore) AdminExists(ctx context.Context) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM users WHERE role = $1)`

	var exists bool
	if err := s.pool.QueryRow(ctx, query, RoleAdmin).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (s *PostgresStore) CreateAdmin(ctx context.Context, user User) error {
	const query = `
		INSERT INTO users (id, name, email, password_hash, role, active)
		VALUES ($1, $2, $3, $4, $5, true)`

	_, err := s.pool.Exec(ctx, query, user.ID, user.Name, user.Email, user.PasswordHash, RoleAdmin)
	return err
}
