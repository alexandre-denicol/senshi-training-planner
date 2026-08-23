CREATE TABLE schedule_entries (
    id uuid PRIMARY KEY,
    workout_id uuid NOT NULL REFERENCES workouts(id) ON DELETE RESTRICT,
    scheduled_date date NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX schedule_entries_workout_date_unique ON schedule_entries (workout_id, scheduled_date);
CREATE INDEX schedule_entries_scheduled_date_idx ON schedule_entries (scheduled_date);

CREATE FUNCTION set_schedule_entries_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER schedule_entries_set_updated_at
BEFORE UPDATE ON schedule_entries
FOR EACH ROW
EXECUTE FUNCTION set_schedule_entries_updated_at();
