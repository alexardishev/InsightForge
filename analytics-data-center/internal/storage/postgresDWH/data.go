package postgresdwh

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lib/pq"
)

func (p *PostgresDWH) InsertDataToDWH(ctx context.Context, query string) error {
	const op = "Storage.PostgreSQL.InsertDataToDWH"
	log := p.Log.With(
		slog.String("op", op),
	)
	_, err := p.Db.ExecContext(ctx, query)
	if err != nil {
		log.Error("ошибка подготовки запроса", slog.String("error", err.Error()))
		return err
	}

	return nil
}
func (p *PostgresDWH) MergeTempTables(ctx context.Context, query string) error {
	const op = "Storage.PostgreSQL.MergeTempTables"
	log := p.Log.With(
		slog.String("op", op),
	)
	_, err := p.Db.ExecContext(ctx, query)
	if err != nil {
		log.Error("ошибка подготовки запроса", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (p *PostgresDWH) ReplicaIdentityFull(ctx context.Context, tableDWHName string) error {
	const op = "Storage.PostgreSQL.ReplicaIdentityFull"
	log := p.Log.With(
		slog.String("op", op),
	)
	query := fmt.Sprintf(`ALTER TABLE %s REPLICA IDENTITY FULL`, pq.QuoteIdentifier(tableDWHName))
	_, err := p.Db.ExecContext(ctx, query)
	if err != nil {
		log.Error("ошибка подготовки запроса", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (p *PostgresDWH) InsertOrUpdateTransactional(
	ctx context.Context,
	schemaName string,
	row map[string]interface{},
	conflictColumns []string,
) error {
	const op = "Storage.PostgreSQL.InsertOrUpdateTransactional"
	log := p.Log.With(
		slog.String("op", op),
	)
	log.Info("Определение обонвления или вставки")
	tx, err := p.Db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}

	defer func() {
		// если ошибка — rollback
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Проверка на существование строки
	where := make([]string, len(conflictColumns))
	values := make([]interface{}, len(conflictColumns))
	for i, col := range conflictColumns {
		where[i] = fmt.Sprintf("%s = $%d", col, i+1)
		values[i] = normalizeSQLValue(row[col])
	}

	selectQuery := fmt.Sprintf(`SELECT 1 FROM %s WHERE %s LIMIT 1`, schemaName, strings.Join(where, " AND "))

	var dummy int
	err = tx.QueryRowContext(ctx, selectQuery, values...).Scan(&dummy)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("ошибка при проверке существования: %w", err)
	}

	// определяем: insert или update
	if err == sql.ErrNoRows {
		// INSERT
		err = insertWithTx(ctx, tx, schemaName, row)
		if err != nil {
			return fmt.Errorf("ошибка при вставке: %w", err)
		}
	} else {
		// UPDATE
		err = updateWithTx(ctx, tx, schemaName, row, conflictColumns)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении: %w", err)
		}
	}

	// COMMIT
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка при коммите транзакции: %w", err)
	}

	return nil
}
func insertWithTx(ctx context.Context, tx *sql.Tx, table string, row map[string]interface{}) error {
	columns := make([]string, 0, len(row))
	placeholders := make([]string, 0, len(row))
	values := make([]interface{}, 0, len(row))

	i := 1
	for col, val := range row {
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, normalizeSQLValue(val))
		i++
	}

	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := tx.ExecContext(ctx, query, values...)
	return err
}

func updateWithTx(ctx context.Context, tx *sql.Tx, table string, row map[string]interface{}, conflictColumns []string) error {
	setParts := []string{}
	whereParts := []string{}
	values := []interface{}{}

	i := 1
	for col, val := range row {
		isKey := false
		for _, k := range conflictColumns {
			if k == col {
				isKey = true
				break
			}
		}
		if isKey {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", col, i))
		values = append(values, normalizeSQLValue(val))
		i++
	}

	for _, col := range conflictColumns {
		whereParts = append(whereParts, fmt.Sprintf("%s = $%d", col, i))
		values = append(values, normalizeSQLValue(row[col]))
		i++
	}

	query := fmt.Sprintf(`UPDATE %s SET %s WHERE %s`,
		table,
		strings.Join(setParts, ", "),
		strings.Join(whereParts, " AND "),
	)

	_, err := tx.ExecContext(ctx, query, values...)
	return err
}

func normalizeSQLValue(val interface{}) interface{} {
	switch v := val.(type) {
	case []interface{}:
		if ints := convertInterfacesToInt64(v); ints != nil {
			return pq.Array(ints)
		}
		strVals := make([]string, 0, len(v))
		for _, item := range v {
			strVals = append(strVals, fmt.Sprintf("%v", item))
		}
		return pq.Array(strVals)
	case []string:
		return pq.Array(v)
	default:
		return val
	}
}

func convertInterfacesToInt64(arr []interface{}) []int64 {
	ints := make([]int64, 0, len(arr))
	for _, item := range arr {
		switch t := item.(type) {
		case int:
			ints = append(ints, int64(t))
		case int64:
			ints = append(ints, t)
		case float64:
			// безопасно конвертируем только целые значения
			if t == float64(int64(t)) {
				ints = append(ints, int64(t))
			} else {
				return nil
			}
		default:
			return nil
		}
	}
	return ints
}
