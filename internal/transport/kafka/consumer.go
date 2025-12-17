package kafka

import (
	"context"
	"encoding/json"
	"log"
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
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(brokers []string, groupID, topic string, h HandleFunc) (*Consumer, error) {
	// не стратую если у кафки нет настроек
	if len(brokers) == 0 || strings.TrimSpace(topic) == "" || strings.TrimSpace(groupID) == "" {
		return nil, nil
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
	}, nil
}

// Run starts the consumer
func (c *Consumer) Run(ctx context.Context) error {
	if c == nil {
		return nil
	}

	h := &groupHandler{c: c}

	for {
		if err := c.group.Consume(ctx, []string{c.topic}, h); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("kafka: consume error: %v", err)
			time.Sleep(time.Second)
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
	for msg := range claim.Messages() {
		var ev orders.Event
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			log.Printf("kafka: bad json: %v", err)
			sess.MarkMessage(msg, "")
			continue
		}
		if strings.TrimSpace(ev.OrderID) == "" {
			log.Printf("kafka: empty order_id")
			sess.MarkMessage(msg, "")
			continue
		}

		if err := h.c.handler(sess.Context(), ev); err != nil {
			log.Printf("kafka: handle failed,retry: order_id=%s status=%s err=%v", ev.OrderID, ev.Status, err)
			return err
		}

		sess.MarkMessage(msg, "")
	}
	return nil
}
