package app

import (
	"context"
	"time"

	ordersgw "course-go-avito-Orurh/internal/gateway/orders"
	"course-go-avito-Orurh/internal/service/orders"
	"course-go-avito-Orurh/internal/transport/kafka"
)

func makeOrdersKafka(p *orders.Processor, gw *ordersgw.GRPCGateway) kafka.HandleFunc {
	return func(ctx context.Context, event orders.Event) error {
		if gw == nil {
			return p.Handle(ctx, event)
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
		return p.Handle(ctx, event)
	}
}

// func actionClass(s string) string {
// 	str := strings.ToLower(strings.TrimSpace(s))
// 	switch str {
// 	case "created", "pending", "confirmed", "cooking", "delivering":
// 		return "assign"
// 	case "canceled", "deleted":
// 		return "unassign"
// 	case "completed":
// 		return "complete"
// 	default:
// 		return "ignore"
// 	}
// }
