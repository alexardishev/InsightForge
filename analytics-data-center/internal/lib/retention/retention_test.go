package retention

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMs(t *testing.T) {
	for _, tc := range []struct {
		retention time.Duration
		value     string
		set       bool
	}{
		{0, "", false},                        // значение брокера по умолчанию
		{336 * time.Hour, "1209600000", true}, // 14 дней
		{720 * time.Hour, "2592000000", true}, // 30 дней
		{-time.Second, "-1", true},            // бессрочно
		{500 * time.Microsecond, "1", true},   // меньше миллисекунды не округляем до 0
	} {
		value, set := Ms(tc.retention)
		require.Equal(t, tc.set, set, tc.retention)
		require.Equal(t, tc.value, value, tc.retention)
	}
}
