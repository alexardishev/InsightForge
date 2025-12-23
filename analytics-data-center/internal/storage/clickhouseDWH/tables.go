package clickhousedwh

import (
	"context"
	"fmt"
	"log/slog"
)

func (c *ClickHouseDB) CreateTempTable(ctx context.Context, query string, tempTableName string) error {
	const op = "Storage.ClickHouseDB.CreateTempTableClickHouseDB"
	log := c.Log.With(
		slog.String("op", op),
		slog.String("query", query),
	)
	log.Info("создание таблицы запущено")
	queryDrop := fmt.Sprintf("DROP TABLE IF EXISTS %s", tempTableName)
	_, err := c.Db.ExecContext(ctx, queryDrop)
	if err != nil {
		log.Error("ошибка при удалении таблицы", slog.String("ошибка", err.Error()))
		return err
	}
	log.Info("Таблица была удалена", slog.String("имя таблицы", tempTableName))

	_, err = c.Db.ExecContext(ctx, query)
	if err != nil {
		log.Error("не удалось удалить создать временную таблицу", slog.String("error", err.Error()))
		return err
	}
	log.Info("Таблица была создана", slog.String("имя таблицы", tempTableName))
	return nil
}

func (c *ClickHouseDB) DeleteTempTable(ctx context.Context, tableName string) error {
	const op = "Storage.ClickHouseDB.DeleteTempTable"
	log := c.Log.With(
		slog.String("op", op),
		slog.String("tableName", tableName),
	)
	log.Info("удаление таблицы запущено")

	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	_, err := c.Db.ExecContext(ctx, query)
	if err != nil {
		log.Error("не удалось удалить таблицу", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (c *ClickHouseDB) GetColumnsTables(ctx context.Context, schemaName string, tempTableName string) ([]string, error) {
	const op = "Storage.ClickHouseDB.GetColumnsTempTables"
	var columns []string
	log := c.Log.With(
		slog.String("op", op),
	)

	query := "SELECT name FROM system.columns WHERE database = ? AND table = ? ORDER BY position"
	rows, err := c.Db.QueryContext(ctx, query, schemaName, tempTableName)
	if err != nil {
		log.Error("Запрос выполнен с ошибкой", slog.String("error", err.Error()))
		return columns, err
	}
	defer rows.Close()

	for rows.Next() {
		var colName string
		err = rows.Scan(&colName)
		if err != nil {
			log.Error("ошибка чтения строки", slog.String("error", err.Error()))
			return columns, err
		}
		columns = append(columns, colName)
	}

	if err := rows.Err(); err != nil {
		log.Error("ошибка при проходе по строкам", slog.String("error", err.Error()))
		return nil, err
	}

	return columns, nil
}

func (p *ClickHouseDB) CreateIndex(ctx context.Context, query string) error {
	panic("CreateIndex")
}

func (p *ClickHouseDB) CreateConstraint(ctx context.Context, query string) error {
	panic("CreateConstraint")
}

func (c *ClickHouseDB) RenameColumn(ctx context.Context, query string) error {
	const op = "Storage.ClickHouseDB.RenameColumn"
	log := c.Log.With(
		slog.String("op", op),
		slog.String("query", query),
	)

	if _, err := c.Db.ExecContext(ctx, query); err != nil {
		log.Error("ошибка переименования колонки", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (c *ClickHouseDB) DropColumn(ctx context.Context, query string) error {
	const op = "Storage.ClickHouseDB.DropColumn"
	log := c.Log.With(
		slog.String("op", op),
		slog.String("query", query),
	)

	if _, err := c.Db.ExecContext(ctx, query); err != nil {
		log.Error("ошибка удаления колонки", slog.String("error", err.Error()))
		return err
	}

	return nil
}
