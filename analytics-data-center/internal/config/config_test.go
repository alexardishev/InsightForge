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
