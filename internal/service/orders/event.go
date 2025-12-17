package orders

import (
	"time"
)

// Event is a single order event
type Event struct {
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
