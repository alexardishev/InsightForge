package cdc

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/require"
)

// Методы lagSource для fakeConsumer — с семантикой librdkafka: Position — смещение
// последнего полученного приложением сообщения + 1 (OffsetInvalid, пока ничего не
// получено), high watermark — смещение последнего сообщения в логе + 1.
func (c *fakeConsumer) Assignment() ([]kafka.TopicPartition, error) {
	var tps []kafka.TopicPartition
	for _, p := range c.parts {
		topic := p.key.topic
		tps = append(tps, kafka.TopicPartition{Topic: &topic, Partition: p.key.partition})
	}
	return tps, nil
}

func (c *fakeConsumer) Position(tps []kafka.TopicPartition) ([]kafka.TopicPartition, error) {
	out := make([]kafka.TopicPartition, 0, len(tps))
	for _, tp := range tps {
		p := c.partition(tp)
		tp.Offset = kafka.OffsetInvalid
		if p.pos > 0 {
			tp.Offset = p.log[p.pos-1].TopicPartition.Offset + 1
		}
		out = append(out, tp)
	}
	return out, nil
}

func (c *fakeConsumer) GetWatermarkOffsets(topic string, partition int32) (int64, int64, error) {
	p := c.partition(kafka.TopicPartition{Topic: &topic, Partition: partition})
	if len(p.log) == 0 {
		return int64(kafka.OffsetInvalid), int64(kafka.OffsetInvalid), nil
	}
	return 0, int64(p.log[len(p.log)-1].TopicPartition.Offset) + 1, nil
}

func capturingLog() (*loggerpkg.Logger, *strings.Builder) {
	out := &strings.Builder{}
	return &loggerpkg.Logger{Logger: slog.New(slog.NewTextHandler(out, nil))}, out
}

func TestCollectLag(t *testing.T) {
	now := time.Now()
	c := newFakeConsumer(func() {}).
		addPartition("cdc", 2, 41, 42, 43).
		addPartition("cdc", 3, 10)
	c.parts[0].pos = 1 // получено 2@41, впереди 42 и 43
	lastEvent := map[partitionKey]time.Time{{"cdc", 2}: now.Add(-2 * time.Hour)}

	lags, err := collectLag(c, lastEvent, now)

	require.NoError(t, err)
	require.Equal(t, []partitionLag{
		{topic: "cdc", partition: 2, messages: 2, age: 2 * time.Hour},
		{topic: "cdc", partition: 3, messages: -1}, // из партиции ещё ничего не получено — неизвестно
	}, lags)
}

func TestCollectLagCaughtUpHasNoAge(t *testing.T) {
	now := time.Now()
	c := newFakeConsumer(func() {}).addPartition("cdc", 2, 41)
	c.parts[0].pos = 1
	lastEvent := map[partitionKey]time.Time{{"cdc", 2}: now.Add(-48 * time.Hour)} // таблица давно не менялась

	lags, err := collectLag(c, lastEvent, now)

	require.NoError(t, err)
	require.Equal(t, []partitionLag{{topic: "cdc", partition: 2, messages: 0}}, lags,
		"a quiet table that is fully read is not behind")
}

func TestLogLagWarnsOnlyForOldBacklog(t *testing.T) {
	log, out := capturingLog()
	l := NewListener(nil, nil, log, nil)
	l.LagWarnAge = time.Hour

	l.logLag([]partitionLag{
		{topic: "cdc.behind", partition: 2, messages: 2, age: 2 * time.Hour}, // отстаёт давно — предупреждение
		{topic: "cdc.fresh", partition: 3, messages: 5, age: 10 * time.Minute},
		// Старое событие само по себе не повод: партиция без отставания или с неизвестным
		// отставанием не предупреждает, даже если возраст большой.
		{topic: "cdc.idle", partition: 4, messages: 0, age: 3 * time.Hour},
		{topic: "cdc.unknown", partition: 5, messages: -1, age: 3 * time.Hour},
	})

	logs := out.String()
	require.Equal(t, 1, strings.Count(logs, loggerpkg.MsgCDCLagHigh.RU))
	require.Contains(t, logs, "topic=cdc.behind")
	require.Contains(t, logs, loggerpkg.MsgCDCLagSummary.RU)
	require.Contains(t, logs, "messages=7")
	require.Contains(t, logs, "unknown=1")
}

func TestLogLagWithoutRetentionDoesNotWarn(t *testing.T) {
	log, out := capturingLog()
	l := NewListener(nil, nil, log, nil) // LagWarnAge = 0: срок хранения брокера по умолчанию или бессрочный

	l.logLag([]partitionLag{{topic: "cdc", partition: 2, messages: 100, age: 1000 * time.Hour}})

	require.NotContains(t, out.String(), loggerpkg.MsgCDCLagHigh.RU)
}

func TestListenerReportsLagPeriodically(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log, out := capturingLog()
	c := newFakeConsumer(cancel).addPartition("cdc", 2, 41, 42)
	stamp := time.Now().Add(-time.Minute)
	for _, m := range c.parts[0].log {
		m.Timestamp = stamp
	}
	c.idle = func() {
		if strings.Contains(out.String(), loggerpkg.MsgCDCLagSummary.RU) {
			cancel()
			return
		}
		// Даём часам сдвинуться: на Windows их шаг грубый, и весь прогон может
		// уложиться в один шаг — тогда интервал отчёта ещё «не истёк».
		time.Sleep(time.Millisecond)
	}
	l := NewListener(c, &fakeDLQ{}, log, func(context.Context, []byte) error { return nil })
	l.LagReportInterval = time.Nanosecond

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Contains(t, out.String(), loggerpkg.MsgCDCLagSummary.RU)
	require.Contains(t, out.String(), "partitions=1")
	require.Equal(t, stamp, l.lastEvent[partitionKey{"cdc", 2}], "event time is taken from the Kafka message timestamp")
}

func TestListenerDoesNotReportLagBeforeInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log, out := capturingLog()
	c := newFakeConsumer(cancel).addPartition("cdc", 2, 41)
	l := NewListener(c, &fakeDLQ{}, log, func(context.Context, []byte) error { return nil })
	l.LagReportInterval = time.Hour

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.NotContains(t, out.String(), loggerpkg.MsgCDCLagSummary.RU)
}

func TestPartitionsRevokedForgetsLastEvent(t *testing.T) {
	l := NewListener(nil, nil, listenerTestLog(), nil)
	l.lastEvent[partitionKey{"cdc", 2}] = time.Now()

	l.PartitionsRevoked([]kafka.TopicPartition{{Topic: ptr("cdc"), Partition: 2}})

	require.Empty(t, l.lastEvent)
}
