ALTER TABLE schedule_entries
ADD COLUMN completed_at timestamptz NULL;

CREATE TABLE training_history (
    id uuid PRIMARY KEY,
    schedule_entry_id uuid NOT NULL UNIQUE REFERENCES schedule_entries(id) ON DELETE RESTRICT,
    training_date date NOT NULL,
    workout_id uuid NULL REFERENCES workouts(id) ON DELETE SET NULL,
    workout_name text NOT NULL CHECK (btrim(workout_name) <> ''),
    completed_by_user_id uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    completed_by_name text NOT NULL CHECK (btrim(completed_by_name) <> ''),
    completed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE training_history_blocks (
    history_id uuid NOT NULL REFERENCES training_history(id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position > 0),
    block_id uuid NULL REFERENCES blocks(id) ON DELETE SET NULL,
    block_name text NOT NULL CHECK (btrim(block_name) <> ''),
    category_id uuid NULL REFERENCES categories(id) ON DELETE SET NULL,
    category_name text NOT NULL CHECK (btrim(category_name) <> ''),
    PRIMARY KEY (history_id, position)
);

CREATE INDEX training_history_training_date_idx ON training_history (training_date);
CREATE INDEX training_history_completed_at_idx ON training_history (completed_at);
