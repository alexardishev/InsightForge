package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
	"strings"
)

func CreateViewQueryPostgres(schema models.View, viewJoin models.ViewJoinTable, logger *slog.Logger) (models.Query, error) {
	const op = "sqlgenerator.CreateViewQuery"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")

	if len(viewJoin.TempTables) == 0 {
		return models.Query{}, fmt.Errorf("нет временных таблиц для формирования вью")
	}

	// SELECT част
	var selectParts []string
	for _, tempTable := range viewJoin.TempTables {
		alias := CleanAndTrim(tempTable.TempTableName, 4)
		for _, col := range tempTable.TempColumns {
			selectParts = append(selectParts, fmt.Sprintf("%s.%s", alias, col.ColumnName))
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("CREATE TABLE %s AS SELECT %s", schema.Name, strings.Join(selectParts, ", ")))

	// Определяем главную таблицу
	mainTableName := ""
	mainAlias := ""
	mainTableSet := false

	for _, join := range schema.Joins {
		if join.Inner != nil && join.Inner.MainTable != "" {
			mainTableName = fmt.Sprintf("temp_%s_%s_%s", join.Inner.Source, join.Inner.Schema, join.Inner.MainTable)
			mainAlias = CleanAndTrim(mainTableName, 4)
			mainTableSet = true
			break
		}
	}
	if !mainTableSet {
		mainTableName = viewJoin.TempTables[0].TempTableName
		mainAlias = CleanAndTrim(mainTableName, 4)
	}

	fromClause := fmt.Sprintf(" FROM %s %s", mainTableName, mainAlias)

	// JOIN части
	var joinClauses []string
	for _, join := range schema.Joins {
		if join.Inner != nil {
			joinTable := fmt.Sprintf("temp_%s_%s_%s", join.Inner.Source, join.Inner.Schema, join.Inner.Table)
			joinAlias := CleanAndTrim(joinTable, 4)
			joinClauses = append(joinClauses, fmt.Sprintf("JOIN %s %s ON %s.%s = %s.%s",
				joinTable, joinAlias,
				mainAlias, join.Inner.ColumnFirst,
				joinAlias, join.Inner.ColumnSecond,
			))
		}
	}

	finalQuery := fmt.Sprintf("%s%s %s", b.String(), fromClause, strings.Join(joinClauses, " "))

	return models.Query{
		Query:     finalQuery,
		TableName: schema.Name,
	}, nil
}
