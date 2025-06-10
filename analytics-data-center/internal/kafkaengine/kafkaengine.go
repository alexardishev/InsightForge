package kafkaengine

import (
	"log/slog"
	"strings"
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

	// Подписка на топики по паттерну (или можно matchedTopics, если точечно)
	err = c.SubscribeTopics([]string{"^.*db.*$"}, rebalanceCallback)
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

func rebalanceCallback(c *kafka.Consumer, ev kafka.Event) error {
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
