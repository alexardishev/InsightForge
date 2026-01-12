package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
	"strings"
)

func CreateViewQueryClickhouse(schema models.View, viewJoin models.ViewJoinTable, logger *slog.Logger) (models.Query, error) {
	const op = "sqlgenerator.CreateViewQueryClickhouse"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")

	if len(viewJoin.TempTables) == 0 {
		return models.Query{}, fmt.Errorf("нет временных таблиц для формирования вью")
	}

	aliasMap := make(map[string]string)
	for idx, tempTable := range viewJoin.TempTables {
		aliasMap[tempTable.TempTableName] = fmt.Sprintf("t%d", idx+1)
	}

	resolveTempTable := func(endpoint models.JoinEndpoint) (models.TempTable, string, error) {
		for _, tempTable := range viewJoin.TempTables {
			if tempTable.Source == endpoint.Source && tempTable.Schema == endpoint.Schema && tempTable.Table == endpoint.Table {
				alias, ok := aliasMap[tempTable.TempTableName]
				if !ok {
					return models.TempTable{}, "", fmt.Errorf("alias not found for temp table %s", tempTable.TempTableName)
				}
				return tempTable, alias, nil
			}
		}
		return models.TempTable{}, "", fmt.Errorf("temp table not found for join endpoint %s.%s.%s", endpoint.Source, endpoint.Schema, endpoint.Table)
	}

	var joins []struct {
		leftTable  models.TempTable
		rightTable models.TempTable
		leftAlias  string
		rightAlias string
		leftCol    string
		rightCol   string
	}

	rightKeys := make(map[string]struct{})

	for _, join := range schema.Joins {
		if join.Inner == nil {
			continue
		}
		left, leftAlias, err := resolveTempTable(join.Inner.Left)
		if err != nil {
			return models.Query{}, err
		}
		right, rightAlias, err := resolveTempTable(join.Inner.Right)
		if err != nil {
			return models.Query{}, err
		}

		joins = append(joins, struct {
			leftTable  models.TempTable
			rightTable models.TempTable
			leftAlias  string
			rightAlias string
			leftCol    string
			rightCol   string
		}{
			leftTable:  left,
			rightTable: right,
			leftAlias:  leftAlias,
			rightAlias: rightAlias,
			leftCol:    join.Inner.Left.Column,
			rightCol:   join.Inner.Right.Column,
		})
		rightKeys[fmt.Sprintf("%s|%s|%s", right.Source, right.Schema, right.Table)] = struct{}{}
	}

	var rootTable models.TempTable
	var rootAlias string

	if len(joins) == 0 {
		rootTable = viewJoin.TempTables[0]
		rootAlias = aliasMap[rootTable.TempTableName]
	} else {
		for _, j := range joins {
			key := fmt.Sprintf("%s|%s|%s", j.leftTable.Source, j.leftTable.Schema, j.leftTable.Table)
			if _, ok := rightKeys[key]; !ok {
				rootTable = j.leftTable
				rootAlias = j.leftAlias
				break
			}
		}

		if rootAlias == "" {
			rootTable = joins[0].leftTable
			rootAlias = joins[0].leftAlias
		}
	}

	known := map[string]struct{}{rootTable.TempTableName: {}}

	var joinClauses []string
	processed := make(map[int]bool)

	for len(processed) < len(joins) {
		progress := false
		for idx, j := range joins {
			if processed[idx] {
				continue
			}
			leftKnown := false
			rightKnown := false
			if _, ok := known[j.leftTable.TempTableName]; ok {
				leftKnown = true
			}
			if _, ok := known[j.rightTable.TempTableName]; ok {
				rightKnown = true
			}

			if leftKnown && !rightKnown {
				joinClauses = append(joinClauses, fmt.Sprintf("JOIN %s %s ON %s.%s = %s.%s",
					j.rightTable.TempTableName, j.rightAlias,
					j.leftAlias, j.leftCol,
					j.rightAlias, j.rightCol,
				))
				known[j.rightTable.TempTableName] = struct{}{}
				processed[idx] = true
				progress = true
				continue
			}

			if rightKnown && !leftKnown {
				joinClauses = append(joinClauses, fmt.Sprintf("JOIN %s %s ON %s.%s = %s.%s",
					j.leftTable.TempTableName, j.leftAlias,
					j.rightAlias, j.rightCol,
					j.leftAlias, j.leftCol,
				))
				known[j.leftTable.TempTableName] = struct{}{}
				processed[idx] = true
				progress = true
			}
		}

		if !progress {
			return models.Query{}, fmt.Errorf("невозможно связать все джоины: отсутствуют исходные таблицы")
		}
	}

	var selectParts []string
	for _, tempTable := range viewJoin.TempTables {
		if _, ok := known[tempTable.TempTableName]; !ok {
			continue
		}
		alias := aliasMap[tempTable.TempTableName]
		for _, col := range tempTable.TempColumns {
			selectParts = append(selectParts, fmt.Sprintf("%s.%s", alias, col.ColumnName))
		}
	}
	selectParts = append(selectParts, "now() AS updated_at")

	orderBy := "tuple()"
	if len(viewJoin.TempTables) > 0 && len(viewJoin.TempTables[0].TempColumns) > 0 {
		orderBy = viewJoin.TempTables[0].TempColumns[0].ColumnName
	}

	engineClause := "ENGINE = ReplacingMergeTree(updated_at)"

	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		"CREATE TABLE %s %s ORDER BY %s AS SELECT %s",
		schema.Name, engineClause, orderBy, strings.Join(selectParts, ", "),
	))

	fromClause := fmt.Sprintf(" FROM %s %s", rootTable.TempTableName, rootAlias)
	finalQuery := fmt.Sprintf("%s%s %s", b.String(), fromClause, strings.Join(joinClauses, " "))

	return models.Query{
		Query:     finalQuery,
		TableName: schema.Name,
	}, nil
}
