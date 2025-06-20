package clickhousedwh

import (
	"context"
	"database/sql"
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
	schemaName string,
	row map[string]interface{},
	conflictColumns []string,
) error {
	panic("InsertOrUpdateTransactional")
}
func insertWithTx(ctx context.Context, tx *sql.Tx, table string, row map[string]interface{}) error {
	columns := make([]string, 0, len(row))
	placeholders := make([]string, 0, len(row))
	values := make([]interface{}, 0, len(row))

	i := 1
	for col, val := range row {
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		i++
	}

	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := tx.ExecContext(ctx, query, values...)
	return err
}

func updateWithTx(ctx context.Context, tx *sql.Tx, table string, row map[string]interface{}, conflictColumns []string) error {
	setParts := []string{}
	whereParts := []string{}
	values := []interface{}{}

	i := 1
	for col, val := range row {
		isKey := false
		for _, k := range conflictColumns {
			if k == col {
				isKey = true
				break
			}
		}
		if isKey {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", col, i))
		values = append(values, val)
		i++
	}

	for _, col := range conflictColumns {
		whereParts = append(whereParts, fmt.Sprintf("%s = $%d", col, i))
		values = append(values, row[col])
		i++
	}

	query := fmt.Sprintf(`UPDATE %s SET %s WHERE %s`,
		table,
		strings.Join(setParts, ", "),
		strings.Join(whereParts, " AND "),
	)

	_, err := tx.ExecContext(ctx, query, values...)
	return err
}
