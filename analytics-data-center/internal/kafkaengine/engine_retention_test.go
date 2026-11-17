package kafkaengine

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/require"
)

const usersTopic = "dbserver_db1.public.users"

func newRetentionEngine(admin topicAdmin, topicRetention time.Duration) *Engine {
	return &Engine{
		log:              &loggerpkg.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))},
		admin:            admin,
		topicRetention:   topicRetention,
		retentionAligned: make(map[string]struct{}),
	}
}

func retentionChange(topic, value string) kafka.ConfigResource {
	return kafka.ConfigResource{
		Type: kafka.ResourceTopic,
		Name: topic,
		Config: []kafka.ConfigEntry{{
			Name: "retention.ms", Value: value, IncrementalOperation: kafka.AlterConfigOpTypeSet,
		}},
	}
}

func TestEngineAlignsRetentionOfExistingTopicOnce(t *testing.T) {
	admin := &fakeTopicAdmin{}
	e := newRetentionEngine(admin, 336*time.Hour)

	e.alignRetention(usersTopic)
	e.alignRetention(usersTopic) // cron ставит каждый топик в очередь каждые topic_subscription_interval

	require.Equal(t, []kafka.ConfigResource{retentionChange(usersTopic, "1209600000")}, admin.altered,
		"only retention.ms is changed, once per topic")
}

func TestEngineSkipsTopicNotCreatedYet(t *testing.T) {
	admin := &fakeTopicAdmin{missing: map[string]bool{usersTopic: true}}
	e := newRetentionEngine(admin, 336*time.Hour)

	e.alignRetention(usersTopic)
	e.alignRetention(usersTopic)

	require.Len(t, admin.altered, 1, "Connect creates the topic with the retention from topic.creation; no retries")
}

func TestEngineRetriesRetentionAfterError(t *testing.T) {
	admin := &fakeTopicAdmin{alterErr: errors.New("broker unavailable")}
	e := newRetentionEngine(admin, 336*time.Hour)

	e.alignRetention(usersTopic)
	admin.alterErr = nil
	e.alignRetention(usersTopic) // следующий тик cron
	e.alignRetention(usersTopic)

	require.Len(t, admin.altered, 2, "a failed attempt is retried, a successful one is not repeated")
}

func TestEngineKeepsBrokerDefaultTopicRetention(t *testing.T) {
	admin := &fakeTopicAdmin{}
	e := newRetentionEngine(admin, 0)

	e.alignRetention(usersTopic)

	require.Empty(t, admin.altered)
}

func TestSubscriptionWorkerAlignsRetentionOfQueuedTopics(t *testing.T) {
	consumer := newOfflineConsumer(t)
	defer consumer.Close()
	admin := &fakeTopicAdmin{}
	e := newRetentionEngine(admin, -time.Second)
	e.consumer = consumer
	e.topicCh = make(chan string, 1)
	e.subscribed = make(map[string]struct{})

	e.topicCh <- usersTopic
	close(e.topicCh)
	e.subscriptionWorker() // отработает очередь и вернётся

	require.Equal(t, []kafka.ConfigResource{retentionChange(usersTopic, "-1")}, admin.altered)
}
