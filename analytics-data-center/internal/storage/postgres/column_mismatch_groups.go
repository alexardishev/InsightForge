package postgres

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const statusOpen = "open"

func (p *PostgresSys) UpsertMismatchGroup(ctx context.Context, group models.ColumnMismatchGroup, items []models.ColumnMismatchItem) (int64, error) {
	const op = "postgres.UpsertMismatchGroup"

	tx, err := p.Db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var existingID sql.NullInt64
	err = tx.QueryRowContext(ctx, `
                SELECT id FROM column_mismatch_groups
                WHERE schema_id=$1 AND database_name=$2 AND schema_name=$3 AND table_name=$4 AND resolved_at IS NULL
        `, group.SchemaID, strings.ToLower(group.DatabaseName), strings.ToLower(group.SchemaName), strings.ToLower(group.TableName)).Scan(&existingID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	groupID := int64(0)
	if existingID.Valid {
		groupID = existingID.Int64
		if _, err = tx.ExecContext(ctx, `DELETE FROM column_mismatch_items WHERE group_id=$1`, groupID); err != nil {
			return 0, err
		}
	} else {
		err = tx.QueryRowContext(ctx, `
                        INSERT INTO column_mismatch_groups (schema_id, database_name, schema_name, table_name, status)
                        VALUES ($1,$2,$3,$4,$5)
                        RETURNING id
                `, group.SchemaID, strings.ToLower(group.DatabaseName), strings.ToLower(group.SchemaName), strings.ToLower(group.TableName), statusOpen).Scan(&groupID)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", op, err)
		}
	}

	for _, item := range items {
		if _, err = tx.ExecContext(ctx, `
                        INSERT INTO column_mismatch_items (group_id, old_column_name, new_column_name, score, suggested)
                        VALUES ($1,$2,$3,$4,$5)
                `, groupID, item.OldColumnName, item.NewColumnName, item.Score, item.SuggestedMatch); err != nil {
			return 0, fmt.Errorf("%s: %w", op, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return groupID, nil
}

func (p *PostgresSys) HasOpenMismatchGroup(ctx context.Context, schemaID int64, database, schema, table string) (bool, error) {
	var exists bool
	err := p.Db.QueryRowContext(ctx, `
                SELECT EXISTS(
                    SELECT 1 FROM column_mismatch_groups
                    WHERE schema_id=$1 AND database_name=$2 AND schema_name=$3 AND table_name=$4 AND resolved_at IS NULL
                )
        `, schemaID, strings.ToLower(database), strings.ToLower(schema), strings.ToLower(table)).Scan(&exists)
	return exists, err
}

func (p *PostgresSys) ListMismatchGroups(ctx context.Context, filter models.ColumnMismatchGroupFilter) ([]models.ColumnMismatchGroup, error) {
	args := []interface{}{}
	where := []string{}

	if filter.SchemaID != nil {
		args = append(args, *filter.SchemaID)
		where = append(where, fmt.Sprintf("schema_id=$%d", len(args)))
	}
	if filter.DatabaseName != nil {
		args = append(args, strings.ToLower(*filter.DatabaseName))
		where = append(where, fmt.Sprintf("database_name=$%d", len(args)))
	}
	if filter.SchemaName != nil {
		args = append(args, strings.ToLower(*filter.SchemaName))
		where = append(where, fmt.Sprintf("schema_name=$%d", len(args)))
	}
	if filter.TableName != nil {
		args = append(args, strings.ToLower(*filter.TableName))
		where = append(where, fmt.Sprintf("table_name=$%d", len(args)))
	}
	if filter.Status != nil {
		args = append(args, *filter.Status)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}

	query := `SELECT id, schema_id, database_name, schema_name, table_name, status, created_at, resolved_at FROM column_mismatch_groups`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	order := "created_at"
	if filter.SortByCreatedAtDesc {
		order += " DESC"
	}
	query += " ORDER BY " + order

	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := p.Db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.ColumnMismatchGroup
	for rows.Next() {
		var g models.ColumnMismatchGroup
		if err := rows.Scan(&g.ID, &g.SchemaID, &g.DatabaseName, &g.SchemaName, &g.TableName, &g.Status, &g.CreatedAt, &g.ResolvedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}

	return groups, rows.Err()
}

func (p *PostgresSys) GetMismatchGroupByID(ctx context.Context, id int64) (models.ColumnMismatchGroup, []models.ColumnMismatchItem, error) {
	var g models.ColumnMismatchGroup
	err := p.Db.QueryRowContext(ctx, `
                SELECT id, schema_id, database_name, schema_name, table_name, status, created_at, resolved_at
                FROM column_mismatch_groups WHERE id=$1
        `, id).Scan(&g.ID, &g.SchemaID, &g.DatabaseName, &g.SchemaName, &g.TableName, &g.Status, &g.CreatedAt, &g.ResolvedAt)
	if err != nil {
		return models.ColumnMismatchGroup{}, nil, err
	}

	rows, err := p.Db.QueryContext(ctx, `
                SELECT id, group_id, old_column_name, new_column_name, score, suggested
                FROM column_mismatch_items WHERE group_id=$1
        `, id)
	if err != nil {
		return g, nil, err
	}
	defer rows.Close()

	var items []models.ColumnMismatchItem
	for rows.Next() {
		var item models.ColumnMismatchItem
		if err := rows.Scan(&item.ID, &item.GroupID, &item.OldColumnName, &item.NewColumnName, &item.Score, &item.SuggestedMatch); err != nil {
			return g, nil, err
		}
		items = append(items, item)
	}

	return g, items, rows.Err()
}

func (p *PostgresSys) ResolveMismatchGroup(ctx context.Context, id int64, status string) error {
	resolvedAt := time.Now()
	_, err := p.Db.ExecContext(ctx, `
                UPDATE column_mismatch_groups SET status=$1, resolved_at=$2 WHERE id=$3
        `, status, resolvedAt, id)
	return err
}
