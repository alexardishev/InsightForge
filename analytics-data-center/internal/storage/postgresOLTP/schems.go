package postgresoltp

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"log/slog"
)

func (p *PostgresOLTP) GetSchemas(ctx context.Context, source string) ([]models.Schema, error) {
	const op = "PostgresOLTP.GetSchems"
	log := p.Log.With(
		slog.String("op", op),
	)
	var schems []models.Schema
	query := `SELECT schema_name
FROM information_schema.schemata
WHERE schema_name NOT LIKE 'pg_%'
  AND schema_name <> 'information_schema';
`

	rows, err := p.Db.QueryContext(ctx, query)
	if err != nil {
		log.Error("ошибка при запросе индексов", slog.String("ошибка", err.Error()))
		return []models.Schema{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var schem models.Schema
		if err := rows.Scan(&schem.Name); err != nil {
			log.Error("ошибка при сканировании строки", slog.String("ошибка", err.Error()))
			return []models.Schema{}, err
		}
		schems = append(schems, schem)
	}
	if err := rows.Err(); err != nil {
		log.Error("ошибка при обходе строк", slog.String("ошибка", err.Error()))
		return nil, err
	}

	return schems, nil

}
