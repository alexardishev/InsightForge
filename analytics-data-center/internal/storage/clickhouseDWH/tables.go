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

func (p *ClickHouseDB) DeleteTempTable(ctx context.Context, tableName string) error {
	panic("DeleteTempTable")
}
func (p *ClickHouseDB) GetColumnsTables(ctx context.Context, schemaName string, tempTableName string) ([]string, error) {
	return []string{}, nil
}

func (p *ClickHouseDB) CreateIndex(ctx context.Context, query string) error {
	panic("CreateIndex")
}

func (p *ClickHouseDB) CreateConstraint(ctx context.Context, query string) error {
	panic("CreateConstraint")
}
