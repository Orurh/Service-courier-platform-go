package kafka

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/service/orders"
	testlog "course-go-avito-Orurh/internal/testutil"
)

func TestNewConsumer_SkipsWhenNoKafkaConfig(t *testing.T) {
	t.Parallel()

	rec := testlog.New()

	got, err := NewConsumer(rec.Logger(), nil, "gid", "topic", func(context.Context, orders.Event) error { return nil })
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = NewConsumer(rec.Logger(), []string{"b:9092"}, "", "topic", nil)
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = NewConsumer(rec.Logger(), []string{"b:9092"}, "gid", "   ", nil)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestNewConsumer_ReturnsErrorWhenSaramaFails(t *testing.T) {
	orig := newConsumerGroup
	t.Cleanup(func() { newConsumerGroup = orig })

	sentinel := errors.New("boom")
	newConsumerGroup = func(_ []string, _ string, _ *sarama.Config) (sarama.ConsumerGroup, error) {
		return nil, sentinel
	}

	rec := testlog.New()
	got, err := NewConsumer(rec.Logger(), []string{"b:9092"}, "gid", "topic", nil)
	require.ErrorIs(t, err, sentinel)
	require.Nil(t, got)
}

type fakeGroup struct {
	mu sync.Mutex

	consumeFn func(context.Context, []string, sarama.ConsumerGroupHandler) error
	errCh     chan error
	closeFn   func() error

	closed bool
}

func (g *fakeGroup) Consume(ctx context.Context, topics []string, h sarama.ConsumerGroupHandler) error {
	if g.consumeFn == nil {
		return nil
	}
	return g.consumeFn(ctx, topics, h)
}

func (g *fakeGroup) Errors() <-chan error {
	if g.errCh == nil {
		g.errCh = make(chan error)
	}
	return g.errCh
}

func (g *fakeGroup) Close() error {
	g.mu.Lock()
	g.closed = true
	g.mu.Unlock()
	if g.closeFn != nil {
		return g.closeFn()
	}
	return nil
}

func (g *fakeGroup) Pause(map[string][]int32)  {}
func (g *fakeGroup) Resume(map[string][]int32) {}
func (g *fakeGroup) PauseAll()                 {}
func (g *fakeGroup) ResumeAll()                {}

func TestNewConsumer_Success_ReturnsConsumer(t *testing.T) {
	orig := newConsumerGroup
	t.Cleanup(func() { newConsumerGroup = orig })

	fg := &fakeGroup{}
	newConsumerGroup = func(_ []string, _ string, _ *sarama.Config) (sarama.ConsumerGroup, error) {
		return fg, nil
	}

	rec := testlog.New()
	got, err := NewConsumer(rec.Logger(), []string{"b:9092"}, "gid", "topic", func(context.Context, orders.Event) error { return nil })
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Same(t, fg, got.group)
	require.Equal(t, "topic", got.topic)
	require.NotNil(t, got.handler)
	require.NotNil(t, got.sleepFn)
}

func TestConsumer_Run_WhenContextCanceled_ReturnsCtxErr(t *testing.T) {
	rec := testlog.New()
	fg := &fakeGroup{errCh: make(chan error, 1)}

	c := &Consumer{
		group:   fg,
		topic:   "t",
		logger:  rec.Logger(),
		sleepFn: func(context.Context, time.Duration) error { return nil },
		handler: func(context.Context, orders.Event) error {
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestConsumer_ConsumeOnce_ErrorPath_UsesSleepOrDone(t *testing.T) {
	t.Parallel()

	rec := testlog.New()
	sentinel := errors.New("consume failed")

	fg := &fakeGroup{
		consumeFn: func(context.Context, []string, sarama.ConsumerGroupHandler) error {
			return sentinel
		},
		errCh: make(chan error, 1),
	}

	var gotSleep time.Duration
	c := &Consumer{
		group:  fg,
		topic:  "t",
		logger: rec.Logger(),
		sleepFn: func(_ context.Context, d time.Duration) error {
			gotSleep = d
			return nil
		},
		handler: func(context.Context, orders.Event) error {
			return nil
		},
	}

	err := c.consumeOnce(context.Background(), &groupHandler{c: c})
	require.NoError(t, err)
	require.Equal(t, time.Second, gotSleep)
	require.True(t, hasMsg(rec.Entries(), "kafka consume error"))
}

func TestConsumer_RunGroupErrorsLogger_Logs(t *testing.T) {
	t.Parallel()
	rec := testlog.New()
	fg := &fakeGroup{errCh: make(chan error, 1)}

	c := &Consumer{
		group:  fg,
		logger: rec.Logger(),
	}

	fg.errCh <- errors.New("kafka internal error")
	c.drainGroupErrors(context.Background())

	require.True(t, hasMsg(rec.Entries(), "kafka consumer group error"))
}

func TestConsumer_Close_NilReceiver_NoPanic(t *testing.T) {
	var c *Consumer
	err := c.Close()
	require.NoError(t, err)
}

func TestConsumer_Close_ClosesGroup(t *testing.T) {
	fg := &fakeGroup{}
	c := &Consumer{group: fg}
	err := c.Close()
	require.NoError(t, err)
	fg.mu.Lock()
	defer fg.mu.Unlock()
	require.True(t, fg.closed)
}
