package cdc

import (
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Listener struct {
	Consumer *kafka.Consumer
	Log      *slog.Logger
	Handler  func(event []byte)
}

func NewListener(consumer *kafka.Consumer, log *slog.Logger, handler func([]byte)) *Listener {
	return &Listener{Consumer: consumer, Log: log, Handler: handler}
}

func (l *Listener) Start() {
	go func() {
		for {
			ev := l.Consumer.Poll(1000)
			if ev == nil {
				continue
			}
			switch e := ev.(type) {
			case *kafka.Message:
				l.Log.Info("CDC сообщение получено", slog.String("topic", *e.TopicPartition.Topic))
				l.Handler(e.Value) // отправляем на обработку
				// ручной коммит
				_, err := l.Consumer.CommitMessage(e)
				if err != nil {
					l.Log.Error("Ошибка при коммите Kafka offset", slog.String("error", err.Error()))
				}
			case kafka.Error:
				l.Log.Error("Kafka ошибка", slog.String("error", e.Error()))
			}
		}
	}()
}
