package debezium

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/lib/pq"
)

// heartbeatTablePattern — имя таблицы heartbeat: "table" или "schema.table".
// Имя подставляется в SQL, поэтому допускаются только буквы, цифры и подчёркивание.
var heartbeatTablePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)

// quoteHeartbeatTable проверяет имя таблицы heartbeat из конфига и экранирует его для SQL.
func quoteHeartbeatTable(name string) (string, error) {
	if !heartbeatTablePattern.MatchString(name) {
		return "", fmt.Errorf("недопустимое имя heartbeat-таблицы %q: ожидается table или schema.table из букв, цифр и _", name)
	}
	parts := strings.Split(name, ".")
	for i, part := range parts {
		parts[i] = pq.QuoteIdentifier(part)
	}
	return strings.Join(parts, "."), nil
}

// heartbeatTableDDL — таблица из одной строки: Debezium раз в heartbeat.interval.ms
// перезаписывает её, и это изменение двигает позицию слота даже в «тихой» базе.
func heartbeatTableDDL(quotedTable string) string {
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id integer PRIMARY KEY, ts timestamptz NOT NULL)", quotedTable)
}

func heartbeatActionQuery(quotedTable string) string {
	return fmt.Sprintf("INSERT INTO %s (id, ts) VALUES (1, now()) ON CONFLICT (id) DO UPDATE SET ts = EXCLUDED.ts", quotedTable)
}

// EnsureHeartbeatTable создаёт таблицу heartbeat в источнике. Подключение — строкой
// подключения Debezium (connection_string_kafka): таблицу пишет Debezium, и владельцем
// должен быть его пользователь, иначе у него может не оказаться прав на запись.
func EnsureHeartbeatTable(ctx context.Context, pgConn, table string) error {
	quoted, err := quoteHeartbeatTable(table)
	if err != nil {
		return err
	}
	db, err := sql.Open("postgres", pgConn)
	if err != nil {
		return fmt.Errorf("heartbeat-таблица %s: %w", table, err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, heartbeatTableDDL(quoted)); err != nil {
		return fmt.Errorf("heartbeat-таблица %s: %w", table, err)
	}
	return nil
}
