package cdc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaConsumer — часть API *kafka.Consumer, которая нужна listener-у.
// Интерфейс позволяет тестировать listener без брокера.
type KafkaConsumer interface {
	Poll(timeoutMs int) kafka.Event
	CommitMessage(message *kafka.Message) ([]kafka.TopicPartition, error)
	SeekPartitions(partitions []kafka.TopicPartition) ([]kafka.TopicPartition, error)
	Pause(partitions []kafka.TopicPartition) error
	Resume(partitions []kafka.TopicPartition) error
}

// DeadLetterQueue сохраняет сообщение, которое невозможно обработать.
// Send возвращает nil только после подтверждения записи брокером:
// после этого listener коммитит offset исходного сообщения.
type DeadLetterQueue interface {
	Send(ctx context.Context, message *kafka.Message, cause error) error
}

// Handler синхронно обрабатывает значение сообщения. nil — событие полностью
// записано в DWH; ошибка, помеченная Permanent, — сообщение уходит в DLQ;
// любая другая ошибка — временная, обработка будет повторена.
type Handler func(ctx context.Context, value []byte) error

const (
	DefaultInitialRetryDelay = 100 * time.Millisecond
	DefaultMaxRetryDelay     = 5 * time.Second
	maxPollTimeout           = time.Second
)

type Listener struct {
	Consumer KafkaConsumer
	DLQ      DeadLetterQueue
	Log      *loggerpkg.Logger
	Handler  Handler

	InitialRetryDelay time.Duration
	MaxRetryDelay     time.Duration

	// LagReportInterval — как часто писать в лог отставание (0 — не писать).
	LagReportInterval time.Duration
	// LagWarnAge — предупреждать, если партиция отстаёт дольше этого (0 — не
	// предупреждать). Приложение задаёт половину срока хранения топиков Debezium.
	LagWarnAge time.Duration

	// retries ворота партиций, на которых обработка упала. Доступ только из
	// горутины Run: rebalance-callback, вызывающий PartitionsRevoked, librdkafka
	// исполняет внутри Poll, то есть в той же горутине.
	retries map[partitionKey]*partitionRetry
	// lastEvent — время события, полученного последним, по партициям (для лога
	// отставания). Доступ только из горутины Run, как и у retries.
	lastEvent     map[partitionKey]time.Time
	nextLagReport time.Time
}

type partitionKey struct {
	topic     string
	partition int32
}

// partitionRetry хранит позицию упавшего сообщения. Пока запись есть, ни одно
// сообщение этой партиции, кроме сообщения на position, не обрабатывается
// и не коммитится — offset не может «перепрыгнуть» необработанное событие.
type partitionRetry struct {
	position kafka.TopicPartition
	cause    error
	attempts int
	delay    time.Duration // пауза перед следующей попыткой
	paused   bool
	resumeAt time.Time
}

func NewListener(consumer KafkaConsumer, dlq DeadLetterQueue, log *loggerpkg.Logger, handler Handler) *Listener {
	return &Listener{
		Consumer:          consumer,
		DLQ:               dlq,
		Log:               log,
		Handler:           handler,
		InitialRetryDelay: DefaultInitialRetryDelay,
		MaxRetryDelay:     DefaultMaxRetryDelay,
		LagReportInterval: DefaultLagReportInterval,
		retries:           make(map[partitionKey]*partitionRetry),
		lastEvent:         make(map[partitionKey]time.Time),
	}
}

// Start запускает Run в отдельной горутине. В канал придёт ровно одна ошибка
// (context.Canceled при штатной остановке), после чего канал закрывается.
// Пока канал не отдал результат, consumer закрывать нельзя: Run может быть внутри Poll.
func (l *Listener) Start(ctx context.Context) <-chan error {
	result := make(chan error, 1)
	go func() {
		defer close(result)
		result <- l.Run(ctx)
	}()
	return result
}

