package cdc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/require"
)

// fakePartition — лог одной партиции и позиция чтения в нём.
type fakePartition struct {
	key partitionKey
	log []*kafka.Message
	pos int
}

// fakeConsumer имитирует поведение librdkafka, важное для listener-а:
// сообщения читаются по позиции партиции, на паузе партиция ничего не отдаёт,
// SeekPartitions переставляет позицию. injected — события, которые Poll отдаёт
// первыми в обход паузы (Kafka-ошибки или «уже скачанное» сообщение).
type fakeConsumer struct {
	parts    []*fakePartition
	paused   map[partitionKey]bool
	injected []kafka.Event
	next     int

	handled []string // заполняет тестовый handler
	commits []string
	seeks   []string
	pauses  []string
	resumes []string

	polls      int
	cancel     context.CancelFunc
	beforePoll func()
	idle       func()
	commitErr  func(*kafka.Message) ([]kafka.TopicPartition, error)
	seekErr    error
	pauseErr   error
}

func msgID(tp kafka.TopicPartition) string {
	return fmt.Sprintf("%s/%d@%d", *tp.Topic, tp.Partition, tp.Offset)
}

// newFakeConsumer: каждая партиция задаётся как "topic", partition, offsets...
func newFakeConsumer(cancel context.CancelFunc) *fakeConsumer {
	c := &fakeConsumer{paused: make(map[partitionKey]bool), cancel: cancel}
	c.idle = func() {
		// Всё прочитано и никто не ждёт повтора — тест закончен.
		for _, paused := range c.paused {
			if paused {
				time.Sleep(time.Millisecond)
				return
			}
		}
		cancel()
	}
	return c
}

func (c *fakeConsumer) addPartition(topic string, partition int32, offsets ...kafka.Offset) *fakeConsumer {
	p := &fakePartition{key: partitionKey{topic: topic, partition: partition}}
	for _, offset := range offsets {
		p.log = append(p.log, fakeMessage(topic, partition, offset))
	}
	c.parts = append(c.parts, p)
	return c
}

func fakeMessage(topic string, partition int32, offset kafka.Offset) *kafka.Message {
	tp := kafka.TopicPartition{Topic: &topic, Partition: partition, Offset: offset}
	return &kafka.Message{TopicPartition: tp, Value: []byte(msgID(tp))}
}

func (c *fakeConsumer) partition(tp kafka.TopicPartition) *fakePartition {
	for _, p := range c.parts {
		if p.key.topic == *tp.Topic && p.key.partition == tp.Partition {
			return p
		}
	}
	return nil
}

// maxFakePolls останавливает зациклившийся listener: тест падает на проверках
// с понятным диффом, а не по таймауту go test.
const maxFakePolls = 2000

func (c *fakeConsumer) Poll(int) kafka.Event {
	c.polls++
	if c.polls > maxFakePolls {
		c.cancel()
		return nil
	}
	if c.beforePoll != nil {
		c.beforePoll()
	}
	if len(c.injected) > 0 {
		ev := c.injected[0]
		c.injected = c.injected[1:]
		return ev
	}
	for i := range c.parts {
		idx := (c.next + i) % len(c.parts)
		p := c.parts[idx]
		if c.paused[p.key] || p.pos >= len(p.log) {
			continue
		}
		msg := p.log[p.pos]
		p.pos++
		c.next = (idx + 1) % len(c.parts)
		return msg
	}
	c.idle()
	return nil
}

func (c *fakeConsumer) CommitMessage(m *kafka.Message) ([]kafka.TopicPartition, error) {
	c.commits = append(c.commits, msgID(m.TopicPartition))
	if c.commitErr != nil {
		return c.commitErr(m)
	}
	return nil, nil
}

func (c *fakeConsumer) SeekPartitions(tps []kafka.TopicPartition) ([]kafka.TopicPartition, error) {
	for _, tp := range tps {
		c.seeks = append(c.seeks, msgID(tp))
		if c.seekErr != nil {
			return nil, c.seekErr
		}
		p := c.partition(tp)
		for i, m := range p.log {
			if m.TopicPartition.Offset == tp.Offset {
				p.pos = i
			}
		}
	}
	return tps, nil
}

func (c *fakeConsumer) Pause(tps []kafka.TopicPartition) error {
	if c.pauseErr != nil {
		return c.pauseErr
	}
	for _, tp := range tps {
		c.pauses = append(c.pauses, msgID(tp))
		c.paused[c.partition(tp).key] = true
	}
	return nil
}

func (c *fakeConsumer) Resume(tps []kafka.TopicPartition) error {
	for _, tp := range tps {
		c.resumes = append(c.resumes, msgID(tp))
		c.paused[c.partition(tp).key] = false
	}
	return nil
}

