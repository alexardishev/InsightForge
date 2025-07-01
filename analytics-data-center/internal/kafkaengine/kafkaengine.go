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

	if len(topics) == 0 {
		log.WarnMsg(loggerpkg.MsgKafkaNoPatternTopics)
	} else {
		log.Info("Kafka consumer subscribing to topics", slog.Any("topics", topics))
		err = c.SubscribeTopics(topics, RebalanceCallback)
		if err != nil {
			log.ErrorMsg(loggerpkg.MsgKafkaSubscribeError, slog.String("error", err.Error()))
			return nil, err
		}
	}

	log.InfoMsg(loggerpkg.MsgKafkaConsumerCreated)

	// Ожидание assign'а
	time.Sleep(5 * time.Second)

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

type TopicLister interface {
	ListTopics(ctx context.Context) ([]string, error)
}

type Engine struct {
	consumer   *kafka.Consumer
	log        *loggerpkg.Logger
	topicCh    chan string
	subscribed map[string]struct{}
	mu         sync.Mutex
}

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
	eng.EnsureKafkaConnectTopics(BootstrapServers, log)
	go eng.subscriptionWorker()
	return eng, nil
}

// Вернет эземпляр консьюмера
func (e *Engine) Consumer() *kafka.Consumer { return e.consumer }

// Событие в очередь и в дальше в воркер
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

func (e *Engine) EnsureKafkaConnectTopics(bootstrapServers string, log *loggerpkg.Logger) error {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	})
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgKafkaAdminClientCreateError, slog.String("error", err.Error()))
		return err
	}
	defer adminClient.Close()

	metadata, err := adminClient.GetMetadata(nil, true, 5000)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgKafkaAdminMetadataError, slog.String("error", err.Error()))
		return err
	}

	requiredTopics := map[string]kafka.TopicSpecification{
		"connect_offsets": {
			Topic:             "connect_offsets",
			NumPartitions:     25,
			ReplicationFactor: 1,
			Config:            map[string]string{"cleanup.policy": "compact"},
		},
		"connect_configs": {
			Topic:             "connect_configs",
			NumPartitions:     1,
			ReplicationFactor: 1,
			Config:            map[string]string{"cleanup.policy": "compact"},
		},
		"connect_statuses": {
			Topic:             "connect_statuses",
			NumPartitions:     5,
			ReplicationFactor: 1,
			Config:            map[string]string{"cleanup.policy": "compact"},
		},
	}

	var toCreate []kafka.TopicSpecification
	var toAlter []kafka.ConfigResource

	for name, spec := range requiredTopics {
		_, exists := metadata.Topics[name]
		if !exists {
			log.InfoMsg(loggerpkg.MsgKafkaAdminTopicMarkedForCreation, slog.String("topic", name))
			toCreate = append(toCreate, spec)
			continue
		}

		// Проверка cleanup.policy
		cfgs, err := adminClient.DescribeConfigs(
			context.Background(),
			[]kafka.ConfigResource{
				{Type: kafka.ResourceTopic, Name: name},
			},
		)
		if err != nil || len(cfgs) == 0 {
			log.WarnMsg(loggerpkg.MsgKafkaAdminConfigReadError, slog.String("topic", name), slog.String("error", err.Error()))
			continue
		}

		cleanup := cfgs[0].Config["cleanup.policy"]
		if cleanup.Value != "compact" {
			log.InfoMsg(loggerpkg.MsgKafkaAdminUpdateNeeded, slog.String("topic", name), slog.String("oldPolicy", cleanup.Value))
			toAlter = append(toAlter, kafka.ConfigResource{
				Type: kafka.ResourceTopic,
				Name: name,
				Config: []kafka.ConfigEntry{
					{Name: "cleanup.policy", Value: "compact"},
				}})
		}
	}

	// Создание новых топиков
	if len(toCreate) > 0 {
		results, err := adminClient.CreateTopics(context.Background(), toCreate)
		if err != nil {
			log.ErrorMsg(loggerpkg.MsgKafkaAdminCreateError, slog.String("error", err.Error()))
			return err
		}
		for _, res := range results {
			if res.Error.Code() != kafka.ErrNoError {
				log.WarnMsg(loggerpkg.MsgKafkaAdminCreateWarning, slog.String("topic", res.Topic), slog.String("error", res.Error.String()))
			} else {
				log.InfoMsg(loggerpkg.MsgKafkaAdminCreateSuccess, slog.String("topic", res.Topic))
			}
		}
	}

	// Обновление конфигураций существующих топиков
	if len(toAlter) > 0 {
		results, err := adminClient.AlterConfigs(context.Background(), toAlter)
		if err != nil {
			log.ErrorMsg(loggerpkg.MsgKafkaAdminUpdateError, slog.String("error", err.Error()))
			return err
		}
		for _, res := range results {
			if res.Error.Code() != kafka.ErrNoError {
				log.WarnMsg(loggerpkg.MsgKafkaAdminUpdateWarning, slog.String("topic", res.Name), slog.String("error", res.Error.String()))
			} else {
				log.InfoMsg(loggerpkg.MsgKafkaAdminUpdateSuccess, slog.String("topic", res.Name))
			}
		}
	}

	return nil
}
