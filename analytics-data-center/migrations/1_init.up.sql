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

-- Пример схемы (без transform, просто 2 колонки)
INSERT INTO schems (id, schema_view) VALUES (
    1,
    '{
      "view_name": "user_basic_info",
      "sources": [
        {
          "name": "prod_data",
          "schemas": [
            {
              "name": "public",
              "tables": [
                {
                  "name": "users",
                  "columns": [
                    {
                      "name": "id",
                      "alias": "user_id",
                      "is_update_key": true
                      "type": "int"
                    },
                    {
                      "name": "email"
                      "type": "text"
                    }
                  ]
                },
                {
                  "name": "profiles",
                  "columns": [
                    {
                      "name": "user_id"
                      "type": "int"
                    },
                    {
                      "name": "age"
                    }
                  ]
                }
              ]
            }
          ]
        }
      ],
      "joins": []
    }'::jsonb
);