type fakeDLQ struct {
	sent   []string
	causes []error
	err    func() error
}

func (d *fakeDLQ) Send(_ context.Context, m *kafka.Message, cause error) error {
	if d.err != nil {
		if err := d.err(); err != nil {
			return err
		}
	}
	d.sent = append(d.sent, msgID(m.TopicPartition))
	d.causes = append(d.causes, cause)
	return nil
}

func listenerTestLog() *loggerpkg.Logger {
	return &loggerpkg.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// newTestListener: failures — сколько раз подряд handler вернёт ошибку для сообщения.
func newTestListener(c *fakeConsumer, dlq DeadLetterQueue, failures map[string][]error) *Listener {
	l := NewListener(c, dlq, listenerTestLog(), func(_ context.Context, value []byte) error {
		id := string(value)
		c.handled = append(c.handled, id)
		if errs := failures[id]; len(errs) > 0 {
			failures[id] = errs[1:]
			return errs[0]
		}
		return nil
	})
	l.InitialRetryDelay = 5 * time.Millisecond
	l.MaxRetryDelay = 20 * time.Millisecond
	return l
}

func TestListenerCommitsOnlyAfterHandlerSucceeds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel).addPartition("cdc", 2, 41)
	completed := false
	c.commitErr = func(*kafka.Message) ([]kafka.TopicPartition, error) {
		require.True(t, completed, "commit must wait for the handler result")
		return nil, nil
	}
	l := NewListener(c, &fakeDLQ{}, listenerTestLog(), func(context.Context, []byte) error {
		completed = true
		return nil
	})

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Equal(t, []string{"cdc/2@41"}, c.commits)
	require.Empty(t, c.seeks)
	require.Empty(t, c.pauses)
}

func TestListenerTransientErrorPausesOnlyFailedPartition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel).
		addPartition("cdc", 2, 41, 42).
		addPartition("cdc", 3, 41, 42)
	l := newTestListener(c, &fakeDLQ{}, map[string][]error{"cdc/2@41": {errors.New("DWH unavailable")}})

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Equal(t, []string{"cdc/2@41", "cdc/3@41", "cdc/3@42", "cdc/2@41", "cdc/2@42"}, c.handled,
		"other partition is processed while the failed one waits; 2@42 is never handled before 2@41 succeeds")
	require.Equal(t, []string{"cdc/3@41", "cdc/3@42", "cdc/2@41", "cdc/2@42"}, c.commits)
	require.Equal(t, []string{"cdc/2@41"}, c.seeks)
	require.Equal(t, []string{"cdc/2@41"}, c.pauses)
	require.Equal(t, []string{"cdc/2@41"}, c.resumes)
}

func TestListenerLongBackoffDoesNotBlockOtherPartitionsOrShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel).
		addPartition("cdc", 2, 41).
		addPartition("cdc", 3, 41, 42, 43)
	c.idle = cancel // партиция 2 на паузе на час — заканчиваем, как только читать больше нечего
	l := newTestListener(c, &fakeDLQ{}, map[string][]error{"cdc/2@41": {errors.New("DWH unavailable")}})
	l.InitialRetryDelay = time.Hour

	started := time.Now()
	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Less(t, time.Since(started), time.Second, "backoff must not sleep inside the poll loop")
	require.Equal(t, []string{"cdc/3@41", "cdc/3@42", "cdc/3@43"}, c.commits)
	require.Empty(t, c.resumes)
}

func TestListenerRetriesUntilSuccessWithGrowingBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel).addPartition("cdc", 2, 41)
	failure := errors.New("DWH unavailable")
	l := newTestListener(c, &fakeDLQ{}, map[string][]error{"cdc/2@41": {failure, failure, failure}})

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Equal(t, []string{"cdc/2@41", "cdc/2@41", "cdc/2@41", "cdc/2@41"}, c.handled)
	require.Equal(t, []string{"cdc/2@41"}, c.commits)
	require.Len(t, c.pauses, 3)
	require.Len(t, c.resumes, 3)
	require.Empty(t, l.retries, "gate is removed after success")
}

