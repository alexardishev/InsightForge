-- Таблица задач
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY,
    create_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL,
    comment TEXT
);

-- Таблица со схемами представлений
CREATE TABLE IF NOT EXISTS schems (
id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    schema_view JSONB NOT NULL
);