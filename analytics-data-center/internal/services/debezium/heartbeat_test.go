package debezium

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConnectorConfigHeartbeatInterval(t *testing.T) {
	cfg, err := postgresConnectorConfig("db1", testConn, ConnectorOptions{HeartbeatInterval: 10 * time.Second})
	require.NoError(t, err)

	require.Equal(t, "10000", cfg["heartbeat.interval.ms"])
	require.NotContains(t, cfg, "heartbeat.action.query", "action query only for sources with a heartbeat table")
}

func TestConnectorConfigWithoutHeartbeat(t *testing.T) {
	cfg, err := postgresConnectorConfig("db1", testConn, ConnectorOptions{})
	require.NoError(t, err)

	require.NotContains(t, cfg, "heartbeat.interval.ms")
	require.NotContains(t, cfg, "heartbeat.action.query")
}

func TestConnectorConfigHeartbeatActionQuery(t *testing.T) {
	cfg, err := postgresConnectorConfig("db1", testConn, ConnectorOptions{
		HeartbeatInterval: 10 * time.Second,
		HeartbeatTable:    "public.insightforge_heartbeat",
	})
	require.NoError(t, err)

	require.Equal(t,
		`INSERT INTO "public"."insightforge_heartbeat" (id, ts) VALUES (1, now()) ON CONFLICT (id) DO UPDATE SET ts = EXCLUDED.ts`,
		cfg["heartbeat.action.query"])
}

func TestConnectorConfigRejectsHeartbeatTableWithoutInterval(t *testing.T) {
	_, err := postgresConnectorConfig("db1", testConn, ConnectorOptions{HeartbeatTable: "insightforge_heartbeat"})

	require.Error(t, err, "Debezium runs the action query only on heartbeats")
}

func TestQuoteHeartbeatTable(t *testing.T) {
	for name, want := range map[string]string{
		"insightforge_heartbeat":        `"insightforge_heartbeat"`,
		"public.insightforge_heartbeat": `"public"."insightforge_heartbeat"`,
		"_hb2":                          `"_hb2"`,
	} {
		got, err := quoteHeartbeatTable(name)
		require.NoError(t, err, name)
		require.Equal(t, want, got)
	}

	for _, name := range []string{
		"", "a.b.c", "1hb", "hb-1", `a"b`, "a'b", "hb; DROP TABLE users", "hb --", "public.", ".hb",
	} {
		_, err := quoteHeartbeatTable(name)
		require.Error(t, err, name)
	}
}

func TestHeartbeatTableDDL(t *testing.T) {
	require.Equal(t,
		`CREATE TABLE IF NOT EXISTS "public"."hb" (id integer PRIMARY KEY, ts timestamptz NOT NULL)`,
		heartbeatTableDDL(`"public"."hb"`))
}

func TestEnsureHeartbeatTableRejectsInvalidNameBeforeConnecting(t *testing.T) {
	// Строка подключения ведёт в никуда: ошибка должна прийти от проверки имени, без сети.
	err := EnsureHeartbeatTable(context.Background(), "postgresql://u:p@127.0.0.1:1/db?connect_timeout=1", "hb; DROP TABLE users")

	require.ErrorContains(t, err, "heartbeat")
}