// PartitionsRevoked снимает ворота с отобранных партиций. После повторного
// назначения чтение начнётся с закоммиченного offset, а держать старую позицию
// опасно: другой экземпляр мог уже продвинуть партицию. Вызывается из
// rebalance-callback (внутри Poll).
func (l *Listener) PartitionsRevoked(partitions []kafka.TopicPartition) {
	for _, tp := range partitions {
		if tp.Topic == nil {
			continue
		}
		key := partitionKey{topic: *tp.Topic, partition: tp.Partition}
		delete(l.retries, key)
		delete(l.lastEvent, key)
	}
}

// Run читает сообщения и обрабатывает их строго по одному. Возвращает
// ctx.Err() при остановке или фатальной ошибке, после которой продолжать
// нельзя (например, не удалось вернуть партицию на упавшее сообщение).
func (l *Listener) Run(ctx context.Context) error {
	l.nextLagReport = time.Now().Add(l.LagReportInterval)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		l.resumeDue(time.Now())
		l.maybeReportLag(time.Now())
		ev := l.Consumer.Poll(l.pollTimeoutMs(time.Now()))
		if err := ctx.Err(); err != nil {
			// Сообщение, если оно пришло, не обработано и не закоммичено — придёт снова.
			return err
		}
		switch e := ev.(type) {
		case *kafka.Message:
			if err := l.handleMessage(ctx, e); err != nil {
				return err
			}
		case kafka.Error:
			l.Log.ErrorMsg(loggerpkg.MsgKafkaError, slog.String("error", e.Error()))
			if e.IsFatal() {
				return fmt.Errorf("CDC consumer failed: %w", e)
			}
		}
	}
}

func (l *Listener) handleMessage(ctx context.Context, msg *kafka.Message) error {
	if msg.TopicPartition.Topic == nil {
		return errors.New("CDC message has no topic")
	}
	key := partitionKey{topic: *msg.TopicPartition.Topic, partition: msg.TopicPartition.Partition}
	log := l.Log.With(
		slog.String("topic", key.topic),
		slog.Any("partition", key.partition),
		slog.String("offset", msg.TopicPartition.Offset.String()),
	)

	retry := l.retries[key]
	if retry != nil && (retry.paused || msg.TopicPartition.Offset != retry.position.Offset) {
		// Сообщение партиции, стоящей на воротах, но не то, которое ждём (или
		// пауза ещё не истекла). Возвращаем партицию на позицию; backoff не трогаем.
		return l.seek(retry)
	}

	log.InfoMsg(loggerpkg.MsgCDCMessageReceived)
	l.lastEvent[key] = eventTime(msg)
	err := l.process(ctx, msg, log)
	if err == nil {
		delete(l.retries, key)
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		// Остановка прервала обработку: offset не закоммичен, после рестарта
		// сообщение будет прочитано снова.
		return ctxErr
	}
	return l.scheduleRetry(key, retry, msg.TopicPartition, err, log)
}

// process: обработка → (DLQ при постоянной ошибке) → коммит.
// Коммит выполняется и при отменённом ctx: запись в DWH к этому моменту уже
// завершена, и коммит избавляет от лишнего повтора после рестарта.
func (l *Listener) process(ctx context.Context, msg *kafka.Message, log *loggerpkg.Logger) error {
	if err := l.Handler(ctx, msg.Value); err != nil {
		if !IsPermanent(err) {
			log.ErrorMsg(loggerpkg.MsgCDCHandlerFailed, slog.String("error", err.Error()))
			return err
		}
		log.ErrorMsg(loggerpkg.MsgCDCPermanentError, slog.String("error", err.Error()))
		if dlqErr := l.sendToDLQ(ctx, msg, err); dlqErr != nil {
			log.ErrorMsg(loggerpkg.MsgCDCDLQSendFailed, slog.String("error", dlqErr.Error()))
			return dlqErr
		}
		log.WarnMsg(loggerpkg.MsgCDCSentToDLQ)
	}
	return l.commit(msg, log)
}

func (l *Listener) sendToDLQ(ctx context.Context, msg *kafka.Message, cause error) error {
	if l.DLQ == nil {
		return fmt.Errorf("DLQ is not configured: %w", cause)
	}
	if err := l.DLQ.Send(ctx, msg, cause); err != nil {
		return fmt.Errorf("DLQ send failed: %w", err)
	}
	return nil
}

