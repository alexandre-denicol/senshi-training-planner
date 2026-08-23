DROP TABLE IF EXISTS training_history_blocks;
DROP TABLE IF EXISTS training_history;

ALTER TABLE schedule_entries
DROP COLUMN IF EXISTS completed_at;
