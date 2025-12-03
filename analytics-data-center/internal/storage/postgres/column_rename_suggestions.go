package postgres

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

func (p *PostgresSys) CreateSuggestion(ctx context.Context, s models.ColumnRenameSuggestion) error {
	const op = "storage.PostgreSQL.CreateSuggestion"
	log := p.Log.With(slog.String("op", op))

	query := `INSERT INTO column_rename_suggestions (
    schema_id, database_name, schema_name, table_name, old_column_name, new_column_name, strategy, task_number
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	if _, err := p.Db.ExecContext(ctx, query,
		s.SchemaID,
		s.DatabaseName,
		s.SchemaName,
		s.TableName,
		s.OldColumnName,
		s.NewColumnName,
		s.Strategy,
		s.TaskNumber,
	); err != nil {
		log.Error("failed to create suggestion", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (p *PostgresSys) ListSuggestions(ctx context.Context, filter models.ColumnRenameSuggestionFilter) ([]models.ColumnRenameSuggestion, error) {
	const op = "storage.PostgreSQL.ListSuggestions"
	log := p.Log.With(slog.String("op", op))

	var args []interface{}
	var conditions []string

	if filter.SchemaID != nil {
		args = append(args, *filter.SchemaID)
		conditions = append(conditions, fmt.Sprintf("schema_id = $%d", len(args)))
	}

	if filter.DatabaseName != nil {
		args = append(args, *filter.DatabaseName)
		conditions = append(conditions, fmt.Sprintf("database_name = $%d", len(args)))
	}

	if filter.SchemaName != nil {
		args = append(args, *filter.SchemaName)
		conditions = append(conditions, fmt.Sprintf("schema_name = $%d", len(args)))
	}

	if filter.TableName != nil {
		args = append(args, *filter.TableName)
		conditions = append(conditions, fmt.Sprintf("table_name = $%d", len(args)))
	}

	baseQuery := `SELECT id, schema_id, database_name, schema_name, table_name, old_column_name, new_column_name, strategy, task_number, created_at FROM column_rename_suggestions`

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	order := "ASC"
	if filter.SortByCreatedAtDesc {
		order = "DESC"
	}

	baseQuery += fmt.Sprintf(" ORDER BY created_at %s", order)

	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		baseQuery += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		baseQuery += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := p.Db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		log.Error("failed to list suggestions", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	var suggestions []models.ColumnRenameSuggestion
	for rows.Next() {
		var s models.ColumnRenameSuggestion
		if err := rows.Scan(
			&s.ID,
			&s.SchemaID,
			&s.DatabaseName,
			&s.SchemaName,
			&s.TableName,
			&s.OldColumnName,
			&s.NewColumnName,
			&s.Strategy,
			&s.TaskNumber,
			&s.CreatedAt,
		); err != nil {
			log.Error("failed to scan suggestion", slog.String("error", err.Error()))
			return nil, err
		}
		suggestions = append(suggestions, s)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", slog.String("error", err.Error()))
		return nil, err
	}

	return suggestions, nil
}

func (p *PostgresSys) HasSuggestion(ctx context.Context, schemaID int64, database, schema, table string) (bool, error) {
	const op = "storage.PostgreSQL.HasSuggestion"
	log := p.Log.With(slog.String("op", op))

	query := `SELECT EXISTS(SELECT 1 FROM column_rename_suggestions WHERE schema_id = $1 AND database_name = $2 AND schema_name = $3 AND table_name = $4)`

	var exists sql.NullBool
	if err := p.Db.QueryRowContext(ctx, query, schemaID, database, schema, table).Scan(&exists); err != nil {
		log.Error("failed to check suggestion", slog.String("error", err.Error()))
		return false, err
	}

	return exists.Valid && exists.Bool, nil
}
