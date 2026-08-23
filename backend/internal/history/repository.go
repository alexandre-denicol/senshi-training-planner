package history

import (
	"context"
	"errors"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const historyScheduleEntryUniqueConstraint = "training_history_schedule_entry_id_key"

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListHistory(ctx context.Context, from string, to string) ([]ListItem, error) {
	const query = `
		SELECT th.id::text, th.training_date::text, th.workout_name, count(thb.position)::int,
			th.participant_count, th.completed_by_name, th.completed_at, th.schedule_entry_id::text
		FROM training_history th
		LEFT JOIN training_history_blocks thb ON thb.history_id = th.id
		WHERE th.training_date >= $1::date AND th.training_date <= $2::date
		GROUP BY th.id
		ORDER BY th.training_date DESC, th.completed_at DESC`

	rows, err := s.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ListItem{}
	for rows.Next() {
		item, err := scanListItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *PostgresStore) GetHistory(ctx context.Context, id string) (Detail, error) {
	return getHistory(ctx, s.pool, id)
}

func (s *PostgresStore) CompleteScheduleEntry(ctx context.Context, historyID string, scheduleEntryID string, completedBy auth.PublicUser, completedAt time.Time, details CompletionDetails) (Detail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Detail{}, err
	}
	defer tx.Rollback(ctx)

	snapshot, err := readScheduleSnapshot(ctx, tx, scheduleEntryID)
	if err != nil {
		return Detail{}, err
	}
	if snapshot.CompletedAt != nil {
		return Detail{}, ErrAlreadyCompleted
	}

	blocks, err := readWorkoutBlockSnapshots(ctx, tx, snapshot.WorkoutID)
	if err != nil {
		return Detail{}, err
	}
	if len(blocks) == 0 {
		return Detail{}, ErrSnapshotUnavailable
	}

	const historyQuery = `
		INSERT INTO training_history (
			id, schedule_entry_id, training_date, workout_id, workout_name,
			completed_by_user_id, completed_by_name, completed_at, participant_count
		)
		VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, $9)`
	if _, err := tx.Exec(ctx, historyQuery,
		historyID,
		scheduleEntryID,
		snapshot.TrainingDate,
		snapshot.WorkoutID,
		snapshot.WorkoutName,
		completedBy.ID,
		completedBy.Name,
		completedAt,
		details.ParticipantCount,
	); historyDuplicateViolation(err) {
		return Detail{}, ErrAlreadyCompleted
	} else if err != nil {
		return Detail{}, err
	}

	const blockQuery = `
		INSERT INTO training_history_blocks (
			history_id, position, block_id, block_name, category_id, category_name
		)
		VALUES ($1, $2, $3, $4, $5, $6)`
	for _, block := range blocks {
		if _, err := tx.Exec(ctx, blockQuery,
			historyID,
			block.Position,
			block.BlockID,
			block.BlockName,
			block.CategoryID,
			block.CategoryName,
		); err != nil {
			return Detail{}, err
		}
	}

	const participantQuery = `
		INSERT INTO training_history_participants (history_id, position, name)
		VALUES ($1, $2, $3)`
	for index, name := range details.ParticipantNames {
		if _, err := tx.Exec(ctx, participantQuery, historyID, index+1, name); err != nil {
			return Detail{}, err
		}
	}

	commandTag, err := tx.Exec(ctx, `UPDATE schedule_entries SET completed_at = $2 WHERE id = $1 AND completed_at IS NULL`, scheduleEntryID, completedAt)
	if err != nil {
		return Detail{}, err
	}
	if commandTag.RowsAffected() == 0 {
		return Detail{}, ErrAlreadyCompleted
	}

	detail, err := getHistory(ctx, tx, historyID)
	if err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, err
	}

	return detail, nil
}

type scheduleSnapshot struct {
	WorkoutID    string
	WorkoutName  string
	TrainingDate string
	CompletedAt  *time.Time
}

