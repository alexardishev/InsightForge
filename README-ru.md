# ⚡ InsightForge

InsightForge — это мощный инструмент для построения аналитических представлений (вьюх) из различных источников данных.
Позволяет собирать таблицы, объединять данные из разных сервисов, применять трансформации и хранить результат в аналитическом хранилище (DWH).

## ❓ Какие проблемы решает
В современных микросервисных системах данные разбросаны по разным базам и схемам. Аналитика требует сводных таблиц, которые часто собираются вручную с помощью сложных ETL‑скриптов. InsightForge автоматизирует этот процесс: объединяет разнородные источники, применяет трансформации и поддерживает вьюхи в актуальном состоянии, избавляя от необходимости писать собственные ETL пайплайны.

![ChatGPT Image 11 апр  2025 г , 12_25_40 (1)](https://github.com/user-attachments/assets/36cdafa8-a9b5-4524-b09b-941059bd7ed8)

---

## 🚀 Возможности

- 📦 Интеграция с внешними OLTP системами через API и события
- 🛠 Конструктор вью: таблицы, колонки, связи (JOIN), трансформации
- 💾 Поддержка PostgreSQL как источника и DWH, а также ClickHouse в роли DWH
- 🔄 Автообновление данных по CDC (Debezium) или событиям
- 🧠 Преобразование данных: enum, alias, text mappings и др.
- 🧪 Асинхронная ETL-обработка с отслеживанием статуса задач
- 📊 Логирование и метрики (OpenTelemetry-ready)
- ⚠️ *На текущий момент UI находится в разработке.*
     🛈 *При этом у вас уже реализованы все функции бэкенда и полноценное использование возможно с использованием INSERT в системную таблицу Schems по примеру, указанному ниже.*
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

## 🏗 Пример вьюхи (JSON)

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

Полный пример конфигурации находится в каталоге `examples/user_basic_info.json`.

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
      - **`alias`** — альтернативное имя колонки в результирующей вьюхе.
      - **`reference`** — ссылка на столбец другой таблицы, включает поля `source`, `schema`, `table`, `column`.
      - **`transform`** — описание преобразования данных:
        - **`type`** — тип трансформации (`JSON`, `FieldTransform` и др.).
        - **`mode`** — режим трансформации (например, `Mapping`, `Alias`).
        - **`output_column`** — имя колонки в результирующей вьюхе.
        - **`mapping`** — логика преобразования, например:
          - для `FieldTransform`: `1 → Создан`, `2 → В работе`
          - для `JSON`: извлечение вложенных значений по ключам
        - **`mapping.type_map`** — тип описания сопоставления (`JSON` или `FieldTransform`).
        - **`mapping.mapping`** — набор `значение из БД → преобразованное значение`.
        - **`mapping.alias_new_column_transform`** — имя новой колонки при создании трансформацией.
        - **`mapping.type_field`** — тип данных при обработке JSON.
        - **`mapping.mapping_json`** — список объектов вида `ключ в JSON → имя колонки`.

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
- `PostgreSQL — для источника (OLTP); может использоваться и как DWH`
- `ClickHouse — альтернативное хранилище DWH`


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

## 🛰 Как работает Kafka и Debezium
При инициализации `analytics-data-center` запускается компонент `cdc-listener`, который подключается к брокеру Kafka. Debezium через Kafka Connect считывает изменения из OLTP базы и отправляет их в топики. Как только в системной таблице Schemas появляется новая схема, приложение автоматически подписывается на соответствующие топики и начинает потреблять события.

Сначала обработчик считывает всю накопившуюся историю сообщений, поэтому сразу после добавления схемы возможны небольшие задержки. Когда очередь будет обработана, события начнут приходить почти в реальном времени. Kafka гарантирует порядок доставки внутри раздела, а потребитель отслеживает смещения, чтобы не пропустить обновления.


## ⚙️ Конфигурация и запуск
- Конфигурация хранится в файле `analytics-data-center/config/local.yaml` и считывается пакетом `internal/config`.
- Запуск осуществляется из `cmd/analytics-data-center/main.go`. Приложение поднимает HTTP сервер на порту `8888` и gRPC сервер на порту, указанном в параметре `grpc.port`.
- Путь к конфигурации передается через флаг `--config` или переменную окружения `CONFIG_PATH`.
- Собрать бинарник можно командой:
```bash
go build ./analytics-data-center/cmd/analytics-data-center
```