func TestListenerCommitFailureReplaysMessage(t *testing.T) {
	for _, partitionLevel := range []bool{false, true} {
		t.Run(fmt.Sprintf("partitionLevel=%v", partitionLevel), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := newFakeConsumer(cancel).addPartition("cdc", 2, 41, 42)
			commitErr := errors.New("commit failed")
			c.commitErr = func(m *kafka.Message) ([]kafka.TopicPartition, error) {
				if len(c.commits) > 1 {
					return nil, nil
				}
				if partitionLevel {
					tp := m.TopicPartition
					tp.Error = commitErr
					return []kafka.TopicPartition{tp}, nil
				}
				return nil, commitErr
			}
			l := newTestListener(c, &fakeDLQ{}, nil)

			require.ErrorIs(t, l.Run(ctx), context.Canceled)
			require.Equal(t, []string{"cdc/2@41", "cdc/2@41", "cdc/2@42"}, c.handled, "ambiguous commit replays the DWH work")
			require.Equal(t, []string{"cdc/2@41", "cdc/2@41", "cdc/2@42"}, c.commits)
			require.Equal(t, []string{"cdc/2@41"}, c.seeks)
		})
	}
}

func TestListenerPermanentErrorGoesToDLQAndCommits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel).addPartition("cdc", 2, 41, 42)
	cause := errors.New("view deleted")
	dlq := &fakeDLQ{}
	l := newTestListener(c, dlq, map[string][]error{"cdc/2@41": {Permanent(cause)}})

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Equal(t, []string{"cdc/2@41"}, dlq.sent)
	require.ErrorIs(t, dlq.causes[0], cause)
	require.Equal(t, []string{"cdc/2@41", "cdc/2@42"}, c.commits)
	require.Empty(t, c.seeks, "permanent error is not retried")
	require.Empty(t, c.pauses)
}

func TestListenerInvalidJSONGoesToDLQ(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel)
	c.injected = []kafka.Event{&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: ptr("cdc"), Partition: 2, Offset: 41},
		Value:          []byte(`{"invalid":`),
	}}
	dlq := &fakeDLQ{}
	log := listenerTestLog()
	l := NewListener(c, dlq, log, func(ctx context.Context, data []byte) error {
		return Dispatch(ctx, data, log, nil)
	})

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Equal(t, []string{"cdc/2@41"}, dlq.sent)
	var syntaxErr *json.SyntaxError
	require.ErrorAs(t, dlq.causes[0], &syntaxErr)
	require.Equal(t, []string{"cdc/2@41"}, c.commits)
}

func TestListenerDLQFailureIsRetriedWithoutCommit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel).addPartition("cdc", 2, 41)
	attempts := 0
	dlq := &fakeDLQ{err: func() error {
		attempts++
		if attempts == 1 {
			require.Empty(t, c.commits)
			return errors.New("broker unavailable")
		}
		return nil
	}}
	l := newTestListener(c, dlq, map[string][]error{"cdc/2@41": {Permanent(errors.New("bad")), Permanent(errors.New("bad"))}})

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Equal(t, 2, attempts)
	require.Equal(t, []string{"cdc/2@41"}, dlq.sent)
	require.Equal(t, []string{"cdc/2@41"}, c.commits)
	require.Equal(t, []string{"cdc/2@41"}, c.seeks)
}

func TestListenerCommitsWhenCanceledAfterSuccessfulHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel).addPartition("cdc", 2, 41, 42)
	l := NewListener(c, &fakeDLQ{}, listenerTestLog(), func(_ context.Context, value []byte) error {
		c.handled = append(c.handled, string(value))
		cancel() // сигнал остановки пришёл, пока шла запись в DWH, но запись завершилась
		return nil
	})

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Equal(t, []string{"cdc/2@41"}, c.handled)
	require.Equal(t, []string{"cdc/2@41"}, c.commits, "completed DWH write is committed to avoid a replay")
}

func TestListenerCancellationDuringHandlerDoesNotCommitOrRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel).addPartition("cdc", 2, 41)
	l := NewListener(c, &fakeDLQ{}, listenerTestLog(), func(ctx context.Context, _ []byte) error {
		cancel()
		return ctx.Err() // запись в DWH прервана отменой
	})

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Empty(t, c.commits)
	require.Empty(t, c.seeks)
	require.Empty(t, c.pauses)
}

func TestListenerSeekOrPauseFailureIsFatal(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*fakeConsumer, error)
	}{
		{"seek", func(c *fakeConsumer, err error) { c.seekErr = err }},
		{"pause", func(c *fakeConsumer, err error) { c.pauseErr = err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newFakeConsumer(func() {}).addPartition("cdc", 2, 41)
			kafkaErr := errors.New("partition is no longer assigned")
			tc.setup(c, kafkaErr)
			handlerErr := errors.New("DWH failed")
			l := newTestListener(c, &fakeDLQ{}, map[string][]error{"cdc/2@41": {handlerErr}})

			result := l.Start(context.Background())
			select {
			case err := <-result:
				require.ErrorIs(t, err, kafkaErr)
				require.ErrorIs(t, err, handlerErr)
			case <-time.After(2 * time.Second):
				t.Fatal("fatal listener failure was not reported")
			}
			_, open := <-result
			require.False(t, open, "Start must report exactly one result and close")
			require.Empty(t, c.commits)
		})
	}
}

