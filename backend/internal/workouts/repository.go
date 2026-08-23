package workouts

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const workoutNameUniqueIndex = "workouts_name_unique_ci"

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListWorkouts(ctx context.Context) ([]WorkoutListItem, error) {
	const query = `
		SELECT w.id::text, w.name, w.active, count(wb.block_id)::int, w.created_at, w.updated_at
		FROM workouts w
		LEFT JOIN workout_blocks wb ON wb.workout_id = w.id
		GROUP BY w.id
		ORDER BY lower(w.name) ASC, w.name ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workouts := []WorkoutListItem{}
	for rows.Next() {
		workout, err := scanWorkoutListItem(rows)
		if err != nil {
			return nil, err
		}
		workouts = append(workouts, workout)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return workouts, nil
}

func (s *PostgresStore) GetWorkout(ctx context.Context, id string) (WorkoutDetail, error) {
	return s.getWorkout(ctx, s.pool, id)
}

func (s *PostgresStore) CreateWorkout(ctx context.Context, workout NewWorkout) (WorkoutDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WorkoutDetail{}, err
	}
	defer tx.Rollback(ctx)

	if err := validateActiveBlocks(ctx, tx, workout.BlockIDs); err != nil {
		return WorkoutDetail{}, err
	}

	const query = `
		INSERT INTO workouts (id, name)
		VALUES ($1, $2)`
	if _, err := tx.Exec(ctx, query, workout.ID, workout.Name); workoutNameUniqueViolation(err) {
		return WorkoutDetail{}, ErrNameExists
	} else if err != nil {
		return WorkoutDetail{}, err
	}

	if err := insertWorkoutBlocks(ctx, tx, workout.ID, workout.BlockIDs); err != nil {
		return WorkoutDetail{}, err
	}

	created, err := s.getWorkout(ctx, tx, workout.ID)
	if err != nil {
		return WorkoutDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkoutDetail{}, err
	}

	return created, nil
}

func (s *PostgresStore) UpdateWorkout(ctx context.Context, id string, name string, blockIDs []string) (WorkoutDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WorkoutDetail{}, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM workouts WHERE id = $1 FOR UPDATE)`, id).Scan(&exists); err != nil {
		return WorkoutDetail{}, err
	}
	if !exists {
		return WorkoutDetail{}, ErrNotFound
	}

	if err := validateBlocksForUpdate(ctx, tx, id, blockIDs); err != nil {
		return WorkoutDetail{}, err
	}

	if _, err := tx.Exec(ctx, `UPDATE workouts SET name = $2 WHERE id = $1`, id, name); workoutNameUniqueViolation(err) {
		return WorkoutDetail{}, ErrNameExists
	} else if err != nil {
		return WorkoutDetail{}, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM workout_blocks WHERE workout_id = $1`, id); err != nil {
		return WorkoutDetail{}, err
	}
	if err := insertWorkoutBlocks(ctx, tx, id, blockIDs); err != nil {
		return WorkoutDetail{}, err
	}

	updated, err := s.getWorkout(ctx, tx, id)
	if err != nil {
		return WorkoutDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkoutDetail{}, err
	}

	return updated, nil
}

func (s *PostgresStore) SetWorkoutStatus(ctx context.Context, id string, active bool) (WorkoutListItem, error) {
	const query = `
		WITH updated AS (
			UPDATE workouts
			SET active = $2
			WHERE id = $1
			RETURNING id, name, active, created_at, updated_at
		)
		SELECT u.id::text, u.name, u.active, count(wb.block_id)::int, u.created_at, u.updated_at
		FROM updated u
		LEFT JOIN workout_blocks wb ON wb.workout_id = u.id
		GROUP BY u.id, u.name, u.active, u.created_at, u.updated_at`

	workout, err := scanWorkoutListItem(s.pool.QueryRow(ctx, query, id, active))
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkoutListItem{}, ErrNotFound
	}
	if err != nil {
		return WorkoutListItem{}, err
	}

	return workout, nil
}

func (s *PostgresStore) DeleteWorkout(ctx context.Context, id string) error {
	commandTag, err := s.pool.Exec(ctx, `DELETE FROM workouts WHERE id = $1`, id)
	if workoutInUseViolation(err) {
		return ErrInUse
	}
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *PostgresStore) getWorkout(ctx context.Context, db queryer, id string) (WorkoutDetail, error) {
	const workoutQuery = `
		SELECT id::text, name, active, created_at, updated_at
		FROM workouts
		WHERE id = $1`

	var workout WorkoutDetail
	err := db.QueryRow(ctx, workoutQuery, id).Scan(
		&workout.ID,
		&workout.Name,
		&workout.Active,
		&workout.CreatedAt,
		&workout.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkoutDetail{}, ErrNotFound
	}
	if err != nil {
		return WorkoutDetail{}, err
	}

	const blocksQuery = `
		SELECT b.id::text, b.name, b.active, wb.position, c.id::text, c.name
		FROM workout_blocks wb
		JOIN blocks b ON b.id = wb.block_id
		JOIN categories c ON c.id = b.category_id
		WHERE wb.workout_id = $1
		ORDER BY wb.position ASC`

	rows, err := db.Query(ctx, blocksQuery, id)
	if err != nil {
		return WorkoutDetail{}, err
	}
	defer rows.Close()

	workout.Blocks = []WorkoutBlock{}
	for rows.Next() {
		var block WorkoutBlock
		if err := rows.Scan(
			&block.ID,
			&block.Name,
			&block.Active,
			&block.Position,
			&block.Category.ID,
			&block.Category.Name,
		); err != nil {
			return WorkoutDetail{}, err
		}
		workout.Blocks = append(workout.Blocks, block)
	}
	if err := rows.Err(); err != nil {
		return WorkoutDetail{}, err
	}

	return workout, nil
}

func validateActiveBlocks(ctx context.Context, tx pgx.Tx, blockIDs []string) error {
	for _, blockID := range blockIDs {
		var active bool
		err := tx.QueryRow(ctx, `SELECT active FROM blocks WHERE id = $1`, blockID).Scan(&active)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidBlocks
		}
		if err != nil {
			return err
		}
		if !active {
			return ErrInvalidBlocks
		}
	}

	return nil
}

func validateBlocksForUpdate(ctx context.Context, tx pgx.Tx, workoutID string, blockIDs []string) error {
	for _, blockID := range blockIDs {
		var active bool
		var retained bool
		const query = `
			SELECT b.active, EXISTS (
				SELECT 1 FROM workout_blocks wb
				WHERE wb.workout_id = $1 AND wb.block_id = b.id
			)
			FROM blocks b
			WHERE b.id = $2`
		err := tx.QueryRow(ctx, query, workoutID, blockID).Scan(&active, &retained)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidBlocks
		}
		if err != nil {
			return err
		}
		if !active && !retained {
			return ErrInvalidBlocks
		}
	}

	return nil
}

func insertWorkoutBlocks(ctx context.Context, tx pgx.Tx, workoutID string, blockIDs []string) error {
	const query = `
		INSERT INTO workout_blocks (workout_id, block_id, position)
		VALUES ($1, $2, $3)`
	for index, blockID := range blockIDs {
		if _, err := tx.Exec(ctx, query, workoutID, blockID, index+1); blockForeignKeyViolation(err) {
			return ErrInvalidBlocks
		} else if err != nil {
			return err
		}
	}

	return nil
}

type workoutScanner interface {
	Scan(dest ...any) error
}

func scanWorkoutListItem(row workoutScanner) (WorkoutListItem, error) {
	var workout WorkoutListItem
	err := row.Scan(
		&workout.ID,
		&workout.Name,
		&workout.Active,
		&workout.BlockCount,
		&workout.CreatedAt,
		&workout.UpdatedAt,
	)
	if err != nil {
		return WorkoutListItem{}, err
	}

	return workout, nil
}

func workoutNameUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == workoutNameUniqueIndex
}

func blockForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "workout_blocks_block_id_fkey"
}

func workoutInUseViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "schedule_entries_workout_id_fkey"
}
