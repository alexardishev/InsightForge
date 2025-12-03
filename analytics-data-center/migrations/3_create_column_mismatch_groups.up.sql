CREATE TABLE IF NOT EXISTS column_mismatch_groups (
    id BIGSERIAL PRIMARY KEY,
    schema_id BIGINT NOT NULL REFERENCES schems(id),
    database_name TEXT NOT NULL,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS column_mismatch_items (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES column_mismatch_groups(id) ON DELETE CASCADE,
    old_column_name TEXT NULL,
    new_column_name TEXT NULL,
    score DOUBLE PRECISION NULL,
    mismatch_type TEXT NOT NULL
);