func TestListenerKafkaErrors(t *testing.T) {
	t.Run("non-fatal error is logged and consumption continues", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		c := newFakeConsumer(cancel).addPartition("cdc", 2, 41)
		c.injected = []kafka.Event{kafka.NewError(kafka.ErrTransport, "broker down", false)}
		l := newTestListener(c, &fakeDLQ{}, nil)

		require.ErrorIs(t, l.Run(ctx), context.Canceled)
		require.Equal(t, []string{"cdc/2@41"}, c.commits)
	})
	t.Run("fatal error stops the listener", func(t *testing.T) {
		c := newFakeConsumer(func() {})
		c.injected = []kafka.Event{kafka.NewError(kafka.ErrFatal, "unrecoverable consumer failure", true)}
		l := newTestListener(c, &fakeDLQ{}, nil)

		var kafkaErr kafka.Error
		require.ErrorAs(t, l.Run(context.Background()), &kafkaErr)
		require.True(t, kafkaErr.IsFatal())
	})
}

func TestListenerRevokedPartitionDropsGate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel).addPartition("cdc", 2, 41, 42, 50)
	l := newTestListener(c, &fakeDLQ{}, map[string][]error{"cdc/2@41": {errors.New("DWH unavailable")}})
	l.InitialRetryDelay = time.Hour
	p := c.parts[0]
	c.beforePoll = func() {
		if len(c.pauses) == 1 && len(l.retries) == 1 {
			// Rebalance: партицию отобрали, другой экземпляр обработал 41..49 и
			// закоммитил 50; затем партицию вернули нам — чтение с 50, без паузы.
			l.PartitionsRevoked([]kafka.TopicPartition{{Topic: ptr("cdc"), Partition: 2}})
			c.paused[p.key] = false
			p.pos = 2
		}
	}

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Equal(t, []string{"cdc/2@41", "cdc/2@50"}, c.handled)
	require.Equal(t, []string{"cdc/2@50"}, c.commits)
	require.Equal(t, []string{"cdc/2@41"}, c.seeks, "no rewind to the stale gate position")
}

func TestListenerStaleMessageFromGatedPartitionIsReseeked(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newFakeConsumer(cancel).addPartition("cdc", 2, 41, 42)
	l := newTestListener(c, &fakeDLQ{}, map[string][]error{"cdc/2@41": {errors.New("DWH unavailable")}})
	c.beforePoll = func() {
		if len(c.pauses) == 1 && len(c.seeks) == 1 {
			// Сообщение, скачанное до паузы, всё же доставлено.
			c.injected = append(c.injected, c.parts[0].log[1])
		}
	}

	require.ErrorIs(t, l.Run(ctx), context.Canceled)
	require.Equal(t, []string{"cdc/2@41", "cdc/2@41", "cdc/2@42"}, c.handled, "stale 2@42 is not handled out of order")
	require.Equal(t, []string{"cdc/2@41", "cdc/2@42"}, c.commits)
	require.Equal(t, []string{"cdc/2@41", "cdc/2@41"}, c.seeks)
	require.Len(t, c.pauses, 1, "stale message does not add a retry attempt")
}

func TestListenerPollTimeoutIsBoundedByNextResume(t *testing.T) {
	l := NewListener(nil, nil, listenerTestLog(), nil)
	now := time.Now()
	require.Equal(t, 1000, l.pollTimeoutMs(now))

	l.retries[partitionKey{"cdc", 1}] = &partitionRetry{paused: true, resumeAt: now.Add(200 * time.Millisecond)}
	l.retries[partitionKey{"cdc", 2}] = &partitionRetry{paused: false, resumeAt: now}
	require.Equal(t, 200, l.pollTimeoutMs(now))

	l.retries[partitionKey{"cdc", 3}] = &partitionRetry{paused: true, resumeAt: now.Add(-time.Second)}
	require.Equal(t, 0, l.pollTimeoutMs(now))
}

func TestRetryDelayGrowsExponentiallyAndIsCapped(t *testing.T) {
	l := NewListener(nil, nil, listenerTestLog(), nil)
	delay := l.InitialRetryDelay
	for _, expected := range []time.Duration{
		200 * time.Millisecond, 400 * time.Millisecond, 800 * time.Millisecond,
		1600 * time.Millisecond, 3200 * time.Millisecond, 5 * time.Second, 5 * time.Second,
	} {
		delay = l.nextRetryDelay(delay)
		require.Equal(t, expected, delay)
	}
}

func ptr(s string) *string { return &s }
