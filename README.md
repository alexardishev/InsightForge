# ⚡ InsightForge

*English documentation*  |  [Русская версия](README-ru.md)  |  [中文版](README-zh.md)

InsightForge is a powerful tool for building analytical views from various data sources. It allows you to gather tables, merge data from different services, apply transformations and store the result in a data warehouse (DWH).

## ❓ Problems solved
Modern applications scatter data across multiple microservices and databases. Analysts often have to manually stitch those pieces together with ad-hoc ETL scripts. InsightForge automates this workflow by consolidating heterogeneous sources into a unified view, applying transformations on the fly and keeping the result up to date. This removes the pain of hand-written ETL pipelines and inconsistent mappings.

## 🚀 Features
- 📦 Integration with external OLTP systems via API and events
- 🛠 View builder with tables, columns, joins and transformations
- 💾 PostgreSQL support as both source and DWH; ClickHouse supported as DWH
- 🔄 Automatic data updates using CDC (Debezium) or events
- 🧠 Data transforms: enums, alias mapping, JSON field extraction and more
- 🧪 Asynchronous ETL processing with task status tracking
- 📊 Logging and metrics (OpenTelemetry-ready)
- ⚠️ The UI is under active development. Full backend functionality is already available via configuration files.

## 🧱 Architecture
- `analytics-view-service` – core service for assembling and refreshing views
- `task-service` – tracks ETL job status
- `sql-generator` – builds SQL scripts (CREATE TABLE, INSERT, etc.)
- `cdc-listener` (optional) – listens to Kafka and reacts to changes
- `config-loader` – loads and validates view configuration (YAML/JSON)

## 📂 Project structure
```
analytics-data-center/
├── cmd/                      # Entry points
├── internal/
│   ├── app/                  # Application initialization
│   ├── config/               # Configuration loader
│   ├── domain/models/        # Domain entities
│   ├── services/             # Business logic
│   ├── storage/              # Database layer
│   └── lib/                  # Utilities (SQL generator, validation)
├── config/                   # YAML configs
├── go.mod
└── README.md
```

The front‑end resides in `client/` and is built with React, TypeScript and Vite.

## 📦 Configuration
Runtime settings are defined in `analytics-data-center/config/local.yaml` and parsed by `internal/config`. The main entry point (`cmd/analytics-data-center/main.go`) launches both an HTTP server (port 8888) and a gRPC server (port specified in the config).

## 🏗 Example view (JSON)
See `README-ru.md` for a detailed JSON example of a view definition. You can also
find the same file in `examples/user_basic_info.json`.

## 🔍 Schema fields
- **`view_name`** – name of the resulting analytical view (table).
- **`main_table`** – main table for joins; defaults to the first table if not set.
- **`sources`** – list of data sources:
  - **`name`** – source name (usually a database name or alias).
  - **`schemas`** – schemas to read tables from.
  - **`tables`** – participating tables:
    - **`columns`** – columns inside a table:
      - **`name`** – column name in the source table.
      - **`type`** – data type.
      - **`view_key`** – key column name in the resulting view used to update rows.
      - **`is_update_key`** – whether column participates in update logic.
      - **`is_primary_key`** – whether column forms the primary key of the view.
      - **`is_nullable`** – whether NULL values are allowed.
      - **`is_deleted`** – marks column as removed in the source but kept as NULL in the view.
      - **`alias`** – custom name of the column in the view.
      - **`reference`** – pointer to another table (`source`, `schema`, `table`, `column`).
      - **`transform`** – transformation rules:
        - **`type`** – transformation type (`JSON`, `FieldTransform`, ...).
        - **`mode`** – transformation mode (e.g. `Mapping`).
        - **`output_column`** – name of the generated column.
        - **`mapping`** – transformation mapping:
          - **`type_map`** – how mapping is described (`JSON` or `FieldTransform`).
          - **`mapping`** – value-to-value mapping.
          - **`alias_new_column_transform`** – new column name when creating from mapping.
          - **`type_field`** – data type for JSON processing.
          - **`mapping_json`** – list of mappings `json field → view column`.
- **`joins`** – table joins:
  - **`inner`** – inner join specification:
    - **`source`**, **`schema`**, **`table`** – location of the joined table.
    - **`column_first`** – column from the main table.
    - **`column_second`** – column from the joined table.

## 🧩 PostgreSQL CDC setup
To enable CDC via Debezium for each OLTP PostgreSQL source:
1. Enable WAL logging (`wal_level = logical`) and replication slots.
2. Create a replication user and grant SELECT permissions.
3. Set `REPLICA IDENTITY FULL` for tracked tables.

```sql
ALTER TABLE public.users REPLICA IDENTITY FULL;
ALTER TABLE public.profiles REPLICA IDENTITY FULL;
```

InsightForge is licensed under the MIT License.
