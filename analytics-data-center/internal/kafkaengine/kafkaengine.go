package kafkaengine

import (
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func NewKafkaConsumer(BootstrapServers string, GroupId string, AutoOffsetReset string, EnableAutoCommit string, SessionTimeoutMs string, ClientId string, log *slog.Logger) (*kafka.Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  BootstrapServers,
		"group.id":           GroupId,
		"auto.offset.reset":  AutoOffsetReset,
		"enable.auto.commit": EnableAutoCommit,
		"session.timeout.ms": SessionTimeoutMs,
		"client.id":          ClientId,
	})

	if err != nil {
		log.Error("Kafka consumer creation failed", slog.String("error", err.Error()))
		return nil, err
	}

	err = c.SubscribeTopics([]string{"^.*"}, nil)
	if err != nil {
		log.Error("Subscribe failed", slog.String("error", err.Error()))
		return nil, err
	}

	meta, err := c.GetMetadata(nil, true, 5000)
	if err != nil {
		log.Error("Failed to get metadata", slog.String("error", err.Error()))
	} else {
		for topic := range meta.Topics {
			log.Info("Topic in cluster", slog.String("topic", topic))
		}
	}
	log.Info("Kafka consumer created and subscribed")

	return c, nil
}
