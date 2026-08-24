package blocks

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const blockNameUniqueIndex = "blocks_category_name_unique_ci"

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListBlocks(ctx context.Context) ([]Block, error) {
	const query = `
		SELECT b.id::text, b.name, b.description, b.active, c.id::text, c.name, b.created_at, b.updated_at
		FROM blocks b
		JOIN categories c ON c.id = b.category_id
		ORDER BY lower(c.name) ASC, c.name ASC, lower(b.name) ASC, b.name ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := []Block{}
	for rows.Next() {
		block, err := scanBlock(rows)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := loadBlockSequences(ctx, s.pool, blocks); err != nil {
		return nil, err
	}

	return blocks, nil
}

func (s *PostgresStore) CreateBlock(ctx context.Context, block NewBlock) (Block, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Block{}, err
	}
	defer tx.Rollback(ctx)

	const query = `
		WITH inserted AS (
			INSERT INTO blocks (id, name, category_id, description)
			SELECT $1, $2, id, $4
			FROM categories
			WHERE id = $3 AND active = true
			RETURNING id, name, description, active, category_id, created_at, updated_at
		)
		SELECT i.id::text, i.name, i.description, i.active, c.id::text, c.name, i.created_at, i.updated_at
		FROM inserted i
		JOIN categories c ON c.id = i.category_id`

	created, err := scanBlock(tx.QueryRow(ctx, query, block.ID, block.Name, block.CategoryID, block.Description))
	if errors.Is(err, pgx.ErrNoRows) {
		return Block{}, ErrInvalidCategory
	}
	if blockNameUniqueViolation(err) {
		return Block{}, ErrNameExists
	}
	if categoryForeignKeyViolation(err) {
		return Block{}, ErrInvalidCategory
	}
	if err != nil {
		return Block{}, err
	}
	if err := insertSequenceItems(ctx, tx, block.ID, block.Sequence); err != nil {
		return Block{}, err
	}
	created.Sequence = sequenceItemsFromNew(block.Sequence)
	if err := tx.Commit(ctx); err != nil {
		return Block{}, err
	}

	return created, nil
}

func (s *PostgresStore) UpdateBlock(ctx context.Context, id string, name string, categoryID string, description *string, sequence []NewSequenceItem) (Block, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Block{}, err
	}
	defer tx.Rollback(ctx)

	const query = `
		WITH active_category AS (
			SELECT id FROM categories WHERE id = $3 AND active = true
		),
		updated AS (
			UPDATE blocks
			SET name = $2, category_id = (SELECT id FROM active_category), description = $4
			WHERE id = $1 AND EXISTS (SELECT 1 FROM active_category)
			RETURNING id, name, description, active, category_id, created_at, updated_at
		)
		SELECT u.id::text, u.name, u.description, u.active, c.id::text, c.name, u.created_at, u.updated_at
		FROM updated u
		JOIN categories c ON c.id = u.category_id`

	updated, err := scanBlock(tx.QueryRow(ctx, query, id, name, categoryID, description))
	if errors.Is(err, pgx.ErrNoRows) {
		if exists, existsErr := s.blockExists(ctx, id); existsErr != nil {
			return Block{}, existsErr
		} else if exists {
			return Block{}, ErrInvalidCategory
		}
		return Block{}, ErrNotFound
	}
	if blockNameUniqueViolation(err) {
		return Block{}, ErrNameExists
	}
	if categoryForeignKeyViolation(err) {
		return Block{}, ErrInvalidCategory
	}
	if err != nil {
		return Block{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM block_sequence_items WHERE block_id = $1`, id); err != nil {
		return Block{}, err
	}
	if err := insertSequenceItems(ctx, tx, id, sequence); err != nil {
		return Block{}, err
	}
	updated.Sequence = sequenceItemsFromNew(sequence)
	if err := tx.Commit(ctx); err != nil {
		return Block{}, err
	}

	return updated, nil
}

func (s *PostgresStore) SetBlockStatus(ctx context.Context, id string, active bool) (Block, error) {
	const query = `
		WITH updated AS (
			UPDATE blocks
			SET active = $2
			WHERE id = $1
			RETURNING id, name, description, active, category_id, created_at, updated_at
		)
		SELECT u.id::text, u.name, u.description, u.active, c.id::text, c.name, u.created_at, u.updated_at
		FROM updated u
		JOIN categories c ON c.id = u.category_id`

	updated, err := scanBlock(s.pool.QueryRow(ctx, query, id, active))
	if errors.Is(err, pgx.ErrNoRows) {
		return Block{}, ErrNotFound
	}
	if err != nil {
		return Block{}, err
	}
	blocks := []Block{updated}
	if err := loadBlockSequences(ctx, s.pool, blocks); err != nil {
		return Block{}, err
	}
	updated = blocks[0]

	return updated, nil
}

func (s *PostgresStore) DeleteBlock(ctx context.Context, id string) error {
	const query = `DELETE FROM blocks WHERE id = $1`

	commandTag, err := s.pool.Exec(ctx, query, id)
	if blockInUseViolation(err) {
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

func (s *PostgresStore) blockExists(ctx context.Context, id string) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM blocks WHERE id = $1)`

	var exists bool
	if err := s.pool.QueryRow(ctx, query, id).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

type blockScanner interface {
	Scan(dest ...any) error
}

func scanBlock(row blockScanner) (Block, error) {
	var block Block
	var description pgtype.Text
	err := row.Scan(
		&block.ID,
		&block.Name,
		&description,
		&block.Active,
		&block.Category.ID,
		&block.Category.Name,
		&block.CreatedAt,
		&block.UpdatedAt,
	)
	if err != nil {
		return Block{}, err
	}
	if description.Valid {
		block.Description = &description.String
	}
	block.Sequence = []SequenceItem{}

	return block, nil
}

func loadBlockSequences(ctx context.Context, db queryer, blocks []Block) error {
	if len(blocks) == 0 {
		return nil
	}

	byID := make(map[string]*Block, len(blocks))
	ids := make([]string, 0, len(blocks))
	for index := range blocks {
		byID[blocks[index].ID] = &blocks[index]
		ids = append(ids, blocks[index].ID)
	}

	const query = `
		SELECT block_id::text, position, text
		FROM block_sequence_items
		WHERE block_id::text = ANY($1)
		ORDER BY block_id, position ASC`
	rows, err := db.Query(ctx, query, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var blockID string
		var item SequenceItem
		if err := rows.Scan(&blockID, &item.Position, &item.Text); err != nil {
			return err
		}
		if block := byID[blockID]; block != nil {
			block.Sequence = append(block.Sequence, item)
		}
	}

	return rows.Err()
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func insertSequenceItems(ctx context.Context, tx pgx.Tx, blockID string, sequence []NewSequenceItem) error {
	const query = `
		INSERT INTO block_sequence_items (block_id, position, text)
		VALUES ($1, $2, $3)`
	for _, item := range sequence {
		if _, err := tx.Exec(ctx, query, blockID, item.Position, item.Text); err != nil {
			return err
		}
	}

	return nil
}

func sequenceItemsFromNew(sequence []NewSequenceItem) []SequenceItem {
	items := make([]SequenceItem, 0, len(sequence))
	for _, item := range sequence {
		items = append(items, SequenceItem{Position: item.Position, Text: item.Text})
	}
	return items
}

func blockNameUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == blockNameUniqueIndex
}

func categoryForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "blocks_category_id_fkey"
}

func blockInUseViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "workout_blocks_block_id_fkey"
}
