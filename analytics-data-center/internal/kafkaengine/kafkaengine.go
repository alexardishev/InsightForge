package kafkaengine

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func NewKafkaConsumer(
	BootstrapServers string,
	GroupId string,
	AutoOffsetReset string,
	EnableAutoCommit string,
	SessionTimeoutMs string,
	ClientId string,
	topics []string,
	log *loggerpkg.Logger,
) (*kafka.Consumer, error) {

	// Создание Kafka consumer
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  BootstrapServers,
		"group.id":           GroupId,
		"auto.offset.reset":  AutoOffsetReset,
		"enable.auto.commit": EnableAutoCommit,
		"session.timeout.ms": SessionTimeoutMs,
		"client.id":          ClientId,
	})
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgKafkaConsumerCreateError, slog.String("error", err.Error()))
		return nil, err
	}

	// Получение всех топиков с "db"
	meta, err := c.GetMetadata(nil, true, 5000)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgKafkaMetadataError, slog.String("error", err.Error()))
		return nil, err
	}

	var matchedTopics []string
	for topic := range meta.Topics {
		if strings.Contains(topic, "db") {
			log.InfoMsg(loggerpkg.MsgKafkaTopicFound, slog.String("topic", topic))
			matchedTopics = append(matchedTopics, topic)
		}
	}

	if len(matchedTopics) == 0 {
		log.WarnMsg(loggerpkg.MsgKafkaNoPatternTopics)
	}

	if len(topics) == 0 {
		topics = []string{"^.*db.*$"}
	}
	err = c.SubscribeTopics(topics, RebalanceCallback)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgKafkaSubscribeError, slog.String("error", err.Error()))
		return nil, err
	}
	log.InfoMsg(loggerpkg.MsgKafkaConsumerCreated)

	// Ожидание initial assign
	time.Sleep(5 * time.Second)

	// Проверка assign'а
	partitions, err := c.Assignment()
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgKafkaAssignError, slog.String("error", err.Error()))
	} else if len(partitions) == 0 {
		log.WarnMsg(loggerpkg.MsgKafkaPartitionsNotAssigned)
	} else {
		for _, p := range partitions {
			log.InfoMsg(loggerpkg.MsgKafkaPartitionAssigned,
				slog.String("topic", *p.Topic),
				slog.Any("partition", p.Partition))
		}
	}

	return c, nil
}

func RebalanceCallback(c *kafka.Consumer, ev kafka.Event) error {
	switch e := ev.(type) {
	case kafka.AssignedPartitions:
		var filtered []kafka.TopicPartition
		for _, tp := range e.Partitions {
			if strings.Contains(*tp.Topic, "temp") {
				continue
			}
			filtered = append(filtered, tp)
		}
		return c.Assign(filtered)
	case kafka.RevokedPartitions:
		return c.Unassign()
	default:
		return nil
	}
}

// TopicLister describes ability to load kafka topics from storage.
type TopicLister interface {
	ListTopics(ctx context.Context) ([]string, error)
}

// Engine manages kafka subscriptions based on topics received from services.
type Engine struct {
	consumer   *kafka.Consumer
	log        *loggerpkg.Logger
	topicCh    chan string
	subscribed map[string]struct{}
	mu         sync.Mutex
}

// NewEngine creates kafka consumer, loads initial topics from lister and starts subscription worker.
func NewEngine(
	BootstrapServers string,
	GroupId string,
	AutoOffsetReset string,
	EnableAutoCommit string,
	SessionTimeoutMs string,
	ClientId string,
	lister TopicLister,
	log *loggerpkg.Logger,
) (*Engine, error) {
	var topics []string
	if lister != nil {
		t, err := lister.ListTopics(context.Background())
		if err != nil {
			log.Warn("Не удалось получить список топиков", slog.String("error", err.Error()))
		} else {
			topics = t
		}
	}

	c, err := NewKafkaConsumer(BootstrapServers, GroupId, AutoOffsetReset, EnableAutoCommit, SessionTimeoutMs, ClientId, topics, log)
	if err != nil {
		return nil, err
	}

	eng := &Engine{
		consumer:   c,
		log:        log,
		topicCh:    make(chan string, 100),
		subscribed: make(map[string]struct{}),
	}
	for _, t := range topics {
		eng.subscribed[t] = struct{}{}
	}
	go eng.subscriptionWorker()
	return eng, nil
}

// Consumer returns underlying kafka consumer.
func (e *Engine) Consumer() *kafka.Consumer { return e.consumer }

// EnqueueTopic notifies engine about new topic to subscribe.
func (e *Engine) EnqueueTopic(topic string) {
	select {
	case e.topicCh <- topic:
	default:
		e.log.Warn("topic queue full", slog.String("topic", topic))
	}
}

func (e *Engine) subscriptionWorker() {
	for topic := range e.topicCh {
		e.mu.Lock()
		if _, ok := e.subscribed[topic]; !ok {
			e.subscribed[topic] = struct{}{}
			var topics []string
			for t := range e.subscribed {
				topics = append(topics, t)
			}
			if err := e.consumer.SubscribeTopics(topics, RebalanceCallback); err != nil {
				e.log.ErrorMsg(loggerpkg.MsgKafkaSubscribeError, slog.String("error", err.Error()))
			}
		}
		e.mu.Unlock()
	}
}
