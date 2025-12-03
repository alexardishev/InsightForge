package postgres

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

func (p *PostgresSys) CreateMismatchGroup(ctx context.Context, group models.ColumnMismatchGroup, items []models.ColumnMismatchItem) (int64, error) {
	const op = "storage.PostgreSQL.CreateMismatchGroup"
	log := p.Log.With(slog.String("op", op))

	tx, err := p.Db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin tx", slog.String("error", err.Error()))
		return 0, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `INSERT INTO column_mismatch_groups (schema_id, database_name, schema_name, table_name, status)
VALUES ($1, $2, $3, $4, $5) RETURNING id`

	var id int64
	if err = tx.QueryRowContext(ctx, query,
		group.SchemaID,
		group.DatabaseName,
		group.SchemaName,
		group.TableName,
		group.Status,
	).Scan(&id); err != nil {
		log.Error("failed to insert mismatch group", slog.String("error", err.Error()))
		return 0, err
	}

	if err = insertMismatchItems(ctx, tx, id, items); err != nil {
		log.Error("failed to insert mismatch items", slog.String("error", err.Error()))
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		log.Error("failed to commit mismatch group", slog.String("error", err.Error()))
		return 0, err
	}

	return id, nil
}

func (p *PostgresSys) ReplaceMismatchItems(ctx context.Context, groupID int64, items []models.ColumnMismatchItem) error {
	const op = "storage.PostgreSQL.ReplaceMismatchItems"
	log := p.Log.With(slog.String("op", op), slog.Int64("group_id", groupID))

	tx, err := p.Db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin tx", slog.String("error", err.Error()))
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM column_mismatch_items WHERE group_id = $1`, groupID); err != nil {
		log.Error("failed to cleanup items", slog.String("error", err.Error()))
		return err
	}

	if err = insertMismatchItems(ctx, tx, groupID, items); err != nil {
		log.Error("failed to insert items", slog.String("error", err.Error()))
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Error("failed to commit items", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (p *PostgresSys) GetOpenMismatchGroup(ctx context.Context, schemaID int64, database, schema, table string) (models.ColumnMismatchGroupWithItems, error) {
	const op = "storage.PostgreSQL.GetOpenMismatchGroup"
	log := p.Log.With(slog.String("op", op))

	groupQuery := `SELECT id, schema_id, database_name, schema_name, table_name, status, created_at, resolved_at
FROM column_mismatch_groups WHERE schema_id = $1 AND database_name = $2 AND schema_name = $3 AND table_name = $4 AND status = $5`

	var group models.ColumnMismatchGroup
	if err := p.Db.QueryRowContext(ctx, groupQuery, schemaID, database, schema, table, models.ColumnMismatchStatusOpen).Scan(
		&group.ID,
		&group.SchemaID,
		&group.DatabaseName,
		&group.SchemaName,
		&group.TableName,
		&group.Status,
		&group.CreatedAt,
		&group.ResolvedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return models.ColumnMismatchGroupWithItems{}, storage.ErrMismatchNotFound
		}
		log.Error("failed to fetch mismatch group", slog.String("error", err.Error()))
		return models.ColumnMismatchGroupWithItems{}, err
	}

	items, err := p.loadMismatchItems(ctx, group.ID)
	if err != nil {
		return models.ColumnMismatchGroupWithItems{}, err
	}

	return models.ColumnMismatchGroupWithItems{Group: group, Items: items}, nil
}

func (p *PostgresSys) ListMismatchGroups(ctx context.Context, filter models.ColumnMismatchFilter) ([]models.ColumnMismatchGroup, error) {
	const op = "storage.PostgreSQL.ListMismatchGroups"
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

	if filter.Status != nil {
		args = append(args, *filter.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}

	query := `SELECT id, schema_id, database_name, schema_name, table_name, status, created_at, resolved_at FROM column_mismatch_groups`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

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
		log.Error("failed to list mismatch groups", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	var groups []models.ColumnMismatchGroup
	for rows.Next() {
		var group models.ColumnMismatchGroup
		if err := rows.Scan(
			&group.ID,
			&group.SchemaID,
			&group.DatabaseName,
			&group.SchemaName,
			&group.TableName,
			&group.Status,
			&group.CreatedAt,
			&group.ResolvedAt,
		); err != nil {
			log.Error("failed to scan mismatch group", slog.String("error", err.Error()))
			return nil, err
		}
		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", slog.String("error", err.Error()))
		return nil, err
	}

	return groups, nil
}

func (p *PostgresSys) GetMismatchGroup(ctx context.Context, id int64) (models.ColumnMismatchGroupWithItems, error) {
	const op = "storage.PostgreSQL.GetMismatchGroup"
	log := p.Log.With(slog.String("op", op), slog.Int64("group_id", id))

	query := `SELECT id, schema_id, database_name, schema_name, table_name, status, created_at, resolved_at
FROM column_mismatch_groups WHERE id = $1`

	var group models.ColumnMismatchGroup
	if err := p.Db.QueryRowContext(ctx, query, id).Scan(
		&group.ID,
		&group.SchemaID,
		&group.DatabaseName,
		&group.SchemaName,
		&group.TableName,
		&group.Status,
		&group.CreatedAt,
		&group.ResolvedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return models.ColumnMismatchGroupWithItems{}, storage.ErrMismatchNotFound
		}
		log.Error("failed to get mismatch group", slog.String("error", err.Error()))
		return models.ColumnMismatchGroupWithItems{}, err
	}

	items, err := p.loadMismatchItems(ctx, group.ID)
	if err != nil {
		return models.ColumnMismatchGroupWithItems{}, err
	}

	return models.ColumnMismatchGroupWithItems{Group: group, Items: items}, nil
}

func (p *PostgresSys) ResolveMismatchGroup(ctx context.Context, id int64) error {
	const op = "storage.PostgreSQL.ResolveMismatchGroup"
	log := p.Log.With(slog.String("op", op), slog.Int64("group_id", id))

	query := `UPDATE column_mismatch_groups SET status = $1, resolved_at = now() WHERE id = $2`
	res, err := p.Db.ExecContext(ctx, query, models.ColumnMismatchStatusResolved, id)
	if err != nil {
		log.Error("failed to resolve mismatch group", slog.String("error", err.Error()))
		return err
	}

	if affected, err := res.RowsAffected(); err == nil && affected == 0 {
		return storage.ErrMismatchNotFound
	}

	return nil
}

func (p *PostgresSys) loadMismatchItems(ctx context.Context, groupID int64) ([]models.ColumnMismatchItem, error) {
	const op = "storage.PostgreSQL.loadMismatchItems"
	log := p.Log.With(slog.String("op", op), slog.Int64("group_id", groupID))

	query := `SELECT id, group_id, old_column_name, new_column_name, score, mismatch_type FROM column_mismatch_items WHERE group_id = $1`
	rows, err := p.Db.QueryContext(ctx, query, groupID)
	if err != nil {
		log.Error("failed to fetch mismatch items", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	var items []models.ColumnMismatchItem
	for rows.Next() {
		var item models.ColumnMismatchItem
		if err := rows.Scan(&item.ID, &item.GroupID, &item.OldColumnName, &item.NewColumnName, &item.Score, &item.Type); err != nil {
			log.Error("failed to scan mismatch item", slog.String("error", err.Error()))
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", slog.String("error", err.Error()))
		return nil, err
	}

	return items, nil
}

func insertMismatchItems(ctx context.Context, tx *sql.Tx, groupID int64, items []models.ColumnMismatchItem) error {
	if len(items) == 0 {
		return nil
	}

	query := `INSERT INTO column_mismatch_items (group_id, old_column_name, new_column_name, score, mismatch_type)
VALUES ($1, $2, $3, $4, $5)`

	for _, item := range items {
		if _, err := tx.ExecContext(ctx, query, groupID, item.OldColumnName, item.NewColumnName, item.Score, item.Type); err != nil {
			return err
		}
	}

	return nil
}
