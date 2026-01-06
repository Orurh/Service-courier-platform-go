package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/service/orders"
	testlog "course-go-avito-Orurh/internal/testutil"
)

type fakeGroup struct{}

func (fakeGroup) Consume(context.Context, []string, sarama.ConsumerGroupHandler) error { return nil }
func (fakeGroup) Errors() <-chan error {
	ch := make(chan error)
	close(ch)
	return ch
}
func (fakeGroup) Close() error { return nil }



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
	t.Parallel()

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


