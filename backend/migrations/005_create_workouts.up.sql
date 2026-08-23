CREATE TABLE workouts (
    id uuid PRIMARY KEY,
    name text NOT NULL CHECK (btrim(name) <> ''),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX workouts_name_unique_ci ON workouts (lower(name));

CREATE TABLE workout_blocks (
    workout_id uuid NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    block_id uuid NOT NULL REFERENCES blocks(id) ON DELETE RESTRICT,
    position integer NOT NULL CHECK (position > 0),
    PRIMARY KEY (workout_id, block_id),
    UNIQUE (workout_id, position)
);

CREATE INDEX workout_blocks_block_id_idx ON workout_blocks (block_id);

CREATE FUNCTION set_workouts_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER workouts_set_updated_at
BEFORE UPDATE ON workouts
FOR EACH ROW
EXECUTE FUNCTION set_workouts_updated_at();
