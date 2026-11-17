package serviceanalytics

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/services/cdc"
	"analyticDataCenter/analytics-data-center/internal/storage"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/require"
)

type cdcUpsert struct {
	view string
	row  map[string]interface{}
	keys []string
}

type cdcDWH struct {
	*mockDWH
	upsert func(context.Context, cdcUpsert) error
}

func (d *cdcDWH) InsertOrUpdateTransactional(ctx context.Context, view string, row map[string]interface{}, keys []string) error {
	return d.upsert(ctx, cdcUpsert{view: view, row: row, keys: keys})
}

// newCDCService: таблица db1.public.users входит в две вьюхи — users_view (id 1) и users_copy (id 2).
func newCDCService(upsert func(context.Context, cdcUpsert) error) (*AnalyticsDataCenterService, *mockSchemaProvider) {
	columns := []models.Column{{Name: "id", Type: "integer", IsUpdateKey: true}, {Name: "name", Type: "text"}}
	schema := &mockSchemaProvider{schems: []int{1, 2}, views: make(map[int]models.View)}
	dwh := &cdcDWH{mockDWH: &mockDWH{columns: make(map[string][]string)}, upsert: upsert}
	for index, name := range []string{"users_view", "users_copy"} {
		schema.views[index+1] = models.View{
			Name: name,
			Sources: []models.Source{{Name: "db1", Schemas: []models.Schema{{
				Name: "public", Tables: []models.Table{{Name: "users", Columns: columns}},
			}}}},
		}
		dwh.columns[name] = []string{"id", "name"}
	}
	return &AnalyticsDataCenterService{
		log:                     getTestLogger(),
		SchemaProvider:          schema,
		RenameSuggestionStorage: schema,
		ColumnMismatchStorage:   schema,
		DWHProvider:             dwh,
		OLTPFactory:             &mockFactory{store: map[string]storage.OLTPDB{"db1": &mockOLTP{columns: columns}}},
		DWHDbName:               DbPostgres,
	}, schema
}

func cdcTestEvent(op string) models.CDCEvent {
	return models.CDCEvent{Data: models.CDCEventData{
		Op:     op,
		Source: models.CDCSource{DB: "db1", Schema: "public", Table: "users"},
		After:  map[string]interface{}{"id": float64(7), "name": "Alice"},
	}}
}

func receiveCDCResult[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for CDC goroutine")
		var zero T
		return zero
	}
}

// cdcConsumer — одна партиция с одним сообщением. Подменяет только Kafka:
// Listener, Dispatch, EventPreprocessing и запись во вьюхи — настоящие.
// Доступ к полям синхронизирован: тесты читают их из своей горутины.
type cdcConsumer struct {
	mu       sync.Mutex
	ctx      context.Context
	message  *kafka.Message
	pending  bool // сообщение ещё не отдано на текущей позиции
	paused   bool
	commits  []*kafka.Message
	seeks    int
	onCommit func()
}

func newCDCConsumer(ctx context.Context, message *kafka.Message) *cdcConsumer {
	return &cdcConsumer{ctx: ctx, message: message, pending: true}
}

func (c *cdcConsumer) Poll(timeoutMs int) kafka.Event {
	c.mu.Lock()
	if c.pending && !c.paused {
		c.pending = false
		c.mu.Unlock()
		return c.message
	}
	c.mu.Unlock()
	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-c.ctx.Done():
	case <-timer.C:
	}
	return nil
}

func (c *cdcConsumer) CommitMessage(message *kafka.Message) ([]kafka.TopicPartition, error) {
	c.mu.Lock()
	c.commits = append(c.commits, message)
	onCommit := c.onCommit
	c.mu.Unlock()
	if onCommit != nil {
		onCommit()
	}
	return nil, nil
}

func (c *cdcConsumer) SeekPartitions(tps []kafka.TopicPartition) ([]kafka.TopicPartition, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seeks++
	c.pending = true // позиция возвращена на сообщение — оно будет прочитано снова
	return tps, nil
}

func (c *cdcConsumer) Pause([]kafka.TopicPartition) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
	return nil
}

func (c *cdcConsumer) Resume([]kafka.TopicPartition) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = false
	return nil
}

func (c *cdcConsumer) snapshot() (commits []*kafka.Message, seeks int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*kafka.Message(nil), c.commits...), c.seeks
}

type recordingDLQ struct {
	mu     sync.Mutex
	causes []error
}

func (d *recordingDLQ) Send(_ context.Context, _ *kafka.Message, cause error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.causes = append(d.causes, cause)
	return nil
}

func cdcKafkaMessage(t *testing.T, op string) *kafka.Message {
	t.Helper()
	payload, err := json.Marshal(cdcTestEvent(op).Data)
	require.NoError(t, err)
	topic := "db1.public.users"
	return &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 0, Offset: 41},
		Value:          payload,
	}
}

func newServiceListener(consumer cdc.KafkaConsumer, dlq cdc.DeadLetterQueue, service *AnalyticsDataCenterService) *cdc.Listener {
	l := cdc.NewListener(consumer, dlq, getTestLogger(), func(ctx context.Context, data []byte) error {
		return cdc.Dispatch(ctx, data, getTestLogger(), service)
	})
	l.InitialRetryDelay = 5 * time.Millisecond
	return l
}