type blockSnapshot struct {
	Position     int
	BlockID      string
	BlockName    string
	CategoryID   string
	CategoryName string
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func readScheduleSnapshot(ctx context.Context, tx pgx.Tx, scheduleEntryID string) (scheduleSnapshot, error) {
	const query = `
		SELECT se.workout_id::text, w.name, se.scheduled_date::text, se.completed_at
		FROM schedule_entries se
		JOIN workouts w ON w.id = se.workout_id
		WHERE se.id = $1
		FOR UPDATE OF se`

	var snapshot scheduleSnapshot
	err := tx.QueryRow(ctx, query, scheduleEntryID).Scan(
		&snapshot.WorkoutID,
		&snapshot.WorkoutName,
		&snapshot.TrainingDate,
		&snapshot.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return scheduleSnapshot{}, ErrScheduleNotFound
	}
	if err != nil {
		return scheduleSnapshot{}, err
	}

	return snapshot, nil
}

func readWorkoutBlockSnapshots(ctx context.Context, tx pgx.Tx, workoutID string) ([]blockSnapshot, error) {
	const query = `
		SELECT wb.position, b.id::text, b.name, c.id::text, c.name
		FROM workout_blocks wb
		JOIN blocks b ON b.id = wb.block_id
		JOIN categories c ON c.id = b.category_id
		WHERE wb.workout_id = $1
		ORDER BY wb.position ASC`

	rows, err := tx.Query(ctx, query, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := []blockSnapshot{}
	for rows.Next() {
		var block blockSnapshot
		if err := rows.Scan(&block.Position, &block.BlockID, &block.BlockName, &block.CategoryID, &block.CategoryName); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}

func getHistory(ctx context.Context, db queryer, id string) (Detail, error) {
	const historyQuery = `
		SELECT id::text, training_date::text, workout_name, participant_count, completed_by_name, completed_at
		FROM training_history
		WHERE id = $1`

	var detail Detail
	var participantCount pgtype.Int4
	err := db.QueryRow(ctx, historyQuery, id).Scan(
		&detail.ID,
		&detail.TrainingDate,
		&detail.WorkoutName,
		&participantCount,
		&detail.CompletedByName,
		&detail.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	if participantCount.Valid {
		value := int(participantCount.Int32)
		detail.ParticipantCount = &value
	}

	const blocksQuery = `
		SELECT position, block_name, category_name
		FROM training_history_blocks
		WHERE history_id = $1
		ORDER BY position ASC`

	rows, err := db.Query(ctx, blocksQuery, id)
	if err != nil {
		return Detail{}, err
	}
	defer rows.Close()

	detail.Blocks = []Block{}
	for rows.Next() {
		var block Block
		if err := rows.Scan(&block.Position, &block.BlockName, &block.CategoryName); err != nil {
			return Detail{}, err
		}
		detail.Blocks = append(detail.Blocks, block)
	}
	if err := rows.Err(); err != nil {
		return Detail{}, err
	}

	const participantsQuery = `
		SELECT name
		FROM training_history_participants
		WHERE history_id = $1
		ORDER BY position ASC`

	participantRows, err := db.Query(ctx, participantsQuery, id)
	if err != nil {
		return Detail{}, err
	}
	defer participantRows.Close()

	detail.ParticipantNames = []string{}
	for participantRows.Next() {
		var name string
		if err := participantRows.Scan(&name); err != nil {
			return Detail{}, err
		}
		detail.ParticipantNames = append(detail.ParticipantNames, name)
	}
	if err := participantRows.Err(); err != nil {
		return Detail{}, err
	}

	return detail, nil
}

type listItemScanner interface {
	Scan(dest ...any) error
}

func scanListItem(row listItemScanner) (ListItem, error) {
	var item ListItem
	var participantCount pgtype.Int4
	err := row.Scan(
		&item.ID,
		&item.TrainingDate,
		&item.WorkoutName,
		&item.BlockCount,
		&participantCount,
		&item.CompletedByName,
		&item.CompletedAt,
		&item.ScheduleEntryID,
	)
	if err != nil {
		return ListItem{}, err
	}
	if participantCount.Valid {
		value := int(participantCount.Int32)
		item.ParticipantCount = &value
	}

	return item, nil
}

func historyDuplicateViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == historyScheduleEntryUniqueConstraint
}
