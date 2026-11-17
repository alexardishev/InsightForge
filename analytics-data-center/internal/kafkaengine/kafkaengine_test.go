package kafkaengine

import (
	"bytes"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/require"
)

func TestNewKafkaConsumerRejectsAutomaticCommit(t *testing.T) {
	for _, setting := range []string{"true", "1", "TRUE"} {
		t.Run(setting, func(t *testing.T) {
			consumer, err := NewKafkaConsumer("", "cdc", "earliest", setting, "6000", "test", nil, nil, loggerpkg.New("test", "en"))
			require.Nil(t, consumer)
			require.ErrorContains(t, err, "set enable.auto.commit=false")
		})
	}
}

func TestNewKafkaConsumerRejectsInvalidAutomaticCommitSetting(t *testing.T) {
	consumer, err := NewKafkaConsumer("", "cdc", "earliest", "invalid", "6000", "test", nil, nil, loggerpkg.New("test", "en"))
	require.Nil(t, consumer)
	var parseErr *strconv.NumError
	require.ErrorAs(t, err, &parseErr)
}

func TestEngineNotifiesOnlyRevokedPartitions(t *testing.T) {
	topic := "db1.public.users"
	partitions := []kafka.TopicPartition{{Topic: &topic, Partition: 3}}
	e := &Engine{}
	e.notifyRevoked(kafka.RevokedPartitions{Partitions: partitions}) // без обработчика — без паники

	var got [][]kafka.TopicPartition
	e.OnPartitionsRevoked(func(tps []kafka.TopicPartition) { got = append(got, tps) })
	e.notifyRevoked(kafka.AssignedPartitions{Partitions: partitions})
	e.notifyRevoked(kafka.RevokedPartitions{Partitions: partitions})

	require.Equal(t, [][]kafka.TopicPartition{partitions}, got)
}

func TestBuildDLQMessage(t *testing.T) {
	topic := "db1.public.users"
	original := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 2, Offset: 41},
		Key:            []byte(`{"id":7}`),
		Value:          []byte(`{"invalid":`),
		Headers:        []kafka.Header{{Key: "trace", Value: []byte("abc")}},
	}

	msg := buildDLQMessage(original, errors.New("JSON parse error"))

	require.Equal(t, "db1.public.users.dlq", *msg.TopicPartition.Topic)
	require.Equal(t, kafka.PartitionAny, msg.TopicPartition.Partition)
	require.Equal(t, original.Key, msg.Key)
	require.Equal(t, original.Value, msg.Value)
	require.Equal(t, []kafka.Header{
		{Key: "trace", Value: []byte("abc")},
		{Key: DLQHeaderTopic, Value: []byte("db1.public.users")},
		{Key: DLQHeaderPartition, Value: []byte("2")},
		{Key: DLQHeaderOffset, Value: []byte("41")},
		{Key: DLQHeaderError, Value: []byte("JSON parse error")},
	}, msg.Headers)
	require.Len(t, original.Headers, 1, "original message headers are not modified")
}

// newOfflineConsumer создаёт настоящий consumer без брокера: Assign, Subscribe и
// Assignment/Subscription работают локально, сеть для них не нужна.
func newOfflineConsumer(t *testing.T) *kafka.Consumer {
	t.Helper()
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "127.0.0.1:1",
		"group.id":          "test",
		"log_level":         0,
	})
	require.NoError(t, err)
	return c
}

// syncBuffer — потокобезопасный приёмник логов для проверок из теста.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestNewKafkaConsumerDoesNotSleepWaitingForAssignment(t *testing.T) {
	started := time.Now()
	c, err := NewKafkaConsumer("127.0.0.1:1", "test", "earliest", "false", "6000", "test", nil, nil, loggerpkg.New("test", "en"))
	require.NoError(t, err)
	defer c.Close()
	require.Less(t, time.Since(started), 2*time.Second,
		"assignment happens inside Poll; sleeping before Poll only delays startup")
}

func TestRebalanceCallbackAssignsEveryPartition(t *testing.T) {
	c := newOfflineConsumer(t)
	defer c.Close()
	var partitions []kafka.TopicPartition
	for _, topic := range []string{
		"dbserver_db1.public.templates",
		"dbserver_db1.public.attempts",
		"dbserver_db1.public.temp_orders", // таблица источника, которую пользователь добавил во вьюху
	} {
		topic := topic
		partitions = append(partitions, kafka.TopicPartition{Topic: &topic, Partition: 0, Offset: kafka.OffsetInvalid})
	}

	require.NoError(t, RebalanceCallback(c, kafka.AssignedPartitions{Partitions: partitions}))

	assigned, err := c.Assignment()
	require.NoError(t, err)
	var topics []string
	for _, tp := range assigned {
		topics = append(topics, *tp.Topic)
	}
	require.ElementsMatch(t, []string{
		"dbserver_db1.public.templates", "dbserver_db1.public.attempts", "dbserver_db1.public.temp_orders",
	}, topics, "an unassigned partition is read by nobody: CDC for the table silently stops")
}

func TestSubscriptionWorkerRetriesFailedSubscription(t *testing.T) {
	failing := newOfflineConsumer(t)
	require.NoError(t, failing.Close()) // SubscribeTopics на закрытом consumer возвращает ошибку
	working := newOfflineConsumer(t)
	defer working.Close()
	logs := &syncBuffer{}
	e := &Engine{
		consumer:   failing,
		log:        &loggerpkg.Logger{Logger: slog.New(slog.NewTextHandler(logs, nil))},
		topicCh:    make(chan string, 10),
		subscribed: make(map[string]struct{}),
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		e.subscriptionWorker()
	}()
	defer func() {
		close(e.topicCh)
		<-done
	}()
	const topic = "dbserver_db1.public.users"

	e.EnqueueTopic(topic)
	require.Eventually(t, func() bool { return strings.Contains(logs.String(), loggerpkg.MsgKafkaSubscribeError.RU) },
		2*time.Second, 5*time.Millisecond, "first subscription attempt must fail")

	e.mu.Lock()
	e.consumer = working // брокер «ожил»
	e.mu.Unlock()
	e.EnqueueTopic(topic) // cron ставит топик в очередь снова на следующем тике

	require.Eventually(t, func() bool {
		subscription, err := working.Subscription()
		return err == nil && len(subscription) == 1 && subscription[0] == topic
	}, 2*time.Second, 5*time.Millisecond, "a failed subscription must be retried, not remembered as done")
}

func TestEngineLogsAssignedAndRevokedPartitions(t *testing.T) {
	logs := &syncBuffer{}
	e := &Engine{log: &loggerpkg.Logger{Logger: slog.New(slog.NewTextHandler(logs, nil))}}
	topic := "dbserver_db1.public.users"
	partitions := []kafka.TopicPartition{{Topic: &topic, Partition: 3}}

	e.logRebalance(kafka.AssignedPartitions{Partitions: partitions})
	e.logRebalance(kafka.RevokedPartitions{Partitions: partitions})

	out := logs.String()
	require.Contains(t, out, loggerpkg.MsgKafkaPartitionAssigned.RU)
	require.Contains(t, out, loggerpkg.MsgKafkaPartitionRevoked.RU)
	require.Equal(t, 2, strings.Count(out, "topic="+topic))
}
