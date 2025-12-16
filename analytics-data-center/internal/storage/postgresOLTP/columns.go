package postgresoltp

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"log/slog"
)

func (p *PostgresOLTP) GetColumns(ctx context.Context, schemaName string, tableName string) ([]models.Column, error) {
	const op = "Storage.PostgreSQL.GetColumns"
	var columns []models.Column
	log := p.Log.With(
		slog.String("op", op),
	)

	query := `
SELECT
    column_name,
    CASE
        WHEN data_type = 'ARRAY' THEN udt_name
        WHEN data_type IN ('character varying', 'character', 'bit', 'bit varying') AND character_maximum_length IS NOT NULL
            THEN format('%s(%s)', data_type, character_maximum_length)
        WHEN data_type IN ('numeric', 'decimal') AND numeric_precision IS NOT NULL
            THEN format('%s(%s,%s)', data_type, numeric_precision, COALESCE(numeric_scale, 0))
        ELSE data_type
    END AS data_type
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2
ORDER BY ordinal_position`
	rows, err := p.Db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		log.Error("Запрос выполнен с ошибкой", slog.String("error", err.Error()))
		return []models.Column{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var row models.Column
		err = rows.Scan(&row.Name, &row.Type)
		if err != nil {
			log.Error("Запрос выполнен с ошибкой", slog.String("error", err.Error()))
			return []models.Column{}, err
		}
		columns = append(columns, row)
	}

	if err := rows.Err(); err != nil {
		log.Error("ошибка при проходе по строкам", slog.String("error", err.Error()))
		return nil, err
	}

	return columns, nil
}

func (p *PostgresOLTP) GetColumnInfo(ctx context.Context, tableName string, columnName string) (models.ColumnInfo, error) {
	const op = "Storage.PostgreSQL.GetColumnInfo"
	var columnInfo models.ColumnInfo
	log := p.Log.With(
		slog.String("op", op),
	)

	query := `
SELECT
    c.column_name,
    c.data_type,
    c.is_nullable,
    c.column_default,
    pgd.description,
    CASE WHEN tc.constraint_type = 'PRIMARY KEY' THEN true ELSE false END as is_primary_key,
    CASE WHEN fkc.constraint_type = 'FOREIGN KEY' THEN true ELSE false END as is_foreign_key,
    CASE WHEN uc.constraint_type = 'UNIQUE' THEN true ELSE false END as is_unique
FROM
    information_schema.columns c
LEFT JOIN
    pg_catalog.pg_statio_all_tables as st on st.relname = c.table_name
LEFT JOIN
    pg_catalog.pg_description pgd on pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
LEFT JOIN
    information_schema.key_column_usage kcu
    ON c.table_name = kcu.table_name
    AND c.column_name = kcu.column_name
LEFT JOIN
    information_schema.table_constraints tc
    ON kcu.constraint_name = tc.constraint_name
    AND tc.constraint_type = 'PRIMARY KEY'
LEFT JOIN
    information_schema.table_constraints fkc
    ON kcu.constraint_name = fkc.constraint_name
    AND fkc.constraint_type = 'FOREIGN KEY'
LEFT JOIN
    information_schema.table_constraints uc
    ON kcu.constraint_name = uc.constraint_name
    AND uc.constraint_type = 'UNIQUE'
WHERE
    c.table_name = $1
    AND c.column_name = $2
LIMIT 1
`

	var isNullStr string
	var defaultValue *string
	var description *string

	err := p.Db.QueryRowContext(ctx, query, tableName, columnName).Scan(
		&columnInfo.ColumnName,
		&columnInfo.Type,
		&isNullStr,
		&defaultValue,
		&description,
		&columnInfo.IsPK,
		&columnInfo.IsFK,
		&columnInfo.IsUnique,
	)
	if err != nil {
		log.Error("Запрос выполнен с ошибкой", slog.String("error", err.Error()))
		return models.ColumnInfo{}, err
	}

	columnInfo.IsNullable = isNullStr == "YES"
	columnInfo.Default = defaultValue
	columnInfo.Description = description

	return columnInfo, nil
}
