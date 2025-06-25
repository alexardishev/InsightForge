package clickhousedwh

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

func (c *ClickHouseDB) InsertDataToDWH(ctx context.Context, query string) error {
	const op = "Storage.ClickHouseDB.InsertDataToDWH"
	log := c.Log.With(
		slog.String("op", op),
	)
	_, err := c.Db.ExecContext(ctx, query)
	if err != nil {
		log.Error("ошибка подготовки запроса", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (c *ClickHouseDB) MergeTempTables(ctx context.Context, query string) error {
	const op = "Storage.ClickHouseDB.MergeTempTables"
	log := c.Log.With(
		slog.String("op", op),
	)
	_, err := c.Db.ExecContext(ctx, query)
	if err != nil {
		log.Error("ошибка подготовки запроса", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (c *ClickHouseDB) ReplicaIdentityFull(ctx context.Context, tableDWHName string) error {
	panic("ReplicaIdentityFull")
}

func (c *ClickHouseDB) InsertOrUpdateTransactional(
	ctx context.Context,
	tableName string,
	row map[string]interface{},
	colStr []string,
) error {
	columns := make([]string, 0, len(row))
	placeholders := make([]string, 0, len(row))
	values := make([]interface{}, 0, len(row))

	for col, val := range row {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	// Всегда добавляем updated_at
	columns = append(columns, "updated_at")
	placeholders = append(placeholders, "now()")

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := c.Db.ExecContext(ctx, query, values...)
	if err != nil {
		c.Log.Error("ошибка при вставке строки", slog.String("query", query), slog.Any("err", err))
		return fmt.Errorf("ошибка вставки в ClickHouse: %w", err)
	}
	return nil
}
