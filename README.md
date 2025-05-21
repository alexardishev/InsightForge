# ⚡ InsightForge

InsightForge — это мощный инструмент для построения аналитических представлений (вьюх) из различных источников данных.  
Позволяет собирать таблицы, объединять данные из разных сервисов, применять трансформации и хранить результат в аналитическом хранилище (DWH).

![ChatGPT Image 11 апр  2025 г , 12_25_40 (1)](https://github.com/user-attachments/assets/36cdafa8-a9b5-4524-b09b-941059bd7ed8)

---

## 🚀 Возможности

- 📦 Интеграция с внешними OLTP системами через API и события
- 🛠 Конструктор вью: таблицы, колонки, связи (JOIN), трансформации
- 💾 Поддержка PostgreSQL как источника и DWH
- 🔄 Автообновление данных по CDC (Debezium) или событиям
- 🧠 Преобразование данных: enum, alias, text mappings и др.
- 🧪 Асинхронная ETL-обработка с отслеживанием статуса задач
- 📊 Логирование и метрики (OpenTelemetry-ready)

---

## 🧱 Архитектура

- `analytics-view-service` — основной сервис для сборки и обновления вьюх
- `task-service` — отслеживание статусов ETL-процессов
- `sql-generator` — генератор SQL-скриптов (CREATE TABLE, INSERT и т.д.)
- `cdc-listener` (опционально) — слушает Kafka и реагирует на изменения
- `config-loader` — загрузка и валидация конфигурации вьюхи (YAML/JSON)

---

## 📂 Структура проекта
```
analytics-data-center/
├── cmd/                      # Точки входа
├── internal/
│   ├── app/                  # Инициализация приложения
│   ├── config/               # Загрузка конфигурации
│   ├── domain/models/        # Основные доменные сущности
│   ├── services/             # Бизнес-логика
│   ├── storage/              # Работа с БД
│   └── lib/                  # Утилиты (SQL генератор, дубликаты)
├── config/                   # YAML конфигурации
├── go.mod
└── README.md
```

---

## 🏗 Пример вьюхи (YAML)

```json
{
  "view_name": "user_basic_info"
  "joins": [
    {
      "inner": {
        "main_table": "users",
        "source": "postgres",
        "schema": "public",
        "table": "profiles",
        "column_first": "id",
        "column_second": "user_id"
      }
    }
  ],
  "sources": [
    {
      "name": "postgres",
      "schemas": [
        {
          "name": "public",
          "tables": [
            {
              "name": "users",
              "columns": [
                {
                  "name": "id",
                  "type": "uuid",
                  "view_key": "user_id",
                  "is_nullable": true,
                  "is_update_key": true,
                  "is_primary_key": true
                },
                {
                  "name": "email",
                  "type": "text",
                  "is_nullable": true
                },
                {
                  "name": "json_transform",
                  "type": "jsonb",
                  "is_nullable": true,
                  "transform": {
                    "type": "JSON",
                    "mode": "Mapping",
                    "output_column": "json_transform",
                    "mapping": {
                      "type_map": "JSON",
                      "mapping_json": [
                        {
                          "type_field": "int",
                          "mapping": {
                            "field1_in_json": "field1_view_column"
                          }
                        },
                        {
                          "type_field": "text",
                          "mapping": {
                            "field2_in_json": "field2_view_column"
                          }
                        }
                      ]
                    }
                  }
                },
                {
                  "name": "status",
                  "type": "int",
                  "transform": {
                    "type": "FieldTransform",
                    "mode": "Mapping",
                    "output_column": "status_label",
                    "mapping": {
                      "type_map": "FieldTransform",
                      "alias_new_column_transform": "status_label",
                      "mapping": {
                        "1": "Создан",
                        "2": "В обработке",
                        "3": "Завершен"
                      }
                    }
                  }
                }
              ]
            },
            {
              "name": "profiles",
              "columns": [
                {
                  "name": "user_id",
                  "type": "uuid",
                  "view_key": "id",
                  "is_update_key": true,
                  "is_primary_key": true
                },
                {
                  "name": "age",
                  "type": "int",
                  "is_nullable": true
                },
                {
                  "name": "test",
                  "type": "varchar",
                  "is_deleted": true,
                  "is_nullable": true
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

## 🔍 Пояснение к ключам JSON-конфигурации
- **`view_name`** — название аналитического представления (итоговая таблица).

- **`main_table`** — основная таблица, к которой будут присоединяться остальные. Если не указана, используется первая таблица из списка.

- **`sources`** — источники данных:
  - **`name`** — имя источника, обычно соответствует имени базы данных.
  - **`schemas`** — список схем, из которых берутся таблицы.
  - **`tables`** — список таблиц, участвующих в построении вьюхи:
    - **`columns`** — список колонок в таблице:
      - **`name`** — имя колонки в исходной таблице.
      - **`type`** — тип данных.
      - **`view_key`** — имя ключа в итоговом представлении, используется для корректного обновления по событиям. Может задаваться несколько взаимозаменяемых ключей, чтобы избежать коллизий при несинхронных событиях.
      - **`is_update_key`** — участвует ли колонка в логике обновления данных.
      - **`is_primary_key`** — часть ли колонка первичного ключа в итоговой вьюхе.
      - **`is_nullable`** — допускает ли значение `NULL`.  
        ⚠️ *На текущий момент не используется — ограничения не мигрируются из OLTP-таблиц.*
      - **`is_deleted`** — колонка считается удалённой, если физически исчезла из OLTP-хранилища.  
        🛈 *При этом она сохраняется в вьюхе и начинает заполняться `NULL`-значениями — это позволяет отследить изменения без нарушения структуры.*
      - **`transform`** — описание преобразования данных:
        - **`type`** — тип трансформации (`JSON`, `FieldTransform` и др.).
        - **`mode`** — режим трансформации (например, `Mapping`, `Alias`).
        - **`output_column`** — имя колонки в результирующей вьюхе.
        - **`mapping`** — логика преобразования, например:
          - для `FieldTransform`: `1 → Создан`, `2 → В работе`
          - для `JSON`: извлечение вложенных значений по ключам

- **`joins`** — объединения таблиц:
  - **`inner`** — внутреннее объединение (INNER JOIN):
    - **`source`** — имя источника второй таблицы.
    - **`schema`** — схема второй таблицы.
    - **`table`** — имя таблицы.
    - **`column_first`** — колонка из основной таблицы (`main_table` или первой в списке).
    - **`column_second`** — колонка из присоединяемой таблицы.
## 🧱 Логика трансформаций

InsightForge поддерживает гибкую систему трансформаций для формирования целевых колонок:
FieldTransform — используется для сопоставления значений поля с текстовыми лейблами (enum-подобные преобразования).
JSON — позволяет распарсить JSONB-поля и вытащить отдельные значения в колонки вьюхи. Используются поля mapping_json, где указывается тип данных и соответствие ключ → имя колонки.

## 📦 Требования и настройка окружения
Для запуска InsightForge и обеспечения корректной работы CDC через Debezium, необходимо развернуть следующие компоненты:

✅ Необходимые сервисы
- `Kafka`
- `Zookeeper`
- `Kafka Connect (с Debezium Connector for PostgreSQL)`
- `PostgreSQL — для источника (OLTP) и хранилища (DWH)`


## 🧩 Настройки PostgreSQL (для CDC)
Для каждого источника (OLTP PostgreSQL), подключаемого к Debezium, необходимо:
1. Включить логирование изменений (WAL)
В postgresql.conf:
- `wal_level = logical`
- `max_replication_slots = 4`
- `max_wal_senders = 4`
2. Создать пользователя с нужными правами
```
```sql
  CREATE ROLE replication_user WITH REPLICATION LOGIN PASSWORD 'password';
  GRANT SELECT ON ALL TABLES IN SCHEMA public TO replication_user;
  ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO replication_user;
  ```

3. Установить REPLICA IDENTITY FULL для всех отслеживаемых таблиц
```
```sql
ALTER TABLE public.users REPLICA IDENTITY FULL;
ALTER TABLE public.profiles REPLICA IDENTITY FULL;
```
