package kafkaengine

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/require"
)

// fakeTopicAdmin записывает запросы к admin-API. exists — топики, которые уже есть в кластере.
type fakeTopicAdmin struct {
	exists   map[string]bool
	created  []kafka.TopicSpecification
	altered  []kafka.ConfigResource
	alterErr error
}

func (a *fakeTopicAdmin) CreateTopics(_ context.Context, topics []kafka.TopicSpecification, _ ...kafka.CreateTopicsAdminOption) ([]kafka.TopicResult, error) {
	var results []kafka.TopicResult
	for _, spec := range topics {
		if a.exists[spec.Topic] {
			results = append(results, kafka.TopicResult{Topic: spec.Topic, Error: kafka.NewError(kafka.ErrTopicAlreadyExists, "exists", false)})
			continue
		}
		a.created = append(a.created, spec)
		results = append(results, kafka.TopicResult{Topic: spec.Topic})
	}
	return results, nil
}

func (a *fakeTopicAdmin) IncrementalAlterConfigs(_ context.Context, resources []kafka.ConfigResource, _ ...kafka.AlterConfigsAdminOption) ([]kafka.ConfigResourceResult, error) {
	a.altered = append(a.altered, resources...)
	if a.alterErr != nil {
		return nil, a.alterErr
	}
	var results []kafka.ConfigResourceResult
	for _, r := range resources {
		results = append(results, kafka.ConfigResourceResult{Type: r.Type, Name: r.Name})
	}
	return results, nil
}

func (a *fakeTopicAdmin) Close() {}

func newTestDLQ(admin topicAdmin, retention time.Duration) *DLQProducer {
	return &DLQProducer{
		admin:     admin,
		retention: retention,
		log:       &loggerpkg.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))},
		topics:    make(map[string]struct{}),
	}
}

func TestBuildDLQMessageWithoutCause(t *testing.T) {
	topic := "db1.public.users"
	msg := buildDLQMessage(&kafka.Message{TopicPartition: kafka.TopicPartition{Topic: &topic}}, nil)

	require.Contains(t, msg.Headers, kafka.Header{Key: DLQHeaderError, Value: []byte("unknown")})
}

func TestDLQSendRejectsMessageWithoutTopic(t *testing.T) {
	d := newTestDLQ(&fakeTopicAdmin{}, 0)
	for _, msg := range []*kafka.Message{nil, {}} {
		require.Error(t, d.Send(context.Background(), msg, errors.New("bad")))
	}
}

func TestRetentionMs(t *testing.T) {
	for _, tc := range []struct {
		retention time.Duration
		value     string
		set       bool
	}{
		{0, "", false},                        // значение брокера по умолчанию
		{720 * time.Hour, "2592000000", true}, // 30 дней
		{-time.Second, "-1", true},            // бессрочно
		{500 * time.Microsecond, "1", true},   // меньше миллисекунды не округляем до 0
	} {
		value, set := retentionMs(tc.retention)
		require.Equal(t, tc.set, set, tc.retention)
		require.Equal(t, tc.value, value, tc.retention)
	}
}

func TestEnsureTopicCreatesTopicWithRetention(t *testing.T) {
	admin := &fakeTopicAdmin{}
	d := newTestDLQ(admin, 720*time.Hour)

	require.NoError(t, d.ensureTopic(context.Background(), "t.dlq"))

	require.Len(t, admin.created, 1)
	require.Equal(t, "t.dlq", admin.created[0].Topic)
	require.Equal(t, map[string]string{"retention.ms": "2592000000"}, admin.created[0].Config)
	require.Empty(t, admin.altered, "a new topic already has the configured retention")
}

func TestEnsureTopicUpdatesRetentionOfExistingTopic(t *testing.T) {
	admin := &fakeTopicAdmin{exists: map[string]bool{"t.dlq": true}}
	d := newTestDLQ(admin, -time.Second)

	require.NoError(t, d.ensureTopic(context.Background(), "t.dlq"))

	require.Equal(t, []kafka.ConfigResource{{
		Type: kafka.ResourceTopic,
		Name: "t.dlq",
		Config: []kafka.ConfigEntry{{
			Name: "retention.ms", Value: "-1", IncrementalOperation: kafka.AlterConfigOpTypeSet,
		}},
	}}, admin.altered, "only retention.ms is changed, other topic settings stay as they are")
}

func TestEnsureTopicKeepsBrokerDefaultRetention(t *testing.T) {
	admin := &fakeTopicAdmin{exists: map[string]bool{"old.dlq": true}}
	d := newTestDLQ(admin, 0)

	require.NoError(t, d.ensureTopic(context.Background(), "new.dlq"))
	require.NoError(t, d.ensureTopic(context.Background(), "old.dlq"))

	require.Len(t, admin.created, 1)
	require.Empty(t, admin.created[0].Config)
	require.Empty(t, admin.altered)
}

func TestEnsureTopicCachesOnlySuccess(t *testing.T) {
	admin := &fakeTopicAdmin{exists: map[string]bool{"t.dlq": true}, alterErr: errors.New("broker unavailable")}
	d := newTestDLQ(admin, time.Hour)

	require.Error(t, d.ensureTopic(context.Background(), "t.dlq"))
	admin.alterErr = nil
	require.NoError(t, d.ensureTopic(context.Background(), "t.dlq"), "failed attempt is not cached")
	require.NoError(t, d.ensureTopic(context.Background(), "t.dlq"))

	require.Len(t, admin.altered, 2, "after success the topic is not checked again")
}
