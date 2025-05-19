package kafkaengine

import (
	"log/slog"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func NewKafkaConsumer(
	BootstrapServers string,
	GroupId string,
	AutoOffsetReset string,
	EnableAutoCommit string,
	SessionTimeoutMs string,
	ClientId string,
	log *slog.Logger,
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
		log.Error("Kafka ошибка создания консюмера", slog.String("error", err.Error()))
		return nil, err
	}

	// Получение всех топиков с "db"
	meta, err := c.GetMetadata(nil, true, 5000)
	if err != nil {
		log.Error("Ошибка получения метаданных", slog.String("error", err.Error()))
		return nil, err
	}

	var matchedTopics []string
	for topic := range meta.Topics {
		if strings.Contains(topic, "db") {
			log.Info("Найденый топик", slog.String("топик", topic))
			matchedTopics = append(matchedTopics, topic)
		}
	}

	if len(matchedTopics) == 0 {
		log.Warn("Нет подходящий топиков с паттерном 'db'")
	}

	// Подписка на топики по паттерну (или можно matchedTopics, если точечно)
	err = c.SubscribeTopics([]string{"^.*db.*$"}, nil)
	if err != nil {
		log.Error("Неудачная подписка", slog.String("error", err.Error()))
		return nil, err
	}
	log.Info("Kafka consumer создан и подписан")

	// Ожидание initial assign
	time.Sleep(5 * time.Second)

	// Проверка assign'а
	partitions, err := c.Assignment()
	if err != nil {
		log.Error("Неудачная попытка назначений", slog.String("error", err.Error()))
	} else if len(partitions) == 0 {
		log.Warn("Разделы пока не назначены — возможно, нет данных или нет соответствующих тем")
	} else {
		for _, p := range partitions {
			log.Info("Assigned topic",
				slog.String("topic", *p.Topic),
				slog.Any("partition", p.Partition))
		}
	}

	return c, nil
}
