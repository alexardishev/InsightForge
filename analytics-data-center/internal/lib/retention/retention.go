// Package retention переводит срок хранения из конфига в настройку топика Kafka retention.ms.
package retention

import (
	"strconv"
	"time"
)

// ConfigKey — настройка топика Kafka со сроком хранения сообщений.
const ConfigKey = "retention.ms"

// Ms переводит срок хранения в значение retention.ms.
// set=false — не задавать, оставить значение брокера по умолчанию (срок 0).
// Отрицательный срок — хранить бессрочно ("-1").
func Ms(d time.Duration) (value string, set bool) {
	switch {
	case d == 0:
		return "", false
	case d < 0:
		return "-1", true
	case d < time.Millisecond:
		return "1", true // 0 у Kafka значит «удалять сразу», а не «по умолчанию»
	default:
		return strconv.FormatInt(d.Milliseconds(), 10), true
	}
}
