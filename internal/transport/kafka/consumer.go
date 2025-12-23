package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"course-go-avito-Orurh/internal/service/orders"

	"github.com/IBM/sarama"
)

// HandleFunc processes a single orders.Event from Kafka
type HandleFunc func(context.Context, orders.Event) error

// Consumer wraps a Sarama consumer group and dispatches events to a handler
type Consumer struct {
	group   sarama.ConsumerGroup
	topic   string
	handler HandleFunc
	logger  *slog.Logger
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(logger *slog.Logger, brokers []string, groupID, topic string, h HandleFunc) (*Consumer, error) {
	// не стратую если у кафки нет настроек
	if len(brokers) == 0 || strings.TrimSpace(topic) == "" || strings.TrimSpace(groupID) == "" {
		return nil, nil
	}
	if logger == nil {
		panic("kafka: loger is nil")
	}

	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	group, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		group:   group,
		topic:   topic,
		handler: h,
		logger:  logger,
	}, nil
}

// Run starts the consumer
func (c *Consumer) Run(ctx context.Context) error {
	if c == nil {
		return nil
	}
	// я подумал логировать ошибки кафки в отдельной горутине...
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-c.group.Errors():
				if !ok {
					return
				}
				c.logger.Error("kafka consumer group error", slog.Any("err", err))
			}
		}
	}()

	h := &groupHandler{c: c}

	for {
		if err := c.group.Consume(ctx, []string{c.topic}, h); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.Error("kafka consume error", slog.Any("err", err))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
			}
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func (c *Consumer) Close() error {
	if c == nil {
		return nil
	}
	return c.group.Close()
}

type groupHandler struct{ c *Consumer }

func (h *groupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *groupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *groupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-sess.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			var dto EventDTO
			if err := json.Unmarshal(msg.Value, &dto); err != nil {
				h.c.logger.Warn("kafka bad json", slog.Any("err", err))
				sess.MarkMessage(msg, "")
				continue
			}

			ev := ToDomain(dto)

			if ev.OrderID == "" {
				h.c.logger.Warn("kafka empty order_id")
				sess.MarkMessage(msg, "")
				continue
			}

			if err := h.c.handler(sess.Context(), ev); err != nil {
				h.c.logger.Error("kafka handle failed, skipping message",
					slog.String("order_id", ev.OrderID),
					slog.String("status", ev.Status),
					slog.Any("err", err),
				)
				sess.MarkMessage(msg, "")
				continue
			}
			sess.MarkMessage(msg, "")
		}
	}
}
