package kafka

import (
	"strings"
	"time"

	"course-go-avito-Orurh/internal/service/orders"
)

// EventDTO is a data transfer object for orders.Event
type EventDTO struct {
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ToDomain converts EventDTO to orders.Event
func ToDomain(dto EventDTO) orders.Event {
	return orders.Event{
		OrderID:   strings.TrimSpace(dto.OrderID),
		Status:    strings.TrimSpace(dto.Status),
		CreatedAt: dto.CreatedAt,
	}
}
