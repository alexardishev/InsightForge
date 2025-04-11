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

```yaml
view_name: user_equipment_report
sources:
  - name: prod_users
    schemas:
      - name: public
        tables:
          - name: users
            columns:
              - name: id
                alias: user_id
                is_update_key: true
              - name: name
              - name: status
                transform:
                  type: enum
                  mode: separate
                  output_column: status_text
                  mapping:
                    "1": Создано
                    "2": Согласовано
                    "3": Ошибка
