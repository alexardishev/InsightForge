package kafkaengine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// DLQTopicSuffix — DLQ для топика <topic> называется <topic>.dlq.
const DLQTopicSuffix = ".dlq"

// Заголовки, которые DLQ-сообщение получает в дополнение к исходным.
const (
	DLQHeaderTopic     = "dlq.original.topic"
	DLQHeaderPartition = "dlq.original.partition"
	DLQHeaderOffset    = "dlq.original.offset"
	DLQHeaderError     = "dlq.error"
)

const (
	dlqFlushTimeoutMs = 5000
	retentionMsConfig = "retention.ms"
)

// topicAdmin — часть *kafka.AdminClient, нужная DLQ; интерфейс позволяет тестировать ensureTopic.
type topicAdmin interface {
	CreateTopics(ctx context.Context, topics []kafka.TopicSpecification, options ...kafka.CreateTopicsAdminOption) ([]kafka.TopicResult, error)
	IncrementalAlterConfigs(ctx context.Context, resources []kafka.ConfigResource, options ...kafka.AlterConfigsAdminOption) ([]kafka.ConfigResourceResult, error)
	Close()
}

// DLQProducer пишет необрабатываемые CDC-сообщения в <topic>.dlq.
// Реализует cdc.DeadLetterQueue.
type DLQProducer struct {
	producer *kafka.Producer
	admin    topicAdmin
	log      *loggerpkg.Logger
	// retention — срок хранения сообщений в DLQ-топиках (kafka.dlq_retention):
	// 0 — значение брокера по умолчанию, отрицательное — бессрочно.
	retention time.Duration

	mu     sync.Mutex
	topics map[string]struct{} // DLQ-топики, которые уже созданы или настроены
}

func NewDLQProducer(bootstrapServers, clientID string, retention time.Duration, log *loggerpkg.Logger) (*DLQProducer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"client.id":         clientID + "-dlq",
		// Коммит исходного offset идёт только после подтверждения записи в DLQ,
		// поэтому ждём подтверждения от всех реплик и не допускаем дублей при ретраях.
		"acks":               "all",
		"enable.idempotence": true,
	})
	if err != nil {
		return nil, fmt.Errorf("create DLQ producer: %w", err)
	}
	admin, err := kafka.NewAdminClientFromProducer(p)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("create DLQ admin client: %w", err)
	}
	d := &DLQProducer{producer: p, admin: admin, log: log, retention: retention, topics: make(map[string]struct{})}
	go d.logEvents()
	return d, nil
}

