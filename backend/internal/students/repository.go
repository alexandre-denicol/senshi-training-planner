package students

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListStudents(ctx context.Context) ([]Student, error) {
	const query = `
		SELECT id::text, name, active, birth_date::text, phone, guardian_name, guardian_phone, notes, created_at, updated_at
		FROM students
		ORDER BY lower(name) ASC, name ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := []Student{}
	for rows.Next() {
		student, err := scanStudent(rows)
		if err != nil {
			return nil, err
		}
		students = append(students, student)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return students, nil
}

func (s *PostgresStore) CreateStudent(ctx context.Context, student NewStudent) (Student, error) {
	const query = `
		INSERT INTO students (id, name, birth_date, phone, guardian_name, guardian_phone, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, name, active, birth_date::text, phone, guardian_name, guardian_phone, notes, created_at, updated_at`

	created, err := scanStudent(s.pool.QueryRow(ctx, query,
		student.ID, student.Name, student.BirthDate, student.Phone, student.GuardianName, student.GuardianPhone, student.Notes))
	if err != nil {
		return Student{}, err
	}

	return created, nil
}

func (s *PostgresStore) UpdateStudent(ctx context.Context, id string, student NewStudent) (Student, error) {
	const query = `
		UPDATE students
		SET name = $2, birth_date = $3, phone = $4, guardian_name = $5, guardian_phone = $6, notes = $7
		WHERE id = $1
		RETURNING id::text, name, active, birth_date::text, phone, guardian_name, guardian_phone, notes, created_at, updated_at`

	updated, err := scanStudent(s.pool.QueryRow(ctx, query,
		id, student.Name, student.BirthDate, student.Phone, student.GuardianName, student.GuardianPhone, student.Notes))
	if errors.Is(err, pgx.ErrNoRows) {
		return Student{}, ErrNotFound
	}
	if err != nil {
		return Student{}, err
	}

	return updated, nil
}

func (s *PostgresStore) SetStudentStatus(ctx context.Context, id string, active bool) (Student, error) {
	const query = `
		UPDATE students
		SET active = $2
		WHERE id = $1
		RETURNING id::text, name, active, birth_date::text, phone, guardian_name, guardian_phone, notes, created_at, updated_at`

	updated, err := scanStudent(s.pool.QueryRow(ctx, query, id, active))
	if errors.Is(err, pgx.ErrNoRows) {
		return Student{}, ErrNotFound
	}
	if err != nil {
		return Student{}, err
	}

	return updated, nil
}

type studentScanner interface {
	Scan(dest ...any) error
}

func scanStudent(row studentScanner) (Student, error) {
	var student Student
	var birthDate, phone, guardianName, guardianPhone, notes pgtype.Text
	err := row.Scan(
		&student.ID,
		&student.Name,
		&student.Active,
		&birthDate,
		&phone,
		&guardianName,
		&guardianPhone,
		&notes,
		&student.CreatedAt,
		&student.UpdatedAt,
	)
	if err != nil {
		return Student{}, err
	}
	if birthDate.Valid {
		student.BirthDate = &birthDate.String
	}
	if phone.Valid {
		student.Phone = &phone.String
	}
	if guardianName.Valid {
		student.GuardianName = &guardianName.String
	}
	if guardianPhone.Valid {
		student.GuardianPhone = &guardianPhone.String
	}
	if notes.Valid {
		student.Notes = &notes.String
	}

	return student, nil
}
