package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/numbergroup/cleanenv"
	"github.com/stretchr/testify/require"
)

func TestKafkaRetentions(t *testing.T) {
	for _, tc := range []struct {
		name string
		yaml string
		want time.Duration
		got  func(Config) time.Duration
	}{
		{"dlq set", "dlq_retention: 720h", 720 * time.Hour, dlqRetention},
		{"dlq forever", "dlq_retention: -1s", -time.Second, dlqRetention},
		{"dlq missing means broker default", "", 0, dlqRetention},
		{"debezium set", "debezium_topic_retention: 336h", 336 * time.Hour, debeziumRetention},
		{"debezium missing means broker default", "", 0, debeziumRetention},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			content := "storage_path: test\nkafka:\n    group.id: \"g\"\n    " + tc.yaml + "\n"
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

			var cfg Config
			require.NoError(t, cleanenv.ReadConfig(path, &cfg))
			require.Equal(t, tc.want, tc.got(cfg))
		})
	}
}

func dlqRetention(cfg Config) time.Duration      { return cfg.Kafka.DLQRetention }
func debeziumRetention(cfg Config) time.Duration { return cfg.Kafka.DebeziumTopicRetention }

func readTestConfig(t *testing.T, content string) Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("storage_path: test\n"+content), 0o600))
	var cfg Config
	require.NoError(t, cleanenv.ReadConfig(path, &cfg))
	return cfg
}

func TestWALWatchDefaults(t *testing.T) {
	cfg := readTestConfig(t, "")

	require.Equal(t, time.Minute, cfg.WALWatch.Interval)
	require.Equal(t, int64(5120), cfg.WALWatch.RetainedWarnMB)
}

func TestWALWatchFromYAML(t *testing.T) {
	cfg := readTestConfig(t, "wal_watch:\n    disabled: true\n    interval: 30s\n    retained_warn_mb: 1024\n")

	require.True(t, cfg.WALWatch.Disabled)
	require.Equal(t, 30*time.Second, cfg.WALWatch.Interval)
	require.Equal(t, int64(1024), cfg.WALWatch.RetainedWarnMB)
}

func TestWALWatchZeroIntervalFallsBackToDefault(t *testing.T) {
	// cleanenv подставляет значение по умолчанию вместо нулевого — поэтому для
	// выключения нужен disabled, а не interval: 0s.
	cfg := readTestConfig(t, "wal_watch:\n    interval: 0s\n")

	require.Equal(t, time.Minute, cfg.WALWatch.Interval)
	require.False(t, cfg.WALWatch.Disabled)
}

func TestHeartbeatTablePerSource(t *testing.T) {
	cfg := readTestConfig(t, `oltp_connections:
  - name: quiet
    connection_string: "postgresql://u:p@h:5432/a"
    heartbeat_table: public.insightforge_heartbeat
  - name: busy
    connection_string: "postgresql://u:p@h:5432/b"
`)

	require.Equal(t, "public.insightforge_heartbeat", cfg.OLTPstorages[0].HeartbeatTable)
	require.Empty(t, cfg.OLTPstorages[1].HeartbeatTable)
}
