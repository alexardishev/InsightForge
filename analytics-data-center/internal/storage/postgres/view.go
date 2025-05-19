package postgres

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

func (p *PostgresSys) GetView(ctx context.Context, idView int64) (models.View, error) {
	var rowSchema []byte

	const op = "Storage.PostgreSQL.GetSchema"
	log := p.Log.With(
		slog.String("op", op),
		slog.Int64("appID", idView),
	)

	log.Info("Operation starting")
	query := "SELECT schema_view FROM schems WHERE id = ($1)"
	err := p.Db.QueryRowContext(ctx, query, idView).Scan(&rowSchema)
	if errors.Is(err, sql.ErrNoRows) {
		log.Warn("конфигурация представления не найдена")
		return models.View{}, storage.ErrSessionNotFound
	}
	if err != nil {
		log.Error("Запрос выполнен с ошибкой", slog.String("error", err.Error()))
		return models.View{}, err
	}
	var view models.View
	if err := json.Unmarshal(rowSchema, &view); err != nil {
		log.Error("Ошибка при разборе JSON", slog.String("error", err.Error()))
		return models.View{}, err
	}

	return view, nil

}

func (p *PostgresSys) GetSchems(ctx context.Context, source string, schema string, table string) ([]int, error) {
	var schemsIds []int
	const op = "Storage.PostgreSQL.GetSchems"
	log := p.Log.With(slog.String("op", op))
	log.Info("Operation starting")

	sanitize := func(s string) string {
		return strings.ReplaceAll(s, `"`, `\"`)
	}

	query := fmt.Sprintf(`
	SELECT id
	FROM schems
	WHERE jsonb_path_exists(schema_view::jsonb,
	'$.sources[*] ? (@.name == "%s").schemas[*] ? (@.name == "%s").tables[*] ? (@.name == "%s")')`,
		sanitize(source), sanitize(schema), sanitize(table))

	rows, err := p.Db.QueryContext(ctx, query)
	if err != nil {
		log.Error("Запрос выполнен с ошибкой", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var idSchema int
		if err := rows.Scan(&idSchema); err != nil {
			log.Error("Ошибка сканирования строки", slog.String("error", err.Error()))
			continue
		}
		schemsIds = append(schemsIds, idSchema)
	}

	if err := rows.Err(); err != nil {
		log.Error("Ошибка после сканирования всех строк", slog.String("error", err.Error()))
		return nil, err
	}

	return schemsIds, nil
}
