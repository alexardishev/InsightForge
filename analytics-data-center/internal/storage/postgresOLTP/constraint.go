package postgresoltp

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"log/slog"
)

func (p *PostgresOLTP) GetConstraint(ctx context.Context, tableName string, schemaName string) (models.Constraints, error) {
	const op = "Storage.PostgresOLTP.GetConstraint"
	log := p.Log.With(
		slog.String("op", op),
		slog.String("tableName", tableName),
	)

	var constraints models.Constraints

	query := `SELECT
    conname AS constraint_name,
    contype AS constraint_type,
    pg_get_constraintdef(c.oid) AS definition
FROM
    pg_constraint c
JOIN
    pg_namespace n ON n.oid = c.connamespace
JOIN
    pg_class t ON t.oid = c.conrelid
WHERE
    n.nspname = $1 AND
    t.relname = $2;`

	rows, err := p.Db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		log.Error("ошибка при запросе индексов", slog.String("ошибка", err.Error()))
		return models.Constraints{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var constraint models.Constarint
		err = rows.Scan(&constraint.ConstarintName, &constraint.ConstarintType, &constraint.Defenition)
		if err != nil {
			log.Error("ошибка сканирования", slog.String("ошибка", err.Error()))
			return models.Constraints{}, err
		}
		constraints.Constraints = append(constraints.Constraints, constraint)
	}

	if err := rows.Err(); err != nil {
		log.Error("ошибка после сканирования всех строк", slog.String("ошибка", err.Error()))
		return models.Constraints{}, err
	}
	return constraints, nil

}
