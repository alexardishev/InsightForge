package postgresdwh

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lib/pq"
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
func (p *PostgresDWH) GetColumnsTables(ctx context.Context, schemaName string, tempTableName string) ([]string, error) {
	const op = "Storage.PostgreSQL.GetColumnsTempTables"
	var columns []string
	log := p.Log.With(
		slog.String("op", op),
	)

	query := "SELECT column_name FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position"
	rows, err := p.Db.QueryContext(ctx, query, schemaName, tempTableName)
	if err != nil {
		log.Error("Запросвыполнен с ошибкой", slog.String("error", err.Error()))
		return columns, err
	}
	defer rows.Close()

	for rows.Next() {
		var row string
		err = rows.Scan(&row)
		if err != nil {
			log.Error("Запросвыполнен с ошибкой", slog.String("error", err.Error()))
			return columns, err
		}
		columns = append(columns, row)
	}

	if err := rows.Err(); err != nil {
		log.Error("ошибка при проходе по строкам", slog.String("error", err.Error()))
		return nil, err
	}

	return columns, nil
}
func (p *PostgresDWH) CreateIndex(ctx context.Context, query string) error {
	const op = "Storage.PostgreSQL.CreateIndex"
	log := p.Log.With(
		slog.String("op", op),
	)

	_, err := p.Db.ExecContext(ctx, query)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "42703" {
			log.Warn("колонка не существует, индекс не создан", slog.String("column", pgErr.Column), slog.String("detail", pgErr.Message))
			return nil
		}

		log.Error("ошибка создания индексов", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (p *PostgresDWH) CreateConstraint(ctx context.Context, query string) error {
	const op = "Storage.PostgreSQL.CreateConstraint"
	log := p.Log.With(
		slog.String("op", op),
	)
	_, err := p.Db.ExecContext(ctx, query)
	if err != nil {
		log.Error("ошибка создания ограничений", slog.String("error", err.Error()))
		return err
	}
	return nil

}

func (p *PostgresDWH) RenameColumn(ctx context.Context, query string) error {
	const op = "Storage.PostgreSQL.RenameColumn"
	log := p.Log.With(
		slog.String("op", op),
		slog.String("query", query),
	)

	if _, err := p.Db.ExecContext(ctx, query); err != nil {
		log.Error("ошибка переименования колонки", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (p *PostgresDWH) DropColumn(ctx context.Context, query string) error {
	const op = "Storage.PostgreSQL.DropColumn"
	log := p.Log.With(
		slog.String("op", op),
		slog.String("query", query),
	)

	if _, err := p.Db.ExecContext(ctx, query); err != nil {
		log.Error("ошибка удаления колонки", slog.String("error", err.Error()))
		return err
	}

	return nil
}
