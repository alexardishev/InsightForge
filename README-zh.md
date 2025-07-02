# ⚡ InsightForge

InsightForge 是一款用于从多个数据源构建分析视图的强大工具。它可以收集表格、合并来自不同服务的数据，执行转换并将结果存储在数据仓库中。

## ❓ 解决的问题
在微服务架构中，数据往往分散在多套数据库和模式中，分析人员需要手动编写 ETL 脚本才能将其汇总。InsightForge 自动整合异构来源的数据，在生成分析视图的同时持续同步更新，减少了维护自建 ETL 流程和字段映射的不便。

## 🚀 功能
- 📦 通过 API 和事件与外部 OLTP 系统集成
- 🛠 视图构建器：表、列、关联 (JOIN) 与数据转换
- 💾 PostgreSQL 既可作为源也可作为 DWH，同时支持 ClickHouse 作为 DWH
- 🔄 通过 Debezium 的 CDC 或事件自动更新数据
- 🧠 数据转换：枚举、别名映射、JSON 字段拆分等
- 🧪 异步 ETL 处理并追踪任务状态
- 📊 日志和指标（兼容 OpenTelemetry）
- ⚠️ UI 仍在开发中，目前可以通过配置文件完整使用后端功能。

## 🧱 架构
- `analytics-view-service` —— 负责组装和刷新视图的核心服务
- `task-service` —— 跟踪 ETL 任务状态
- `sql-generator` —— 生成 SQL 脚本（CREATE TABLE、INSERT 等）
- `cdc-listener`（可选）—— 监听 Kafka 变更
- `config-loader` —— 载入并校验 YAML/JSON 配置

## 📂 项目结构
```
analytics-data-center/
├── cmd/
├── internal/
│   ├── app/
│   ├── config/
│   ├── domain/models/
│   ├── services/
│   ├── storage/
│   └── lib/
├── config/
└── README.md
```
前端位于 `client/`，使用 React、TypeScript 和 Vite 构建。

## 📦 配置
所有运行参数在 `analytics-data-center/config/local.yaml` 中定义，并由 `internal/config` 读取。`cmd/analytics-data-center/main.go` 会同时启动 HTTP（8888 端口）和 gRPC 服务（端口在配置中指定）。

## 🏗 示例视图 (JSON)
详细的配置文件示例位于 `examples/user_basic_info.json`，`README-ru.md` 中也有同样的内容。

## 🔍 配置字段说明
- **`view_name`** —— 最终视图（表）的名称。
- **`main_table`** —— 作为连接基准的主表，未指定时默认为第一张表。
- **`sources`** —— 数据源列表：
  - **`name`** —— 数据源名称（通常是数据库或其别名）。
  - **`schemas`** —— 包含表的模式列表。
  - **`tables`** —— 参与构建的表：
    - **`columns`** —— 表中的列：
      - **`name`** —— 源表中的列名。
      - **`type`** —— 数据类型。
      - **`view_key`** —— 用于更新的视图字段名。
      - **`is_update_key`** —— 是否参与更新逻辑。
      - **`is_primary_key`** —— 是否构成视图的主键。
      - **`is_nullable`** —— 是否允许 `NULL`。
      - **`is_deleted`** —— 在源中被删除但在视图中保留为空。
      - **`alias`** —— 在视图中的自定义列名。
      - **`reference`** —— 指向其他表的字段（包含 `source`、`schema`、`table`、`column`）。
      - **`transform`** —— 数据转换规则：
        - **`type`** —— 转换类型（如 `JSON`、`FieldTransform`）。
        - **`mode`** —— 转换模式（例如 `Mapping`）。
        - **`output_column`** —— 结果列名。
        - **`mapping`** —— 转换映射：
          - **`type_map`** —— 映射描述方式（`JSON` 或 `FieldTransform`）。
          - **`mapping`** —— 值到值的映射表。
          - **`alias_new_column_transform`** —— 新列的名称。
          - **`type_field`** —— 处理 JSON 时的字段类型。
          - **`mapping_json`** —— `JSON 字段 → 视图列` 的对应关系列表。
- **`joins`** —— 表连接：
  - **`inner`** —— INNER JOIN 描述：
    - **`source`**、**`schema`**、**`table`** —— 连接表的位置。
    - **`column_first`** —— 主表中的列。
    - **`column_second`** —— 连接表中的列。

## 🧩 PostgreSQL CDC 设置
为每个需要 CDC 的 PostgreSQL 数据库：
1. 打开 WAL 日志并配置复制槽（`wal_level = logical` 等）。
2. 创建复制用户并授予 SELECT 权限。
3. 为跟踪表设置 `REPLICA IDENTITY FULL`。

```sql
ALTER TABLE public.users REPLICA IDENTITY FULL;
ALTER TABLE public.profiles REPLICA IDENTITY FULL;
```

## 🛰 Kafka 事件流
初始化时，`cdc-listener` 会连接到 Kafka 集群。Debezium 借助 Kafka Connect 将 OLTP 数据库的变更写入多个主题。只要在系统表 Schemas 创建了新模式，应用就会自动订阅相应主题并开始消费事件。

监听器会先读取所有历史消息，因此在最初阶段可能出现延迟。随着积压处理完成，后续的变更将几乎实时到达。Kafka 在分区内保证消息顺序，消费者会记录偏移量以避免数据丢失。

InsightForge 采用 MIT 许可证发布。
