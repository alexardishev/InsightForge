package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lib/pq"
)

func CreateViewQueryPostgres(schema models.View, viewJoin models.ViewJoinTable, logger *slog.Logger) (models.Query, error) {
	const op = "sqlgenerator.CreateViewQuery"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")

	if len(viewJoin.TempTables) == 0 {
		return models.Query{}, fmt.Errorf("нет временных таблиц для формирования вью")
	}

	aliasMap := make(map[string]string)
	assignAlias := func(name string, idx int) string {
		if existing, ok := aliasMap[name]; ok {
			return existing
		}
		alias := fmt.Sprintf("t%d", idx+1)
		aliasMap[name] = alias
		return alias
	}

	var selectParts []string
	for idx, tempTable := range viewJoin.TempTables {
		alias := assignAlias(tempTable.TempTableName, idx)
		for _, col := range tempTable.TempColumns {
			selectParts = append(selectParts, fmt.Sprintf("%s.%s", pq.QuoteIdentifier(alias), pq.QuoteIdentifier(col.ColumnName)))
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("CREATE TABLE %s AS SELECT %s", pq.QuoteIdentifier(schema.Name), strings.Join(selectParts, ", ")))

	mainTableName := ""
	mainAlias := ""
	mainTableSet := false

	for _, join := range schema.Joins {
		if join.Inner != nil && join.Inner.MainTable != "" {
			mainTableName = fmt.Sprintf("temp_%s_%s_%s", join.Inner.Source, join.Inner.Schema, join.Inner.MainTable)
			mainAlias = assignAlias(mainTableName, len(aliasMap))
			mainTableSet = true
			break
		}
	}
	if !mainTableSet {
		mainTableName = viewJoin.TempTables[0].TempTableName
		mainAlias = assignAlias(mainTableName, len(aliasMap))
	}

	fromClause := fmt.Sprintf(" FROM %s %s", pq.QuoteIdentifier(mainTableName), pq.QuoteIdentifier(mainAlias))

	var joinClauses []string
	for _, join := range schema.Joins {
		if join.Inner != nil {
			joinTable := fmt.Sprintf("temp_%s_%s_%s", join.Inner.Source, join.Inner.Schema, join.Inner.Table)
			joinAlias := assignAlias(joinTable, len(aliasMap))
			joinClauses = append(joinClauses, fmt.Sprintf("JOIN %s %s ON %s.%s = %s.%s",
				pq.QuoteIdentifier(joinTable), pq.QuoteIdentifier(joinAlias),
				pq.QuoteIdentifier(mainAlias), pq.QuoteIdentifier(join.Inner.ColumnFirst),
				pq.QuoteIdentifier(joinAlias), pq.QuoteIdentifier(join.Inner.ColumnSecond),
			))
		}
	}

	joinSQL := strings.Join(joinClauses, " ")
	if joinSQL != "" {
		joinSQL = " " + joinSQL
	}

	finalQuery := fmt.Sprintf("%s%s%s", b.String(), fromClause, joinSQL)

	return models.Query{
		Query:     finalQuery,
		TableName: schema.Name,
	}, nil
}
