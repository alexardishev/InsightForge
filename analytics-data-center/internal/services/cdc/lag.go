package cdc

import (
	"log/slog"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// DefaultLagReportInterval — как часто писать в лог отставание CDC.
const DefaultLagReportInterval = 30 * time.Second

// lagSource — данные librdkafka для расчёта отставания. *kafka.Consumer его реализует;
// все три метода работают локально, без запросов к брокеру.
type lagSource interface {
	Assignment() ([]kafka.TopicPartition, error)
	Position(partitions []kafka.TopicPartition) ([]kafka.TopicPartition, error)
	GetWatermarkOffsets(topic string, partition int32) (low, high int64, err error)
}

// partitionLag — отставание одной партиции.
type partitionLag struct {
	topic     string
	partition int32
	// messages — сколько сообщений ещё не прочитано; -1 — пока неизвестно
	// (из партиции ещё ничего не получено).
	messages int64
	// age — сколько прошло с момента события, полученного последним. Пока партиция
	// отстаёт, примерно настолько DWH отстаёт от источника. 0 — догнали или неизвестно.
	age time.Duration
}

// collectLag считает отставание по назначенным партициям: непрочитанные = high
// watermark − позиция. High watermark librdkafka обновляет в ответах fetch, поэтому
// у партиции на паузе (ждёт повтора) он не растёт и отставание может быть занижено.
func collectLag(src lagSource, lastEvent map[partitionKey]time.Time, now time.Time) ([]partitionLag, error) {
	assigned, err := src.Assignment()
	if err != nil || len(assigned) == 0 {
		return nil, err
	}
	positions, err := src.Position(assigned)
	if err != nil {
		return nil, err
	}
	lags := make([]partitionLag, 0, len(positions))
	for _, tp := range positions {
		if tp.Topic == nil {
			continue
		}
		lag := partitionLag{topic: *tp.Topic, partition: tp.Partition, messages: -1}
		_, high, err := src.GetWatermarkOffsets(*tp.Topic, tp.Partition)
		if err == nil && high >= 0 && tp.Offset >= 0 {
			lag.messages = max(high-int64(tp.Offset), 0)
		}
		// Возраст считаем, только если есть непрочитанные сообщения (или их число неизвестно).
		// Если прочитано всё, партиция не отстаёт, сколько бы времени ни прошло с последнего события.
		if lag.messages != 0 {
			if seen, ok := lastEvent[partitionKey{topic: *tp.Topic, partition: tp.Partition}]; ok {
				lag.age = now.Sub(seen)
			}
		}
		lags = append(lags, lag)
	}
	return lags, nil
}

// maybeReportLag пишет отставание в лог не чаще LagReportInterval.
func (l *Listener) maybeReportLag(now time.Time) {
	if l.LagReportInterval <= 0 || now.Before(l.nextLagReport) {
		return
	}
	l.nextLagReport = now.Add(l.LagReportInterval)
	src, ok := l.Consumer.(lagSource)
	if !ok {
		return
	}
	lags, err := collectLag(src, l.lastEvent, now)
	if err != nil {
		l.Log.WarnMsg(loggerpkg.MsgCDCLagFailed, slog.String("error", err.Error()))
		return
	}
	l.logLag(lags)
}

// logLag: одна сводная строка; предупреждение по каждой партиции, которая отстаёт
// дольше LagWarnAge — её события могут удалиться из Kafka непрочитанными.
func (l *Listener) logLag(lags []partitionLag) {
	var (
		total   int64
		unknown int
		maxAge  time.Duration
	)
	for _, lag := range lags {
		if lag.messages < 0 {
			unknown++
		} else {
			total += lag.messages
		}
		maxAge = max(maxAge, lag.age)
		if l.LagWarnAge > 0 && lag.messages > 0 && lag.age > l.LagWarnAge {
			l.Log.WarnMsg(loggerpkg.MsgCDCLagHigh,
				slog.String("topic", lag.topic),
				slog.Any("partition", lag.partition),
				slog.Int64("messages", lag.messages),
				slog.Duration("age", lag.age.Round(time.Second)),
				slog.Duration("warn_age", l.LagWarnAge),
			)
		}
	}
	l.Log.InfoMsg(loggerpkg.MsgCDCLagSummary,
		slog.Int("partitions", len(lags)),
		slog.Int64("messages", total),
		slog.Int("unknown", unknown),
		slog.Duration("max_age", maxAge.Round(time.Second)),
	)
}

// eventTime — время события по timestamp сообщения Kafka; если его нет — время получения.
func eventTime(msg *kafka.Message) time.Time {
	if msg.Timestamp.IsZero() {
		return time.Now()
	}
	return msg.Timestamp
}
