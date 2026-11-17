package kafkaengine

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"analyticDataCenter/analytics-data-center/internal/lib/retention"
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
	rebalanceCb kafka.RebalanceCb,
	log *loggerpkg.Logger,
) (*kafka.Consumer, error) {
	autoCommit, err := strconv.ParseBool(EnableAutoCommit)
	if err != nil {
		return nil, fmt.Errorf("invalid enable.auto.commit: %w", err)
	}
	if autoCommit {
		return nil, fmt.Errorf("CDC commits offsets manually after the DWH write: set enable.auto.commit=false")
	}
	if rebalanceCb == nil {
		rebalanceCb = RebalanceCallback
	}

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": BootstrapServers,
		"group.id":          GroupId,
		"auto.offset.reset": AutoOffsetReset,
		// Offset коммитит только cdc.Listener (CommitMessage) после записи в DWH.
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,
		"session.timeout.ms":       SessionTimeoutMs,
		"client.id":                ClientId,
	})
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgKafkaConsumerCreateError, slog.String("error", err.Error()))
		return nil, err
	}

	if len(topics) == 0 {
		log.WarnMsg(loggerpkg.MsgKafkaNoPatternTopics)
	} else {
		log.Info("Kafka consumer subscribing to topics", slog.Any("topics", topics))
		err = c.SubscribeTopics(topics, rebalanceCb)
		if err != nil {
			log.ErrorMsg(loggerpkg.MsgKafkaSubscribeError, slog.String("error", err.Error()))
			return nil, err
		}
	}

	// Ждать назначения партиций здесь бессмысленно: rebalance-callback, который их
	// назначает, librdkafka вызывает только внутри Poll. Назначенные партиции
	// логирует Engine.rebalance.
	log.InfoMsg(loggerpkg.MsgKafkaConsumerCreated)
	return c, nil
}

// RebalanceCallback принимает все назначенные партиции. Отбрасывать их нельзя:
// координатор группы считает партицию нашей, и её не будет читать никто.
// Ненужные топики отсекаются раньше — подписка идёт только на таблицы вьюх (ListTopics).
func RebalanceCallback(c *kafka.Consumer, ev kafka.Event) error {
	switch e := ev.(type) {
	case kafka.AssignedPartitions:
		return c.Assign(e.Partitions)
	case kafka.RevokedPartitions:
		return c.Unassign()
	default:
		return nil
	}
}

type TopicLister interface {
	ListTopics(ctx context.Context) ([]string, error)
}

// adminRequestTimeout — сколько ждать ответа admin-API Kafka.
const adminRequestTimeout = 10 * time.Second

type Engine struct {
	consumer   *kafka.Consumer
	log        *loggerpkg.Logger
	topicCh    chan string
	subscribed map[string]struct{}
	mu         sync.Mutex
	// onRevoked вызывается из rebalance-callback, то есть внутри Poll.
	onRevoked atomic.Pointer[func([]kafka.TopicPartition)]

	// admin создан из consumer-а и закрывается раньше него (Close).
	admin topicAdmin
	// topicRetention — срок хранения топиков Debezium (kafka.debezium_topic_retention).
	topicRetention time.Duration
	// retentionAligned — топики, срок хранения которых уже приведён к конфигу.
	// Доступ только из subscriptionWorker.
	retentionAligned map[string]struct{}
}

func NewEngine(
	BootstrapServers string,
	GroupId string,
	AutoOffsetReset string,
	EnableAutoCommit string,
	SessionTimeoutMs string,
	ClientId string,
	topicRetention time.Duration,
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

	eng := &Engine{
		log:              log,
		topicCh:          make(chan string, 100),
		subscribed:       make(map[string]struct{}),
		topicRetention:   topicRetention,
		retentionAligned: make(map[string]struct{}),
	}
	c, err := NewKafkaConsumer(BootstrapServers, GroupId, AutoOffsetReset, EnableAutoCommit, SessionTimeoutMs, ClientId, topics, eng.rebalance, log)
	if err != nil {
		return nil, err
	}
	admin, err := kafka.NewAdminClientFromConsumer(c)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("create Kafka admin client: %w", err)
	}
	eng.consumer = c
	eng.admin = admin
	for _, t := range topics {
		eng.subscribed[t] = struct{}{}
	}
	eng.EnsureKafkaConnectTopics(BootstrapServers, log)
	go eng.subscriptionWorker()
	return eng, nil
}

