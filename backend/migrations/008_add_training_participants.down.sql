DROP TABLE IF EXISTS training_history_participants;

ALTER TABLE training_history
DROP CONSTRAINT IF EXISTS training_history_participant_count_non_negative,
DROP COLUMN IF EXISTS participant_count;
