package kafkaengine

import (
	"context"
	"errors"

	"analyticDataCenter/analytics-data-center/internal/lib/retention"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// topicAdmin — часть *kafka.AdminClient для создания топиков и смены их настроек.
// Интерфейс позволяет тестировать DLQ и Engine без брокера.
type topicAdmin interface {
	CreateTopics(ctx context.Context, topics []kafka.TopicSpecification, options ...kafka.CreateTopicsAdminOption) ([]kafka.TopicResult, error)
	IncrementalAlterConfigs(ctx context.Context, resources []kafka.ConfigResource, options ...kafka.AlterConfigsAdminOption) ([]kafka.ConfigResourceResult, error)
	Close()
}

// setTopicRetention меняет у топика только retention.ms: IncrementalAlterConfigs, в отличие
// от AlterConfigs, не сбрасывает остальные настройки к значениям по умолчанию.
// Ошибка по самому топику возвращается как kafka.Error (см. isUnknownTopic).
func setTopicRetention(ctx context.Context, admin topicAdmin, topic, value string) error {
	results, err := admin.IncrementalAlterConfigs(ctx, []kafka.ConfigResource{{
		Type: kafka.ResourceTopic,
		Name: topic,
		Config: []kafka.ConfigEntry{{
			Name: retention.ConfigKey, Value: value, IncrementalOperation: kafka.AlterConfigOpTypeSet,
		}},
	}})
	if err != nil {
		return err
	}
	for _, res := range results {
		if res.Error.Code() != kafka.ErrNoError {
			return res.Error
		}
	}
	return nil
}

// isUnknownTopic — топика ещё нет в кластере.
func isUnknownTopic(err error) bool {
	var kafkaErr kafka.Error
	return errors.As(err, &kafkaErr) && kafkaErr.Code() == kafka.ErrUnknownTopicOrPart
}
