package postgresdwh

import (
	"context"
	"fmt"
	"log/slog"
)

func (p *PostgresDWH) CreateTempTable(ctx context.Context, query string, tempTableName string) error {
	const op = "Storage.PostgreSQL.CreateTempTablePostgres"
	log := p.Log.With(
		slog.String("op", op),
		slog.String("query", query),
	)
	log.Info("создание таблицы запущено")
	queryDrop := fmt.Sprintf("DROP TABLE IF EXISTS %s", tempTableName)
	_, err := p.Db.ExecContext(ctx, queryDrop)
	if err != nil {
		log.Error("ошибка при удалении таблицы", slog.String("ошибка", err.Error()))
		return err
	}
	log.Info("Таблица была удалена", slog.String("имя таблицы", tempTableName))

	stmt, err := p.Db.Prepare(query)
	if err != nil {
		log.Error("ошибка подготовки запроса", slog.String("error", err.Error()))
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		log.Error("не удалось удалить создать временную таблицу", slog.String("error", err.Error()))
		return err
	}
	log.Info("Таблица была создана", slog.String("имя таблицы", tempTableName))
	return nil
}

func (p *PostgresDWH) DeleteTempTable(ctx context.Context, tableName string) error {
	const op = "Storage.PostgreSQL.DeleteTempTablePostgres"
	log := p.Log.With(
		slog.String("op", op),
		slog.String("tableName", tableName),
	)
	log.Info("удаление таблицы запущено")

	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	stmt, err := p.Db.Prepare(query)
	if err != nil {
		log.Error("ошибка подготовки запроса", slog.String("error", err.Error()))
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		log.Error("не удалось удалить таблицу", slog.String("error", err.Error()))
		return err
	}
	return nil
}
