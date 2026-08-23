package categories

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const categoryNameUniqueIndex = "categories_name_unique_ci"

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListCategories(ctx context.Context) ([]Category, error) {
	const query = `
		SELECT id::text, name, active, created_at, updated_at
		FROM categories
		ORDER BY lower(name) ASC, name ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var category Category
		if err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Active,
			&category.CreatedAt,
			&category.UpdatedAt,
		); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

func (s *PostgresStore) CreateCategory(ctx context.Context, category NewCategory) (Category, error) {
	const query = `
		INSERT INTO categories (id, name)
		VALUES ($1, $2)
		RETURNING id::text, name, active, created_at, updated_at`

	created, err := scanCategory(s.pool.QueryRow(ctx, query, category.ID, category.Name))
	if categoryNameUniqueViolation(err) {
		return Category{}, ErrNameExists
	}
	if err != nil {
		return Category{}, err
	}

	return created, nil
}

func (s *PostgresStore) UpdateCategory(ctx context.Context, id string, name string) (Category, error) {
	const query = `
		UPDATE categories
		SET name = $2
		WHERE id = $1
		RETURNING id::text, name, active, created_at, updated_at`

	updated, err := scanCategory(s.pool.QueryRow(ctx, query, id, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	if categoryNameUniqueViolation(err) {
		return Category{}, ErrNameExists
	}
	if err != nil {
		return Category{}, err
	}

	return updated, nil
}

func (s *PostgresStore) SetCategoryStatus(ctx context.Context, id string, active bool) (Category, error) {
	const query = `
		UPDATE categories
		SET active = $2
		WHERE id = $1
		RETURNING id::text, name, active, created_at, updated_at`

	updated, err := scanCategory(s.pool.QueryRow(ctx, query, id, active))
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	if err != nil {
		return Category{}, err
	}

	return updated, nil
}

func (s *PostgresStore) DeleteCategory(ctx context.Context, id string) error {
	const query = `DELETE FROM categories WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

type categoryScanner interface {
	Scan(dest ...any) error
}

func scanCategory(row categoryScanner) (Category, error) {
	var category Category
	err := row.Scan(
		&category.ID,
		&category.Name,
		&category.Active,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		return Category{}, err
	}

	return category, nil
}

func categoryNameUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == categoryNameUniqueIndex
}
