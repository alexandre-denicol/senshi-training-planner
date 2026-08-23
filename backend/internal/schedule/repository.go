package schedule

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const scheduleWorkoutDateUniqueIndex = "schedule_entries_workout_date_unique"

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListEntries(ctx context.Context, from string, to string) ([]Entry, error) {
	const query = `
		SELECT se.id::text, se.scheduled_date::text, w.id::text, w.name, w.active, se.completed_at, se.created_at, se.updated_at
		FROM schedule_entries se
		JOIN workouts w ON w.id = se.workout_id
		WHERE se.scheduled_date >= $1::date AND se.scheduled_date <= $2::date
		ORDER BY se.scheduled_date ASC, lower(w.name) ASC, w.name ASC`

	rows, err := s.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (s *PostgresStore) CreateEntry(ctx context.Context, entry NewEntry) (Entry, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Entry{}, err
	}
	defer tx.Rollback(ctx)

	if err := validateActiveWorkout(ctx, tx, entry.WorkoutID); err != nil {
		return Entry{}, err
	}

	const query = `
		INSERT INTO schedule_entries (id, workout_id, scheduled_date)
		VALUES ($1, $2, $3::date)`
	if _, err := tx.Exec(ctx, query, entry.ID, entry.WorkoutID, entry.ScheduledDate); scheduleDuplicateViolation(err) {
		return Entry{}, ErrDuplicate
	} else if workoutForeignKeyViolation(err) {
		return Entry{}, ErrInvalidWorkout
	} else if err != nil {
		return Entry{}, err
	}

	created, err := getEntry(ctx, tx, entry.ID)
	if err != nil {
		return Entry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Entry{}, err
	}

	return created, nil
}

func (s *PostgresStore) UpdateEntry(ctx context.Context, id string, workoutID string, scheduledDate string) (Entry, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Entry{}, err
	}
	defer tx.Rollback(ctx)

	var currentWorkoutID string
	var completedAt *time.Time
	if err := tx.QueryRow(ctx, `SELECT workout_id::text, completed_at FROM schedule_entries WHERE id = $1 FOR UPDATE`, id).Scan(&currentWorkoutID, &completedAt); errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	} else if err != nil {
		return Entry{}, err
	}
	if completedAt != nil {
		return Entry{}, ErrCompleted
	}

	if currentWorkoutID != workoutID {
		if err := validateActiveWorkout(ctx, tx, workoutID); err != nil {
			return Entry{}, err
		}
	}

	const query = `
		UPDATE schedule_entries
		SET workout_id = $2, scheduled_date = $3::date
		WHERE id = $1`
	if _, err := tx.Exec(ctx, query, id, workoutID, scheduledDate); scheduleDuplicateViolation(err) {
		return Entry{}, ErrDuplicate
	} else if workoutForeignKeyViolation(err) {
		return Entry{}, ErrInvalidWorkout
	} else if err != nil {
		return Entry{}, err
	}

	updated, err := getEntry(ctx, tx, id)
	if err != nil {
		return Entry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Entry{}, err
	}

	return updated, nil
}

func (s *PostgresStore) DeleteEntry(ctx context.Context, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var completedAt *time.Time
	if err := tx.QueryRow(ctx, `SELECT completed_at FROM schedule_entries WHERE id = $1 FOR UPDATE`, id).Scan(&completedAt); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if completedAt != nil {
		return ErrCompleted
	}

	commandTag, err := tx.Exec(ctx, `DELETE FROM schedule_entries WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func validateActiveWorkout(ctx context.Context, tx pgx.Tx, workoutID string) error {
	var active bool
	err := tx.QueryRow(ctx, `SELECT active FROM workouts WHERE id = $1`, workoutID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidWorkout
	}
	if err != nil {
		return err
	}
	if !active {
		return ErrInvalidWorkout
	}

	return nil
}

func getEntry(ctx context.Context, db queryer, id string) (Entry, error) {
	const query = `
		SELECT se.id::text, se.scheduled_date::text, w.id::text, w.name, w.active, se.completed_at, se.created_at, se.updated_at
		FROM schedule_entries se
		JOIN workouts w ON w.id = se.workout_id
		WHERE se.id = $1`

	entry, err := scanEntry(db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, err
	}

	return entry, nil
}

type entryScanner interface {
	Scan(dest ...any) error
}

func scanEntry(row entryScanner) (Entry, error) {
	var entry Entry
	err := row.Scan(
		&entry.ID,
		&entry.ScheduledDate,
		&entry.Workout.ID,
		&entry.Workout.Name,
		&entry.Workout.Active,
		&entry.CompletedAt,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)
	if err != nil {
		return Entry{}, err
	}

	return entry, nil
}

func scheduleDuplicateViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == scheduleWorkoutDateUniqueIndex
}

func workoutForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "schedule_entries_workout_id_fkey"
}
