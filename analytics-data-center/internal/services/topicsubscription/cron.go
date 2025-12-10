package topicsubscription

import (
	"context"
	"log/slog"
	"sync"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
)

const (
	defaultInterval = 5 * time.Second
)

// TopicLister retrieves topics that should be subscribed in Kafka.
type TopicLister interface {
	ListTopics(ctx context.Context) ([]string, error)
}

// TopicNotifier enqueues topic subscriptions for Kafka consumers.
type TopicNotifier interface {
	EnqueueTopic(topic string)
}

// Cron periodically refreshes Kafka topic subscriptions.
type Cron struct {
	log      *loggerpkg.Logger
	lister   TopicLister
	notifier TopicNotifier
	interval time.Duration

	stop     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once
}

// NewCron constructs a new Cron job for topic subscription refreshes.
func NewCron(log *loggerpkg.Logger, lister TopicLister, notifier TopicNotifier, interval time.Duration) *Cron {
	if interval <= 0 {
		interval = defaultInterval
	}

	return &Cron{
		log:      log,
		lister:   lister,
		notifier: notifier,
		interval: interval,
		stop:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}
}

// Start launches the cron job.
func (c *Cron) Start() {
	go c.run()
}

// Stop gracefully stops the cron job.
func (c *Cron) Stop() {
	c.stopOnce.Do(func() {
		close(c.stop)
		<-c.stopped
	})
}

func (c *Cron) run() {
	defer close(c.stopped)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.refresh()

	for {
		select {
		case <-ticker.C:
			c.refresh()
		case <-c.stop:
			return
		}
	}
}

func (c *Cron) refresh() {
	if c.lister == nil || c.notifier == nil {
		c.log.Warn("topic subscription cron is not fully configured")
		return
	}

	ctx := context.Background()
	topics, err := c.lister.ListTopics(ctx)
	if err != nil {
		c.log.Warn("failed to list topics for subscription", slog.String("error", err.Error()))
		return
	}

	if len(topics) == 0 {
		c.log.Warn("no topics found for subscription refresh")
		return
	}

	for _, topic := range topics {
		c.notifier.EnqueueTopic(topic)
	}
}
