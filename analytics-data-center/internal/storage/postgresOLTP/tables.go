package postgresoltp

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"fmt"
	"log/slog"
)

func (p *PostgresOLTP) GetTables(ctx context.Context, schema string) ([]models.Table, error) {
	const op = "PostgresOLTP.GetTables"
	log := p.Log.With(
		slog.String("op", op),
	)
	var tables []models.Table
	query := fmt.Sprintf("SELECT table_name FROM information_schema.tables WHERE table_schema = '%s'", schema)
	rows, err := p.Db.QueryContext(ctx, query)
	if err != nil {
		log.Error("ошибка при запросе индексов", slog.String("ошибка", err.Error()))
		return []models.Table{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var table models.Table
		if err := rows.Scan(&table.Name); err != nil {
			log.Error("ошибка при сканировании строки", slog.String("ошибка", err.Error()))
			return []models.Table{}, err
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		log.Error("ошибка при обходе строк", slog.String("ошибка", err.Error()))
		return nil, err
	}

	return tables, nil
}
