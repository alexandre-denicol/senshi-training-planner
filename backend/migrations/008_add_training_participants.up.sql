ALTER TABLE training_history
ADD COLUMN participant_count integer NULL,
ADD CONSTRAINT training_history_participant_count_non_negative CHECK (participant_count IS NULL OR participant_count >= 0);

CREATE TABLE training_history_participants (
    history_id uuid NOT NULL REFERENCES training_history(id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position > 0),
    name text NOT NULL CHECK (btrim(name) <> ''),
    PRIMARY KEY (history_id, position)
);
