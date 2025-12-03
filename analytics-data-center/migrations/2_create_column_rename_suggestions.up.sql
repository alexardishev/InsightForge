CREATE TABLE IF NOT EXISTS column_rename_suggestions (
    id BIGSERIAL PRIMARY KEY,
    schema_id BIGINT NOT NULL REFERENCES schems(id),
    database_name TEXT NOT NULL,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    old_column_name TEXT NOT NULL,
    new_column_name TEXT NOT NULL,
    strategy TEXT NOT NULL,
    task_number TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
