ALTER TABLE blocks
ADD COLUMN description text NULL;

CREATE TABLE block_sequence_items (
    block_id uuid NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position > 0),
    text text NOT NULL CHECK (btrim(text) <> ''),
    PRIMARY KEY (block_id, position)
);

ALTER TABLE training_history_blocks
ADD COLUMN block_description text NULL;

CREATE TABLE training_history_block_sequence_items (
    history_id uuid NOT NULL,
    block_position integer NOT NULL,
    item_position integer NOT NULL CHECK (item_position > 0),
    text text NOT NULL CHECK (btrim(text) <> ''),
    PRIMARY KEY (history_id, block_position, item_position),
    FOREIGN KEY (history_id, block_position)
        REFERENCES training_history_blocks(history_id, position)
        ON DELETE CASCADE
);