func (l *Listener) commit(msg *kafka.Message, log *loggerpkg.Logger) error {
	committed, err := l.Consumer.CommitMessage(msg)
	if err == nil {
		for _, tp := range committed {
			if tp.Error != nil {
				err = tp.Error
				break
			}
		}
	}
	if err != nil {
		// Результат коммита неизвестен — повторяем обработку целиком.
		// Это безопасно: запись в DWH идемпотентна (upsert по ключу).
		log.ErrorMsg(loggerpkg.MsgKafkaCommitError, slog.String("error", err.Error()))
		return fmt.Errorf("commit failed: %w", err)
	}
	return nil
}

// scheduleRetry ставит ворота на партицию, возвращает её на упавшее сообщение
// и ставит на паузу. Остальные партиции продолжают читаться без задержки.
func (l *Listener) scheduleRetry(key partitionKey, retry *partitionRetry, tp kafka.TopicPartition, cause error, log *loggerpkg.Logger) error {
	if retry == nil {
		retry = &partitionRetry{
			position: kafka.TopicPartition{Topic: tp.Topic, Partition: tp.Partition, Offset: tp.Offset},
			delay:    l.InitialRetryDelay,
		}
		l.retries[key] = retry
	}
	retry.attempts++
	retry.cause = cause
	if err := l.seek(retry); err != nil {
		return err
	}
	if err := l.Consumer.Pause([]kafka.TopicPartition{retry.position}); err != nil {
		return fmt.Errorf("CDC retry %s[%d] at offset %s after %w: pause failed: %w",
			key.topic, key.partition, retry.position.Offset, cause, err)
	}
	retry.paused = true
	retry.resumeAt = time.Now().Add(retry.delay)
	log.WarnMsg(loggerpkg.MsgCDCRetryScheduled,
		slog.Int("attempt", retry.attempts),
		slog.Duration("delay", retry.delay),
		slog.String("error", cause.Error()),
	)
	retry.delay = l.nextRetryDelay(retry.delay)
	return nil
}

func (l *Listener) seek(retry *partitionRetry) error {
	// SeekPartitions блокирует до обновления позиции fetcher-а и отбрасывает
	// сообщения, уже скачанные со старой позиции.
	result, err := l.Consumer.SeekPartitions([]kafka.TopicPartition{retry.position})
	if err == nil {
		for _, tp := range result {
			if tp.Error != nil {
				err = tp.Error
				break
			}
		}
	}
	if err != nil {
		return fmt.Errorf("CDC retry %s[%d] at offset %s after %w: seek failed: %w",
			*retry.position.Topic, retry.position.Partition, retry.position.Offset, retry.cause, err)
	}
	return nil
}

// resumeDue снимает паузу с партиций, у которых истёк backoff.
func (l *Listener) resumeDue(now time.Time) {
	for _, retry := range l.retries {
		if !retry.paused || now.Before(retry.resumeAt) {
			continue
		}
		if err := l.Consumer.Resume([]kafka.TopicPartition{retry.position}); err != nil {
			// Не фатально: партиция могла быть отобрана, тогда ворота снимет PartitionsRevoked.
			l.Log.WarnMsg(loggerpkg.MsgCDCResumeFailed,
				slog.String("topic", *retry.position.Topic),
				slog.Any("partition", retry.position.Partition),
				slog.String("error", err.Error()),
			)
		}
		retry.paused = false
	}
}

// pollTimeoutMs ограничивает ожидание в Poll ближайшим моментом снятия паузы,
// чтобы повтор не опаздывал больше чем на время одного Poll.
func (l *Listener) pollTimeoutMs(now time.Time) int {
	timeout := maxPollTimeout
	for _, retry := range l.retries {
		if !retry.paused {
			continue
		}
		if wait := retry.resumeAt.Sub(now); wait < timeout {
			timeout = wait
		}
	}
	if timeout < 0 {
		timeout = 0
	}
	return int(timeout.Milliseconds())
}

func (l *Listener) nextRetryDelay(delay time.Duration) time.Duration {
	if delay >= l.MaxRetryDelay/2 {
		return l.MaxRetryDelay
	}
	return delay * 2
}
