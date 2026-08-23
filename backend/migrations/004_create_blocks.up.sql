CREATE TABLE blocks (
    id uuid PRIMARY KEY,
    category_id uuid NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    name text NOT NULL CHECK (btrim(name) <> ''),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX blocks_category_name_unique_ci ON blocks (category_id, lower(name));

CREATE FUNCTION set_blocks_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER blocks_set_updated_at
BEFORE UPDATE ON blocks
FOR EACH ROW
EXECUTE FUNCTION set_blocks_updated_at();
