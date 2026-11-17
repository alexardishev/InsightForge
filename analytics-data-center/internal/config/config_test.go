package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/numbergroup/cleanenv"
	"github.com/stretchr/testify/require"
)

func TestKafkaDLQRetention(t *testing.T) {
	for _, tc := range []struct {
		name string
		yaml string
		want time.Duration
	}{
		{"set", "dlq_retention: 720h", 720 * time.Hour},
		{"forever", "dlq_retention: -1s", -time.Second},
		{"missing means broker default", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			content := "storage_path: test\nkafka:\n    group.id: \"g\"\n    " + tc.yaml + "\n"
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

			var cfg Config
			require.NoError(t, cleanenv.ReadConfig(path, &cfg))
			require.Equal(t, tc.want, cfg.Kafka.DLQRetention)
		})
	}
}