func TestCDCListenerWaitsForEveryViewBeforeCommit(t *testing.T) {
	for _, op := range []string{"c", "u"} {
		t.Run(op, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			started := make(chan cdcUpsert, 2)
			release := make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			service, _ := newCDCService(func(ctx context.Context, call cdcUpsert) error {
				started <- call
				if call.view == "users_copy" {
					select {
					case <-release:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				return nil
			})
			message := cdcKafkaMessage(t, op)
			consumer := newCDCConsumer(ctx, message)
			committed := make(chan struct{}, 1)
			consumer.onCommit = func() {
				committed <- struct{}{}
				cancel()
			}
			done := make(chan error, 1)
			go func() { done <- newServiceListener(consumer, &recordingDLQ{}, service).Run(ctx) }()
			t.Cleanup(func() {
				unblock()
				cancel()
				receiveCDCResult(t, done)
			})

			for _, view := range []string{"users_view", "users_copy"} {
				call := receiveCDCResult(t, started)
				require.Equal(t, view, call.view)
				require.Equal(t, cdcTestEvent(op).Data.After, call.row)
				require.Equal(t, []string{"id"}, call.keys)
			}
			select {
			case <-committed:
				t.Fatal("offset committed before the last DWH view completed")
			case <-time.After(25 * time.Millisecond):
			}

			unblock()
			receiveCDCResult(t, committed)
			commits, seeks := consumer.snapshot()
			require.Equal(t, []*kafka.Message{message}, commits)
			require.Zero(t, seeks)
		})
	}
}

func TestCDCListenerReplaysUpsertsAfterPartialDWHFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dwhErr := errors.New("second view temporarily unavailable")
	var mu sync.Mutex
	var calls []cdcUpsert
	service, _ := newCDCService(func(_ context.Context, call cdcUpsert) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, call)
		if len(calls) == 2 {
			return dwhErr
		}
		return nil
	})
	message := cdcKafkaMessage(t, "c")
	consumer := newCDCConsumer(ctx, message)
	consumer.onCommit = cancel

	require.ErrorIs(t, newServiceListener(consumer, &recordingDLQ{}, service).Run(ctx), context.Canceled)
	commits, seeks := consumer.snapshot()
	require.Equal(t, []*kafka.Message{message}, commits, "committed once, only after the replay succeeded")
	require.Equal(t, 1, seeks)
	require.Len(t, calls, 4)
	require.Equal(t, []string{"users_view", "users_copy", "users_view", "users_copy"},
		[]string{calls[0].view, calls[1].view, calls[2].view, calls[3].view})
	require.Equal(t, calls[0], calls[2], "already applied view is safely retried with the same upsert keys and row")
	require.Equal(t, calls[1], calls[3])
}

func TestCDCListenerSendsEventForDeletedViewToDLQ(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	service, schema := newCDCService(func(context.Context, cdcUpsert) error {
		t.Error("no view must be written when one of them is missing")
		return nil
	})
	delete(schema.views, 2) // вьюха удалена, а связь таблицы с ней осталась
	message := cdcKafkaMessage(t, "u")
	consumer := newCDCConsumer(ctx, message)
	consumer.onCommit = cancel
	dlq := &recordingDLQ{}

	require.ErrorIs(t, newServiceListener(consumer, dlq, service).Run(ctx), context.Canceled)
	require.Len(t, dlq.causes, 1)
	require.ErrorIs(t, dlq.causes[0], ErrInvalidSchemID)
	commits, seeks := consumer.snapshot()
	require.Equal(t, []*kafka.Message{message}, commits)
	require.Zero(t, seeks, "permanent error is not retried")
}

func TestCDCListenerShutdownAbortsDWHWriteWithoutCommit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writing := make(chan struct{})
	service, _ := newCDCService(func(ctx context.Context, _ cdcUpsert) error {
		close(writing)
		<-ctx.Done() // запись «висит», пока её не прервёт остановка
		return ctx.Err()
	})
	consumer := newCDCConsumer(ctx, cdcKafkaMessage(t, "c"))
	done := make(chan error, 1)
	go func() { done <- newServiceListener(consumer, &recordingDLQ{}, service).Run(ctx) }()

	receiveCDCResult(t, writing)
	cancel()
	require.ErrorIs(t, receiveCDCResult(t, done), context.Canceled)
	commits, seeks := consumer.snapshot()
	require.Empty(t, commits)
	require.Zero(t, seeks)
}

func TestEventPreprocessingPreservesDeleteAndSnapshotNoop(t *testing.T) {
	// Nil storage providers make any newly introduced DWH work fail this test.
	service := &AnalyticsDataCenterService{log: getTestLogger()}
	for _, op := range []string{"d", "r"} {
		t.Run(op, func(t *testing.T) {
			require.NoError(t, service.EventPreprocessing(context.Background(), cdcTestEvent(op)))
		})
	}
}
