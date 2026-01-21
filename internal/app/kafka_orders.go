package app

import (
	"context"
	"time"

	ordersgw "course-go-avito-Orurh/internal/gateway/orders"
	"course-go-avito-Orurh/internal/service/orders"
	"course-go-avito-Orurh/internal/transport/kafka"
)

type ordersGateway interface {
	GetByID(ctx context.Context, id string) (*ordersgw.Order, error)
}

type ordersHandler interface {
	Handle(context.Context, orders.Event) error
}

func makeOrdersKafka(h ordersHandler, gw ordersGateway) kafka.HandleFunc {
	return func(ctx context.Context, event orders.Event) error {
		if gw == nil {
			return h.Handle(ctx, event)
		}

		gwCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		ord, err := gw.GetByID(gwCtx, event.OrderID)
		if err != nil {
			return err
		}

		if ord == nil {
			return nil
		}

		event.Status = ord.Status
		event.CreatedAt = ord.CreatedAt
		return h.Handle(ctx, event)
	}
}