// Send пишет копию сообщения в DLQ и ждёт подтверждения брокера.
func (d *DLQProducer) Send(ctx context.Context, msg *kafka.Message, cause error) error {
	if msg == nil || msg.TopicPartition.Topic == nil {
		// Без исходного топика не из чего собрать имя DLQ-топика.
		return errors.New("DLQ: message has no topic")
	}
	dlqMsg := buildDLQMessage(msg, cause)
	if err := d.ensureTopic(ctx, *dlqMsg.TopicPartition.Topic); err != nil {
		return err
	}
	// Буфер 1: если ctx отменят раньше отчёта о доставке, librdkafka не заблокируется.
	delivery := make(chan kafka.Event, 1)
	if err := d.producer.Produce(dlqMsg, delivery); err != nil {
		return fmt.Errorf("produce to %s: %w", *dlqMsg.TopicPartition.Topic, err)
	}
	select {
	case ev := <-delivery:
		m, ok := ev.(*kafka.Message)
		if !ok {
			return fmt.Errorf("unexpected DLQ delivery event: %v", ev)
		}
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("deliver to %s: %w", *dlqMsg.TopicPartition.Topic, m.TopicPartition.Error)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close дожидается отправки сообщений в очереди и закрывает продюсер.
func (d *DLQProducer) Close() {
	d.producer.Flush(dlqFlushTimeoutMs)
	d.admin.Close()
	d.producer.Close()
}

// ensureTopic создаёт DLQ-топик со сроком хранения из конфига (нельзя полагаться
// на auto.create.topics.enable брокера). У существующего топика срок хранения
// приводится к конфигу, поэтому изменение kafka.dlq_retention действует после
// рестарта. Проверка выполняется один раз на топик за время работы процесса.
func (d *DLQProducer) ensureTopic(ctx context.Context, topic string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.topics[topic]; ok {
		return nil
	}
	retention, setRetention := retentionMs(d.retention)
	spec := kafka.TopicSpecification{
		Topic:         topic,
		NumPartitions: 1, // ReplicationFactor 0 — значение брокера по умолчанию
	}
	if setRetention {
		spec.Config = map[string]string{retentionMsConfig: retention}
	}
	results, err := d.admin.CreateTopics(ctx, []kafka.TopicSpecification{spec})
	if err != nil {
		d.log.ErrorMsg(loggerpkg.MsgKafkaDLQTopicFailed, slog.String("topic", topic), slog.String("error", err.Error()))
		return fmt.Errorf("create DLQ topic %s: %w", topic, err)
	}
	exists := false
	for _, res := range results {
		switch res.Error.Code() {
		case kafka.ErrNoError:
		case kafka.ErrTopicAlreadyExists:
			exists = true
		default:
			d.log.ErrorMsg(loggerpkg.MsgKafkaDLQTopicFailed, slog.String("topic", topic), slog.String("error", res.Error.String()))
			return fmt.Errorf("create DLQ topic %s: %w", topic, res.Error)
		}
	}
	if exists && setRetention {
		if err := d.setRetention(ctx, topic, retention); err != nil {
			return err
		}
	}
	d.topics[topic] = struct{}{}
	return nil
}

// setRetention меняет только retention.ms: IncrementalAlterConfigs, в отличие от
// AlterConfigs, не сбрасывает остальные настройки топика к значениям по умолчанию.
func (d *DLQProducer) setRetention(ctx context.Context, topic, retention string) error {
	results, err := d.admin.IncrementalAlterConfigs(ctx, []kafka.ConfigResource{{
		Type: kafka.ResourceTopic,
		Name: topic,
		Config: []kafka.ConfigEntry{{
			Name: retentionMsConfig, Value: retention, IncrementalOperation: kafka.AlterConfigOpTypeSet,
		}},
	}})
	if err == nil {
		for _, res := range results {
			if res.Error.Code() != kafka.ErrNoError {
				err = res.Error
				break
			}
		}
	}
	if err != nil {
		d.log.ErrorMsg(loggerpkg.MsgKafkaDLQRetentionFailed, slog.String("topic", topic), slog.String("error", err.Error()))
		return fmt.Errorf("set retention of DLQ topic %s: %w", topic, err)
	}
	d.log.InfoMsg(loggerpkg.MsgKafkaDLQRetentionSet, slog.String("topic", topic), slog.String(retentionMsConfig, retention))
	return nil
}

// retentionMs переводит kafka.dlq_retention в значение retention.ms.
// false — оставить значение брокера по умолчанию.
func retentionMs(retention time.Duration) (string, bool) {
	switch {
	case retention == 0:
		return "", false
	case retention < 0:
		return "-1", true // бессрочно
	case retention < time.Millisecond:
		return "1", true // 0 у Kafka значит «удалять сразу», а не «по умолчанию»
	default:
		return strconv.FormatInt(retention.Milliseconds(), 10), true
	}
}

// logEvents логирует ошибки продюсера (отчёты о доставке приходят в канал delivery Send).
// Канал закрывается в producer.Close.
func (d *DLQProducer) logEvents() {
	for ev := range d.producer.Events() {
		if e, ok := ev.(kafka.Error); ok {
			d.log.ErrorMsg(loggerpkg.MsgKafkaDLQEventError, slog.String("error", e.Error()))
		}
	}
}

// buildDLQMessage копирует ключ, значение и заголовки исходного сообщения и
// добавляет, откуда оно пришло и почему не обработано.
func buildDLQMessage(msg *kafka.Message, cause error) *kafka.Message {
	source := *msg.TopicPartition.Topic
	topic := source + DLQTopicSuffix
	headers := make([]kafka.Header, 0, len(msg.Headers)+4)
	headers = append(headers, msg.Headers...)
	headers = append(headers,
		kafka.Header{Key: DLQHeaderTopic, Value: []byte(source)},
		kafka.Header{Key: DLQHeaderPartition, Value: []byte(strconv.Itoa(int(msg.TopicPartition.Partition)))},
		kafka.Header{Key: DLQHeaderOffset, Value: []byte(msg.TopicPartition.Offset.String())},
		kafka.Header{Key: DLQHeaderError, Value: []byte(errorText(cause))},
	)
	return &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            msg.Key,
		Value:          msg.Value,
		Headers:        headers,
	}
}

// errorText защищает от nil: Send — публичный метод, причину может не передать вызывающий.
func errorText(err error) string {
	if err == nil {
		return "unknown"
	}
	return err.Error()
}
