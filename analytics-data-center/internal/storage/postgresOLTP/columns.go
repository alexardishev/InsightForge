package postgresoltp

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"database/sql"
	"log/slog"
	"strings"
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
    is_nullable,
    data_type,
    udt_schema,
    udt_name,
    character_maximum_length,
    numeric_precision,
    numeric_scale
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
		var nullable string
		var charMax sql.NullInt64
		var numPrec sql.NullInt64
		var numScale sql.NullInt64
		err = rows.Scan(
			&row.Name,
			&nullable,
			&row.DataType,
			&row.UdtSchema,
			&row.UdtName,
			&charMax,
			&numPrec,
			&numScale,
		)
		if err != nil {
			log.Error("Запрос выполнен с ошибкой", slog.String("error", err.Error()))
			return []models.Column{}, err
		}

		if charMax.Valid {
			row.CharMaxLen = &charMax.Int64
		}
		if numPrec.Valid {
			row.NumPrecision = &numPrec.Int64
		}
		if numScale.Valid {
			row.NumScale = &numScale.Int64
		}

		row.IsNullable = nullable == "YES"
		// Сохраняем Type для обратной совместимости: массивы получают имя элемента, остальные — data_type.
		if strings.EqualFold(row.DataType, "ARRAY") && row.UdtName != "" {
			row.Type = row.UdtName
		} else {
			row.Type = row.DataType
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
