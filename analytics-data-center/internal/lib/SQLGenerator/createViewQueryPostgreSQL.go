package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"
)

func CreateViewQuery(schema models.View, viewJoin models.ViewJoinTable, logger *slog.Logger) (models.Query, error) {
	const op = "sqlgenerator.CreateViewQuery"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")

	if len(viewJoin.TempTables) == 0 {
		return models.Query{}, fmt.Errorf("нет временных таблиц для формирования вью")
	}

	var selectParts []string
	for _, tempTable := range viewJoin.TempTables {
		alias := CleanAndTrim(tempTable.TempTableName, 4)
		for _, col := range tempTable.TempColumns {
			selectParts = append(selectParts, fmt.Sprintf("%s.%s", alias, col.ColumnName))
		}
	}

	// SELECT часть
	var b strings.Builder
	b.WriteString(fmt.Sprintf("CREATE TABLE %s AS SELECT %s", schema.Name, strings.Join(selectParts, ", ")))

	// FROM часть
	firstTable := viewJoin.TempTables[0]
	firstAlias := CleanAndTrim(firstTable.TempTableName, 4)
	fromClause := fmt.Sprintf(" FROM %s %s", firstTable.TempTableName, firstAlias)

	// JOIN части
	var joinClauses []string
	for _, join := range schema.Joins {
		if join.Inner != nil {
			joinAlias := CleanAndTrim(fmt.Sprintf("temp_%s_%s_%s", join.Inner.Source, join.Inner.Schema, join.Inner.Table), 4)
			joinTable := fmt.Sprintf("temp_%s_%s_%s", join.Inner.Source, join.Inner.Schema, join.Inner.Table)
			joinClauses = append(joinClauses, fmt.Sprintf("JOIN %s %s ON %s.%s = %s.%s",
				joinTable, joinAlias,
				firstAlias, join.Inner.ColumnFirst,
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

func CleanAndTrim(input string, maxLen int) string {
	cleaned := strings.TrimSpace(input)
	cleaned = strings.Join(strings.Fields(cleaned), "_")
	if utf8.RuneCountInString(cleaned) > maxLen {
		runes := []rune(cleaned)
		return string(runes[21:])
	}
	return cleaned
}
