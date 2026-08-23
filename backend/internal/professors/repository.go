package professors

import (
	"context"
	"errors"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListProfessors(ctx context.Context) ([]Professor, error) {
	const query = `
		SELECT id::text, name, email, active, created_at, updated_at
		FROM users
		WHERE role = $1
		ORDER BY name ASC, email ASC`

	rows, err := s.pool.Query(ctx, query, auth.RoleProfessor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	professors := []Professor{}
	for rows.Next() {
		var professor Professor
		if err := rows.Scan(
			&professor.ID,
			&professor.Name,
			&professor.Email,
			&professor.Active,
			&professor.CreatedAt,
			&professor.UpdatedAt,
		); err != nil {
			return nil, err
		}
		professors = append(professors, professor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return professors, nil
}

func (s *PostgresStore) CreateProfessor(ctx context.Context, professor NewProfessor) (Professor, error) {
	const query = `
		INSERT INTO users (id, name, email, password_hash, role, active)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id::text, name, email, active, created_at, updated_at`

	created, err := scanProfessor(
		s.pool.QueryRow(
			ctx,
			query,
			professor.ID,
			professor.Name,
			professor.Email,
			professor.PasswordHash,
			auth.RoleProfessor,
		),
	)
	if uniqueViolation(err) {
		return Professor{}, ErrEmailExists
	}
	if err != nil {
		return Professor{}, err
	}

	return created, nil
}

func (s *PostgresStore) UpdateProfessor(ctx context.Context, id string, name string, email string) (Professor, error) {
	const query = `
		UPDATE users
		SET name = $2,
			email = $3
		WHERE id = $1
			AND role = $4
		RETURNING id::text, name, email, active, created_at, updated_at`

	updated, err := scanProfessor(s.pool.QueryRow(ctx, query, id, name, email, auth.RoleProfessor))
	if errors.Is(err, pgx.ErrNoRows) {
		return Professor{}, ErrNotFound
	}
	if uniqueViolation(err) {
		return Professor{}, ErrEmailExists
	}
	if err != nil {
		return Professor{}, err
	}

	return updated, nil
}

func (s *PostgresStore) SetProfessorStatus(ctx context.Context, id string, active bool) (Professor, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Professor{}, err
	}
	defer tx.Rollback(ctx)

	const updateQuery = `
		UPDATE users
		SET active = $2
		WHERE id = $1
			AND role = $3
		RETURNING id::text, name, email, active, created_at, updated_at`

	professor, err := scanProfessor(tx.QueryRow(ctx, updateQuery, id, active, auth.RoleProfessor))
	if errors.Is(err, pgx.ErrNoRows) {
		return Professor{}, ErrNotFound
	}
	if err != nil {
		return Professor{}, err
	}

	if !active {
		if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, id); err != nil {
			return Professor{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Professor{}, err
	}

	return professor, nil
}

func (s *PostgresStore) SetProfessorPassword(ctx context.Context, id string, passwordHash string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const updateQuery = `
		UPDATE users
		SET password_hash = $2
		WHERE id = $1
			AND role = $3`

	commandTag, err := tx.Exec(ctx, updateQuery, id, passwordHash, auth.RoleProfessor)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *PostgresStore) DeleteProfessor(ctx context.Context, id string) error {
	const query = `
		DELETE FROM users
		WHERE id = $1
			AND role = $2`

	commandTag, err := s.pool.Exec(ctx, query, id, auth.RoleProfessor)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

type professorScanner interface {
	Scan(dest ...any) error
}

func scanProfessor(row professorScanner) (Professor, error) {
	var professor Professor
	err := row.Scan(
		&professor.ID,
		&professor.Name,
		&professor.Email,
		&professor.Active,
		&professor.CreatedAt,
		&professor.UpdatedAt,
	)
	if err != nil {
		return Professor{}, err
	}

	return professor, nil
}

func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