// Вернет эземпляр консьюмера
func (e *Engine) Consumer() *kafka.Consumer { return e.consumer }

// Close закрывает admin-клиент, затем consumer (admin создан из consumer-а).
// Вызывать только после остановки cdc.Listener: закрывать consumer, пока listener
// внутри Poll, нельзя.
func (e *Engine) Close() error {
	if e.admin != nil {
		e.admin.Close()
	}
	return e.consumer.Close()
}

// OnPartitionsRevoked регистрирует обработчик отзыва партиций (cdc.Listener.PartitionsRevoked).
// Обработчик вызывается из rebalance-callback внутри Poll, в горутине, которая вызывает Poll.
func (e *Engine) OnPartitionsRevoked(handler func([]kafka.TopicPartition)) {
	e.onRevoked.Store(&handler)
}

func (e *Engine) rebalance(c *kafka.Consumer, ev kafka.Event) error {
	e.logRebalance(ev)
	e.notifyRevoked(ev)
	return RebalanceCallback(c, ev)
}

func (e *Engine) logRebalance(ev kafka.Event) {
	var (
		msg        loggerpkg.Message
		partitions []kafka.TopicPartition
	)
	switch ev := ev.(type) {
	case kafka.AssignedPartitions:
		msg, partitions = loggerpkg.MsgKafkaPartitionAssigned, ev.Partitions
	case kafka.RevokedPartitions:
		msg, partitions = loggerpkg.MsgKafkaPartitionRevoked, ev.Partitions
	default:
		return
	}
	for _, tp := range partitions {
		e.log.InfoMsg(msg, slog.String("topic", *tp.Topic), slog.Any("partition", tp.Partition))
	}
}

func (e *Engine) notifyRevoked(ev kafka.Event) {
	revoked, ok := ev.(kafka.RevokedPartitions)
	if !ok {
		return
	}
	if handler := e.onRevoked.Load(); handler != nil {
		(*handler)(revoked.Partitions)
	}
}

// Событие в очередь и дальше в воркер
func (e *Engine) EnqueueTopic(topic string) {
	select {
	case e.topicCh <- topic:
	default:
		e.log.Warn("topic queue full", slog.String("topic", topic))
	}
}

func (e *Engine) subscriptionWorker() {
	for topic := range e.topicCh {
		e.subscribe(topic)
		e.alignRetention(topic)
	}
}

// alignRetention приводит срок хранения уже существующего топика к конфигу (меняет
// только retention.ms), один раз на топик за время работы процесса. Новые топики
// создаёт Connect сразу с нужным сроком (topic.creation.* в конфиге коннектора),
// поэтому отсутствующий топик пропускается без повторов. При другой ошибке попытка
// повторится, когда cron снова поставит топик в очередь.
func (e *Engine) alignRetention(topic string) {
	value, set := retention.Ms(e.topicRetention)
	if !set || e.admin == nil {
		return
	}
	if _, ok := e.retentionAligned[topic]; ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), adminRequestTimeout)
	defer cancel()
	err := setTopicRetention(ctx, e.admin, topic, value)
	switch {
	case err == nil:
		e.log.InfoMsg(loggerpkg.MsgKafkaTopicRetentionSet, slog.String("topic", topic), slog.String(retention.ConfigKey, value))
	case isUnknownTopic(err):
		e.log.InfoMsg(loggerpkg.MsgKafkaTopicNotCreatedYet, slog.String("topic", topic))
	default:
		e.log.WarnMsg(loggerpkg.MsgKafkaTopicRetentionFailed, slog.String("topic", topic), slog.String("error", err.Error()))
		return
	}
	e.retentionAligned[topic] = struct{}{}
}

// subscribe добавляет топик к подписке. В subscribed топик попадает только после
// успешного SubscribeTopics: при ошибке cron через topic_subscription_interval
// снова поставит его в очередь, и попытка повторится.
func (e *Engine) subscribe(topic string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.subscribed[topic]; ok {
		return
	}
	topics := make([]string, 0, len(e.subscribed)+1)
	for t := range e.subscribed {
		topics = append(topics, t)
	}
	topics = append(topics, topic)
	sort.Strings(topics)
	if err := e.consumer.SubscribeTopics(topics, e.rebalance); err != nil {
		e.log.ErrorMsg(loggerpkg.MsgKafkaSubscribeError, slog.String("topic", topic), slog.String("error", err.Error()))
		return
	}
	e.subscribed[topic] = struct{}{}
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
