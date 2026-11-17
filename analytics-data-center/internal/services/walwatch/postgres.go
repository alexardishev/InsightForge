package walwatch

import (
	"context"
	"database/sql"
	"strings"

	"analyticDataCenter/analytics-data-center/internal/services/debezium"
)

// slotsQuery — логические слоты сервера с именами по шаблону $1. Удерживаемый WAL —
// от restart_lsn слота до текущей позиции журнала; у потерянного слота restart_lsn
// пустой, тогда -1. wal_status есть с Postgres 13.
const slotsQuery = `SELECT slot_name, active, COALESCE(wal_status, ''),
       COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)::bigint, -1)
FROM pg_replication_slots
WHERE slot_type = 'logical' AND slot_name LIKE $1
ORDER BY slot_name`

// postgresSource — SourceDB поверх пула соединений источника.
type postgresSource struct {
	db *sql.DB
}

func NewPostgresSource(db *sql.DB) SourceDB {
	return postgresSource{db: db}
}

// slotNamePattern — шаблон LIKE для слотов Debezium: префикс из debezium.SlotName,
// подчёркивание экранировано (в LIKE «_» — любой символ).
func slotNamePattern() string {
	return strings.ReplaceAll(debezium.SlotName(""), "_", `\_`) + "%"
}

func (p postgresSource) Slots(ctx context.Context) ([]Slot, error) {
	rows, err := p.db.QueryContext(ctx, slotsQuery, slotNamePattern())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var slots []Slot
	for rows.Next() {
		var s Slot
		if err := rows.Scan(&s.Name, &s.Active, &s.WALStatus, &s.RetainedBytes); err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}
	return slots, rows.Err()
}

func (p postgresSource) MaxSlotWALKeepSize(ctx context.Context) (string, error) {
	var value string
	err := p.db.QueryRowContext(ctx, "SHOW max_slot_wal_keep_size").Scan(&value)
	return value, err
}
