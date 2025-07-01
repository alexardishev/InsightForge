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

func (p *PostgresSys) UpdateView(ctx context.Context, view models.View, schemaId int) error {
	const op = "Storage.PostgreSQL.UpdateView"
	log := p.Log.With(slog.String("op", op))
	log.Info("Operation starting")
	unmarshalView, err := json.Marshal(view)
	if err != nil {
		log.Error("Ошибка перевода в JSON", slog.String("error", err.Error()))
		return err
	}

	query := "UPDATE schems set schema_view = ($1) where id = ($2)"
	_, err = p.Db.ExecContext(ctx, query, unmarshalView, schemaId)
	if err != nil {
		log.Error("Ошибка обновления схемы", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (p *PostgresSys) UploadView(ctx context.Context, view models.View) (int64, error) {
	const op = "Storage.PostgreSQL.UploadVIew"
	log := p.Log.With(slog.String("op", op))
	log.Info("Operation starting")
	var id int64
	unmarshalView, err := json.Marshal(view)
	if err != nil {
		log.Error("Ошибка перевода в JSON", slog.String("error", err.Error()))
		return 0, err
	}
	query := "INSERT INTO schems (schema_view) VALUES ($1) RETURNING id"
	err = p.Db.QueryRowContext(ctx, query, unmarshalView).Scan(&id)
	if err != nil {
		log.Error("Ошибка создания схемы", slog.String("error", err.Error()))
		return 0, nil
	}
	return id, nil
}

func (p *PostgresSys) ListTopics(ctx context.Context) ([]string, error) {
	const op = "Storage.PostgreSQL.ListTopics"
	log := p.Log.With(slog.String("op", op))

	query := `SELECT DISTINCT
               src->>'name' AS source,
               sch->>'name' AS schema,
               tbl->>'name' AS table
       FROM schems,
               LATERAL jsonb_array_elements(schema_view->'sources') AS s(src),
               LATERAL jsonb_array_elements(src->'schemas') AS sc(sch),
               LATERAL jsonb_array_elements(sch->'tables') AS t(tbl)`

	rows, err := p.Db.QueryContext(ctx, query)
	if err != nil {
		log.Error("Запрос выполнен с ошибкой", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	var topics []string
	for rows.Next() {
		var source, schema, table string
		if err := rows.Scan(&source, &schema, &table); err != nil {
			log.Error("Ошибка сканирования строки", slog.String("error", err.Error()))
			return nil, err
		}
		topic := fmt.Sprintf("dbserver_%s.%s.%s", source, schema, table)
		topics = append(topics, topic)
	}

	if err := rows.Err(); err != nil {
		log.Error("Ошибка после сканирования всех строк", slog.String("error", err.Error()))
		return nil, err
	}
	return topics, nil
}
