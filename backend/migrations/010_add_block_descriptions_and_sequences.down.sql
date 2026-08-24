DROP TABLE IF EXISTS training_history_block_sequence_items;

ALTER TABLE training_history_blocks
DROP COLUMN IF EXISTS block_description;

DROP TABLE IF EXISTS block_sequence_items;

ALTER TABLE blocks
DROP COLUMN IF EXISTS description;
